package switchtec

import (
	"unsafe"

	"github.com/HewlettPackard/structex"
)

const (
	NvmeAdminPassthruMaxDataLength = 4096 + 16*4
)

// NvmeAdminPassthru will send an ADMIN PASSTHRU command to device and get reply
// `pdfid` is the Physical Device Function Identifier
// `data` is payload transmitted to the device
// `expRspLen` is the expected response length from the command
func (dev *Device) NvmeAdminPassthru(pdfid uint16, data []byte, expRspLen int) ([]byte, error) {

	dev.Lock()
	defer dev.Unlock()

	rspLen, err := dev.adminPassthruStart(pdfid, data, expRspLen)
	if err != nil {
		return nil, err
	}

	rsp := make([]byte, 0)
	if rspLen != 0 {
		rsp, err = dev.adminPassthruData(pdfid, rspLen)
		if err != nil {
			return nil, err
		}
	}

	err = dev.adminPassthruEnd(pdfid)

	return rsp, err
}

func (dev *Device) adminPassthruStart(pdfid uint16, data []byte, expRspLen int) (uint16, error) {
	type passthruStartCmd struct {
		Subcmd         uint8
		Reserved0      [3]uint8
		Pdfid          uint16
		ExpectedRspLen uint16
		MoreData       uint8
		Reserved1      [3]uint8
		DataOffset     uint16
		DataLen        uint16
		Data           [maxDataLength - 16]byte
	}

	cmd := passthruStartCmd{
		Subcmd: uint8(NvmeAdminPassthruStart),
		Pdfid:  pdfid,
	}

	if sz, _ := structex.Size(cmd); sz != maxDataLength {
		panic("adminPassthruStart cmd is incorrect size, use structure packing")
	}

	dataOffset := 0

	if data != nil && len(data) != 0 {

		cmd.MoreData = 0
		if len(data) > int(unsafe.Sizeof(cmd.Data)) {
			cmd.MoreData = 1
		}

		for cmd.MoreData != 0 {
			dataLen := copy(cmd.Data[:], data[dataOffset:])
			cmd.DataOffset = uint16(dataOffset)
			cmd.DataLen = uint16(dataLen)

			if err := dev.RunCommand(NvmeAdminPassthruCommand, &cmd, nil); err != nil {
				return 0, err
			}

			dataOffset += dataLen
			cmd.MoreData = 0
			if len(data)-dataOffset > int(unsafe.Sizeof(cmd.Data)) {
				cmd.MoreData = 1
			}
		}

		if dataOffset < len(data) {
			dataLen := copy(cmd.Data[:], data[dataOffset:])

			cmd.DataOffset = uint16(dataOffset)
			cmd.DataLen = uint16(dataLen)
		} else {
			cmd.DataOffset = 0
			cmd.DataLen = 0
		}
	}

	type passthruStartRsp struct {
		RspLen    uint16
		Reserved0 uint16
	}

	rsp := new(passthruStartRsp)

	if sz, _ := structex.Size(rsp); sz != 4 {
		panic("adminPassthruStart rsp is incorrect size, use structure packing")
	}

	cmd.ExpectedRspLen = uint16(expRspLen)

	if err := dev.RunCommand(NvmeAdminPassthruCommand, &cmd, rsp); err != nil {
		return 0, err
	}

	return rsp.RspLen, nil
}

func (dev *Device) adminPassthruData(pdfid uint16, rspLen uint16) ([]byte, error) {

	type passthruDataCmd struct {
		SubCommand uint8
		Reserved0  [3]uint8
		Pdfid      uint16
		Offset     uint16
	}

	cmd := passthruDataCmd{
		SubCommand: uint8(NvmeAdminPassthruData),
		Pdfid:      pdfid,
		Offset:     0,
	}

	if sz, _ := structex.Size(cmd); sz != 8 {
		panic("adminPassthruData cmd is incorrect size, use structure packing")
	}

	type passthruDataRsp struct {
		Offset uint16
		Len    uint16
		Data   [maxDataLength - 4]byte
	}

	rsp := new(passthruDataRsp)

	if sz, _ := structex.Size(rsp); sz != maxDataLength {
		panic("adminPassthruData rsp is incorrect size, use structure packing")
	}

	rspData := make([]byte, rspLen)

	for offset := uint16(0); offset < rspLen; offset += rsp.Len {
		cmd.Offset = offset

		if err := dev.RunCommand(NvmeAdminPassthruCommand, cmd, rsp); err != nil {
			return nil, err
		}

		copy(rspData[offset:], rsp.Data[:rsp.Len])
	}

	return rspData, nil
}

func (dev *Device) adminPassthruEnd(pdfid uint16) error {
	type passthruEndCmd struct {
		SubCommand uint8
		Reserved0  [3]uint8
		Pdfid      uint16
		Reserved1  uint16
	}

	cmd := passthruEndCmd{
		SubCommand: uint8(NvmeAdminPassthruEnd),
		Pdfid:      pdfid,
	}

	if sz, _ := structex.Size(cmd); sz != 8 {
		panic("adminPassthruEnd cmd is incorrect size, use structure packing")
	}

	return dev.RunCommand(NvmeAdminPassthruCommand, cmd, nil)
}
