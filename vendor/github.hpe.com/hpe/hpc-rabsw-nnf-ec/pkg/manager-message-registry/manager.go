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
	"fmt"

	ec "github.hpe.com/hpe/hpc-rabsw-nnf-ec/pkg/ec"
	sf "github.hpe.com/hpe/hpc-rabsw-nnf-ec/pkg/rfsf/pkg/models"
)

var MessageRegistryManager = manager{}

type manager struct {
	registries []Registry
}

func (m *manager) findRegistry(id string) *Registry {
	for idx, reg := range m.registries {
		if reg.Id() == id {
			return &m.registries[idx]
		}
	}

	return nil
}

func (m *manager) Initialize() error {
	m.registries = RegistryFiles

	for idx := range m.registries {
		if err := m.registries[idx].initialize(); err != nil {
			return err
		}
	}

	return nil
}

// Get - Provides the collection of registry files maintained by the Message Registry Manager
func (m *manager) Get(model *sf.MessageRegistryFileCollectionMessageRegistryFileCollection) error {

	model.MembersodataCount = int64(len(m.registries))
	model.Members = make([]sf.OdataV4IdRef, model.MembersodataCount)
	for idx, reg := range m.registries {
		model.Members[idx] = sf.OdataV4IdRef{OdataId: reg.OdataId()}
	}

	return nil
}

// RegistryIdGet - Provides the registry file identified by the id maintained by the Message Registry Manager
func (m *manager) RegistryIdGet(id string, model *sf.MessageRegistryFileV113MessageRegistryFile) error {
	r := m.findRegistry(id)
	if r == nil {
		return ec.NewErrNotFound().WithCause(fmt.Sprintf("registry id %s not found", id))
	}

	model.Id = r.Id()
	model.Description = r.Model.Description
	model.Registry = r.Model.RegistryPrefix + r.Model.RegistryVersion

	model.Location = []sf.MessageRegistryFileV113Location{
		{
			Language:       r.Model.Language,
			PublicationUri: r.PublicationUri,
			Uri:            fmt.Sprintf("%s/Registry", r.OdataId()),
		},
	}

	return nil
}
