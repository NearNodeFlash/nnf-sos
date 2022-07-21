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

var ErrServerControllerDisabled = fmt.Errorf("Server Controller Disabled")

// Server Controller API for when there is no server present, or the server is disabled
type DisabledServerController struct{}

func NewDisabledServerController() ServerControllerApi {
	return &DisabledServerController{}
}

func (*DisabledServerController) Connected() bool {
	return false
}

func (*DisabledServerController) GetServerInfo() ServerInfo {
	return ServerInfo{}
}

func (*DisabledServerController) NewStorage(uuid.UUID, []StorageNamespace) *Storage {
	return nil
}

func (*DisabledServerController) Delete(*Storage) error {
	return ErrServerControllerDisabled
}

func (*DisabledServerController) GetStatus(*Storage) (StorageStatus, error) {
	return StorageStatus_Disabled, nil
}

func (*DisabledServerController) CreateFileSystem(*Storage, FileSystemApi, FileSystemOptions) error {
	return ErrServerControllerDisabled
}

func (*DisabledServerController) MountFileSystem(*Storage, FileSystemApi, FileSystemOptions) error {
	return ErrServerControllerDisabled
}

func (*DisabledServerController) UnmountFileSystem(*Storage, FileSystemApi) error {
	return ErrServerControllerDisabled
}

func (*DisabledServerController) DeleteFileSystem(*Storage, FileSystemApi) error {
	return ErrServerControllerDisabled
}
