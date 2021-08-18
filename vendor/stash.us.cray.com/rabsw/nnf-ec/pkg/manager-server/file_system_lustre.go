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
		index:      oem.Index,
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
	f.devices = devices
	switch f.targetType {
	case TargetMGT:
		err = runCmd(f, fmt.Sprintf("mkfs.lustre --mgs %s", f.devices[0]))
	case TargetMDT:
		err = runCmd(f, fmt.Sprintf("mkfs.lustre --mdt --fsname=%s --mgsnode=%s --index=%d %s", f.name, f.mgsNode, f.index, f.devices[0]))
	case TargetOST:
		err = runCmd(f, fmt.Sprintf("mkfs.lustre --ost --fsname=%s --mgsnode=%s --index=%d %s", f.name, f.mgsNode, f.index, f.devices[0]))
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

func (f *FileSystemLustre) Delete() error { return nil }

func (f *FileSystemLustre) Mount(mountpoint string) error {

	if err := os.MkdirAll(mountpoint, 0755); err != nil {
		// Skip anything other than ErrExist.
		if os.IsExist(err) == false {
			log.Error(err, "Unable to create mountpoint", " mountpoint ", mountpoint)
			return err
		}
	}

	err := runCmd(f, fmt.Sprintf("mount -t lustre %s %s", f.devices[0], mountpoint))
	if err != nil {
		return err
	}
	f.mountpoint = mountpoint
	return nil
}

func (f *FileSystemLustre) Unmount() error {
	if len(f.mountpoint) > 0 {
		err := runCmd(f, fmt.Sprintf("umount %s", f.mountpoint))
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
