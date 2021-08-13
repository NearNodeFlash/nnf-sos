package server

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"os"
)

type LustreTargetType string

const (
	TargetMGT LustreTargetType = "MGT"
	TargetMDT LustreTargetType = "MDT"
	TargetOST LustreTargetType = "OST"
)

var targetTypes = map[string]LustreTargetType{
	"MGT": TargetMGT,
	"MDT": TargetMDT,
	"OST": TargetOST,
}

type FileSystemLustre struct {
	// Satisfy FileSystemApi interface.
	FileSystem

	targetType LustreTargetType
	mgsNode    string
	index      int
}

func NewFileSystemLustre(oem FileSystemOem) FileSystemApi {
	fs := &FileSystemLustre{
		FileSystem: FileSystem{name: oem.Name},
		mgsNode:    oem.MgsNode,
		index:      0, //oem.Index,
		// TargetType is already verified by IsType() below.
		targetType: targetTypes[oem.TargetType],
	}
	return fs
}

func (*FileSystemLustre) IsType(oem FileSystemOem) bool {
	_, ok := targetTypes[oem.TargetType]
	return ok
}
func (*FileSystemLustre) Type() string   { return "lustre" }
func (f *FileSystemLustre) Name() string { return f.name }

func (f *FileSystemLustre) Create(devices []string, options FileSystemOptions) error {

	var err error
	f.devices = make([]string, 1)
	f.devices[0] = "/dev/vdb"
	switch f.targetType {
	case TargetMGT:
		cmd := fmt.Sprintf("mkfs.lustre --mgs %s", f.devices[0])
		log.Info("EEE mkfs MGT ", " cmd ", cmd, " devices ", f.devices)
		out, err := f.run(cmd)
		if err != nil {
			log.Error(err, " EEE mkfs MGT")
		}
		if out != nil {
			log.Info("EEE mkfs MGT", "out", out)
		}
	case TargetMDT:
		_, err = f.run(fmt.Sprintf("mkfs.lustre --mdt --fsname=%s --mgsnode=%s --index=%d %s", f.name, f.mgsNode, f.index, f.devices[0]))
	case TargetOST:
		_, err = f.run(fmt.Sprintf("mkfs.lustre --ost --fsname=%s --mgsnode=%s --index=%d %s", f.name, f.mgsNode, f.index, f.devices[0]))
	}

	return err
}

func (f *FileSystemLustre) Delete() error { return nil }

func (f *FileSystemLustre) Mount(mountpoint string) error {

	log.Info("EEE mount mkdir", "dir", mountpoint)
	if err := os.MkdirAll(mountpoint, 0755); err != nil {
		// Skip anything other than ErrExist.
		if os.IsExist(err) == false {
			log.Error(err, " EEE mount mkdir")
			return err
		}
	}

	cmd := fmt.Sprintf("mount -t lustre %s %s",
		f.devices[0], mountpoint)
	log.Info("EEE mount", "cmd", cmd)
	out, err := f.run(cmd)
	if err != nil {
		log.Error(err, " EEE mount")
		return err
	}
	if out != nil {
		log.Info("EEE mount", "out", out)
	}
	f.mountpoint = mountpoint
	return nil
}

func (f *FileSystemLustre) Unmount() error {
	if len(f.mountpoint) > 0 {
		_, err := f.run(fmt.Sprintf("umount %s", f.mountpoint))
		if err != nil {
			return err
		}
	}
	if err := os.Remove(f.mountpoint); err != nil {
		// Log anything other than ErrNotExist.
		if os.IsNotExist(err) == false {
			// Just log it, don't fuss over it.
			log.Info("Unable to remove mountpoint; continuing", "mountpoint", f.mountpoint, "err", err)
		}
	}
	return nil
}
