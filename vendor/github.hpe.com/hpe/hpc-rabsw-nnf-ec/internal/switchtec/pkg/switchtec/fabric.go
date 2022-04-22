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
	"bytes"
	"encoding/binary"
	"errors"
	"unsafe"

	"github.com/HewlettPackard/structex"
)

const (
	dumpFabric = iota
	dumpPaxAll
	dumpPax
	dumpHvd
	dumpFabPort
	dumpEpPort
	dumpHvdDetail
)

// DumpSectionHeader is returned as part of every dump section
type DumpSectionHeader struct {
	SectionClass uint8
	PAXIndex     uint8
	SoftwareFID  uint16
	ResponseSize uint32
	Rsvd         uint32
} // 12 bytes

// DumpEpPortHeader is returned as part of a Ep Port Dump
type DumpEpPortHeader struct {
	Typ          uint8
	PhysicalPort uint8
	Count        uint16
	Size         uint32 `bitfield:"24"`
	Rsvd         uint32 `bitfield:"8,reserved"`
}

// EpPortType describes the type of return data for an GFMS EP Dump
type EpPortType uint8

const (
	DeviceEpPortType EpPortType = 0
	SwitchEpPortType EpPortType = 1
	NoneEpPortType   EpPortType = 2
)

type DumpEpPortAttachmentHeader struct {
	FunctionCount     uint16 `countOf:"Functions"`
	AttachedDSPEnumID uint16
	Size              uint32 `bitfield:"24"`
	Rsvd              uint32 `bitfield:"8,reserved"`
}

type DumpEpPortAttachedDeviceFunction struct {
	FunctionID     uint16
	PDFID          uint16
	SRIOVCapPF     uint8
	VFNum          uint8
	Rsvd           uint16
	Bound          uint8
	BoundPAXID     uint8
	BoundHVDPhyPID uint8
	BoundHVDLogPID uint8
	VID            uint16
	DID            uint16
	SubSysVID      uint16
	SubSysDID      uint16
	DeviceClass    uint32 `bitfield:"24"`
	BARNumber      uint32 `bitfield:"8"`
	BAR            [6]struct {
		Typ  uint8
		Size uint8
	}
}

// DumpEpPortDevice is returned after a DumpSectionHeader of type Device(0)
type DumpEpPortEp struct {
	Hdr       DumpEpPortAttachmentHeader
	Functions []DumpEpPortAttachedDeviceFunction
}

//
// DumpEpPort Type specific return types
//

// DumpEpPortDevice is returned when DumpEpPortHeader.Type is DeviceEpPortType
type DumpEpPortDevice struct {
	Section DumpSectionHeader
	Hdr     DumpEpPortHeader
	Ep      DumpEpPortEp
}

// DumpEpPortSwitch is returned when DumpEpPortHeader.Type is SwitchEpPortType
type DumpEpPortSwitch struct {
	// TODO
}

// DumpEpPortNone is returned when  DumpEpPortHeader.Type is NoneEpPortType
type DumpEpPortNone struct {
	Section DumpSectionHeader
	Hdr     DumpEpPortHeader
}

// Bind does some stuff
func (dev *Device) Bind(hostSwIdx uint8, hostPhysPortID uint8, hostLogPortID uint8, pdfid uint16) error {

	type function struct {
		Pdfid     uint16
		NextValid uint8
		Reserved  uint8
	}
	cmd := struct {
		SubCmd         uint8
		HostSwIdx      uint8
		HostPhysPortID uint8
		HostLogPortID  uint8
		Functions      [8]function
	}{
		SubCmd:         uint8(PortBindSubCommand),
		HostSwIdx:      hostSwIdx,
		HostPhysPortID: hostPhysPortID,
		HostLogPortID:  hostLogPortID,
		Functions: [8]function{
			{
				Pdfid:     pdfid,
				NextValid: 0,
			},
		},
	}

	return dev.RunCommand(BindUnbindCommand, &cmd, nil)
}

// Unbind does some unbind stuff
func (dev *Device) Unbind(hostSwIdx uint8, hostPhysPortID uint8, hostLogPortID uint8) error {
	cmd := struct {
		subCmd         uint8
		hostSwIdx      uint8
		hostPhysPortID uint8
		hostLogPortID  uint8
		pdfid          uint16
		option         uint8
		reserved       uint8
	}{
		subCmd:         uint8(PortUnbindSubCommand),
		hostSwIdx:      hostSwIdx,
		hostPhysPortID: hostPhysPortID,
		hostLogPortID:  hostLogPortID,
	}

	return dev.RunCommand(BindUnbindCommand, cmd, nil)
}

func (dev *Device) GfmsEpPortStart(pid uint8) (uint32, error) {
	cmd := struct {
		subCmd uint8
		pid    uint8
		rsvd   uint16
		typ    uint32
	}{
		subCmd: dumpEpPort,
		pid:    pid,
		typ:    GfmsDumpStartSubCommand,
	}

	type response struct {
		Len uint32
		Num uint32
	}

	var rsp = new(response)

	err := dev.RunCommand(DumpCommand, cmd, rsp)
	len := rsp.Len

	return len, err
}

func (dev *Device) GfmsEpPortGet(pid uint8, len uint32) (*bytes.Buffer, error) {
	cmd := struct {
		subCmd   uint8
		pid      uint8
		reserved uint16
		typ      uint32
		offset   uint32
	}{
		subCmd: dumpEpPort,
		pid:    pid,
		typ:    GfmsDumpGetSubCommand,
		offset: 0,
	}

	type response struct {
		Offset   uint32
		Size     uint32
		Reserved uint32
		Data     [maxDataLength - 12]byte
	}

	var rsp = new(response)

	var ret = new(bytes.Buffer)
	for offset := uint32(0); offset < len; offset += rsp.Size {

		rsp.Offset = offset

		err := dev.RunCommand(DumpCommand, cmd, rsp)
		if err != nil {
			return ret, err
		}

		rsp.Size -= 3 // account for "offset, size, reserved" dword fields

		ret.Write(rsp.Data[:rsp.Size*4])
	}

	return ret, nil
}

func (dev *Device) GfmsEpPortFinish() error {
	cmd := struct {
		subCmd   uint8
		reserved [3]uint8
		typ      uint32
	}{
		subCmd: dumpEpPort,
		typ:    GfmsDumpFinishSubCommand,
	}

	return dev.RunCommand(DumpCommand, cmd, nil)
}

var (
	EpPortNotDevice = errors.New("End-point is not a device")
)

func (dev *Device) GfmsEpPortDeviceEnumerate(pid uint8, f func(*DumpEpPortDevice) error) error {
	l, err := dev.GfmsEpPortStart(pid)
	if err != nil {
		return err
	}

	defer dev.GfmsEpPortFinish()

	buf, err := dev.GfmsEpPortGet(pid, l)
	if err != nil {
		return err
	}

	// Need to take special consideration when a device is not attached
	// as structex can't decode dynamic types and a DumpEpPort can return
	// Device, Switch, or None.
	//
	// When None type is returned, structex fails to parse the full type
	// so we default to decoding a none-type, then expand for the
	// particular Device type.
	var bufCpy bytes.Buffer = *buf // Buffer copy so it can be used again later

	epPort := new(DumpEpPortNone)
	if err := structex.DecodeByteBuffer(buf, epPort); err != nil {
		return err
	}

	switch EpPortType(epPort.Hdr.Typ) {
	case NoneEpPortType:
		return f(&DumpEpPortDevice{
			Section: epPort.Section,
			Hdr:     epPort.Hdr,
		})
	case SwitchEpPortType:
		return EpPortNotDevice
	case DeviceEpPortType:
		epPortDevice := new(DumpEpPortDevice)

		if err := structex.DecodeByteBuffer(&bufCpy, epPortDevice); err != nil {
			return err
		}

		return f(epPortDevice)
	}

	return EpPortNotDevice
}

type EpTunnelSubCommand uint16

const (
	EnableEpTunnelSubCommand  EpTunnelSubCommand = 0
	DisableEpTunnelSubCommand EpTunnelSubCommand = 1
	StatusEpTunnelSubCommand  EpTunnelSubCommand = 2
)

type EpTunnelStatus uint8

const (
	DisabledEpTunnelStatus EpTunnelStatus = 0
	EnabledEpTunnelStatus  EpTunnelStatus = 1
	UnknownEpTunnelStatus  EpTunnelStatus = 2
)

func (dev *Device) endPointTunnelConfig(subcmd EpTunnelSubCommand, pdfid uint16, expResponseLen uint16) ([]byte, error) {

	cmd := struct {
		Subcmd         uint16
		Pdfid          uint16
		ExpResponseLen uint16
		// TODO: I don't see a need for the Metadata paramete
		//       for now I keep this out of structex
		MetadataLen uint16 //`countOf:"Metadata"`
		//Metadata       [MrpcPayloadSize - 8]byte
	}{
		Subcmd:         uint16(subcmd),
		Pdfid:          pdfid,
		ExpResponseLen: expResponseLen,
		MetadataLen:    0,
	}

	type response struct {
		Len  uint32 `countOf:"Data"`
		Data [MrpcPayloadSize - 4]byte
	}

	rsp := new(response)

	if err := dev.RunCommand(EpTunnelConfigCommand, cmd, rsp); err != nil {
		return nil, err
	}

	if rsp.Len != 0 {
		return rsp.Data[0:rsp.Len], nil
	}

	return nil, nil
}

func (dev *Device) EndPointTunnelEnable(pdfid uint16) error {
	_, err := dev.endPointTunnelConfig(EnableEpTunnelSubCommand, pdfid, 0)
	return err
}

func (dev *Device) EndPointTunnelDisable(pdfid uint16) error {
	_, err := dev.endPointTunnelConfig(DisableEpTunnelSubCommand, pdfid, 0)
	return err
}

func (dev *Device) EndPointTunnelStatus(pdfid uint16) (EpTunnelStatus, error) {
	var status uint32 = 0
	rsp, err := dev.endPointTunnelConfig(StatusEpTunnelSubCommand, pdfid, uint16(unsafe.Sizeof(status)))
	if err != nil {
		return UnknownEpTunnelStatus, err
	}

	status = binary.LittleEndian.Uint32(rsp)
	return EpTunnelStatus(status), nil
}
