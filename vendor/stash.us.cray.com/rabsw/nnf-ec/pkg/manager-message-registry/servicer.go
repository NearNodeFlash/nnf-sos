package messageregistry

import (
	"fmt"
	"net/http"

	. "stash.us.cray.com/rabsw/nnf-ec/pkg/common"
	sf "stash.us.cray.com/rabsw/nnf-ec/pkg/rfsf/pkg/models"
)

type DefaultApiService struct{}

func NewDefaultApiService() Api {
	return &DefaultApiService{}
}

func (*DefaultApiService) RedfishV1RegistriesGet(w http.ResponseWriter, r *http.Request) {

	model := sf.MessageRegistryFileCollectionMessageRegistryFileCollection{
		OdataId:   "/redfish/v1/Registries",
		OdataType: "#MessageRegistryFileCollection.v1_0_0.MessageRegistryFileCollection",
		Name:      "Message Registry File Collection",
	}

	err := MessageRegistryManager.Get(&model)

	EncodeResponse(model, err, w)
}

func (*DefaultApiService) RedfishV1RegistriesRegistryIdGet(w http.ResponseWriter, r *http.Request) {
	params := Params(r)
	registryId := params["RegistryId"]

	model := sf.MessageRegistryFileV113MessageRegistryFile{
		OdataId:   fmt.Sprintf("/redfish/v1/Registries/%s", registryId),
		OdataType: "#MessageRegistryFile.v1_1_3.MessageRegistryFile",
		Name:      "Message Registry File",
	}

	err := MessageRegistryManager.RegistryIdGet(registryId, &model)

	EncodeResponse(model, err, w)
}

func (*DefaultApiService) RedfishV1RegistriesRegistryIdRegistryGet(w http.ResponseWriter, r *http.Request) {
	EncodeResponse(nil, nil, w)
}
