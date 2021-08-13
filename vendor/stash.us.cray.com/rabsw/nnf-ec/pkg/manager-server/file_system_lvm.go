package server

import (
	"fmt"
	"strings"
)

type FileSystemLvm struct {
	// Satisfy FileSystemApi interface.
	FileSystem

	leader bool
}

func NewFileSystemLvm(oem FileSystemOem) FileSystemApi {
	return &FileSystemLvm{FileSystem: FileSystem{name: oem.Name}, leader: true}
}

type FileSystemCreateOptionsLvm struct {
	FileSystemOem `json:",inline"`
}

func (*FileSystemLvm) IsType(oem FileSystemOem) bool { return oem.Type == "lvm" }
func (*FileSystemLvm) Type() string                  { return "lvm" }
func (f *FileSystemLvm) Name() string                { return f.name }

func (f *FileSystemLvm) Create(devices []string, opts FileSystemOptions) error {

	vg := f.vg()
	lv := f.lv()

	// TODO: Some sort of rollback mechanism on failure condition

	// Ensure the existing volume groups are scanned. Use this information to determine
	// if initilization is required for the Volume Group, or if it exists and can be
	// activated.
	if _, err := f.run("vgscan"); err != nil {
		return err
	}

	rsp, _ := f.run(fmt.Sprintf("lvdisplay %s || echo 'not found'", vg))
	if len(rsp) != 0 && !strings.Contains(string(rsp), "not found") {
		f.leader = false

		// Volume Group is present, activate the volume
		if _, err := f.run(fmt.Sprintf("vgchange --activate y %s", vg)); err != nil {
			return err
		}

		return nil
	}

	// Only create the file system if it doesn't exist yet
	for _, device := range devices {
		if _, err := f.run(fmt.Sprintf("pvcreate %s", device)); err != nil {
			return err
		}
	}

	if _, err := f.run(fmt.Sprintf("vgcreate %s %s", vg, strings.Join(devices, " "))); err != nil {
		return err
	}

	// Get the size and use this to program the maximum size for
	// the logical volume.
	size, err := f.run(fmt.Sprintf("vgdisplay %s | grep 'Total PE' | awk '{printf $3;}'", vg))
	if err != nil {
		return err
	}

	// Create the logical volume
	if _, err := f.run(fmt.Sprintf("lvcreate --size %s --stripes %d --stripesize=32KiB --name %s %s", size, len(devices), lv, vg)); err != nil {
		return err
	}

	return nil
}

func (f *FileSystemLvm) Delete() error {
	if !f.leader {
		return nil
	}

	vg := f.vg()
	lv := f.lv()

	if _, err := f.run(fmt.Sprintf("lvremove --yes /dev/%s/%s", vg, lv)); err != nil {
		return err
	}

	if _, err := f.run(fmt.Sprintf("vgremove --yes %s", vg)); err != nil {
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
	f.mountpoint = mountpoint
	_, err := f.run(fmt.Sprintf("ln -s /dev/%s/%s %s", f.vg(), f.lv(), mountpoint))
	return err
}

func (f *FileSystemLvm) Unmount() error {
	_, err := f.run(fmt.Sprintf("unlink %s", f.mountpoint))
	return err
}

func (f *FileSystemLvm) vg() string {
	return fmt.Sprintf("%s_vg", f.Name())
}

func (f *FileSystemLvm) lv() string {
	return fmt.Sprintf("%s_lv", f.Name())
}
