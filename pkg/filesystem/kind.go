/*
 * Copyright 2023-2024 Hewlett Packard Enterprise Development LP
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
	"path/filepath"

	"github.com/NearNodeFlash/nnf-sos/pkg/blockdevice"
	"github.com/go-logr/logr"
)

type KindFileSystem struct {
	Log  logr.Logger
	Path string

	BlockDevice blockdevice.BlockDevice
}

// Check that LustreFileSystem implements the FileSystem interface
var _ FileSystem = &KindFileSystem{}

func (m *KindFileSystem) Create(ctx context.Context, complete bool) (bool, error) {
	if complete {
		return false, nil
	}

	if err := os.MkdirAll(m.Path, 0755); err != nil {
		return false, fmt.Errorf("could not create mount directory %s: %w", m.Path, err)
	}

	m.Log.Info("Created mock file system in kind", "path", m.Path)
	return true, nil
}

func (m *KindFileSystem) Destroy(ctx context.Context) (bool, error) {
	// Remove the directory. If it fails don't worry about it.
	_ = os.RemoveAll(m.Path)

	m.Log.Info("Destroyed mock file system in kind")
	return true, nil
}

func (m *KindFileSystem) Activate(ctx context.Context, complete bool) (bool, error) {
	if complete {
		return false, nil
	}

	m.Log.Info("Activated mock file system in kind")
	return true, nil
}

func (m *KindFileSystem) Deactivate(ctx context.Context) (bool, error) {
	m.Log.Info("Deactivated mock file system in kind")
	return true, nil
}

func (m *KindFileSystem) Mount(ctx context.Context, path string, complete bool) (bool, error) {
	if complete {
		return false, nil
	}

	bn := filepath.Dir(path)
	if err := os.MkdirAll(bn, 0755); err != nil {
		return false, fmt.Errorf("could not create directory for symlink %s: %w", bn, err)
	}

	if err := os.Symlink(m.Path, path); err != nil {
		return false, fmt.Errorf("could not create symlink mount %s: %w", path, err)
	}

	m.Log.Info("Mounted mock file system in kind", "filesystem", m.Path, "mount", path)
	return true, nil
}

func (m *KindFileSystem) Unmount(ctx context.Context, path string) (bool, error) {
	// Remove the directory. If it fails don't worry about it.
	_ = os.Remove(path)

	m.Log.Info("Unmounted mock file system in kind")
	return true, nil
}

func (m *KindFileSystem) PostActivate(ctx context.Context, complete bool) (bool, error) {
	if complete {
		return false, nil
	}

	m.Log.Info("Ran PostActivate")

	return true, nil
}

func (m *KindFileSystem) PreDeactivate(ctx context.Context, complete bool) (bool, error) {
	if complete {
		return false, nil
	}

	m.Log.Info("Ran PreDeactivate")

	return true, nil
}

func (m *KindFileSystem) PreMount(ctx context.Context, complete bool) (bool, error) {
	if complete {
		return false, nil
	}

	m.Log.Info("Ran PreMount")

	return true, nil
}

func (m *KindFileSystem) PostMount(ctx context.Context, path string, complete bool) (bool, error) {
	if complete {
		return false, nil
	}

	m.Log.Info("Ran PostMount")

	return true, nil
}

func (m *KindFileSystem) PreUnmount(ctx context.Context, path string, complete bool) (bool, error) {
	if complete {
		return false, nil
	}

	m.Log.Info("Ran PreUnmount")

	return true, nil
}

func (m *KindFileSystem) PostUnmount(ctx context.Context, complete bool) (bool, error) {
	if complete {
		return false, nil
	}

	m.Log.Info("Ran PostUnmount")

	return true, nil
}

func (m *KindFileSystem) PostSetup(ctx context.Context, complete bool) (bool, error) {
	if complete {
		return false, nil
	}

	m.Log.Info("Ran PostMount")

	return true, nil
}

func (m *KindFileSystem) PreTeardown(ctx context.Context, complete bool) (bool, error) {
	if complete {
		return false, nil
	}

	m.Log.Info("Ran PreUnmount")

	return true, nil
}
