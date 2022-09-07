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
	"os"
	"path/filepath"
	"strings"
)

const (
	VolumeGroupName   = "volumeGroupName"
	LogicalVolumeName = "logicalVolumeName"
)

type FileSystemLvm struct {
	// Satisfy FileSystemApi interface.
	FileSystem
	lvName string
	vgName string
	shared bool
}

func init() {
	FileSystemRegistry.RegisterFileSystem(&FileSystemLvm{})
}

func (*FileSystemLvm) New(oem FileSystemOem) (FileSystemApi, error) {
	return &FileSystemLvm{
		FileSystem: FileSystem{name: oem.Name},
		shared:     false,
	}, nil
}

func (*FileSystemLvm) IsType(oem FileSystemOem) bool { return oem.Type == "lvm" }
func (*FileSystemLvm) IsMockable() bool              { return false }

func (*FileSystemLvm) Type() string   { return "lvm" }
func (f *FileSystemLvm) Name() string { return f.name }

func (f *FileSystemLvm) Create(devices []string, opts FileSystemOptions) error {

	if _, exists := opts[VolumeGroupName]; exists {
		f.vgName = opts[VolumeGroupName].(string)
	} else {
		f.vgName = fmt.Sprintf("%s_vg", f.Name())
	}

	if _, exists := opts[LogicalVolumeName]; exists {
		f.lvName = opts[LogicalVolumeName].(string)
	} else {
		f.lvName = fmt.Sprintf("%s_lv", f.Name())
	}

	// TODO: Some sort of rollback mechanism on failure condition

	// Ensure the existing volume groups are scanned. Use this information to determine
	// if initialization is required for the Volume Group, or if it exists and can be
	// activated.
	if _, err := f.run("vgscan"); err != nil {
		return err
	}

	rsp, _ := f.run(fmt.Sprintf("lvdisplay %s || echo 'not found'", f.vgName))
	if len(rsp) != 0 && !strings.Contains(string(rsp), "not found") {
		// Volume Group is present, activate the volume. This is for the case where another
		// node created the volume group and we just want to share it.
		shared := ""
		if f.shared {
			// Activate the shared lock
			if _, err := f.run(fmt.Sprintf("vgchange --lock-start %s", f.vgName)); err != nil {
				return err
			}

			shared = "s" // activate with shared option
		}

		if _, err := f.run(fmt.Sprintf("vgchange --activate %sy %s", shared, f.vgName)); err != nil {
			return err
		}

		return nil
	}

	// Create the physical volumes.
	for _, device := range devices {
		if _, err := f.run(fmt.Sprintf("pvcreate %s", device)); err != nil {
			return err
		}
	}

	// Check if we are creating a shared volume group and prepend the necessary flag
	shared := ""
	if f.shared {
		shared = "--shared"
	}

	if _, err := f.run(fmt.Sprintf("vgcreate %s %s %s", shared, f.vgName, strings.Join(devices, " "))); err != nil {
		return err
	}

	shared = ""
	if f.shared {
		if _, err := f.run(fmt.Sprintf("vgchange --lockstart %s", f.vgName)); err != nil {
			return err
		}

		shared = "s"
	}

	// Create the logical volume
	// -Zn - don't zero the volume, it will fail.
	// We are depending on the drive behavior for newly allocated blocks to track
	// NVM Command Set spec, Section 3.2.3.2.1 Deallocated or Unwritten Logical Blocks
	// The Kioxia drives support DLFEAT=001b
	// 	3.2.3.2.1 Deallocated or Unwritten Logical Blocks
	// A logical block that has never been written to, or which has been deallocated using the Dataset
	// Management command, the Write Zeroes command or the Sanitize command is called a deallocated or unwritten logical block.
	// Using the Error Recovery feature (refer to section 4.1.3.2), host software may select the behavior
	// of the controller when reading deallocated or unwritten blocks. The controller shall abort Copy, Read, Verify,
	// or Compare commands that include deallocated or unwritten blocks with a status of Deallocated or Unwritten Logical Block
	// if that error has been enabled using the DULBE bit in the Error Recovery feature. If the Deallocated or Unwritten Logical
	// error is not enabled, the values read from a deallocated or unwritten block and its metadata (excluding protection information)
	// shall be:
	// • all bytes cleared to 0h if bits 2:0 in the DLFEAT field are set to 001b;
	zeroOpt := "-Zn"
	if _, ok := os.LookupEnv("NNF_SUPPLIED_DEVICES"); ok {
		// On non-NVMe, let the zeroing happen so devices can be reused.
		zeroOpt = "--yes"
	}

	if _, err := f.run(fmt.Sprintf("lvcreate %s -l 100%%VG --stripes %d --stripesize=32KiB --name %s %s", zeroOpt, len(devices), f.lvName, f.vgName)); err != nil {
		return err
	}

	// Activate the volume group.
	if _, err := f.run(fmt.Sprintf("vgchange --activate %sy %s", shared, f.vgName)); err != nil {
		return err
	}

	return nil
}

func (f *FileSystemLvm) Delete() error {

	if _, err := f.run(fmt.Sprintf("lvremove --yes %s", f.devPath())); err != nil {
		return err
	}

	if _, err := f.run(fmt.Sprintf("vgchange --activate n %s", f.vgName)); err != nil {
		return err
	}

	if _, err := f.run(fmt.Sprintf("vgremove --yes %s", f.vgName)); err != nil {
		return err
	}

	for _, device := range f.devices {
		if _, err := f.run(fmt.Sprintf("pvremove --yes %s", device)); err != nil {
			return err
		}
	}

	return nil
}

func (f *FileSystemLvm) Mount(mountpoint string) error {
	if _, err := f.run(fmt.Sprintf("mkdir -p %s", filepath.Dir(mountpoint))); err != nil {
		return err
	}

	if _, err := f.run(fmt.Sprintf("touch %s", mountpoint)); err != nil {
		return err
	}

	mounted, err := f.IsMountPoint(mountpoint)
	if err != nil {
		return err
	}
	
	if !mounted {
		if _, err := f.run(fmt.Sprintf("mount --bind %s %s", f.devPath(), mountpoint)); err != nil {
			return err
		}
	}

	return nil
}

func (f *FileSystemLvm) GenerateRecoveryData() map[string]string {
	return map[string]string{
		VolumeGroupName:   f.vgName,
		LogicalVolumeName: f.lvName,
	}
}

func (f *FileSystemLvm) LoadRecoveryData(data map[string]string) {
	f.vgName = data[VolumeGroupName]
	f.lvName = data[LogicalVolumeName]
}

func (f *FileSystemLvm) devPath() string {
	return filepath.Join("/dev", f.vgName, f.lvName)
}
