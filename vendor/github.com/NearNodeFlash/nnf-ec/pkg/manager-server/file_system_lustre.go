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

	log "github.com/sirupsen/logrus"
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
	"ldiskfs": BackFsLdiskfs,
	"zfs":     BackFsZfs,
}

func init() {
	FileSystemRegistry.RegisterFileSystem(&FileSystemLustre{})
}

type FileSystemLustre struct {
	// Satisfy FileSystemApi interface.
	FileSystem

	Oem              FileSystemOemLustre
	MkfsMountAddOpts FileSystemOemMkfsMountAddOpts
}

func (*FileSystemLustre) New(oem FileSystemOem) (FileSystemApi, error) {
	return &FileSystemLustre{
		FileSystem: FileSystem{name: oem.Name},
		// TargetType and BackFs are already verified by IsType() below.
		Oem:              oem.Lustre,
		MkfsMountAddOpts: oem.MkfsMountAddOpts,
	}, nil
}

func (*FileSystemLustre) IsType(oem FileSystemOem) bool {
	_, ok := targetTypes[oem.Lustre.TargetType]
	if ok {
		_, ok = backFsTypes[oem.Lustre.BackFs]
	}
	return ok
}

func (*FileSystemLustre) IsMockable() bool { return false }
func (*FileSystemLustre) Type() string     { return "lustre" }
func (f *FileSystemLustre) Name() string   { return f.name }

func (f *FileSystemLustre) Create(devices []string, options FileSystemOptions) error {

	var err error
	var backFs string
	f.devices = devices
	if f.Oem.BackFs == BackFsZfs {
		backFs = fmt.Sprintf("--backfstype=%s %s", f.Oem.BackFs, f.zfsVolName())
	}
	devsStr := strings.Join(devices, " ")
	if _, ok := os.LookupEnv("NNF_SUPPLIED_DEVICES"); ok {
		// On non-NVMe.
		devsStr = devices[0]
	}
	var mkfsArgs string
	switch f.Oem.TargetType {
	case TargetMGT:
		mkfsArgs = "--mgs"
	case TargetMDT:
		mkfsArgs = fmt.Sprintf("--mdt --fsname=%s --mgsnode=%s --index=%d", f.name, f.Oem.MgsNode, f.Oem.Index)
	case TargetMGTMDT:
		mkfsArgs = fmt.Sprintf("--mgs --mdt --fsname=%s --index=%d", f.name, f.Oem.Index)
	case TargetOST:
		mkfsArgs = fmt.Sprintf("--ost --fsname=%s --mgsnode=%s --index=%d", f.name, f.Oem.MgsNode, f.Oem.Index)
	}
	addMkfs := strings.Join(f.MkfsMountAddOpts.Mkfs, " ")
	mkfsCmd := fmt.Sprintf("mkfs.lustre %s %s %s %s", mkfsArgs, backFs, addMkfs, devsStr)
	err = runCmd(f, mkfsCmd)

	return err
}

func runCmd(f *FileSystemLustre, cmd string) error {
	out, err := f.run(cmd)
	if err != nil {
		log.Error(err, cmd)
	}
	log.Info(cmd, " output ", string(out))

	return err
}

func (f *FileSystemLustre) Delete() error {
	var err error
	if f.Oem.BackFs == BackFsZfs {
		zpool := f.zfsPoolName()
		// Query the existence of the pool.
		err = runCmd(f, fmt.Sprintf("zpool list %s", zpool))
		if err != nil {
			return err
		}
		err = runCmd(f, fmt.Sprintf("zpool destroy %s", zpool))
		if err != nil {
			return err
		}
	}

	if _, ok := os.LookupEnv("NNF_SUPPLIED_DEVICES"); ok {
		// On non-NVMe, wipe so devices can be reused.
		var devName string = f.devices[0]
		err = runCmd(f, fmt.Sprintf("wipefs --all %s", devName))
		if err != nil {
			return err
		}
	}
	// Inform the OS of partition table changes.
	err = runCmd(f, "partprobe")
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

	return f.mount(devName, mountpoint, "lustre", f.MkfsMountAddOpts.Mount)
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
	return fmt.Sprintf("%s-%spool", f.name, f.zfsTargType())
}

func (f *FileSystemLustre) zfsVolName() string {
	return fmt.Sprintf("%s/%s%d", f.zfsPoolName(), f.zfsTargType(), f.Oem.Index)
}
