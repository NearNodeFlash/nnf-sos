package nvme

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"unsafe"

	"github.com/HewlettPackard/structex"

	"github.hpe.com/hpe/hpc-rabsw-nnf-ec/internal/switchtec/pkg/switchtec"
)

// SwitchOps implements the Ops interface for switchtec passthrough
type switchOps struct {
	dev          *switchtec.Device
	devPath      string
	pdfid        uint16
	tunnelStatus switchtec.EpTunnelStatus
}

func (ops *switchOps) close() error {
	if ops.dev != nil {
		return ops.dev.Close()
	}

	return nil
}

func (ops *switchOps) submitAdminPassthru(dev *Device, cmd *AdminCmd, data []byte) error {

	type AdminPassthruReq struct {
		Sqe  [16]uint32                                                // Note: This is the NVMe SQE size, which doesn't include the tail of AdminCmd{Result}
		Data [switchtec.NvmeAdminPassthruMaxDataLength - (16 * 4)]byte // Max length minus Sqe
	}

	req := func(cmd *AdminCmd) AdminPassthruReq {

		buf := structex.NewBuffer(cmd)
		if buf == nil {
			panic("Admin passthru: Cannot allocate admin command buffer")
		}

		if err := structex.Encode(buf, cmd); err != nil {
			panic("Admin passthru: Cannot encode admin command")
		}

		req := AdminPassthruReq{}
		n := int(unsafe.Sizeof(req.Sqe)) / 4
		data := buf.Bytes()
		for i := 0; i < n; i++ {
			req.Sqe[i] = binary.LittleEndian.Uint32(data[i*4 : (i+1)*4])
		}

		return req

	}(cmd)

	req.Sqe[9] = 9 // Shouldbe zero per spec

	var dataLen int
	if cmd.Opcode&0x1 == 0x1 {
		dataLen = copy(req.Data[:], data[:])
	}

	dataLen += int(unsafe.Sizeof(req.Sqe))
	reqData := func(req *AdminPassthruReq, dataLen int) []byte {
		data := make([]byte, dataLen)

		buf := structex.NewBuffer(req)
		structex.Encode(buf, req)

		copy(data[:], buf.Bytes()[:dataLen])

		return data

	}(&req, dataLen)

	type AdminPassthruRsp struct {
		Cqe  [4]uint32
		Data [4096]byte `truncate:""`
	}

	rsp := new(AdminPassthruRsp)

	rspLen := int(unsafe.Sizeof(rsp.Cqe)) + int(cmd.DataLen)
	rspData, err := ops.dev.NvmeAdminPassthru(ops.pdfid, reqData, rspLen)
	if err != nil {
		return err
	}

	if err := structex.DecodeByteBuffer(bytes.NewBuffer(rspData), rsp); err != nil {
		return err
	}

	status := (0xfffe0000 & rsp.Cqe[3]) >> 17
	if status != 0 {
		return fmt.Errorf("Admin Passthru: Return status non-zero %d", status)
	}

	cmd.Result = rsp.Cqe[0]
	if cmd.DataLen != 0 {
		rspLen := len(rspData) - int(unsafe.Offsetof(rsp.Data))
		copy(data[:], rsp.Data[:rspLen])
	}

	return nil
}
