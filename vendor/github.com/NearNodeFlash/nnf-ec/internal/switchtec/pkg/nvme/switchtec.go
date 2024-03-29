/*
 * Copyright 2020, 2021, 2022 Hewlett Packard Enterprise Development LP
 * Other additional copyright holders may be indicated within.
 *
 * The entirety of this work is licensed under the Apache License,
 * Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License.
 *
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package nvme

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"unsafe"

	"github.com/HewlettPackard/structex"

	"github.com/NearNodeFlash/nnf-ec/internal/switchtec/pkg/switchtec"
)

// SwitchOps implements the Ops interface for switchtec passthrough
type switchOps struct {
	dev          *switchtec.Device
	devPath      string
	pdfid        uint16
	tunnelStatus switchtec.EpTunnelStatus
}

type SwitchtecNvmeCommandError struct {
	dev *Device
	cmd *AdminCmd
	err error
}

func newSwitchtecNvmeCommandError(dev *Device, cmd *AdminCmd, err error) *SwitchtecNvmeCommandError {
	return &SwitchtecNvmeCommandError{
		dev: dev, cmd: cmd, err: err,
	}
}

func (e *SwitchtecNvmeCommandError) Error() string {
	return fmt.Sprintf("Device %s: Failed NVMe Command: %s: Error: %s", e.dev, e.cmd, e.err)
}

func (e *SwitchtecNvmeCommandError) Unwrap() error {
	return e.err
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
		return newSwitchtecNvmeCommandError(dev, cmd, err)
	}

	if err := structex.DecodeByteBuffer(bytes.NewBuffer(rspData), rsp); err != nil {
		return err
	}

	status := (0xfffe0000 & rsp.Cqe[3]) >> 17
	if status != 0 {
		return newSwitchtecNvmeCommandError(dev, cmd, newCommandError(status))
	}

	cmd.Result = rsp.Cqe[0]
	if cmd.DataLen != 0 {
		rspLen := len(rspData) - int(unsafe.Offsetof(rsp.Data))
		copy(data[:], rsp.Data[:rspLen])
	}

	return nil
}
