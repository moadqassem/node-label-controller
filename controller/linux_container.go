package controller

import (
	"errors"
	"strings"
	"time"

	"node-label-controller/config"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/typed/core/v1"
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
	configs       *config.LinuxContainerController
	nodeInterface v1.NodeInterface
	stop          chan struct{}
	errors        chan error
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

	return &LinuxContainerController{
		configs:       configs,
		nodeInterface: client.CoreV1().Nodes(),
		stop:          make(chan struct{}),
		errors:        make(chan error),
	}, nil
}

func (lc *LinuxContainerController) Stop() {
	lc.stop <- struct{}{}
}

func (lc *LinuxContainerController) Run() {
	defer runtime.HandleCrash()

	klog.Info("running node-labeling controller")

	for i := 0; i < lc.configs.WorkersNumber; i++ {
		go wait.Until(lc.nodeLabelingSync, time.Second, lc.stop)
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
	nodes, err := lc.nodeInterface.List(metav1.ListOptions{})
	if err != nil {
		lc.errors <- err
		return
	}

	for _, node := range nodes.Items {
		if strings.Contains(node.Status.NodeInfo.OSImage, containerImage) {
			labels := node.GetLabels()
			if labels == nil {
				labels = make(map[string]string)
			}

			if value := labels[linuxContainerLabel]; value != "" {
				continue
			}

			labels[linuxContainerLabel] = linuxContainerValue
			node.SetLabels(labels)
			updatedNode, err := lc.nodeInterface.Update(&node)
			if err != nil {
				lc.errors <- err
				continue
			}

			klog.Infof("node %v has been updated", updatedNode.Name)
		}
	}
}
