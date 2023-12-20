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

type LustreFileSystemCommandArgs struct {
	Mkfs        string
	MountTarget string
	Mount       string

	Vars map[string]string
}

type LustreFileSystem struct {
	Log         logr.Logger
	CommandArgs LustreFileSystemCommandArgs

	Name       string
	TargetType string
	MgsAddress string
	Index      int
	BackFs     string

	BlockDevice blockdevice.BlockDevice
}

// Check that LustreFileSystem implements the FileSystem interface
var _ FileSystem = &LustreFileSystem{}

func (l *LustreFileSystem) parseArgs(args string) string {
	m := map[string]string{
		"$DEVICE":    l.BlockDevice.GetDevice(),
		"$ZVOL_NAME": l.BlockDevice.GetDevice(),
		"$MGS_NID":   l.MgsAddress,
		"$INDEX":     fmt.Sprintf("%d", l.Index),
		"$FS_NAME":   l.Name,
		"$BACKFS":    l.BackFs,
	}

	for k, v := range l.CommandArgs.Vars {
		m[k] = v
	}

	// Initialize the VarHandler substitution variables
	varHandler := var_handler.NewVarHandler(m)
	return varHandler.ReplaceAll(args)
}

func (l *LustreFileSystem) Create(ctx context.Context, complete bool) (bool, error) {
	if complete == true {
		return false, nil
	}

	// If the device is already formatted, don't run the mkfs again
	formatted, err := l.BlockDevice.CheckFormatted()
	if err != nil {
		return false, fmt.Errorf("could not determine if device is formatted: %w", err)
	}
	if formatted {
		return false, nil
	}

	if _, err := command.Run(fmt.Sprintf("mkfs -t lustre %s", l.parseArgs(l.CommandArgs.Mkfs)), l.Log); err != nil {
		if err != nil {
			return false, fmt.Errorf("could not create file system: %w", err)
		}
	}
	return true, nil
}

func (l *LustreFileSystem) Destroy(ctx context.Context) (bool, error) {

	return false, nil
}

func (l *LustreFileSystem) Activate(ctx context.Context, complete bool) (bool, error) {
	mounter := mount.New("")
	mounts, err := mounter.List()
	if err != nil {
		return false, err
	}

	path := filepath.Clean(fmt.Sprintf("/mnt/nnf/lustre/%s.%s.%d", l.Name, l.TargetType, l.Index))
	for _, m := range mounts {
		if m.Path != path {
			continue
		}

		// Found an existing mount at this path. Check if it's the mount we expect
		if m.Device != l.BlockDevice.GetDevice() || m.Type != "lustre" {
			return false, fmt.Errorf("unexpected mount at path %s. Device %s type %s", path, m.Device, m.Type)
		}

		// The Lustre target is already mounted. Nothing left to do
		return false, nil
	}

	// Create the mount directory
	if err := os.MkdirAll(path, 0755); err != nil {
		return false, fmt.Errorf("could not create mount directory %s: %w", path, err)
	}

	if _, err := l.BlockDevice.Activate(ctx); err != nil {
		return false, fmt.Errorf("could not activate block device for mounting %s: %w", path, err)
	}

	// Build the mount command from the args provided
	l.CommandArgs.Vars["$MOUNT_PATH"] = path
	mountCmd := fmt.Sprintf("mount -t lustre %s", l.parseArgs(l.CommandArgs.MountTarget))

	// Run the mount command
	if _, err := command.Run(mountCmd, l.Log); err != nil {
		if _, err := l.BlockDevice.Deactivate(ctx); err != nil {
			return false, fmt.Errorf("could not deactivate block device after failed mount %s: %w", path, err)
		}

		return false, fmt.Errorf("could not mount file system %s: %w", path, err)
	}

	return true, nil
}

func (l *LustreFileSystem) Deactivate(ctx context.Context) (bool, error) {
	mounter := mount.New("")
	mounts, err := mounter.List()
	if err != nil {
		return false, err
	}

	path := filepath.Clean(fmt.Sprintf("/mnt/nnf/lustre/%s.%s.%d", l.Name, l.TargetType, l.Index))
	for _, m := range mounts {
		if m.Path != path {
			continue
		}

		// Found an existing mount at this path. Check if it's the mount we expect
		if m.Device != l.BlockDevice.GetDevice() || m.Type != "lustre" {
			return false, fmt.Errorf("unexpected mount at path %s. Device %s type %s", path, m.Device, m.Type)
		}

		if _, err := command.Run(fmt.Sprintf("umount %s", path), l.Log); err != nil {
			return false, fmt.Errorf("could not unmount file system %s: %w", path, err)
		}

		// Remove the directory. If it fails don't worry about it.
		_ = os.Remove(path)

		if _, err := l.BlockDevice.Deactivate(ctx); err != nil {
			return false, fmt.Errorf("could not deactivate block device after unmount %s: %w", path, err)
		}

		return true, nil
	}

	// Try to deactivate the block device in case the deactivate failed after the unmount above
	if _, err := l.BlockDevice.Deactivate(ctx); err != nil {
		return false, fmt.Errorf("could not deactivate block device after unmount %s: %w", path, err)
	}

	// file system already unmounted
	return false, nil
}

func (l *LustreFileSystem) Mount(ctx context.Context, path string, complete bool) (bool, error) {
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

		// Found an existing mount at this path. Check if it's the mount we expect
		if m.Type != "lustre" {
			return false, fmt.Errorf("unexpected mount at path %s. Device %s type %s", path, m.Device, m.Type)
		}

		// The file system is already mounted. Nothing left to do
		return false, nil
	}

	// Create the mount directory
	if err := os.MkdirAll(path, 0755); err != nil {
		return false, fmt.Errorf("could not create mount directory %s: %w", path, err)
	}

	// Build the mount command from the args provided
	l.CommandArgs.Vars["$MOUNT_PATH"] = path
	mountCmd := fmt.Sprintf("mount -t lustre %s", l.parseArgs(l.CommandArgs.Mount))

	// Run the mount command
	if _, err := command.Run(mountCmd, l.Log); err != nil {
		return false, fmt.Errorf("could not mount file system %s: %w", path, err)
	}

	return true, nil
}

func (l *LustreFileSystem) Unmount(ctx context.Context, path string) (bool, error) {
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

		// Found an existing mount at this path. Check if it's the mount we expect
		if m.Device != fmt.Sprintf("%s:/%s", l.MgsAddress, l.Name) || m.Type != "lustre" {
			return false, fmt.Errorf("unexpected mount at path %s. Device %s type %s", path, m.Device, m.Type)
		}

		if _, err := command.Run(fmt.Sprintf("umount %s", path), l.Log); err != nil {
			return false, fmt.Errorf("could not unmount file system %s: %w", path, err)
		}

		// Remove the directory. If it fails don't worry about it.
		_ = os.Remove(path)

		return true, nil
	}

	// file system already unmounted
	return false, nil
}

func (l *LustreFileSystem) SetPermissions(ctx context.Context, uid uint32, gid uint32, complete bool) (bool, error) {
	return false, nil
}
