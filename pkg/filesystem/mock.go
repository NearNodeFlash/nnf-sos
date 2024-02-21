/*
 * Copyright 2023 Hewlett Packard Enterprise Development LP
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

package filesystem

import (
	"context"

	"github.com/NearNodeFlash/nnf-sos/pkg/blockdevice"
	"github.com/go-logr/logr"
)

type MockFileSystem struct {
	Log  logr.Logger
	Path string

	BlockDevice blockdevice.BlockDevice
}

// Check that LustreFileSystem implements the FileSystem interface
var _ FileSystem = &MockFileSystem{}

func (m *MockFileSystem) Create(ctx context.Context, complete bool) (bool, error) {
	if complete == true {
		return false, nil
	}

	m.Log.Info("Created mock file system", "path", m.Path)
	return true, nil
}

func (m *MockFileSystem) Destroy(ctx context.Context) (bool, error) {
	m.Log.Info("Destroyed mock file system")

	return true, nil
}

func (m *MockFileSystem) Activate(ctx context.Context, complete bool) (bool, error) {
	if complete == true {
		return false, nil
	}

	m.Log.Info("Activated mock file system")
	return true, nil
}

func (m *MockFileSystem) Deactivate(ctx context.Context) (bool, error) {
	m.Log.Info("Deactivated mock file system")

	return true, nil
}

func (m *MockFileSystem) Mount(ctx context.Context, path string, complete bool) (bool, error) {
	if complete == true {
		return false, nil
	}

	m.Log.Info("Mounted mock file system", "filesystem", m.Path, "mount", path)
	return true, nil
}

func (m *MockFileSystem) Unmount(ctx context.Context, path string) (bool, error) {
	m.Log.Info("Unmounted mock file system")

	return true, nil
}

func (m *MockFileSystem) SetPermissions(ctx context.Context, uid uint32, gid uint32, complete bool) (bool, error) {
	m.Log.Info("Set mock file system permissions")

	return true, nil
}
