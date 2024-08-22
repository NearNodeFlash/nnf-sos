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
	"math/bits"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

type EventType int

const (
	Invalid_EventType   EventType = -1
	Global_EventType              = 0
	Partition_EventType           = 1
	Port_EventType                = 2
)

func (e EventType) Name() string {
	switch e {
	case Global_EventType:
		return "Global"
	case Partition_EventType:
		return "Partition"
	case Port_EventType:
		return "Port"
	default:
		return "Invalid"
	}
}

type EventBit uint64

type EventId int

const (
	StackError_EventId       EventId = 0
	PpuError_EventId                 = 1
	IspError_EventId                 = 2
	SysReset_EventId                 = 3
	FwExc_EventId                    = 4
	FwNmi_EventId                    = 5
	FwNonFatal_EventId               = 6
	FwFatal_EventId                  = 7
	TwiMrpcComp_EventId              = 8
	TwiMrpcCompAsync_EventId         = 9
	CliMrpcComp_EventId              = 10
	CliMrpcCompAsync_EventId         = 11
	GpioInt_EventId                  = 12
	PartionReset_EventId             = 13
	MrpcComp_EventId                 = 14
	MrpcCompAsync_EventId            = 15
	DynPartBindComp_EventId          = 16
	AerInP2p_EventId                 = 17
	AerInVep_EventId                 = 18
	Dpc_EventId                      = 19
	Cts_EventId                      = 20
	Hotplug_EventId                  = 21
	Ier_EventId                      = 22
	Thresh_EventId                   = 23
	PowerMgmt_EventId                = 24
	TlpThrottling_EventId            = 25
	ForceSpeed_EventId               = 26
	CreditTimeout_EventId            = 27
	LinkState_EventId                = 28
	Gfms_EventId                     = 29
	Uec_EventId                      = 31

	MaxEventId     = 31
	InvalidEventId = -1
)

const (
	LocalPartitionIndex = -1
	IndexAll            = -2
)

type EventFlags int32

const (
	Clear_EventFlag        EventFlags = (1 << 0)
	EnablePoll_EventFlag              = (1 << 1)
	EnableLog_EventFlag               = (1 << 2)
	EnableCli_EventFlag               = (1 << 3)
	EnableFatal_EventFlag             = (1 << 4)
	DisablePoll_EventFlag             = (1 << 5)
	DisableLog_EventFlag              = (1 << 6)
	DisableCli_EventFlag              = (1 << 7)
	DisableFatal_EventFlag            = (1 << 8)
)

type EventOptions int32

const (
	Ignore_EventOption EventOptions = (1 << 0)
)

type EventDef struct {
	Type        EventType
	Id          EventId
	Options     EventOptions
	Bit         int
	Description string
}

var events = []EventDef{
	{Type: Global_EventType, Id: StackError_EventId, Bit: 0, Description: "Stack Error"},
	{Type: Global_EventType, Id: PpuError_EventId, Bit: 1, Description: "PPU Error"},
	{Type: Global_EventType, Id: IspError_EventId, Bit: 2, Description: "ISP Error"},
	{Type: Global_EventType, Id: SysReset_EventId, Bit: 3, Description: "System Reset"},
	{Type: Global_EventType, Id: FwExc_EventId, Bit: 4, Description: "Firmware Exception"},
	{Type: Global_EventType, Id: FwNmi_EventId, Bit: 5, Description: "Firmware Non-Maskable Interrupt"},
	{Type: Global_EventType, Id: FwNonFatal_EventId, Bit: 6, Description: "Firmware Non-Fatal Error"},
	{Type: Global_EventType, Id: FwFatal_EventId, Bit: 7, Description: "Firmware Fatal Error"},
	{Type: Global_EventType, Id: TwiMrpcComp_EventId, Bit: 8, Description: "TWI MRPC Completion"},
	{Type: Global_EventType, Id: TwiMrpcCompAsync_EventId, Bit: 9, Description: "TWI MRPC Async Completion"},
	{Type: Global_EventType, Id: CliMrpcComp_EventId, Bit: 10, Description: "CLI MRPC Completion"},
	{Type: Global_EventType, Id: CliMrpcCompAsync_EventId, Bit: 11, Description: "CLI MRPC Async Completion"},
	{Type: Global_EventType, Id: GpioInt_EventId, Bit: 12, Description: "GPIO Interrupt"},
	{Type: Global_EventType, Id: Gfms_EventId, Options: Ignore_EventOption, Bit: 13, Description: "Global Fabric Manager Server Event"},

	{Type: Partition_EventType, Id: PartionReset_EventId, Bit: 0, Description: "Partition Reset"},
	{Type: Partition_EventType, Id: MrpcComp_EventId, Bit: 1, Description: "MRPC Completion"},
	{Type: Partition_EventType, Id: MrpcCompAsync_EventId, Bit: 2, Description: "MRPC Async Completion"},
	{Type: Partition_EventType, Id: DynPartBindComp_EventId, Bit: 3, Description: "Dynamic Partition Binding Completion"},

	{Type: Port_EventType, Id: AerInP2p_EventId, Bit: 0, Description: "Advanced Error Reporting in P2P Port"},
	{Type: Port_EventType, Id: AerInVep_EventId, Bit: 1, Description: "Advanced Error Reporting in vEP"},
	{Type: Port_EventType, Id: Dpc_EventId, Bit: 2, Description: "Downstream Port Containment Error"},
	{Type: Port_EventType, Id: Cts_EventId, Bit: 3, Description: "Completion Timeout Synthesis Event"},
	{Type: Port_EventType, Id: Uec_EventId, Bit: 4, Description: "Upstream Error Containment Event"},
	{Type: Port_EventType, Id: Hotplug_EventId, Bit: 5, Description: "Hotplug Event"},
	{Type: Port_EventType, Id: Ier_EventId, Bit: 6, Description: "Internal Error Reporting Event"},
	{Type: Port_EventType, Id: Thresh_EventId, Bit: 7, Description: "Event Counter Threshold Reached"},
	{Type: Port_EventType, Id: PowerMgmt_EventId, Bit: 8, Description: "Power Management Event"},
	{Type: Port_EventType, Id: TlpThrottling_EventId, Bit: 9, Description: "TLP Throttling Event"},
	{Type: Port_EventType, Id: ForceSpeed_EventId, Bit: 10, Description: "Force Speed Event"},
	{Type: Port_EventType, Id: CreditTimeout_EventId, Bit: 11, Description: "Credit Timeout"},
	{Type: Port_EventType, Id: LinkState_EventId, Bit: 12, Description: "Link State Change Event"},
}

func (id *EventId) Type() EventType {
	for _, e := range events {
		if e.Id == *id {
			return e.Type
		}
	}

	return Invalid_EventType
}

func (id *EventId) Bit() int {
	for _, e := range events {
		if e.Id == *id {
			return e.Bit
		}
	}

	return -1
}

func (id *EventId) Description() string {
	for _, e := range events {
		if e.Id == *id {
			return e.Description
		}
	}

	return "Unknown Event"
}

func (dev *Device) EventSummary() (*IoctlEventSummary, error) {
	summary := new(IoctlEventSummary)

	_, _, err := syscall.Syscall(syscall.SYS_IOCTL, dev.file.Fd(), SwitchtecIoctlEventSummary, uintptr(unsafe.Pointer(summary)))
	if err != 0 {
		return nil, syscall.Errno(err)
	}

	return summary, nil
}

func (dev *Device) EventCtl(id EventId, index int32, flags EventFlags, data *[5]uint32) (uint32, error) {

	ctl := IoctlEventCtl{
		EventId: uint32(id),
		Flags:   uint32(flags),
		Index:   index,
	}

	_, _, err := syscall.Syscall(syscall.SYS_IOCTL, dev.file.Fd(), SwitchtecIoctlEventCtl, uintptr(unsafe.Pointer(&ctl)))

	if err != 0 {
		return 0, syscall.Errno(err)
	}

	if data != nil {
		copy((*data)[:], ctl.Data[:])
	}

	return ctl.Count, nil
}

func (dev *Device) PffToPort(index uint32) (int32, int32, error) {
	p := IoctlPffPort{
		Pff: index,
	}

	_, _, err := syscall.Syscall(syscall.SYS_IOCTL, dev.file.Fd(), SwitchtecIoctlPffToPort, uintptr(unsafe.Pointer(&p)))
	if err != 0 {
		return 0, 0, syscall.Errno(err)
	}

	return p.Partition, p.Port, nil
}

func (dev *Device) PortToPff(partition, port int32) (int32, error) {
	p := IoctlPffPort{
		Partition: partition,
		Port:      port,
	}

	_, _, err := syscall.Syscall(syscall.SYS_IOCTL, dev.file.Fd(), SwitchtecIoctlPortToPff, uintptr(unsafe.Pointer(&p)))
	if err != 0 {
		return 0, syscall.Errno(err)
	}

	return int32(p.Pff), nil
}

func (dev *Device) EventWait(timeoutMs int) (bool, error) {

	fds := []unix.PollFd{
		{Fd: int32(dev.file.Fd()), Events: unix.POLLPRI},
	}

	_, err := unix.Poll(fds, timeoutMs)
	if err != nil {
		return false, err
	}

	if (fds[0].Revents & unix.POLLERR) != 0 {
		return false, fmt.Errorf("No Device")
	}

	if (fds[0].Revents & unix.POLLPRI) != 0 {
		return true, nil
	}

	return false, nil
}

func (dev *Device) EventCheck(chk *IoctlEventSummary) (*IoctlEventSummary, error) {

	sum, err := dev.EventSummary()
	if err != nil {
		return nil, err
	}

	if chk.globalBitmap == sum.globalBitmap {
		return sum, nil
	}

	if chk.partitionBitmap == sum.partitionBitmap {
		return sum, nil
	}

	if chk.localPartitionBitmap == sum.localPartitionBitmap {
		return sum, nil
	}

	for i := range chk.partition {
		if (chk.partition[i] & sum.partition[i]) != 0 {
			return sum, nil
		}
	}

	for i := range chk.pff {
		if (chk.pff[i] & sum.pff[i]) != 0 {
			return sum, nil
		}
	}

	return nil, nil
}

type eventSummaryIterator struct {
	dev     *Device
	summary *IoctlEventSummary
	index   int
}

func newEventSummaryIterator(dev *Device, summary *IoctlEventSummary) *eventSummaryIterator {
	return &eventSummaryIterator{
		dev:     dev,
		summary: summary,
		index:   0,
	}
}

// Loop through available events listed in the Event Summary. This will process each Event
// one at a time. Events are listed in a bitmap field for each event type where a set bit
// indicates the presence of an event. When an event is detected, we return the event
// information and clear the bit.
//
// Returns and (Event, false) if the event should be processed. (Event, true) if there
// are no further events.
func (itr *eventSummaryIterator) Next() (Event, bool) {
	s := itr.summary

	definition := func(t EventType, bit int) *EventDef {
		for _, ev := range events {
			if ev.Type == t && ev.Bit == bit {
				return &ev
			}
		}

		return nil
	}

	for {

		var event Event
		var eventDef = &EventDef{
			Type: Invalid_EventType,
		}

		bit := bits.TrailingZeros64(s.globalBitmap)
		if bit >= 0 && bit < 64 {
			s.globalBitmap &= ^(1 << bit) // Clear the event presence bit

			eventDef = definition(Global_EventType, bit)

			event.Index = 0
			event.Partition = -1
			event.Port = -1

			goto EvaluateEvent
		}

		for idx := 0; idx < len(s.partition); idx++ {
			bit := bits.TrailingZeros32(s.partition[idx])
			if bit >= 0 && bit < 32 {
				s.partition[idx] &= ^(1 << bit) // Clear the event presence bit

				eventDef = definition(Partition_EventType, bit)

				event.Index = idx
				event.Partition = idx
				event.Port = -1

				goto EvaluateEvent
			}
		}

		for idx := 0; idx < len(s.pff); idx++ {
			bit := bits.TrailingZeros32(s.pff[idx])
			if bit >= 0 && bit < 32 {
				s.pff[idx] &= ^(1 << bit) // Clear the event presence bit

				eventDef = definition(Port_EventType, bit)
				partition, port, _ := itr.dev.PffToPort(uint32(idx))

				event.Index = idx
				event.Partition = int(partition)
				event.Port = int(port)

				goto EvaluateEvent
			}
		}

	EvaluateEvent:
		if (eventDef.Options & Ignore_EventOption) != 0 {
			continue
		}

		if eventDef.Type == Invalid_EventType {
			return event, true
		}

		event.Type = eventDef.Type
		event.Id = eventDef.Id

		return event, false

	}
}

type Event struct {
	Id        EventId
	Type      EventType
	Count     int
	Index     int
	Partition int
	Port      int
}

func (e *Event) String() string {
	return fmt.Sprintf("%-38s\t(%d)\t%-3d", e.Id.Description(), uint32(e.Id), e.Count)
}

func (e *Event) Compare(e2 Event) bool {
	if e.Partition != e2.Partition {
		return e.Partition < e2.Partition
	}

	if e.Port != e2.Port {
		return e.Port < e2.Port
	}

	return uint32(e.Id) < uint32(e2.Id)
}

type Events []Event

func (e Events) Len() int           { return len(e) }
func (e Events) Less(i, j int) bool { return e[i].Compare(e[j]) }
func (e Events) Swap(i, j int)      { e[i], e[j] = e[j], e[i] }

func (dev *Device) GetEvents(summary *IoctlEventSummary, id EventId, showAll bool, clearAll bool, index int32) ([]Event, error) {

	ret := make([]Event, 0, 1024)
	itr := newEventSummaryIterator(dev, summary)

	// Iterate over the switchtec events provided in the summary
	for event, done := itr.Next(); !done; event, done = itr.Next() {

		if index >= 0 && index != int32(event.Index) {
			continue
		}

		if showAll == false && event.Partition != int(dev.partition) {
			continue
		}

		flags := EventFlags(0)
		if clearAll == true || id == event.Id {
			flags |= Clear_EventFlag
		}

		cnt, err := dev.EventCtl(event.Id, int32(event.Index), flags, nil)
		if err != nil {
			continue
		}

		if err != nil {
			return nil, err
		}

		event.Count = int(cnt)
		ret = append(ret, event)
	}

	return ret, nil
}

// EventWaitFor will wait for the the occurrence of the EventId at the location specified by Index
// or a timeout to occur. The Event summary is returned, empty on timeout or with the value of the
// event otherwise. timeout = -1 is infinite wait.
func (dev *Device) EventWaitFor(id EventId, index int32, timeout int64) (*IoctlEventSummary, error) {

	chk, err := newEventSummary(id, index)
	if err != nil {
		return nil, err
	}

	if _, err := dev.EventCtl(id, index, EnablePoll_EventFlag|Clear_EventFlag, nil); err != nil {
		return nil, err
	}

	now := time.Now()
	start := now

	for true {

		eventTimeout := -1

		// Compute the timeout value from Now()
		elapsed := now.Sub(start).Microseconds()
		if timeout > 0 && timeout >= elapsed {
			eventTimeout = int(elapsed - timeout)
		}

		// Wait for the event to occur or a timeout
		occurred, err := dev.EventWait(eventTimeout)
		if err != nil {
			return nil, err
		}

		if occurred {

			sum, err := dev.EventCheck(chk)
			if err != nil {
				return nil, err
			}

			if sum != nil {
				return sum, nil
			}
		}

		now = time.Now()
		if timeout > 0 && now.Sub(start).Microseconds() >= timeout {
			return dev.EventSummary()
		}
	}

	return nil, nil
}

const (
	HostLinkUp_GfmsEvent = iota
	HostLinkDown_GfmsEvent
	DeviceAdd_GfmsEvent
	DeviceDelete_GfmsEvent
	FabricLinkUp_GfmsEvent
	FabricLinkDown_GfmsEvent
	Bind_GfmsEvent
	Unbind_GfmsEvent
	DatabaseChanged_GfmsEvent
	HvdInstEnable_GfmsEvent
	HvdInstDisable_GfmsEvent
	EpPortRemove_GfmsEvent
	EpPortAdd_GfmsEvent
	Aer_GfmsEvent

	Max_GfmsEvent
)

type GfmsEvent struct {
	Code int
	Id   int
	Data [8 * 4]byte
}

type GfmsEventInterface interface {
	Print()
}

// Event Data for Switchtec GFMS Host Link Up / Down Event
type HostGfmsEvent struct {
	PhysPortId uint16
}

func NewHostGfmsEvent(e GfmsEvent) GfmsEventInterface {
	return &HostGfmsEvent{
		PhysPortId: binary.LittleEndian.Uint16(e.Data[0:2]),
	}
}

func (event *HostGfmsEvent) Print() {
	fmt.Printf("    %30s:                  %4d\n", "Physical Port Id", event.PhysPortId)
}

// Event Data for Switchtec GFMS Device Add / Delete Event
type DeviceGfmsEvent struct {
	PhysPortId    uint16
	FunctionCount uint16
}

func NewDeviceGfmsEvent(e GfmsEvent) GfmsEventInterface {
	return &DeviceGfmsEvent{
		PhysPortId:    binary.LittleEndian.Uint16(e.Data[0:2]),
		FunctionCount: binary.LittleEndian.Uint16(e.Data[2:4]),
	}
}

func (event *DeviceGfmsEvent) Print() {
	fmt.Printf("    %30s:                  %4d\n", "Physical Port Id", event.PhysPortId)
	fmt.Printf("    %30s:                  %4d\n", "Function Count", event.FunctionCount)
}

// Event Data for Switchtec GFMS Bind / Unbind Event
type BindGfmsEvent struct {
	HostSwIdx      uint8
	HostPhysPortId uint8
	LogPortId      uint8
	Reserved       uint8
	PDFID          uint16
}

func NewBindGfmsEvent(e GfmsEvent) GfmsEventInterface {
	return &BindGfmsEvent{
		HostSwIdx:      e.Data[0],
		HostPhysPortId: e.Data[1],
		LogPortId:      e.Data[2],
		Reserved:       e.Data[3],
		PDFID:          binary.LittleEndian.Uint16(e.Data[4:6]),
	}
}

func (event *BindGfmsEvent) Print() {
	fmt.Printf("    %30s:                  %4d\n", "Host Switch Index", event.HostSwIdx)
	fmt.Printf("    %30s:                  %4d\n", "Host Physical Port ID", event.HostPhysPortId)
	fmt.Printf("    %30s:                  %4d\n", "Logical Port ID", event.LogPortId)
	fmt.Printf("    %30s:                0x%4x\n", "PDFID", event.PDFID)

}

// Event Data for Switchtec GFMS HVD Instance Enable / Disable Event
type HvdGfmsEvent struct {
	HvdInstId  uint8
	PhysPortId uint8
	ClockChan  uint8
}

func NewHvdGfmsEvent(e GfmsEvent) GfmsEventInterface {
	return &HvdGfmsEvent{
		HvdInstId:  e.Data[0],
		PhysPortId: e.Data[1],
		ClockChan:  e.Data[2],
	}
}

func (event *HvdGfmsEvent) Print() {
	fmt.Printf("    %30s:                  %4d\n", "HVD Instance ID", event.HvdInstId)
	fmt.Printf("    %30s:                  %4d\n", "Physical Port ID", event.PhysPortId)
	fmt.Printf("    %30s:                  %4d\n", "Clock Channel", event.ClockChan)
}

// Event Data for Switchtec GFMS EP Port Add / Remove Event
type PortGfmsEvent struct {
	PhysPortId uint8
}

func NewPortGfmsEvent(e GfmsEvent) GfmsEventInterface {
	return &PortGfmsEvent{
		PhysPortId: e.Data[0],
	}
}

func (event *PortGfmsEvent) Print() {
	fmt.Printf("    %30s:                  %4d\n", "Physical Port ID", event.PhysPortId)
}

// Event Data for Switchtec GFMS AER Event
type AerGfmsEvent struct {
	PhysPortId             uint16
	Handle                 uint8
	Reserved               uint8
	CeUeErrSts             uint32
	AerErrLogTimeStampHigh uint32
	AerErrLogTimeStampLow  uint32
	AerHeaderLog           [4]uint32
}

func NewAerGfmsEvent(e GfmsEvent) GfmsEventInterface {
	return &AerGfmsEvent{
		PhysPortId:             binary.LittleEndian.Uint16(e.Data[0:2]),
		Handle:                 e.Data[2],
		Reserved:               e.Data[3],
		CeUeErrSts:             binary.LittleEndian.Uint32(e.Data[4:8]),
		AerErrLogTimeStampHigh: binary.BigEndian.Uint32(e.Data[8:12]),
		AerErrLogTimeStampLow:  binary.LittleEndian.Uint32(e.Data[12:16]),
		AerHeaderLog: [4]uint32{
			binary.BigEndian.Uint32(e.Data[16:20]),
			binary.BigEndian.Uint32(e.Data[20:24]),
			binary.BigEndian.Uint32(e.Data[24:28]),
			binary.BigEndian.Uint32(e.Data[28:32]),
		},
	}
}

func (event *AerGfmsEvent) Dpc() int  { return int((event.Handle & 0x02) >> 1) }
func (event *AerGfmsEvent) CeUe() int { return int((event.Handle & 0x04) >> 2) }
func (event *AerGfmsEvent) Log() bool { return (event.Handle & 0x01) != 0x00 }

func (event *AerGfmsEvent) Print() {
	fmt.Printf("    %30s:                  %4d\n", "Physical Port Id", event.PhysPortId)
	fmt.Printf("    %30s:                  %4s\n", "DPC Triggered", [2]string{"No", "Yes"}[event.Dpc()])
	fmt.Printf("    %30s:                  %4s\n", "CE/UE", [2]string{"CE", "UE"}[event.CeUe()])
	fmt.Printf("    %30s:          0x%08x\n", "CE/UE Error Status", event.CeUeErrSts)
	fmt.Printf("    %30s:  0x%08x%08x\n", "Time Stamp (In Clock Ticks)", event.AerErrLogTimeStampHigh, event.AerErrLogTimeStampLow)
	if event.Log() {
		fmt.Printf("    %30s:           0x%08x\n", "AER TLP Header Log", event.AerHeaderLog[0])
		fmt.Printf("    %30s:           0x%08x\n", "", event.AerHeaderLog[1])
		fmt.Printf("    %30s:           0x%08x\n", "", event.AerHeaderLog[2])
		fmt.Printf("    %30s:           0x%08x\n", "", event.AerHeaderLog[3])
	}
}

func (dev *Device) ClearGfmsEvents() error {
	cmd := struct {
		subCmd uint8
	}{
		subCmd: uint8(ClearGfmsEventsSubCommand),
	}

	return dev.RunCommand(GfmsEventCommand, cmd, nil)
}

func (dev *Device) GetGfmsEvents() ([]GfmsEvent, error) {

	events := make([]GfmsEvent, 0)

	cmd := struct {
		SubCmd        uint8
		Reserved      uint8
		RequestNumber uint16 // Requested number of GFMS Event Entries
	}{
		SubCmd:        uint8(GetGfmsEventsSubCommand),
		RequestNumber: 128,
	}

	type response struct {
		ResponseNumber       uint16 // Number of GFMS Event Entries in this response
		RemainingNumberFlags uint16 // [0:14] Number of GFMS Event Entries remaining in GFMS Event Queue
		// [15]   Flag to indicate whether the event entry buffer queue has
		//        been overwritten as a result of not being read in time.
		Data [maxDataLength - 4]byte
	}

	var rsp = new(response)

	// Each GFMS Event Entry contains a GFMS Event Entry Header and GFMS Event Entry Data. The
	// GFMS Event Header format is the same 1 DWORD length for all GFMS events. The GFMS Event
	// Entry Data is indicated in the GFMS Event Header
	type entry struct {
		entryLength uint16 // Valid Event Data length, including this DWORD
		eventCode   uint8  // GFMS Event Code
		srcSwId     uint8  // GFMS Event Source Switch Index
		data        []byte // GFMS Event data for each GFMS event.
	}

	entryDataOffset := int(unsafe.Offsetof(entry{}.data))

	for true {
		err := dev.RunCommand(GfmsEventCommand, cmd, rsp)
		if err != nil {
			return nil, err
		}

		offset := 0
		for i := 0; i < int(rsp.ResponseNumber); i++ {
			event := GfmsEvent{
				Code: int(rsp.Data[offset+2]),
				Id:   int(rsp.Data[offset+3]),
			}

			entryLength := int(binary.LittleEndian.Uint16(rsp.Data[offset : offset+2]))
			dataLen := entryLength - entryDataOffset
			copy(event.Data[:], rsp.Data[offset+4:offset+4+dataLen])

			events = append(events, event)

			offset += entryLength
		}

		remainingCount := rsp.RemainingNumberFlags & 0x7fff

		if remainingCount == 0 {
			break
		}
	}

	return events, nil
}
