package server

import (
	_ "embed"

	"gopkg.in/yaml.v2"
)

//go:embed config.yaml
var configFile []byte

type ConfigFile struct {
	Version  string
	Metadata struct {
		Name string
	}

	FileSystems []FileSystemConfig `yaml:"fileSystems"` // yaml sucks, and doesnt understand camelCase
}

type FileSystemConfig struct {
	Name       string
	Install    []FileSystemInstallConfig
	Operations []FileSystemOperationConfig
}

type FileSystemInstallConfig struct {
	Dist    string
	Command string
	Gpg     string
}

type FileSystemOperationConfig struct {
	Label     string
	Command   string
	Arguments []string
	Options   []map[string]string
}

func loadConfig() (*ConfigFile, error) {

	var config = new(ConfigFile)
	if err := yaml.UnmarshalStrict(configFile, config); err != nil {
		return config, err
	}

	return config, nil
}
