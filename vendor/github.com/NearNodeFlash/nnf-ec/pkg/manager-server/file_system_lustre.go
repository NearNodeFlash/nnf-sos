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
	"strings"

	"github.com/NearNodeFlash/nnf-ec/pkg/var_handler"
)

const (
	TargetMGT    string = "MGT"
	TargetMDT    string = "MDT"
	TargetMGTMDT string = "MGTMDT"
	TargetOST    string = "OST"
)

var targetTypes = map[string]string{
	"MGT":    TargetMGT,
	"MDT":    TargetMDT,
	"MGTMDT": TargetMGTMDT,
	"OST":    TargetOST,
}

const (
	BackFsLdiskfs string = "ldiskfs"
	BackFsZfs     string = "zfs"
)

var backFsTypes = map[string]string{
	"zfs": BackFsZfs,
}

func init() {
	FileSystemRegistry.RegisterFileSystem(&FileSystemLustre{})
}

type FileSystemLustre struct {
	// Satisfy FileSystemApi interface.
	FileSystem

	TargetType string

	Oem       FileSystemOemLustre
	MkfsMount FileSystemOemMkfsMount
	ZfsArgs   FileSystemOemZfs
}

func (*FileSystemLustre) New(oem FileSystemOem) (FileSystemApi, error) {
	return &FileSystemLustre{
		FileSystem: FileSystem{name: oem.Name},
		TargetType: targetTypes[oem.Lustre.TargetType],
		Oem:        oem.Lustre,
		MkfsMount:  oem.MkfsMount,
		ZfsArgs:    oem.ZfsCmd,
	}, nil
}

func (*FileSystemLustre) IsType(oem *FileSystemOem) bool {
	_, ok := targetTypes[oem.Lustre.TargetType]
	if ok {
		_, ok = backFsTypes[oem.Lustre.BackFs]
	}
	return ok
}

func (*FileSystemLustre) IsMockable() bool { return false }
func (*FileSystemLustre) Type() string     { return "lustre" }

func (f *FileSystemLustre) Name() string { return f.name }

func (f *FileSystemLustre) MkfsDefault() string {
	return map[string]string{
		TargetMGT:    "--mgs $VOL_NAME",
		TargetMDT:    "--mdt --fsname=$FS_NAME --mgsnode=$MGS_NID --index=$INDEX $VOL_NAME",
		TargetMGTMDT: "--mgs --mdt --fsname=$FS_NAME --index=$INDEX $VOL_NAME",
		TargetOST:    "--ost --fsname=$FS_NAME --mgsnode=$MGS_NID --index=$INDEX $VOL_NAME",
	}[f.TargetType]
}

func (f *FileSystemLustre) Create(devices []string, options FileSystemOptions) error {

	var err error
	f.devices = devices
	if f.Oem.BackFs != BackFsZfs {
		return fmt.Errorf("The backing FS must be ZFS")
	}

	devsStr := strings.Join(devices, " ")
	if _, ok := os.LookupEnv("NNF_SUPPLIED_DEVICES"); ok {
		// On non-NVMe.
		devsStr = devices[0]
	}

	varHandler := var_handler.NewVarHandler(map[string]string{
		"$DEVICE_NUM":  fmt.Sprintf("%d", len(devices)),
		"$DEVICE_LIST": devsStr,
		"$POOL_NAME":   f.zfsPoolName(),
		"$VOL_NAME":    f.zfsVolName(), // <pool_name>/<dataset_name>
		"$MGS_NID":     f.Oem.MgsNode,
		"$INDEX":       fmt.Sprintf("%d", f.Oem.Index),
		"$FS_NAME":     f.name,
	})
	if err := varHandler.ListToVars("$DEVICE_LIST", "$DEVICE"); err != nil {
		return fmt.Errorf("invalid internal device list: %w", err)
	}

	zpoolArgs := varHandler.ReplaceAll(f.ZfsArgs.ZpoolCreate)
	zpoolCreate := fmt.Sprintf("zpool create %s", zpoolArgs)
	_, err = f.run(zpoolCreate)
	if err != nil {
		return err
	}

	mkfsArgs := varHandler.ReplaceAll(f.MkfsMount.Mkfs)
	mkfsCmd := fmt.Sprintf("mkfs.lustre --backfstype=%s %s", f.Oem.BackFs, mkfsArgs)
	_, err = f.run(mkfsCmd)
	if err != nil {
		// Destroy the zpool we created above.
		_, _ = f.run(fmt.Sprintf("zpool destroy %s", f.zfsPoolName()))
	}

	return err
}

func (f *FileSystemLustre) Delete() error {
	var err error
	if f.Oem.BackFs == BackFsZfs {
		zpool := f.zfsPoolName()
		// Query the existence of the pool.
		_, err = f.run(fmt.Sprintf("zpool list %s", zpool))
		if err != nil {
			return err
		}
		_, err = f.run(fmt.Sprintf("zpool destroy %s", zpool))
		if err != nil {
			return err
		}
	}

	if _, ok := os.LookupEnv("NNF_SUPPLIED_DEVICES"); ok {
		// On non-NVMe, wipe so devices can be reused.
		var devName string = f.devices[0]
		_, err = f.run(fmt.Sprintf("wipefs --all %s", devName))
		if err != nil {
			return err
		}
	}
	// Inform the OS of partition table changes.
	_, err = f.run("partprobe")
	if err != nil {
		return err
	}

	return nil
}

func (f *FileSystemLustre) Mount(mountpoint string) error {

	var devName string
	if f.Oem.BackFs == BackFsZfs {
		devName = f.zfsVolName()
	} else {
		devName = f.devices[0]
	}

	return f.mount(devName, mountpoint, "lustre", f.MkfsMount.Mount)
}

func (f *FileSystemLustre) GenerateRecoveryData() map[string]string {
	return map[string]string{}
}

func (f *FileSystemLustre) LoadRecoveryData(map[string]string) {

}

func (f *FileSystemLustre) zfsTargType() string {
	return strings.ToLower(string(f.Oem.TargetType))
}

func (f *FileSystemLustre) zfsPoolName() string {
	return fmt.Sprintf("%s-%spool-%d", f.name, f.zfsTargType(), f.Oem.Index)
}

func (f *FileSystemLustre) zfsVolName() string {
	return fmt.Sprintf("%s/%s%d", f.zfsPoolName(), f.zfsTargType(), f.Oem.Index)
}
