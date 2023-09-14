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
	"fmt"
	"os"

	"github.com/NearNodeFlash/nnf-sos/pkg/blockdevice"
	"github.com/go-logr/logr"
)

type KindFileSystem struct {
	Log  logr.Logger
	Path string

	BlockDevice blockdevice.BlockDevice
}

// Check that LustreFileSystem implements the FileSystem interface
var _ FileSystem = &MockFileSystem{}

func (m *KindFileSystem) Create(ctx context.Context, complete bool) (bool, error) {
	if complete == true {
		return false, nil
	}

	if err := os.MkdirAll(m.Path, 0755); err != nil {
		return false, fmt.Errorf("could not create mount directory %s: %w", m.Path, err)
	}

	m.Log.Info("Created mock file system", "path", m.Path)
	return true, nil
}

func (m *KindFileSystem) Destroy(ctx context.Context) (bool, error) {
	// Remove the directory. If it fails don't worry about it.
	_ = os.RemoveAll(m.Path)

	m.Log.Info("Destroyed mock file system")
	return true, nil
}

func (m *KindFileSystem) Activate(ctx context.Context, complete bool) (bool, error) {
	if complete == true {
		return false, nil
	}

	m.Log.Info("Activated mock file system")
	return true, nil
}

func (m *KindFileSystem) Deactivate(ctx context.Context) (bool, error) {
	m.Log.Info("Deactivated mock file system")
	return true, nil
}

func (m *KindFileSystem) Mount(ctx context.Context, path string, options string, complete bool) (bool, error) {
	if complete == true {
		return false, nil
	}

	if err := os.Symlink(m.Path, path); err != nil {
		return false, fmt.Errorf("could not create symlink mount %s: %w", path, err)
	}

	m.Log.Info("Mounted mock file system", "filesystem", m.Path, "mount", path)
	return true, nil
}

func (m *KindFileSystem) Unmount(ctx context.Context, path string) (bool, error) {
	// Remove the directory. If it fails don't worry about it.
	_ = os.Remove(path)

	m.Log.Info("Unmounted mock file system")
	return true, nil
}

func (m *KindFileSystem) SetPermissions(ctx context.Context, uid uint32, gid uint32, complete bool) (bool, error) {
	m.Log.Info("Set mock file system permissions")

	return false, nil
}
