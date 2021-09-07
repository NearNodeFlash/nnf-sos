package messageregistry

import (
	_ "embed"
	"encoding/json"
	"fmt"

	"stash.us.cray.com/rabsw/nnf-ec/pkg/ec"
	sf "stash.us.cray.com/rabsw/rfsf-openapi/pkg/models"
)

//go:embed registry-files/Base.1.10.0.json
var BaseMessageFile []byte

//go:embed registry-files/ResourceEvent.1.0.3.json
var ResourceEventMessageFile []byte

var MessageRegistryManager = manager{}

type manager struct {
	files []file
}

type file struct {
	id   string
	data *[]byte
}

func (f *file) OdataId() string { return fmt.Sprintf("/redfish/v1/Registries/%s", f.id) }

func (m *manager) Initialize() error {
	files := []*[]byte{
		&BaseMessageFile,
		&ResourceEventMessageFile,
	}

	m.files = make([]file, len(files))
	for idx, data := range files {
		model := sf.MessageRegistryFileV113MessageRegistryFile{}

		if err := json.Unmarshal(*data, &model); err != nil {
			return err
		}

		m.files[idx] = file{
			id:   model.Id,
			data: data,
		}
	}

	return nil
}

func (m *manager) Get(model *sf.MessageRegistryFileCollectionMessageRegistryFileCollection) error {

	model.MembersodataCount = int64(len(m.files))
	model.Members = make([]sf.OdataV4IdRef, model.MembersodataCount)
	for idx, f := range m.files {
		model.Members[idx] = sf.OdataV4IdRef{OdataId: f.OdataId()}
	}

	return nil
}

func (m *manager) RegistryIdGet(id string, model *sf.MessageRegistryFileV113MessageRegistryFile) error {
	for _, file := range m.files {
		if file.id == id {
			if err := json.Unmarshal(*file.data, &model); err != nil {
				return ec.NewErrInternalServerError().WithError(err).WithCause(fmt.Sprintf("Failed to unmarshal registry file %s", id))
			}

			model.OdataId = file.OdataId()

			return nil
		}
	}

	return ec.NewErrNotFound().WithCause(fmt.Sprintf("Registry file %s not found", id))
}
