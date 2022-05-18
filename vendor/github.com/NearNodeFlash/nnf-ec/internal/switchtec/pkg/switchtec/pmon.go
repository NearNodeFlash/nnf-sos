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
	"unsafe"
)

type BandwidthType int

const (
	Raw_BandwidthType     BandwidthType = 0
	Payload_BandwidthType               = 1
)

type BandwidthCounterDirectional struct {
	Posted     uint64
	Completion uint64
	NonPosted  uint64
}

type BandwidthCounter struct {
	TimeInMicroseconds uint64
	Egress             BandwidthCounterDirectional
	Ingress            BandwidthCounterDirectional
}

func (from *BandwidthCounter) Subtract(c *BandwidthCounter) {
	from.TimeInMicroseconds -= c.TimeInMicroseconds
	from.Egress.Posted -= c.Egress.Posted
	from.Egress.NonPosted -= c.Egress.NonPosted
	from.Egress.Completion -= c.Egress.Completion
	from.Ingress.Posted -= c.Ingress.Posted
	from.Ingress.NonPosted -= c.Ingress.NonPosted
	from.Ingress.Completion -= c.Ingress.Completion
}

func (d *BandwidthCounterDirectional) Total() uint64 {
	return d.Posted + d.NonPosted + d.Completion
}

// Set Bandwidth Type for particular physical ports in the system
func (dev *Device) BandwidthCounterSetMany(physPortIds []int, bwType BandwidthType) error {

	type port struct {
		Id     uint8
		BwType uint8
	}
	cmd := struct {
		SubCmd uint8
		Count  uint8          `structex:"countOf='Ports'"`
		Ports  [maxPorts]port `structex:"truncate"`
	}{
		SubCmd: uint8(SetBandwidthCounterSubCommand),
		Count:  uint8(len(physPortIds)),
	}

	for i := range physPortIds {
		cmd.Ports[i] = port{
			Id:     uint8(physPortIds[i]),
			BwType: uint8(bwType),
		}
	}

	return dev.RunCommand(PerformanceMonitorCommand, cmd, nil)
}

// Set Bandwidth Type performance monitor for all ports in the system
func (dev *Device) BandwidthCounterSetAll(bwType BandwidthType) error {

	ports, err := dev.LinkStat()
	if err != nil {
		return err
	}

	ids := make([]int, len(ports))
	for i := range ports {
		ids[i] = int(ports[i].PhysPortId)
	}

	return dev.BandwidthCounterSetMany(ids, bwType)
}

type BandwidthCounterPort struct {
	Id    uint8
	Clear uint8
}

type BandwidthCounterCommand struct {
	SubCmd uint8
	Count  uint8                          `structex:"countOf='Ports'"`
	Ports  [maxPorts]BandwidthCounterPort `structex:"truncate"`
}

const (
	MaxBandwidthCounterPorts = uint8((maxDataLength - int(unsafe.Sizeof(BandwidthCounterCommand{}))) / int(unsafe.Sizeof(BandwidthCounter{})))
)

func (dev *Device) BandwidthCounterMany(physPortIds []uint8, clear bool) ([]BandwidthCounter, error) {

	cmd := BandwidthCounterCommand{
		SubCmd: uint8(GetBandwidthCounterSubCommand),
	}

	remain := len(physPortIds)
	physPortIdIdx := 0

	clearFunc := func(clear bool) uint8 {
		if clear {
			return 1
		}
		return 0
	}

	results := []BandwidthCounter{}

	for remain != 0 {
		cmd.Count = uint8(remain)
		if cmd.Count > MaxBandwidthCounterPorts {
			cmd.Count = MaxBandwidthCounterPorts
		}

		for i := 0; i < int(cmd.Count); i++ {
			cmd.Ports[i] = BandwidthCounterPort{
				Id:    physPortIds[physPortIdIdx+i],
				Clear: clearFunc(clear),
			}
		}

		type PerformanceMonitorResponse struct {
			Counters [MaxBandwidthCounterPorts]BandwidthCounter
		}

		rsp := PerformanceMonitorResponse{}

		if err := dev.RunCommand(PerformanceMonitorCommand, cmd, &rsp); err != nil {
			return nil, err
		}

		remain -= int(cmd.Count)
		physPortIdIdx += int(cmd.Count)

		results = append(results, rsp.Counters[:cmd.Count]...)
	}

	return results, nil
}

func (dev *Device) BandwidthCounterAll(clear bool) ([]PortId, []BandwidthCounter, error) {

	ports, err := dev.Status()
	if err != nil {
		return nil, nil, err
	}

	physPortIds := make([]uint8, len(ports))
	for i := range ports {
		physPortIds[i] = ports[i].PhysPortId
	}

	results, err := dev.BandwidthCounterMany(physPortIds, clear)

	return ports, results, err
}
