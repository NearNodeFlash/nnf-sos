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

package switchtec

import (
	"encoding/binary"
	"fmt"
)

const (
	csrMaxReadWriteLengthInBytes = 4

	csrReadSubCommand  uint8 = 0
	csrWriteSubCommand uint8 = 1
)

func (dev *Device) CsrRead(pdfid uint16, addr uint16, length uint8) ([]byte, error) {

	if length > csrMaxReadWriteLengthInBytes {
		return nil, fmt.Errorf("csr read of length %d exceeds limit %d", length, csrMaxReadWriteLengthInBytes)
	}

	cmd := struct {
		SubCmd    uint8
		Reserved0 uint8
		Pdfid     uint16
		Addr      uint16
		Length    uint8
		Reserved1 uint8
	}{
		SubCmd: csrReadSubCommand,
		Pdfid:  pdfid,
		Addr:   addr,
		Length: length,
	}

	rsp := struct {
		Data [4]byte
	}{}

	if err := dev.RunCommand(EpResourceAccessCommand, &cmd, &rsp); err != nil {
		return nil, err
	}


	return rsp.Data[0:length], nil
}

func (dev *Device) CsrRead8(pdfid uint16, addr uint16) (uint8, error) {
	data, err := dev.CsrRead(pdfid, addr, 1)
	if err != nil {
		return 0, err
	}

	return data[0], err
}

func (dev *Device) CsrRead16(pdfid uint16, addr uint16) (uint16, error) {
	data, err := dev.CsrRead(pdfid, addr, 2)
	if err != nil {
		return 0, err
	}

	return binary.LittleEndian.Uint16(data), nil
}

func (dev *Device) CsrRead32(pdfid uint16, addr uint16) (uint32, error) {
	data, err := dev.CsrRead(pdfid, addr, 4)
	if err != nil {
		return 0, err
	}

	return binary.LittleEndian.Uint32(data), nil
}

func (dev *Device) CsrWrite(pdfid uint16, addr uint16, payload []byte) error {

	length := len(payload)
	if length > csrMaxReadWriteLengthInBytes {
		return fmt.Errorf("csr write of length %d exceeds limit %d", length, csrMaxReadWriteLengthInBytes)
	}

	cmd := struct {
		SubCmd    uint8
		Reserved0 uint8
		Pdfid     uint16
		Addr      uint16
		Length    uint8
		Reserved1 uint8
		Data      [4]byte
	}{
		SubCmd: csrWriteSubCommand,
		Pdfid:  pdfid,
		Addr:   addr,
		Length: uint8(length),
	}

	copy(cmd.Data[:], payload)

	return dev.RunCommand(EpResourceAccessCommand, &cmd, nil)
}

func (dev *Device) CsrWrite8(pdfid uint16, addr uint16, data uint8) error {
	payload := [1]byte{data}

	return dev.CsrWrite(pdfid, addr, payload[:])
}

func (dev *Device) CsrWrite16(pdfid uint16, addr uint16, data uint16) error {
	payload := [2]byte{}
	binary.LittleEndian.PutUint16(payload[:], data)

	return dev.CsrWrite(pdfid, addr, payload[:])
}

func (dev *Device) CsrWrite32(pdfid uint16, addr uint16, data uint32) error {
	payload := [4]byte{}
	binary.LittleEndian.PutUint32(payload[:], data)

	return dev.CsrWrite(pdfid, addr, payload[:])
}
