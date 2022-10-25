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
	"regexp"

	"github.com/NearNodeFlash/nnf-ec/pkg/var_handler"
)

type FileSystemGfs2 struct {
	FileSystemLvm

	Oem       FileSystemOemGfs2
	MkfsMount FileSystemOemMkfsMount
}

func init() {
	FileSystemRegistry.RegisterFileSystem(&FileSystemGfs2{})
}

func (*FileSystemGfs2) New(oem FileSystemOem) (FileSystemApi, error) {

	// From mksfs.gfs2(8)
	//    Fsname is a unique file system name used to distinguish this GFS2
	//    file system from others created (1 to 16 characters). Valid
	//    clusternames and fsnames may only contain alphanumeric characters,
	//    hyphens (-) and underscores (_)

	// Length checks...
	if len(oem.Name) == 0 {
		return nil, fmt.Errorf("File Name not provided")
	}

	if len(oem.Name) > 16 {
		return nil, fmt.Errorf("File Name '%s' overflows 16 character limit", oem.Name)
	}

	if len(oem.Gfs2.ClusterName) == 0 {
		return nil, fmt.Errorf("Cluster Name not provided")
	}

	// Pattern checks ...
	exp := regexp.MustCompile("[a-zA-Z0-9_\\-]*")

	if !exp.MatchString(oem.Name) {
		return nil, fmt.Errorf("File System Name '%s' is invalid. Must match pattern '%s'", oem.Name, exp.String())
	}

	if !exp.MatchString(oem.Gfs2.ClusterName) {
		return nil, fmt.Errorf("Cluster Name '%s' is invalid. Must match pattern '%s'", oem.Gfs2.ClusterName, exp.String())
	}

	return &FileSystemGfs2{
		FileSystemLvm: FileSystemLvm{
			FileSystem: FileSystem{name: oem.Name},
			CmdArgs:    oem.LvmCmd,
			shared:     true,
		},
		Oem:       oem.Gfs2,
		MkfsMount: oem.MkfsMount,
	}, nil
}

func (*FileSystemGfs2) IsType(oem FileSystemOem) bool { return oem.Type == "gfs2" }
func (*FileSystemGfs2) IsMockable() bool              { return false }
func (*FileSystemGfs2) Type() string                  { return "gfs2" }

func (f *FileSystemGfs2) Name() string { return f.name }

func (f *FileSystemGfs2) Create(devices []string, opts FileSystemOptions) error {

	if err := f.FileSystemLvm.Create(devices, opts); err != nil {
		return err
	}

	varHandler := var_handler.NewVarHandler(map[string]string{
		"$DEVICE":       f.FileSystemLvm.devPath(),
		"$CLUSTER_NAME": f.Oem.ClusterName,
		"$LOCK_SPACE":   f.Name(),
		"$PROTOCOL":     "lock_dlm",
	})
	mkfsArgs := varHandler.ReplaceAll(f.MkfsMount.Mkfs)

	if _, err := f.run(fmt.Sprintf("mkfs.gfs2 -O %s", mkfsArgs)); err != nil {
		return err
	}

	return nil
}

func (f *FileSystemGfs2) Mount(mountpoint string) error {
	return f.mount(f.devPath(), mountpoint, "", f.MkfsMount.Mount)
}
