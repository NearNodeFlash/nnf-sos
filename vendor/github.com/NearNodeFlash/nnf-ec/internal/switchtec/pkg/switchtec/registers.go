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

type register uint32

const (
	gasMrpcOffset      = 0x0000
	gasTopCfgOffset    = 0x1000
	gasSwEventOffset   = 0x1800
	gasSysInfoOffset   = 0x2000
	gasFlashInfoOffset = 0x2200
	gasPartCfgOffset   = 0x4000
	gasNtbOffset       = 0x10000
	gasPffCsrOffset    = 0x134000
)

const (
	MrpcPayloadSize = 1024
)

type mrpcRegs struct {
	input   [MrpcPayloadSize]byte
	output  [MrpcPayloadSize]byte
	command uint32
	status  uint32
	ret     uint32
}

const (
	mrpcInProgressStatus  = 1
	mrpcDoneStatus        = 2
	mrpcErrorStatus       = 0xff
	mrpcInterruptedStatus = 0x100
)

type topRegs struct {
	bifurValid     uint8
	stackValid     [6]uint8
	partitionCount uint8
	partitionId    uint8
	pffCount       uint8
	pffPort        [255]uint8
}

type sysInfoRegs struct {
	deviceID            uint32
	deviceVersion       uint32
	firmwareVersion     uint32
	_                   uint32
	vendorTableRevision uint32
	tableFormatVersion  uint32
	partitionId         uint32
	cfgFileFmtVersion   uint32
	cfgRunning          uint16
	imgRunning          uint16
	_                   [57]uint32
	vendorID            [8]byte
	productID           [16]byte
	productRevision     [4]byte
	componentVendor     [8]byte
	componentID         uint16
	componentRevision   uint8
}
