package controller

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"node-label-controller/config"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
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
	configs  *config.LinuxContainerController
	client   *k8s.Clientset
	indexer  cache.Indexer
	queue    workqueue.RateLimitingInterface
	informer cache.Controller
	stop     chan struct{}
	errors   chan error
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

	c := &LinuxContainerController{
		client:  client,
		configs: configs,
		errors:  make(chan error),
		stop:    make(chan struct{}),
	}

	nodeListWatcher := cache.NewListWatchFromClient(client.CoreV1().RESTClient(), "nodes", metav1.NamespaceAll, fields.Everything())
	if nodeListWatcher == nil {
		return nil, errors.New("created watcher is nil")
	}

	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	c.queue = queue

	indexer, informer := cache.NewIndexerInformer(nodeListWatcher, &v1.Node{}, 0, cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(key)
			}
		},
		UpdateFunc: func(old interface{}, new interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(new)
			if err == nil {
				queue.Add(key)
			}
		},
		DeleteFunc: func(obj interface{}) {
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(key)
			}
		},
	}, cache.Indexers{})

	c.indexer = indexer
	c.informer = informer

	c.warmUp()

	return c, nil
}

func (lc *LinuxContainerController) Stop() {
	lc.queue.ShutDown()

	lc.stop <- struct{}{}

}

func (lc *LinuxContainerController) Run() {
	defer runtime.HandleCrash()

	klog.Info("running node-labeling controller")

	go lc.informer.Run(lc.stop)

	if !cache.WaitForCacheSync(lc.stop, lc.informer.HasSynced) {
		runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
		return
	}

	for i := 0; i < lc.configs.WorkersNumber; i++ {
		go wait.Until(lc.runWorker, time.Second, lc.stop)
	}

	<-lc.stop
	klog.Info("stopping node-labeling controller")
}

func (lc *LinuxContainerController) runWorker() {
	for lc.processNextItem() {
	}
}

func (lc *LinuxContainerController) Name() string {
	return lc.configs.Name
}

func (lc *LinuxContainerController) Errors() <-chan error {
	return lc.errors
}

func (lc *LinuxContainerController) processNextItem() bool {
	key, quit := lc.queue.Get()
	if quit {
		return false
	}

	defer lc.queue.Done(key)

	err := lc.nodeLabelingSync(key.(string))

	lc.handleErr(err, key)
	return true
}

// handleErr checks if an error happened and makes sure we will retry later.
func (lc *LinuxContainerController) handleErr(err error, key interface{}) {
	if err == nil {
		lc.queue.Forget(key)
		return
	}

	if lc.queue.NumRequeues(key) < lc.configs.MaxRetries {
		klog.Infof("error while labeling the node %v: %v", key, err)

		lc.queue.AddRateLimited(key)
		return
	}

	lc.queue.Forget(key)
	runtime.HandleError(err)
	klog.Infof("dropping node %q out of the queue: %v", key, err)
}

func (lc *LinuxContainerController) nodeLabelingSync(key string) error {
	obj, ok, err := lc.indexer.GetByKey(key)
	if err != nil {
		klog.Errorf("fetching object with key %s from store failed with %v", key, err)
		return err
	}

	if !ok {
		fmt.Printf("node %s does not exist anymore\n", key)
	} else {
		node := obj.(*v1.Node)
		if node != nil && strings.Contains(node.Status.NodeInfo.OSImage, containerImage) {
			labels := obj.(*v1.Node).GetLabels()
			if labels == nil {
				labels = make(map[string]string)
			}

			labels[linuxContainerLabel] = linuxContainerValue
			node.SetLabels(labels)
		}
	}
	return nil
}

func (lc *LinuxContainerController) warmUp() {
	nodes, err := lc.client.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		lc.errors <- err
	}

	for _, node := range nodes.Items {
		if err := lc.indexer.Add(&node); err != nil {
			lc.errors <- err
		}
	}
}
