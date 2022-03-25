package server

import (
	"fmt"
	"regexp"
)

type FileSystemGfs2 struct {
	FileSystemLvm
	clusterName string
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

	if len(oem.ClusterName) == 0 {
		return nil, fmt.Errorf("Cluster Name not provided")
	}

	// Pattern checks ...
	exp := regexp.MustCompile("[a-zA-Z0-9_\\-]*")

	if !exp.MatchString(oem.Name) {
		return nil, fmt.Errorf("File System Name '%s' is invalid. Must match pattern '%s'", oem.Name, exp.String())
	}

	if !exp.MatchString(oem.ClusterName) {
		return nil, fmt.Errorf("Cluster Name '%s' is invalid. Must match pattern '%s'", oem.ClusterName, exp.String())
	}

	return &FileSystemGfs2{
		FileSystemLvm: FileSystemLvm{
			FileSystem: FileSystem{name: oem.Name},
		},
		clusterName: oem.ClusterName,
	}, nil
}

func (*FileSystemGfs2) IsType(oem FileSystemOem) bool { return oem.Type == "gfs2" }
func (*FileSystemGfs2) IsMockable() bool              { return false }
func (*FileSystemGfs2) Type() string                  { return "gfs2" }

func (f *FileSystemGfs2) Name() string { return f.name }

func (f *FileSystemGfs2) Create(devices []string, opts FileSystemOptions) error {
	opts["shared"] = true
	if err := f.FileSystemLvm.Create(devices, opts); err != nil {
		return err
	}

	if _, err := f.run(fmt.Sprintf("mkfs.gfs2 -O -j2 -p lock_dlm -t %s:%s %s", f.clusterName, f.Name(), f.FileSystemLvm.devPath())); err != nil {
		return err
	}

	return nil
}

func (f *FileSystemGfs2) Delete() error {
	return f.FileSystemLvm.Delete()
}

func (f *FileSystemGfs2) Mount(mountpoint string) error {
	if _, err := f.run(fmt.Sprintf("mkdir -p %s", mountpoint)); err != nil {
		return err
	}

	if _, err := f.run(fmt.Sprintf("mount %s %s", f.FileSystemLvm.devPath(), mountpoint)); err != nil {
		return err
	}

	f.mountpoint = mountpoint

	return nil
}

func (f *FileSystemGfs2) Unmount() error {
	if f.mountpoint == "" {
		return nil
	}

	_, err := f.run(fmt.Sprintf("unmount %s", f.mountpoint))

	f.mountpoint = ""

	return err
}
