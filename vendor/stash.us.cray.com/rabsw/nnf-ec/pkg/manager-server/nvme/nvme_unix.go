//go:build linux
// +build linux

package nvme

import (
	"golang.org/x/sys/unix"
)

// https://elixir.bootlin.com/linux/latest/source/include/uapi/asm-generic/ioctl.h
const (
	_IOC_NONE  = 0x0
	_IOC_WRITE = 0x1
	_IOC_READ  = 0x2

	_IOC_NRBITS   = 8
	_IOC_TYPEBITS = 8
	_IOC_SIZEBITS = 14
	_IOC_DIRBITS  = 2

	_IOC_NRSHIFT   = 0
	_IOC_TYPESHIFT = _IOC_NRSHIFT + _IOC_NRBITS
	_IOC_SIZESHIFT = _IOC_TYPESHIFT + _IOC_TYPEBITS
	_IOC_DIRSHIFT  = _IOC_SIZESHIFT + _IOC_SIZEBITS
)

func _IOC(dir uint, t uint, nr uint, size uint) uint {
	return (dir << _IOC_DIRSHIFT) |
		(t << _IOC_TYPESHIFT) |
		(nr << _IOC_NRSHIFT) |
		(size << _IOC_SIZESHIFT)
}

func _IO(t uint, nr uint) uint { return _IOC(_IOC_NONE, t, nr, 0) }

// https://github.com/linux-nvme/nvme-cli/blob/master/linux/nvme_ioctl.h
var _NVME_IOCTL_ID = func() uint { return _IO('N', 0x40) }()

func GetNamespaceId(path string) (int, error) {
	fd, err := unix.Open(path, unix.O_RDONLY, 0)
	if err != nil {
		return -1, err
	}
	defer unix.Close(fd)

	return unix.IoctlRetInt(fd, _NVME_IOCTL_ID)
}
