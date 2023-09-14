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
	"path/filepath"

	"github.com/NearNodeFlash/nnf-sos/pkg/blockdevice"
	"github.com/NearNodeFlash/nnf-sos/pkg/command"
	"github.com/NearNodeFlash/nnf-sos/pkg/var_handler"
	"github.com/go-logr/logr"

	mount "k8s.io/mount-utils"
)

type SimpleFileSystemCommandArgs struct {
	Mkfs  string
	Mount string

	Vars map[string]string
}

type SimpleFileSystem struct {
	Log         logr.Logger
	CommandArgs SimpleFileSystemCommandArgs

	Type        string
	MountTarget string
	TempDir     string

	BlockDevice blockdevice.BlockDevice
}

// Check that SimpleFileSystem implements the FileSystem interface
var _ FileSystem = &SimpleFileSystem{}

func (f *SimpleFileSystem) parseArgs(args string) string {
	m := map[string]string{
		"$DEVICE": f.BlockDevice.GetDevice(),
	}

	for k, v := range f.CommandArgs.Vars {
		m[k] = v
	}

	// Initialize the VarHandler substitution variables
	varHandler := var_handler.NewVarHandler(m)
	return varHandler.ReplaceAll(args)
}

func (f *SimpleFileSystem) Create(ctx context.Context, complete bool) (bool, error) {
	if complete == true {
		return false, nil
	}

	if f.Type == "none" {
		return false, nil
	}

	if _, err := f.BlockDevice.Activate(ctx); err != nil {
		return false, fmt.Errorf("could not activate block device for mounting: %w", err)
	}

	// If the device is already formatted, don't run the mkfs again
	if f.BlockDevice.CheckFormatted() {
		if _, err := f.BlockDevice.Deactivate(ctx); err != nil {
			return false, fmt.Errorf("could not deactivate block device after format shows completed: %w", err)
		}

		return false, nil
	}

	if _, err := command.Run(fmt.Sprintf("mkfs -t %s %s", f.Type, f.parseArgs(f.CommandArgs.Mkfs))); err != nil {
		if err != nil {
			return false, fmt.Errorf("could not create file system: %w", err)
		}
	}

	if _, err := f.BlockDevice.Deactivate(ctx); err != nil {
		return false, fmt.Errorf("could not deactivate block device after mkfs: %w", err)
	}

	return true, nil
}

func (f *SimpleFileSystem) Destroy(ctx context.Context) (bool, error) {
	return false, nil
}

func (f *SimpleFileSystem) Activate(ctx context.Context, complete bool) (bool, error) {
	return false, nil
}

func (f *SimpleFileSystem) Deactivate(ctx context.Context) (bool, error) {
	return false, nil
}

func (f *SimpleFileSystem) Mount(ctx context.Context, path string, options string, complete bool) (bool, error) {
	path = filepath.Clean(path)
	mounter := mount.New("")
	mounts, err := mounter.List()
	if err != nil {
		return false, err
	}

	for _, m := range mounts {
		if m.Path != path {
			continue
		}

		if f.Type == "none" {
			return false, nil
		}

		// Found an existing mount at this path. Check if it's the mount we expect
		if m.Device != f.BlockDevice.GetDevice() || m.Type != f.Type {
			return false, fmt.Errorf("unexpected mount at path %s. Device %s type %s", path, m.Device, m.Type)
		}

		// The file system is already mounted. Nothing left to do
		return false, nil
	}

	// Create the mount file or directory
	switch f.MountTarget {
	case "directory":
		if err := os.MkdirAll(path, 0755); err != nil {
			return false, fmt.Errorf("could not create mount directory %s: %w", path, err)
		}
	case "file":
		// Create the parent directory and then the file
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return false, fmt.Errorf("could not create parent mount directory %s: %w", filepath.Dir(path), err)
		}

		if err := os.WriteFile(path, []byte(""), 0644); err != nil {
			return false, fmt.Errorf("could not create mount file %s: %w", path, err)
		}
	}

	if _, err := f.BlockDevice.Activate(ctx); err != nil {
		return false, fmt.Errorf("could not activate block device for mounting %s: %w", path, err)
	}

	// Run the mount command
	if len(options) == 0 {
		options = f.CommandArgs.Mount
	}
	mountCmd := fmt.Sprintf("mount -t %s %s %s", f.Type, f.BlockDevice.GetDevice(), path)
	if len(options) > 0 {
		mountCmd = mountCmd + " -o " + f.parseArgs(options)
	}

	if _, err := command.Run(mountCmd); err != nil {
		if _, err := f.BlockDevice.Deactivate(ctx); err != nil {
			return false, fmt.Errorf("could not deactivate block device after failed mount %s: %w", path, err)
		}

		return false, fmt.Errorf("could not mount file system %s: %w", path, err)
	}

	return true, nil
}

func (f *SimpleFileSystem) Unmount(ctx context.Context, path string) (bool, error) {
	path = filepath.Clean(path)
	mounter := mount.New("")
	mounts, err := mounter.List()
	if err != nil {
		return false, err
	}

	for _, m := range mounts {
		if m.Path != path {
			continue
		}

		// Found an existing mount at this path. If it's not a bind mount, check if it's the mount device we expect
		if f.Type != "none" && (m.Device != f.BlockDevice.GetDevice() || m.Type != f.Type) {
			return false, fmt.Errorf("unexpected mount at path %s. Device %s type %s", path, m.Device, m.Type)
		}

		if _, err := command.Run(fmt.Sprintf("umount %s", path)); err != nil {
			return false, fmt.Errorf("could not unmount file system %s: %w", path, err)
		}

		// Remove the file/directory. If it fails don't worry about it.
		_ = os.Remove(path)

		if _, err := f.BlockDevice.Deactivate(ctx); err != nil {
			return false, fmt.Errorf("could not deactivate block device after unmount %s: %w", path, err)
		}

		return true, nil
	}

	// Try to deactivate the block device in case the deactivate failed after the unmount above
	if _, err := f.BlockDevice.Deactivate(ctx); err != nil {
		return false, fmt.Errorf("could not deactivate block device after unmount %s: %w", path, err)
	}

	// file system already unmounted
	return false, nil
}

func (f *SimpleFileSystem) SetPermissions(ctx context.Context, userID uint32, groupID uint32, complete bool) (bool, error) {
	if complete {
		return false, nil
	}

	if f.Type == "none" {
		return false, nil
	}

	if _, err := f.Mount(ctx, f.TempDir, "", false); err != nil {
		return false, fmt.Errorf("could not mount temp dir '%s' to set permissions: %w", f.TempDir, err)
	}

	if err := os.Chown(f.TempDir, int(userID), int(groupID)); err != nil {
		if _, unmountErr := f.Unmount(ctx, f.TempDir); unmountErr != nil {
			return false, fmt.Errorf("could not unmount after setting owner permissions failed '%s': %w", f.TempDir, unmountErr)
		}
		return false, fmt.Errorf("could not set owner permissions '%s': %w", f.TempDir, err)
	}

	if _, err := f.Unmount(ctx, f.TempDir); err != nil {
		return false, fmt.Errorf("could not unmount after setting owner permissions '%s': %w", f.TempDir, err)
	}

	return false, nil
}
