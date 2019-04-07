package config

import (
	"encoding/json"
	"io/ioutil"
)

// Config contains all the configurations for the controllers engine.
type Config struct {
	KubeConfigPath           string                    `json:"kube_config_path"`
	LinuxContainerController *LinuxContainerController `json:"linux_container_controller"`
}

// LinuxContainerController contains the configurations of the linux container controller.
type LinuxContainerController struct {
	WorkersNumber int    `json:"workers_number"`
	Name          string `json:"name"`
	Resource      string `json:"resource"`
	MaxRetries    int    `json:"max_retries"`
}

// LoadConfig load the configurations from the config.json file into the Config struct.
func LoadConfig(path string) (*Config, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	configs := &Config{}
	err = json.Unmarshal(data, &configs)
	if err != nil {
		return nil, err
	}

	return configs, nil
}
