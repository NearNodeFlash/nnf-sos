package messageregistry

import (
	_ "embed"
	"encoding/json"
	"fmt"

	sf "stash.us.cray.com/rabsw/nnf-ec/pkg/rfsf/pkg/models"
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

type Registry struct {
	PublicationUri string

	Data *[]byte

	Model sf.MessageRegistryV142MessageRegistry
}

var RegistryFiles = []Registry{
	{"https://redfish.dmtf.org/registries/Base.1.10.0.json", &BaseMessageRegistry, sf.MessageRegistryV142MessageRegistry{}},
	{"https://redfish.dmtf.org/registries/ResourceEvent.1.0.3.json", &ResourceEventMessageRegistry, sf.MessageRegistryV142MessageRegistry{}},
	{"https://redfish.dmtf.org/registries/Fabric.1.0.0.json", &FabricMessageRegistry, sf.MessageRegistryV142MessageRegistry{}},
}

func (r *Registry) initialize() error {
	return json.Unmarshal(*r.Data, &r.Model)
}

func (r *Registry) Id() string      { return r.Model.Id }
func (r *Registry) OdataId() string { return fmt.Sprintf("/redfish/v1/Registries/%s", r.Model.Id) }
