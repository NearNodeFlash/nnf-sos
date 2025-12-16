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
	"github.com/NearNodeFlash/nnf-sos/pkg/command"
	"github.com/NearNodeFlash/nnf-sos/pkg/var_handler"
	"github.com/go-logr/logr"

	mount "k8s.io/mount-utils"
)

type SimpleFileSystemCommandArgs struct {
	Mkfs           string
	Mount          string
	PreActivate    []string
	PostActivate   []string
	PreDeactivate  []string
	PostDeactivate []string
	PreMount       []string
	PostMount      []string
	PreUnmount     []string
	PostUnmount    []string
	PostSetup      []string
	PreTeardown    []string

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

	// If the device is already formatted, don't run the mkfs again
	formatted, err := f.BlockDevice.CheckFormatted()
	if err != nil {
		return false, fmt.Errorf("could not determine if device is formatted: %w", err)
	}
	if formatted {
		return false, nil
	}

	if _, err := command.Run(fmt.Sprintf("mkfs -t %s %s", f.Type, f.parseArgs(f.CommandArgs.Mkfs)), f.Log); err != nil {
		if err != nil {
			return false, fmt.Errorf("could not create file system: %w", err)
		}
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

func (f *SimpleFileSystem) Mount(ctx context.Context, path string, complete bool) (bool, error) {
	if len(f.CommandArgs.Mount) == 0 {
		return false, nil
	}

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

	// Build the mount command from the args provided
	if f.CommandArgs.Vars == nil {
		f.CommandArgs.Vars = make(map[string]string)
	}
	f.CommandArgs.Vars["$MOUNT_PATH"] = path
	mountCmd := fmt.Sprintf("mount -t %s %s", f.Type, f.parseArgs(f.CommandArgs.Mount))

	if _, err := command.Run(mountCmd, f.Log); err != nil {
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

		if _, err := command.Run(fmt.Sprintf("umount %s", path), f.Log); err != nil {
			return false, fmt.Errorf("could not unmount file system %s: %w", path, err)
		}

		// Remove the file/directory. If it fails don't worry about it.
		_ = os.Remove(path)

		return true, nil
	}

	// file system already unmounted
	return false, nil
}

func (f *SimpleFileSystem) SimpleRunCommands(ctx context.Context, complete bool, commands []string, phase string, args map[string]string) (bool, error) {
	if len(commands) == 0 {
		return false, nil
	}

	if complete {
		return false, nil
	}

	// Build the commands from the args provided
	if f.CommandArgs.Vars == nil {
		f.CommandArgs.Vars = make(map[string]string)
	}

	for k, v := range args {
		f.CommandArgs.Vars[k] = v
	}

	for _, rawCommand := range commands {
		formattedCommand := f.parseArgs(rawCommand)
		f.Log.Info(phase, "command", formattedCommand)

		output, err := command.Run(formattedCommand, f.Log)
		if err != nil {
			return false, fmt.Errorf("could not run %s command: %s: %w", phase, formattedCommand, err)
		}

		f.Log.Info(phase, "output", output)
	}

	return true, nil
}

func (f *SimpleFileSystem) PreMount(ctx context.Context, path string, complete bool) (bool, error) {
	return f.SimpleRunCommands(ctx, complete, f.CommandArgs.PreMount, "PreMount", map[string]string{"$MOUNT_PATH": path})
}

func (f *SimpleFileSystem) PostMount(ctx context.Context, path string, complete bool) (bool, error) {
	return f.SimpleRunCommands(ctx, complete, f.CommandArgs.PostMount, "PostMount", map[string]string{"$MOUNT_PATH": path})
}

func (f *SimpleFileSystem) PreUnmount(ctx context.Context, path string, complete bool) (bool, error) {
	return f.SimpleRunCommands(ctx, complete, f.CommandArgs.PreUnmount, "PreUnmount", map[string]string{"$MOUNT_PATH": path})
}

func (f *SimpleFileSystem) PostUnmount(ctx context.Context, complete bool) (bool, error) {
	return f.SimpleRunCommands(ctx, complete, f.CommandArgs.PostUnmount, "PostUnmount", nil)
}

func (f *SimpleFileSystem) PostActivate(ctx context.Context, complete bool) (bool, error) {
	return f.SimpleRunCommands(ctx, complete, f.CommandArgs.PostActivate, "PostActivate", nil)
}

func (f *SimpleFileSystem) PreDeactivate(ctx context.Context, complete bool) (bool, error) {
	return f.SimpleRunCommands(ctx, complete, f.CommandArgs.PreDeactivate, "PreDeactivate", nil)
}

func (f *SimpleFileSystem) PostSetup(ctx context.Context, complete bool) (bool, error) {
	if len(f.CommandArgs.PostSetup) == 0 {
		return false, nil
	}

	if complete {
		return false, nil
	}

	if f.Type == "none" {
		return false, nil
	}

	if _, err := f.Mount(ctx, f.TempDir, false); err != nil {
		return false, fmt.Errorf("could not mount temp dir '%s' for PostSetup: %w", f.TempDir, err)
	}

	// Build the commands from the args provided
	if f.CommandArgs.Vars == nil {
		f.CommandArgs.Vars = make(map[string]string)
	}
	f.CommandArgs.Vars["$MOUNT_PATH"] = f.TempDir

	for _, rawCommand := range f.CommandArgs.PostSetup {
		formattedCommand := f.parseArgs(rawCommand)
		f.Log.Info("PostSetup", "command", formattedCommand)

		output, err := command.Run(formattedCommand, f.Log)
		if err != nil {
			if _, unmountErr := f.Unmount(ctx, f.TempDir); unmountErr != nil {
				return false, fmt.Errorf("could not unmount after PostSetup command failed: %s: %w", formattedCommand, unmountErr)
			}
			return false, fmt.Errorf("could not run PostSetup command: %s: %w", formattedCommand, err)
		}

		f.Log.Info("PostSetup", "output", output)
	}

	if _, err := f.Unmount(ctx, f.TempDir); err != nil {
		return false, fmt.Errorf("could not unmount after PostSetup '%s': %w", f.TempDir, err)
	}

	return true, nil
}

func (f *SimpleFileSystem) PreTeardown(ctx context.Context, complete bool) (bool, error) {
	if len(f.CommandArgs.PreTeardown) == 0 {
		return false, nil
	}

	if complete {
		return false, nil
	}

	if f.Type == "none" {
		return false, nil
	}

	if _, err := f.Mount(ctx, f.TempDir, false); err != nil {
		return false, fmt.Errorf("could not mount temp dir '%s' for PreTeardown: %w", f.TempDir, err)
	}

	// Build the commands from the args provided
	if f.CommandArgs.Vars == nil {
		f.CommandArgs.Vars = make(map[string]string)
	}
	f.CommandArgs.Vars["$MOUNT_PATH"] = f.TempDir

	for _, rawCommand := range f.CommandArgs.PreTeardown {
		formattedCommand := f.parseArgs(rawCommand)
		f.Log.Info("PreTeardown", "command", formattedCommand)

		output, err := command.Run(formattedCommand, f.Log)
		if err != nil {
			if _, unmountErr := f.Unmount(ctx, f.TempDir); unmountErr != nil {
				return false, fmt.Errorf("could not unmount after PreTeardown command failed: %s: %w", formattedCommand, unmountErr)
			}
			return false, fmt.Errorf("could not run PreTeardown command: %s: %w", formattedCommand, err)
		}

		f.Log.Info("PreTeardown", "output", output)
	}

	if _, err := f.Unmount(ctx, f.TempDir); err != nil {
		return false, fmt.Errorf("could not unmount after PreTeardown'%s': %w", f.TempDir, err)
	}

	return true, nil
}
