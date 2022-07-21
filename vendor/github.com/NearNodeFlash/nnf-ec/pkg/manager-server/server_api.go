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

package server

import (
	"github.com/google/uuid"
)

// Server Controller Provider defines the interface for providing Server
// Controllers to the NNF Manager.
//
// A Server Controller is responsible for receiving commands from the NNF
// Manager and dictating to the underlying server hardware the necessary
// actions to fullfil the NNF Manager request.
//
// The Server Controller supports three modes of operation.
//   1) Local Server Controller - This is expected to run locally to the NNF
//      manager. The necessary hardware and software components are expected
//      to be readily accessable to the local server controller.
//      See server_local.go
//
//   2) Remote Server Controller - This controller is running apart from the
//      NNF Manager, with access to this controller through some non-local
//      means like TCP/IP, Non-Transparent Bridge, Remote DMA, etc.
//      See server_remote.go
//
//   3) Mock Server Controller - This controller is used in a mock environment
//      to be used for simulating server behavior.
//      See server_mock.go

type ServerControllerProvider interface {
	NewServerController(ServerControllerOptions) ServerControllerApi
}

type ServerControllerOptions struct {
	Local   bool   // Set to true if the server controller is local to the running program
	Address string // IP Address of the Server
}

type ServerInfo struct {
	LNetNids []string
}

// Server Controller API defines the interface for interacting with and controlling
// a Server in the Rabbit NNF topology. That is - A remote initiator endpoint or
// the local NNF server.
type ServerControllerApi interface {
	Connected() bool

	GetServerInfo() ServerInfo

	// Allocate a new Storage object to be managed by this controller
	NewStorage(uuid.UUID, []StorageNamespace) *Storage

	Delete(*Storage) error

	// Retrieve the status of the provided Storage object from the controller
	GetStatus(*Storage) (StorageStatus, error)

	CreateFileSystem(*Storage, FileSystemApi, FileSystemOptions) error
	DeleteFileSystem(*Storage, FileSystemApi) error
	MountFileSystem(*Storage, FileSystemApi, FileSystemOptions) error
	UnmountFileSystem(*Storage, FileSystemApi) error
}
