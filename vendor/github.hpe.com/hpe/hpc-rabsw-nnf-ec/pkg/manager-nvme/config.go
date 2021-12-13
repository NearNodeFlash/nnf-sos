package nvme

import (
	_ "embed"

	"gopkg.in/yaml.v2"
)

//go:embed config.yaml
var configFile []byte

type ControllerConfig struct {
	Functions uint32
	Resources uint32
}

type StorageConfig struct {
	Controller ControllerConfig
	Devices    []string `yaml:",flow"`
}

type ConfigFile struct {
	Version  string
	Metadata struct {
		Name string
	}
	DebugLevel string `yaml:"debugLevel"`
	Storage    StorageConfig
}

func loadConfig() (*ConfigFile, error) {
	var config = new(ConfigFile)
	if err := yaml.Unmarshal(configFile, config); err != nil {
		return config, err
	}

	return config, nil
}
