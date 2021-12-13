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
