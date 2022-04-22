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
	"strings"
)

type FileSystemZfs struct {
	// Satisfy FileSystemApi interface.
	FileSystem
}

func init() {
	FileSystemRegistry.RegisterFileSystem(&FileSystemZfs{})
}

func (*FileSystemZfs) New(oem FileSystemOem) (FileSystemApi, error) {
	return &FileSystemZfs{FileSystem: FileSystem{name: oem.Name}}, nil
}

func (*FileSystemZfs) IsType(oem FileSystemOem) bool { return oem.Type == "zfs" }
func (*FileSystemZfs) IsMockable() bool              { return false }

func (*FileSystemZfs) Type() string   { return "zfs" }
func (f *FileSystemZfs) Name() string { return f.name }

func (f *FileSystemZfs) Create(devices []string, options FileSystemOptions) error {

	f.devices = devices

	// For ZFS, there are no creation steps necessary for a file system.
	// All the work is done in a single call; this is deferred until
	// the Mount() is executed

	return nil
}

func (f *FileSystemZfs) Delete() error { return nil }

func (f *FileSystemZfs) Mount(mountpoint string) error {
	_, err := f.run(fmt.Sprintf("zpool create -m %s %s %s", f.mountpoint, f.name, strings.Join(f.devices, " ")))

	return err
}

func (f *FileSystemZfs) Unmount() error {
	_, err := f.run(fmt.Sprintf("zpool destroy %s", f.name))

	return err
}
