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
	"fmt"

	server "github.com/NearNodeFlash/nnf-ec/pkg/manager-server"

	sf "github.com/NearNodeFlash/nnf-ec/pkg/rfsf/pkg/models"
)

type Endpoint struct {
	id           string
	name         string
	controllerId uint16
	state        sf.ResourceState

	fabricId string

	// This is the Server Controller used for managing the endpoint
	serverCtrl server.ServerControllerApi

	config         *ServerConfig
	storageService *StorageService
}

func (ep *Endpoint) OdataId() string {
	return fmt.Sprintf("%s/Endpoints/%s", ep.storageService.OdataId(), ep.id)
}
