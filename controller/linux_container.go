package controller

import (
	"errors"
	"strings"
	"time"

	"node-label-controller/config"

	v12 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/klog"
)

const (
	defaultWorkersNumber = 1
	defaultMaxRetries    = 5
	linuxContainerLabel  = "kubermatic.io/uses-container-linux"
	linuxContainerValue  = "true"
	containerImage       = "Container Linux"
)

// LinuxContainerController labels the the node objects in case of node operating system is ContainerLinux.
type LinuxContainerController struct {
	configs *config.LinuxContainerController
	client  *k8s.Clientset
	watcher watch.Interface
	stop    chan struct{}
	errors  chan error
}

// NewLinuxContainerControllerFromClientSet creates and configure a new linux container controller based on the sent client set.
func NewLinuxContainerControllerFromClientSet(configs *config.LinuxContainerController, client *k8s.Clientset) (Controller, error) {
	if configs.Name == "" {
		return nil, errors.New("controller name cannot be empty")
	}

	if configs.WorkersNumber == 0 {
		configs.WorkersNumber = defaultWorkersNumber
	}

	if configs.MaxRetries == 0 {
		configs.MaxRetries = defaultMaxRetries
	}

	watcher, err := client.CoreV1().Nodes().Watch(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	return &LinuxContainerController{
		configs: configs,
		client:  client,
		watcher: watcher,
		stop:    make(chan struct{}),
		errors:  make(chan error),
	}, nil
}

func (lc *LinuxContainerController) Stop() {
	lc.stop <- struct{}{}
}

func (lc *LinuxContainerController) Run() {
	defer runtime.HandleCrash()

	klog.Info("running node-labeling controller")

	for i := 0; i < lc.configs.WorkersNumber; i++ {
		if lc.configs.Watcher {
			klog.Info("using node labeler watcher")
			go lc.nodeLabelingWatcher()
		} else {
			klog.Info("using node labeler syncing")
			go wait.Until(lc.nodeLabelingSync, time.Second, lc.stop)
		}
	}

	<-lc.stop
	klog.Info("stopping node-labeling controller")
}

func (lc *LinuxContainerController) Name() string {
	return lc.configs.Name
}

func (lc *LinuxContainerController) Errors() <-chan error {
	return lc.errors
}

func (lc *LinuxContainerController) nodeLabelingSync() {
	nodes, err := lc.client.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		lc.errors <- err
		return
	}

	for _, node := range nodes.Items {
		lc.nodeLabeling(&node)
	}
}

func (lc *LinuxContainerController) nodeLabelingWatcher() {
	for event := range lc.watcher.ResultChan() {
		switch obj := event.Object.(type) {
		case *v12.Node:
			lc.nodeLabeling(obj)
		}
	}
}

func (lc *LinuxContainerController) nodeLabeling(obj *v12.Node) {
	if strings.Contains(obj.Status.NodeInfo.OSImage, containerImage) {
		labels := obj.GetLabels()
		if labels == nil {
			labels = make(map[string]string)
		}

		if value := labels[linuxContainerLabel]; value != "" {
			return
		}

		labels[linuxContainerLabel] = linuxContainerValue
		obj.SetLabels(labels)
		updatedNode, err := lc.client.CoreV1().Nodes().Update(obj)
		if err != nil {
			lc.errors <- err
			return
		}

		klog.Infof("node %v has been updated", updatedNode.Name)
	}
}
