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
	"fmt"
	"net/http"

	. "github.com/NearNodeFlash/nnf-ec/pkg/common"
	sf "github.com/NearNodeFlash/nnf-ec/pkg/rfsf/pkg/models"
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
