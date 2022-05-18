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
	server "github.com/NearNodeFlash/nnf-ec/pkg/manager-server"
)

type NnfControllerInterface interface {
	ServerControllerProvider() server.ServerControllerProvider
	PersistentControllerProvider() PersistentControllerApi
}

type NnfController struct {
	serverControllerProvider server.ServerControllerProvider
	persistentController     PersistentControllerApi
}

func NewNnfController(persistence bool) NnfControllerInterface {
	return &NnfController{
		server.DefaultServerControllerProvider{},
		getPersistentControllerApi(persistence),
	}
}

func (c *NnfController) ServerControllerProvider() server.ServerControllerProvider {
	return c.serverControllerProvider
}

func (c *NnfController) PersistentControllerProvider() PersistentControllerApi {
	return c.persistentController
}

// TODO: See if mock makes sense here - really the lower-layers should provide the mocking
// of physical devices; NNF Controller might be fine building off that.
func NewMockNnfController(persistence bool) NnfControllerInterface {
	return &NnfController{
		serverControllerProvider: server.MockServerControllerProvider{},
		persistentController:     getPersistentControllerApi(persistence),
	}
}

func getPersistentControllerApi(persistence bool) PersistentControllerApi {
	if persistence {
		return NewDefaultPersistentController()
	}
	return NewMockPersistentController()
}
