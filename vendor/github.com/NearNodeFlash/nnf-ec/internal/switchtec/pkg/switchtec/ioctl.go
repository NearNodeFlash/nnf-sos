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
	"fmt"
	"math"
	"unsafe"
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

func _IOC(dir, t, nr, size uintptr) uintptr {
	return (dir << _IOC_DIRSHIFT) |
		(t << _IOC_TYPESHIFT) |
		(nr << _IOC_NRSHIFT) |
		(size << _IOC_SIZESHIFT)
}

func _IOR(t, nr, size uintptr) uintptr  { return _IOC(_IOC_READ, t, nr, size) }
func _IOWR(t, nr, size uintptr) uintptr { return _IOC(_IOC_READ|_IOC_WRITE, t, nr, size) }

// TODO: Source from switchtec-user
type IoctlEventSummary struct {
	globalBitmap         uint64 // Bitmap of global events
	partitionBitmap      uint64 // Bitmap of partitions with active events
	localPartitionBitmap uint32 // Bitmap of events in the local partition
	padding              uint32
	partition            [48]uint32  // Bitmap of events in each partition (stack)
	pff                  [255]uint32 // Bitmap of events in each port function
}

func (sum *IoctlEventSummary) setAllPartitions(bit uint32) {
	for i := range sum.partition {
		sum.partition[i] |= bit
	}
}

func (sum *IoctlEventSummary) setAllPffs(bit uint32) {
	for i := range sum.pff {
		sum.pff[i] |= bit
	}
}

type IoctlEventCtl struct {
	EventId  uint32
	Index    int32
	Flags    uint32
	Occurred uint32
	Count    uint32
	Data     [5]uint32
}

const (
	PffVep int = 100
)

type IoctlPffPort struct {
	Pff       uint32
	Partition int32
	Port      int32
}

var (
	SwitchtecIoctlEventSummary uintptr
	SwitchtecIoctlEventCtl     uintptr
	SwitchtecIoctlPffToPort    uintptr
	SwitchtecIoctlPortToPff    uintptr
)

func init() {
	SwitchtecIoctlEventSummary = _IOR(uintptr('W'), 0x42, unsafe.Sizeof(IoctlEventSummary{}))
	SwitchtecIoctlEventCtl = _IOWR(uintptr('W'), 0x43, unsafe.Sizeof(IoctlEventCtl{}))
	SwitchtecIoctlPffToPort = _IOWR(uintptr('W'), 0x44, unsafe.Sizeof(IoctlPffPort{}))
	SwitchtecIoctlPortToPff = _IOWR(uintptr('W'), 0x45, unsafe.Sizeof(IoctlPffPort{}))

	roundUpToMultiple := func(val int, mul int) int {
		return int(math.Ceil(float64(val)/float64(mul)) * float64(mul))
	}

	var expectedSize int = 0
	expectedSize = 4 /*bytes per uint32*/ * roundUpToMultiple( /*structure long-form uint32 fields*/ 2+2+1+1+48+255, 2 /*rounds up to largest member, which is uint64*/)
	if int(unsafe.Sizeof(IoctlEventSummary{})) != expectedSize {
		panic(fmt.Sprintf("IoctlEventSummary structure is not of expected size. Expected: %d Actual: %d", expectedSize, unsafe.Sizeof(IoctlEventSummary{})))
	}

	expectedSize = 4 /*bytes per uint32*/ * ( /*structure long-form uint32 fields*/ 1 + 1 + 1 + 1 + 1 + 5)
	if int(unsafe.Sizeof(IoctlEventCtl{})) != expectedSize {
		panic(fmt.Sprintf("IoctlEventCtl structure is not of expected size. Expected: %d Actual: %d", expectedSize, unsafe.Sizeof(IoctlEventCtl{})))
	}

	expectedSize = 4 /*bytes per uint32*/ * ( /*structure long-form uint32 fields*/ 1 + 1 + 1)
	if int(unsafe.Sizeof(IoctlPffPort{})) != expectedSize {
		panic(fmt.Sprintf("IoctlPffPort structure is not of expected size. Expected: %d Actual: %d", expectedSize, unsafe.Sizeof(IoctlPffPort{})))
	}

}

// Create an event summary with the specified EventId and the given index.
func newEventSummary(id EventId, index int32) (*IoctlEventSummary, error) {
	sum := new(IoctlEventSummary)

	bit := uint64(1 << id.Bit())
	switch id.Type() {
	case Global_EventType:
		sum.globalBitmap |= bit
	case Partition_EventType:
		if index == LocalPartitionIndex {
			sum.localPartitionBitmap |= uint32(bit)
		} else if index == IndexAll {
			sum.setAllPartitions(uint32(bit))
		} else if index < 0 || int(index) >= len(sum.partition) {
			return nil, fmt.Errorf("Index value overflows partition length")
		} else {
			sum.partition[index] |= uint32(bit)
		}
	case Port_EventType:
		if index == IndexAll {
			sum.setAllPffs(uint32(bit))
		} else if index < 0 || int(index) >= len(sum.pff) {
			return nil, fmt.Errorf("Index value overflows pff length")
		} else {
			sum.pff[index] |= uint32(bit)
		}
	}

	return sum, nil
}
