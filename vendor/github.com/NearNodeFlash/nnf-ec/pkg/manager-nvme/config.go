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
