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

type LustreTargetType string

const (
	TargetMGT    LustreTargetType = "MGT"
	TargetMDT    LustreTargetType = "MDT"
	TargetMGTMDT LustreTargetType = "MGTMDT"
	TargetOST    LustreTargetType = "OST"
)

var targetTypes = map[string]LustreTargetType{
	"MGT":    TargetMGT,
	"MDT":    TargetMDT,
	"MGTMDT": TargetMGTMDT,
	"OST":    TargetOST,
}

type LustreBackFsType string

const (
	BackFsLdiskfs LustreBackFsType = "ldiskfs"
	BackFsZfs     LustreBackFsType = "zfs"
)

var backFsTypes = map[string]LustreBackFsType{
	"ldiskfs": BackFsLdiskfs,
	"zfs":     BackFsZfs,
}

func init() {
	FileSystemRegistry.RegisterFileSystem(&FileSystemLustre{})
}

type FileSystemLustre struct {
	// Satisfy FileSystemApi interface.
	FileSystem

	targetType LustreTargetType
	mgsNode    string
	index      int
	backFs     LustreBackFsType
}

func (*FileSystemLustre) New(oem FileSystemOem) (FileSystemApi, error) {
	return &FileSystemLustre{
		FileSystem: FileSystem{name: oem.Name},
		mgsNode:    oem.MgsNode,
		index:      oem.Index,
		// TargetType and BackFs are already verified by IsType() below.
		targetType: targetTypes[oem.TargetType],
		backFs:     backFsTypes[oem.BackFs],
	}, nil
}

func (*FileSystemLustre) IsType(oem FileSystemOem) bool {
	_, ok := targetTypes[oem.TargetType]
	if ok {
		_, ok = backFsTypes[oem.BackFs]
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
	if f.backFs == BackFsZfs {
		backFs = fmt.Sprintf("--backfstype=%s %s", f.backFs, f.zfsVolName())
	}
	devsStr := strings.Join(devices, " ")
	if _, ok := os.LookupEnv("NNF_SUPPLIED_DEVICES"); ok {
		// On non-NVMe.
		devsStr = devices[0]
	}
	switch f.targetType {
	case TargetMGT:
		err = runCmd(f, fmt.Sprintf("mkfs.lustre --mgs %s %s", backFs, devsStr))
	case TargetMDT:
		err = runCmd(f, fmt.Sprintf("mkfs.lustre --mdt --fsname=%s --mgsnode=%s --index=%d %s %s", f.name, f.mgsNode, f.index, backFs, devsStr))
	case TargetMGTMDT:
		err = runCmd(f, fmt.Sprintf("mkfs.lustre --mgs --mdt --fsname=%s --index=%d %s %s", f.name, f.index, backFs, devsStr))
	case TargetOST:
		err = runCmd(f, fmt.Sprintf("mkfs.lustre --ost --fsname=%s --mgsnode=%s --index=%d %s %s", f.name, f.mgsNode, f.index, backFs, devsStr))
	}

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
	if f.backFs == BackFsZfs {
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

	if err := os.MkdirAll(mountpoint, 0755); err != nil {
		// Skip anything other than ErrExist.
		if os.IsExist(err) == false {
			log.Error(err, "Unable to create mountpoint", " mountpoint ", mountpoint)
			return err
		}
	}

	var devName string = f.devices[0]
	if f.backFs == BackFsZfs {
		devName = f.zfsVolName()
	}
	err := runCmd(f, fmt.Sprintf("mount -t lustre %s %s", devName, mountpoint))
	if err != nil {
		return err
	}

	return nil
}

func (f *FileSystemLustre) Unmount(mountpoint string) error {
	if len(mountpoint) > 0 {
		err := runCmd(f, fmt.Sprintf("umount %s", mountpoint))
		if err != nil {
			return err
		}
	}
	if err := os.Remove(mountpoint); err != nil {
		// Log anything other than ErrNotExist.
		if os.IsNotExist(err) == false {
			// Just log it, don't fuss over it.
			log.Info("Unable to remove mountpoint; continuing", "mountpoint", mountpoint, "err", err)
		}
	}

	return nil
}

func (f *FileSystemLustre) GenerateRecoveryData() map[string]string {
	return map[string]string{}
}

func (f *FileSystemLustre) LoadRecoveryData(map[string]string) {

}

func (f *FileSystemLustre) zfsTargType() string {
	return strings.ToLower(string(f.targetType))
}

func (f *FileSystemLustre) zfsPoolName() string {
	return fmt.Sprintf("%s-%spool", f.name, f.zfsTargType())
}

func (f *FileSystemLustre) zfsVolName() string {
	return fmt.Sprintf("%s/%s%d", f.zfsPoolName(), f.zfsTargType(), f.index)
}
