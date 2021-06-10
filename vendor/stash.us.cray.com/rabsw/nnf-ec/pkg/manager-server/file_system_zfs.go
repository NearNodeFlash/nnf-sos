package server

import (
	"fmt"
	"strings"
)

type FileSystemZfs struct {
	FileSystem

	devices    []string
	mountpoint string
}

func NewFileSystemZfs(name string) FileSystemApi {
	return &FileSystemZfs{FileSystem: FileSystem{name: name}}
}

func (*FileSystemZfs) Type() string   { return "zfs" }
func (f *FileSystemZfs) Name() string { return f.name }

func (f *FileSystemZfs) Create(devices []string, options FileSystemOptions) error {

	f.devices = devices

	// For ZFS, there are no creation steps necessary for a file system.
	// All the work is done in a single call; this is deferred until
	// the Mount() is executed

	return nil
}

func (f *FileSystemZfs) Delete() error { return nil }

func (f *FileSystemZfs) Mount(mountpoint string) error {
	_, err := f.run(fmt.Sprintf("zpool create -m %s %s %s",
		f.mountpoint, f.name, strings.Join(f.devices, " ")))

	return err
}

func (f *FileSystemZfs) Unmount() error {
	_, err := f.run(fmt.Sprintf("zpool destroy %s", f.name))

	return err
}
