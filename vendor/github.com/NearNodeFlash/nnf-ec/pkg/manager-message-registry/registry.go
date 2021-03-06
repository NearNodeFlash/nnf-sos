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

package messageregistry

import (
	_ "embed"
	"encoding/json"
	"fmt"

	sf "github.com/NearNodeFlash/nnf-ec/pkg/rfsf/pkg/models"
)

//go:generate ./generator/msgenerator ./registries/Base.1.10.0.json
//go:embed registries/Base.1.10.0.json
var BaseMessageRegistry []byte

//go:generate ./generator/msgenerator ./registries/ResourceEvent.1.0.3.json
//go:embed registries/ResourceEvent.1.0.3.json
var ResourceEventMessageRegistry []byte

//go:generate ./generator/msgenerator ./registries/Fabric.1.0.0.json
//go:embed registries/Fabric.1.0.0.json
var FabricMessageRegistry []byte

//go:generate ./generator/msgenerator ./registries/Nnf.1.0.0.json
//go:embed registries/Nnf.1.0.0.json
var NnfMessageRegistry []byte

type Registry struct {
	PublicationUri string

	Data *[]byte

	Model sf.MessageRegistryV142MessageRegistry
}

var RegistryFiles = []Registry{
	{"https://redfish.dmtf.org/registries/Base.1.10.0.json", &BaseMessageRegistry, sf.MessageRegistryV142MessageRegistry{}},
	{"https://redfish.dmtf.org/registries/ResourceEvent.1.0.3.json", &ResourceEventMessageRegistry, sf.MessageRegistryV142MessageRegistry{}},
	{"https://redfish.dmtf.org/registries/Fabric.1.0.0.json", &FabricMessageRegistry, sf.MessageRegistryV142MessageRegistry{}},
	{"", &NnfMessageRegistry, sf.MessageRegistryV142MessageRegistry{}},
}

func (r *Registry) initialize() error {
	return json.Unmarshal(*r.Data, &r.Model)
}

func (r *Registry) Id() string      { return r.Model.Id }
func (r *Registry) OdataId() string { return fmt.Sprintf("/redfish/v1/Registries/%s", r.Model.Id) }
