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

import "github.com/google/uuid"

type MockServerControllerProvider struct{}

func (MockServerControllerProvider) NewServerController(ServerControllerOptions) ServerControllerApi {
	return NewMockServerController()
}

func NewMockServerController() ServerControllerApi {
	return &MockServerController{}
}

type MockServerController struct {
	storage []*Storage
}

func (c *MockServerController) Connected() bool { return true }

func (m *MockServerController) GetServerInfo() ServerInfo {
	return ServerInfo{
		LNetNids: []string{"1.2.3.4@tcp0"},
	}
}

func (c *MockServerController) NewStorage(pid uuid.UUID, expectedNamespaces []StorageNamespace) *Storage {
	storage := &Storage{
		Id:                 pid,
		expectedNamespaces: expectedNamespaces,
		ctrl:               c,
	}
	c.storage = append(c.storage, storage)
	return storage
}

func (c *MockServerController) Delete(s *Storage) error {
	for storageIdx, storage := range c.storage {
		if storage.Id == s.Id {
			c.storage = append(c.storage[:storageIdx], c.storage[storageIdx+1:]...)
			break
		}
	}

	return nil
}

func (*MockServerController) GetStatus(s *Storage) (StorageStatus, error) {
	return StorageStatus_Ready, nil
}

func (*MockServerController) CreateFileSystem(s *Storage, fs FileSystemApi, opts FileSystemOptions) error {
	s.fileSystem = fs

	if fs.IsMockable() {
		if err := fs.Create(s.Devices(), opts); err != nil {
			return err
		}
	}

	return nil
}

func (*MockServerController) MountFileSystem(s *Storage, opts FileSystemOptions) error {
	if s.fileSystem.IsMockable() {
		mountPoint := opts["mountpoint"].(string)
		if mountPoint == "" {
			return nil
		}

		if err := s.fileSystem.Mount(mountPoint); err != nil {
			return err
		}
	}

	return nil
}

func (*MockServerController) UnmountFileSystem(s *Storage) error {
	if s.fileSystem.IsMockable() {
		if err := s.fileSystem.Unmount(); err != nil {
			return err
		}
	}

	return nil
}

func (*MockServerController) DeleteFileSystem(s *Storage) error {
	if s.fileSystem.IsMockable() {
		return s.fileSystem.Delete()
	}
	return nil
}
