package server

import "fmt"

// FileSystemXfs establishes an XFS file system on an underlying LVM logical volume.
type FileSystemXfs struct {
	FileSystemLvm
}

func init() {
	FileSystemRegistry.RegisterFileSystem(&FileSystemXfs{})
}

func (*FileSystemXfs) New(oem FileSystemOem) FileSystemApi {
	return &FileSystemXfs{
		FileSystemLvm: FileSystemLvm{
			FileSystem: FileSystem{name: oem.Name},
		},
	}
}

func (*FileSystemXfs) IsType(oem FileSystemOem) bool { return oem.Type == "xfs" }
func (*FileSystemXfs) IsMockable() bool              { return false }

func (*FileSystemXfs) Type() string   { return "xfs" }
func (f *FileSystemXfs) Name() string { return f.name }

func (f *FileSystemXfs) Create(devices []string, opts FileSystemOptions) error {
	if err := f.FileSystemLvm.Create(devices, opts); err != nil {
		return err
	}

	if _, err := f.run(fmt.Sprintf("mkfs.xfs %s", f.FileSystemLvm.devPath())); err != nil {
		return err
	}

	return nil
}

func (f *FileSystemXfs) Delete() error {
	return f.FileSystemLvm.Delete()
}

func (f *FileSystemXfs) Mount(mountpoint string) error {
	if _, err := f.run(fmt.Sprintf("mkdir -p %s", mountpoint)); err != nil {
		return err
	}

	if _, err := f.run(fmt.Sprintf("mount %s %s", f.FileSystemLvm.devPath(), mountpoint)); err != nil {
		return err
	}

	f.mountpoint = mountpoint

	return nil
}

func (f *FileSystemXfs) Unmount() error {
	if f.mountpoint == "" {
		return nil
	}

	_, err := f.run(fmt.Sprintf("umount %s", f.mountpoint))
	return err
}
