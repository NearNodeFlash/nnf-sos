package server

import (
	"fmt"
	"path/filepath"
	"strings"
)

type FileSystemLvm struct {
	// Satisfy FileSystemApi interface.
	FileSystem
}

func init() {
	FileSystemRegistry.RegisterFileSystem(&FileSystemLvm{})
}

func (*FileSystemLvm) New(oem FileSystemOem) FileSystemApi {
	return &FileSystemLvm{FileSystem: FileSystem{name: oem.Name}}
}

func (*FileSystemLvm) IsType(oem FileSystemOem) bool { return oem.Type == "lvm" }
func (*FileSystemLvm) IsMockable() bool              { return false }

func (*FileSystemLvm) Type() string   { return "lvm" }
func (f *FileSystemLvm) Name() string { return f.name }

func (f *FileSystemLvm) Create(devices []string, opts FileSystemOptions) error {

	volumeGroup := f.volumeGroup()
	logicalVolume := f.logicalVolume()

	// TODO: Some sort of rollback mechanism on failure condition

	// Ensure the existing volume groups are scanned. Use this information to determine
	// if initialization is required for the Volume Group, or if it exists and can be
	// activated.
	if _, err := f.run("vgscan"); err != nil {
		return err
	}

	rsp, _ := f.run(fmt.Sprintf("lvdisplay %s || echo 'not found'", volumeGroup))
	if len(rsp) != 0 && !strings.Contains(string(rsp), "not found") {
		// Volume Group is present, activate the volume
		if _, err := f.run(fmt.Sprintf("vgchange --activate y %s", volumeGroup)); err != nil {
			return err
		}

		return nil
	}

	// Create the physical volumes.
	for _, device := range devices {
		if _, err := f.run(fmt.Sprintf("pvcreate %s", device)); err != nil {
			return err
		}
	}

	if _, err := f.run(fmt.Sprintf("vgcreate %s %s", volumeGroup, strings.Join(devices, " "))); err != nil {
		return err
	}

	// Get the size and use this to program the maximum size for
	// the logical volume.
	size, err := f.run(fmt.Sprintf("vgdisplay %s | grep 'Total PE' | awk '{printf $3;}'", volumeGroup))
	if err != nil {
		return err
	}

	// Create the logical volume
	// -Zn - don't zero the volume, it will fail.
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
	if _, err := f.run(fmt.Sprintf("lvcreate -Zn --size %s --stripes %d --stripesize=32KiB --name %s %s", size, len(devices), logicalVolume, volumeGroup)); err != nil {
		return err
	}

	// Activate the volume group.
	if _, err := f.run(fmt.Sprintf("vgchange --activate y %s", volumeGroup)); err != nil {
		return err
	}

	return nil
}

func (f *FileSystemLvm) Delete() error {
	if _, err := f.run(fmt.Sprintf("lvremove --yes %s", f.devPath())); err != nil {
		return err
	}

	if _, err := f.run(fmt.Sprintf("vgremove --yes %s", f.volumeGroup())); err != nil {
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
	_, err := f.run(fmt.Sprintf("ln -s %s %s", f.devPath(), mountpoint))
	return err
}

func (f *FileSystemLvm) Unmount() error {
	_, err := f.run(fmt.Sprintf("unlink %s", f.mountpoint))
	return err
}

func (f *FileSystemLvm) devPath() string {
	return filepath.Join("/dev", f.volumeGroup(), f.logicalVolume())
}

func (f *FileSystemLvm) volumeGroup() string {
	return fmt.Sprintf("%s_vg", f.Name())
}

func (f *FileSystemLvm) logicalVolume() string {
	return fmt.Sprintf("%s_lv", f.Name())
}
