package controller

import (
	"errors"
	"sync"

	"node-label-controller/config"

	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// Controller manages all the interactions with of the custom controllers.
type Controller interface {
	Stop()
	Run()
	Name() string
}

type engine struct {
	controllers sync.Map
	kubeConfig  string
	configs     *config.Config
}

// NewEngine creates and load the controllers engine.
func NewEngine(configs *config.Config) (*engine, error) {
	if configs == nil {
		return nil, errors.New("engine configs cannot be nil")
	}

	clientConfig, err := clientcmd.BuildConfigFromFlags("", configs.KubeConfigPath)
	if err != nil {
		return nil, err
	}

	clientSet, err := k8s.NewForConfig(clientConfig)
	if err != nil {
		return nil, err
	}

	lcController, err := NewLinuxContainerControllerFromClientSet(configs.LinuxContainerController, clientSet)
	if err != nil {
		return nil, err
	}

	var sm sync.Map

	sm.Store(lcController.Name(), lcController)

	e := &engine{
		kubeConfig:  configs.KubeConfigPath,
		controllers: sm,
		configs:     configs,
	}

	return e, nil
}

// Start starts the controller engine based on the sent configs.
func (e *engine) Start() {
	e.controllers.Range(
		func(key, value interface{}) bool {
			go value.(*LinuxContainerController).Run()
			return true
		})
}

// Stop stops the controller engine and shutdown the controllers gracefully.
func (e *engine) Stop() {
	e.controllers.Range(
		func(key, value interface{}) bool {
			go value.(*LinuxContainerController).Stop()
			return true
		})
}
