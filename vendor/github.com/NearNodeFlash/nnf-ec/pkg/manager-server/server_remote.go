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
	"fmt"

	"github.com/google/uuid"
)

const (
	RemoteStorageServiceId   = "NNFServer"
	RemoteStorageServicePort = 60050
)

type StoragePoolOem struct {
	Namespaces []StorageNamespace `json:"Namespaces"`
}

// Remote Server Controller defines a Server Controller API for a connected server
// that is not local to the running process. This contains many "do nothing" methods
// as management of storage, file systems, and related content is done "off-rabbit".
type RemoteServerController struct{}

func NewRemoteServerController(opts ServerControllerOptions) ServerControllerApi {
	return &RemoteServerController{}
}

func (c *RemoteServerController) Connected() bool { return true }

func (c *RemoteServerController) GetServerInfo() ServerInfo {
	return ServerInfo{}
}

func (c *RemoteServerController) NewStorage(pid uuid.UUID, expectedNamespaces []StorageNamespace) *Storage {
	return &Storage{
		Id:   pid,
		ctrl: c,
	}
}

func (c *RemoteServerController) Delete(s *Storage) error {
	return nil
}

func (c *RemoteServerController) GetStatus(s *Storage) (StorageStatus, error) {
	return StorageStatus_Ready, nil
}

func (c *RemoteServerController) CreateFileSystem(*Storage, FileSystemApi, FileSystemOptions) error {
	return fmt.Errorf("Cannot create file system on remote server")
}

func (r *RemoteServerController) MountFileSystem(*Storage, FileSystemApi, string) error {
	return nil
}

func (r *RemoteServerController) UnmountFileSystem(*Storage, FileSystemApi, string) error {
	return nil
}

func (r *RemoteServerController) DeleteFileSystem(*Storage, FileSystemApi) error {
	return nil
}
