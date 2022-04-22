/*
 * Copyright 2020, 2021, 2022 Hewlett Packard Enterprise Development LP
 * Other additional copyright holders may be indicated within.
 *
 * The entirety of this work is licensed under the Apache License,
 * Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License.
 *
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

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
