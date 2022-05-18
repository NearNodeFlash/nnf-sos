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

package nnf

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

	Id string

	DebugLevel string `yaml:"debugLevel"`

	AllocationConfig AllocationConfig `yaml:"allocationConfig"`
	RemoteConfig     RemoteConfig     `yaml:"remoteConfig"`
}

type AllocationConfig struct {
	// This is the default allocation policy for the NNF controller. An allocation policy
	// defines the way in which underlying storage is allocated when a user requests storage from
	// the NNF Controller. Valid values are "spares", "global", "switch-local", or "compute-local"
	// with the default being "spares". For more information see allocation_policy.go
	Policy string `yaml:"policy,omitempty"`

	// The Standard defines the level at which the allocation policy should function.
	// Valid values are "strict" or "relaxed", with the default being "strict". See allocation_policy.go
	Standard string `yaml:"standard,omitempty"`
}

type RemoteConfig struct {
	AccessMode string         `yaml:"accessMode"`
	Servers    []ServerConfig `yaml:"servers"`
}

type ServerConfig struct {
	Label   string `yaml:"label"`
	Address string `yaml:"address"`
}

func loadConfig() (*ConfigFile, error) {
	var config = new(ConfigFile)
	if err := yaml.Unmarshal(configFile, config); err != nil {
		return config, err
	}

	return config, nil
}
