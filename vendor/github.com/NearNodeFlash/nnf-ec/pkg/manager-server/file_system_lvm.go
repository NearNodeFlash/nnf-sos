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
	"path"
	"strings"
	"time"

	"github.com/NearNodeFlash/nnf-ec/pkg/var_handler"
)

const (
	VolumeGroupName   = "volumeGroupName"
	LogicalVolumeName = "logicalVolumeName"
)

type FileSystemLvm struct {
	// Satisfy FileSystemApi interface.
	FileSystem
	lvName  string
	vgName  string
	shared  bool
	CmdArgs FileSystemOemLvm
}

func init() {
	FileSystemRegistry.RegisterFileSystem(&FileSystemLvm{})
}

func (*FileSystemLvm) New(oem FileSystemOem) (FileSystemApi, error) {
	return &FileSystemLvm{
		FileSystem: FileSystem{name: oem.Name},
		CmdArgs:    oem.LvmCmd,
		shared:     false,
	}, nil
}

func (*FileSystemLvm) IsType(oem *FileSystemOem) bool { return oem.Type == "lvm" }
func (*FileSystemLvm) IsMockable() bool               { return false }
func (*FileSystemLvm) Type() string                   { return "lvm" }

func (f *FileSystemLvm) Name() string { return f.name }

func (f *FileSystemLvm) MkfsDefault() string { return "" }

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

	activateVolumeGroup := func() error {
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

	rsp, _ := f.run(fmt.Sprintf("lvdisplay %s || echo 'not found'", f.vgName))
	if len(rsp) != 0 && !strings.Contains(string(rsp), "not found") {
		// Volume Group is present, activate the volume. This is for the case where another
		// node created the volume group and we just want to share it.
		return activateVolumeGroup()
	}

	varHandler := var_handler.NewVarHandler(map[string]string{
		"$DEVICE_NUM":  fmt.Sprintf("%d", len(devices)),
		"$DEVICE_LIST": strings.Join(devices, " "),
		"$LV_NAME":     f.lvName,
		"$VG_NAME":     f.vgName,
	})
	if err := varHandler.ListToVars("$DEVICE_LIST", "$DEVICE"); err != nil {
		return fmt.Errorf("invalid internal device list: %w", err)
	}

	// Create the physical volumes.
	for _, device := range devices {
		varHandler.VarMap["$DEVICE"] = device
		pvArgs := varHandler.ReplaceAll(f.CmdArgs.PvCreate)
		if _, err := f.run(fmt.Sprintf("pvcreate %s", pvArgs)); err != nil {
			return err
		}
	}
	delete(varHandler.VarMap, "$DEVICE")

	// Check if we are creating a shared volume group and prepend the necessary flag
	shared := ""
	if f.shared {
		shared = "--shared"
	}

	vgArgs := varHandler.ReplaceAll(f.CmdArgs.VgCreate)
	if _, err := f.run(fmt.Sprintf("vgcreate %s %s", shared, vgArgs)); err != nil {
		return err
	}

	if err := activateVolumeGroup(); err != nil {
		return err
	}

	// Create the logical volume
	// --zero n - don't zero the volume, it will fail.
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
	zeroOpt := "--zero n"
	if _, ok := os.LookupEnv("NNF_SUPPLIED_DEVICES"); ok {
		// On non-NVMe, let the zeroing happen so devices can be reused.
		zeroOpt = "--zero y"
	}

	lvArgs := varHandler.ReplaceAll(f.CmdArgs.LvCreate)
	if _, err := f.run(fmt.Sprintf("lvcreate --yes %s %s", zeroOpt, lvArgs)); err != nil {
		return err
	}

	// Ensure that the Logical Volume exists prior returning; otherwise a future 'mkfs' command may fail.
	/*
		NJT: Can't do this yet since to Persistent() controller all errors are fatal and perform a Rollback.
		if _, err := os.Stat(f.devPath()); os.IsNotExist(err) {
			return ec.NewErrorNotReady().WithCause(fmt.Sprintf("logical volume '%s' does not exist", f.devPath()))
		}
	*/

	// HACK: Wait up to 10 seconds for the path to come ready
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(f.devPath()); !os.IsNotExist(err) {
			return nil
		}

		time.Sleep(time.Second)
	}

	return fmt.Errorf("logical volume path '%s' does not exist after 10s", f.devPath())
}

func (f *FileSystemLvm) Delete() error {

	if _, err := f.run(fmt.Sprintf("lvremove --yes %s", f.devPath())); err != nil {
		return err
	}

	if _, err := f.run(fmt.Sprintf("vgchange --nolocking --activate n %s", f.vgName)); err != nil {
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
	return f.mount(f.devPath(), mountpoint, "", []string{"--bind"})
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
	return path.Join("/dev", f.vgName, f.lvName)
}
