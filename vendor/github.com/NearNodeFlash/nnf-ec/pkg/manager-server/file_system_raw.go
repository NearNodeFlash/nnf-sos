/*
 * Copyright 2020, 2021, 2022, 2023 Hewlett Packard Enterprise Development LP
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
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"k8s.io/mount-utils"
)

type FileSystemRaw struct {
	FileSystemLvm

	uid     int
	gid     int
}

func init() {
	FileSystemRegistry.RegisterFileSystem(&FileSystemRaw{})
}

func (*FileSystemRaw) New(oem FileSystemOem) (FileSystemApi, error) {
	return &FileSystemRaw{
		FileSystemLvm: FileSystemLvm{
			FileSystem: FileSystem{name: oem.Name},
			CmdArgs:    oem.LvmCmd,
			shared:     false,
		},
	}, nil
}

func (*FileSystemRaw) IsType(oem *FileSystemOem) bool { return oem.Type == "raw" }
func (*FileSystemRaw) Type() string                   { return "raw" }


func (f *FileSystemRaw) Create(devices []string, opts FileSystemOptions) error {
	if err := f.FileSystemLvm.Create(devices, opts); err != nil {
		return err
	}

	if _, exists := opts[UserID]; exists {
		f.uid = opts[UserID].(int)
	} else {
		f.uid = 0
	}

	if _, exists := opts[GroupID]; exists {
		f.gid = opts[GroupID].(int)
	} else {
		f.gid = 0
	}

	return nil
}

func (f *FileSystemRaw) Mount(mountpoint string) error {
	// Make the parent directory
	if err := os.MkdirAll(filepath.Dir(mountpoint), 0755); err != nil {
		return err
	}

	// Create an empty file to bind mount the device onto
	if err := os.WriteFile(mountpoint, []byte(""), 0644); err != nil {
		return err
	}

	mounter := mount.New("")
	mounted, err := mounter.IsMountPoint(mountpoint)
	if err != nil {
		return err
	}

	if !mounted {
		if err := mounter.Mount(f.devPath(), mountpoint, "none", []string{"bind"}); err != nil {
			log.Errorf("Mount failed: %v", err)
			return err
		}

		if err := os.Chown(f.devPath(), f.uid, f.gid); err != nil {
			return err
		}
	}

	return nil
}
