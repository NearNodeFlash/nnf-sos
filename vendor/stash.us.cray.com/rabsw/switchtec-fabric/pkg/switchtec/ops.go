package switchtec

import (
	"encoding/binary"
	"fmt"
	"os"
	"path"
	"strconv"
	"syscall"
	"unsafe"
)

// ops defines the device operations unique to the particular transport type.
//  - character device (i.e. /dev/switchtec0)
//  - UART (i.e. /dev/ttyUSB0)
//  - others (i.e. ethernet, i2c)
type ops interface {
	close(dev *Device) error
	getDeviceID(dev *Device) (uint32, error)
	submitCommand(dev *Device, cmd Command, payload []byte) error
	readResponse(dev *Device, response []byte) error

	gasMap(dev *Device, readonly bool) error
	gasUnmap(dev *Device) error

	// GAS direct access
	gasWrite(dev *Device, offset int, data []byte) error
	gasRead(dev *Device, offset int, size int) ([]byte, error)

	// GAS MRPC access
	gasWriteMrpcPayload(dev *Device, data []byte) error
	gasReadMrpcPayload(dev *Device, size int) ([]byte, error)

	// GAS uint32 access shortcuts
	gasRead32(dev *Device, offset int) (uint32, error)
	gasWrite32(dev *Device, offset int, data uint32) error
}

type gasMap struct {
	config    *[]byte
	resources *[]byte
	size      int64
}

type charOps struct {
	gas *gasMap
}

func (ops *charOps) close(dev *Device) error {
	if ops.gas != nil {
		ops.gasUnmap(dev)
	}

	return nil
}

func (ops *charOps) getDeviceID(dev *Device) (uint32, error) {
	linkPath := fmt.Sprintf("%s/%s/device/device", "/sys/class/switchtec", path.Base(dev.name))
	ret, err := sysFsReadInt(linkPath, 16)
	return uint32(ret), err
}

func (ops *charOps) submitCommand(dev *Device, cmd Command, payload []byte) error {
	cmd = Command(int32(cmd) | (dev.paxID << PAXIdShift))

	sz := len(payload) + int(unsafe.Sizeof(cmd))
	buf := make([]byte, sz)

	binary.LittleEndian.PutUint32(buf[:], uint32(cmd))

	copy(buf[unsafe.Sizeof(cmd):], payload)

	_, err := dev.file.Write(buf)

	return err
}

func (ops *charOps) readResponse(dev *Device, response []byte) error {
	var ret uint32 = 0

	sz := len(response) + int(unsafe.Sizeof(ret))
	buf := make([]byte, sz)

	if _, err := dev.file.Read(buf); err != nil {
		return err
	}

	ret = binary.LittleEndian.Uint32(buf[0:4])
	if ret != 0 {
		return fmt.Errorf("Failed Command: Response Code %#x (%d)", ret, ret)
	}

	copy(response, buf[unsafe.Sizeof(ret):])

	return nil
}

func (ops *charOps) gasMap(dev *Device, readonly bool) error {
	size, err := dev.ResourceSize("device/resource0")
	if err != nil {
		return err
	}

	config, err := dev.MapResource("device/resource0_wc", 0, gasTopCfgOffset, readonly)
	if err != nil {
		return err
	}

	resources, err := dev.MapResource("device/resource0", gasTopCfgOffset, uint64(size-gasTopCfgOffset), readonly)
	if err != nil {
		return err
	}

	ops.gas = &gasMap{
		config:    config,
		resources: resources,
		size:      size,
	}

	return nil
}

func (gas *gasMap) close() {
	gas.config = nil
	gas.resources = nil
	gas.size = -1
}

func (ops *charOps) gasUnmap(dev *Device) error {
	defer ops.gas.close()
	if err := syscall.Munmap(*(ops.gas.config)); err != nil {
		return err
	}
	if err := syscall.Munmap(*(ops.gas.resources)); err != nil {
		return err
	}

	return nil
}

func (ops *charOps) gasWrite(dev *Device, offset int, data []byte) error {
	if offset < gasTopCfgOffset {
		if offset+len(data) > gasTopCfgOffset {
			return fmt.Errorf("GAS Write cannot span individual resoureces")
		}

		copy((*ops.gas.config)[offset:], data)
	} else {
		copy((*ops.gas.resources)[offset-gasTopCfgOffset:], data)
	}

	return nil
}

func (ops *charOps) gasWriteMrpcPayload(dev *Device, data []byte) error {
	return ops.gasWrite(dev, gasMrpcOffset, data)
}

func (ops *charOps) gasWrite32(dev *Device, offset int, data uint32) error {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, data)
	return ops.gasWrite(dev, offset, b)
}

func (ops *charOps) gasRead(dev *Device, offset int, size int) ([]byte, error) {
	b := make([]byte, size)
	if offset < gasTopCfgOffset {
		if offset+size > gasTopCfgOffset {
			return nil, fmt.Errorf("GAS Read cannot span individual resources")
		}

		copy(b, (*ops.gas.config)[offset:])
	} else {
		copy(b, (*ops.gas.resources)[offset-gasTopCfgOffset:])
	}

	return b, nil
}

func (ops *charOps) gasReadMrpcPayload(dev *Device, size int) ([]byte, error) {
	return ops.gasRead(dev, gasMrpcOffset, size)
}

func (ops *charOps) gasRead32(dev *Device, offset int) (uint32, error) {
	ret, err := ops.gasRead(dev, offset, 4)
	return binary.LittleEndian.Uint32(ret), err
}

func (dev *Device) ResourceSize(resource string) (int64, error) {
	path, err := dev.SystemPath(resource)
	if err != nil {
		return 0, err
	}

	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return 0, err
	}

	return info.Size(), err
}

func (dev *Device) SystemPath(resource string) (string, error) {
	var stat syscall.Stat_t

	/*
		Translation from C to GO of the huge_encode_dev/huge_decode_dev
		routines. This is some hacky level shit here.
		here: https://github.com/torvalds/linux/blob/master/include/linux/kdev_t.h#L9
	*/

	major := func(stat syscall.Stat_t) int {
		return int((stat.Rdev & 0xfff00) >> 8)
	}

	minor := func(stat syscall.Stat_t) int {
		return int((stat.Rdev & 0xff) | ((stat.Rdev >> 12) & 0xfff00))
	}

	if err := syscall.Fstat(int(dev.file.Fd()), &stat); err != nil {
		return "", err
	}

	return fmt.Sprintf("/sys/dev/char/%d:%d/%s", major(stat), minor(stat), resource), nil
}

func (dev *Device) MapResource(path string, offset uint64, length uint64, readonly bool) (*[]byte, error) {
	path, err := dev.SystemPath(path)
	if err != nil {
		return nil, err
	}

	openFlags := func(readonly bool) int {
		if readonly {
			return syscall.O_RDONLY
		}
		return syscall.O_WRONLY
	}

	file, err := os.OpenFile(path, openFlags(readonly), 0)
	if err != nil {
		fmt.Printf("Cannot open file %s\n", path)
		return nil, err
	}
	defer file.Close()

	mmapFlags := func(readonly bool) int {
		if readonly {
			return syscall.PROT_READ
		}
		return syscall.PROT_WRITE
	}

	mmap, err := syscall.Mmap(int(file.Fd()), int64(offset), int(length), mmapFlags(readonly), syscall.MAP_SHARED)
	if err != nil {
		fmt.Printf("Cannot mmap file %s\n", path)
		return nil, err
	}

	return &mmap, nil
}

func sysFsReadString(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	buf := make([]byte, 1024)
	n, err := f.Read(buf)
	if err != nil {
		return "", err
	}

	return string(buf[:n]), err
}

func sysFsReadInt(path string, base int) (int64, error) {
	str, err := sysFsReadString(path)
	if err != nil {
		return -1, err
	}

	return strconv.ParseInt(str, base, 64)
}
