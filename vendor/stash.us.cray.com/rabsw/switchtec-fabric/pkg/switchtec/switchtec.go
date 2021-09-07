package switchtec

import (
	"fmt"
	"math"
	"os"
	"sort"
	"syscall"

	"github.com/mattn/go-isatty"

	"github.com/HewlettPackard/structex"
)

// Device describing the switchtec device
type Device struct {
	file           *os.File
	name           string
	paxID          int32
	ops            ops
	partition      uint8
	partitionCount uint8

	ctrl *commandController
}

const (
	PAXIDMask  = 0x1f
	PAXIdShift = 18
)

// Open returns the device such that it is accessible to other
// API functions.
func Open(path string) (*Device, error) {

	fd, err := syscall.Open(path, os.O_RDWR, 0666)
	if err != nil {
		return nil, err
	}

	f := os.NewFile(uintptr(fd), path)
	if f == nil {
		return nil, fmt.Errorf("Cannot open new file %s", path)
	}

	syscall.SetNonblock(int(f.Fd()), false)

	var dev = new(Device)
	dev.file = f
	dev.name = path
	dev.ops = &charOps{}
	dev.paxID = 0 // -1 & PAXIDMask
	dev.ctrl = NewCommandController()

	if isatty.IsTerminal(f.Fd()) {
		if err := dev.OpenUart(path); err != nil {
			return dev, err
		}
	}

	id, err := dev.Identify()
	if err != nil {
		return dev, err
	}

	dev.paxID = id

	return dev, err
}

func (dev *Device) Lock()   { dev.ctrl.Lock() }
func (dev *Device) Unlock() { dev.ctrl.Unlock() }

// Close -
func (dev *Device) Close() error {

	if dev.ops != nil {
		dev.ops.close(dev)
		dev.ops = nil
	}

	return dev.file.Close()
}

// RunCommand takes a command parameter and payload to send to the device. Optionally
// can take a response to return data back to the caller. Payload and response can be
// any structex annotated structure.
func (dev *Device) RunCommand(cmd Command, payload interface{}, response interface{}) error {

	payloadBuf := structex.NewBuffer(payload)

	if payload != nil {
		if err := structex.Encode(payloadBuf, payload); err != nil {
			return err
		}
	}

	dev.Lock()
	defer dev.Unlock()

	if err := dev.ops.submitCommand(dev, cmd, payloadBuf.Bytes()); err != nil {
		return err
	}

	responseBuf := structex.NewBuffer(response)

	if err := dev.ops.readResponse(dev, responseBuf.Bytes()); err != nil {
		return err
	}

	if response == nil {
		return nil
	}

	err := structex.Decode(responseBuf, response)

	return err
}

// RunCommandRawBytes function will run the command on the device
func (dev *Device) RunCommandRawBytes(cmd Command, payload []byte, response []byte) error {

	dev.Lock()
	defer dev.Unlock()

	if err := dev.ops.submitCommand(dev, cmd, payload); err != nil {
		return err
	}

	if err := dev.ops.readResponse(dev, response); err != nil {
		return err
	}

	return nil
}

func (dev *Device) ID() int32 {
	return dev.paxID
}

func (dev *Device) SetID(id int32) {
	dev.paxID = id
}

func (dev *Device) IsLocal() bool {
	/*
		TODO: Support for remote fabric control - routing to non-local
		PAX chip.
	*/
	return true
}

// Identify will return the PAX Identifer
func (dev *Device) Identify() (int32, error) {

	type identify struct {
		PaxID int32
	}

	id := new(identify)

	if err := dev.RunCommand(GetPaxIDCommand, nil, id); err != nil {
		return -1, err
	}

	return id.PaxID, nil
}

// Echo will send a pattern to the device and expect to receive its inverse,
// confirming that the device is functioning properly.
func (dev *Device) Echo(pattern uint32) error {
	type Echo struct {
		Pattern uint32
	}

	cmd := Echo{
		Pattern: pattern,
	}

	rsp := Echo{}

	if err := dev.RunCommand(EchoCommand, cmd, &rsp); err != nil {
		return err
	}

	if cmd.Pattern != ^rsp.Pattern {
		return fmt.Errorf("Echo pattern does not match: Sent: %#08x Received: %#08x", cmd.Pattern, rsp.Pattern)
	}

	return nil
}

// Port Identification
type PortId struct {
	Partition  uint8
	Stack      uint8
	Upstream   uint8
	StackId    uint8
	PhysPortId uint8
	LogPortId  uint8
}

func (p1 *PortId) Less(p2 *PortId) bool {
	if p1.Partition != p2.Partition {
		return p1.Partition < p2.Partition
	}

	if p1.Upstream != p2.Upstream {
		return p1.Upstream == 0 || p2.Upstream != 0
	}

	return p1.LogPortId < p2.LogPortId
}

type PortIds []PortId

func (p PortIds) Len() int           { return len(p) }
func (p PortIds) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p PortIds) Less(i, j int) bool { return p[i].Less(&p[j]) }

// Get the status of all the ports on the switchtec device
func (dev *Device) Status() (PortIds, error) {
	stats, err := dev.LinkStat()
	if err != nil {
		return nil, err
	}

	portIds := make(PortIds, len(stats))

	for i, stat := range stats {
		portIds[i] = PortId{
			Partition:  stat.Partition,
			Stack:      stat.Stack,
			StackId:    stat.StackId,
			Upstream:   stat.Upstream,
			PhysPortId: stat.PhysPortId,
			LogPortId:  stat.LogPortId,
		}
	}

	sort.Sort(portIds)

	return portIds, nil
}

// PortLinkStat -
type PortLinkStat struct {
	Partition uint8
	Stack     uint8
	StackId   uint8
	Upstream  uint8

	PhysPortId uint8
	LogPortId  uint8

	CfgLinkWidth uint8
	NegLinkWidth uint8

	LinkUp    bool
	LinkGen   uint8
	LinkState PortLinkState

	CurLinkRateGBps float64
}

type PortLinkState uint16

const (
	PortLinkState_Detect   PortLinkState = 0x0000
	PortLinkState_Polling  PortLinkState = 0x0001
	PortLinkState_Config   PortLinkState = 0x0002
	PortLinkState_L0       PortLinkState = 0x0003
	PortLinkState_Recovery PortLinkState = 0x0004
	PortLinkState_Disable  PortLinkState = 0x0005
	PortLinkState_LoopBack PortLinkState = 0x0006
	PortLinkState_HotReset PortLinkState = 0x0007
	PortLinkState_TxL0s    PortLinkState = 0x0008
	PortLinkState_L1       PortLinkState = 0x0009
	PortLinkState_L2       PortLinkState = 0x000A
	PortLinkState_Unknown  PortLinkState = 0x00FF

	linkStateMask uint16 = 0x00FF // Upper byte is for minor details
)

const (
	maxStacks = 8
	maxPorts  = 48

	UnboundPort uint8 = math.MaxUint8
)

const (
	MaxSupportedTransferRates = 6
)

var (
	PciGenDataRateGTps = [MaxSupportedTransferRates]float64{0, 2.5, 5, 8, 16, 32}
	PciGenDataRateGBps = [MaxSupportedTransferRates]float64{0, 250, 500, 985, 1969, 3938}
)

func GetDataRateGBps(pciGen uint8) float64 {
	if int(pciGen) < len(PciGenDataRateGBps) {
		return PciGenDataRateGBps[pciGen]
	}

	return 0
}

// Link Stat will get the status of all the ports on a switchtec device
func (dev *Device) LinkStat() ([]PortLinkStat, error) {

	type LinkStat struct {
		Bitmap uint64
	}

	cmd := LinkStat{
		Bitmap: 0,
	}

	type portLinkStat struct {
		PhysPortId      uint8
		PartitionId     uint8
		LogicalPortId   uint8
		PortStackId     uint8 `bitfield:"4"`
		StackId         uint8 `bitfield:"4"`
		CfgLinkWidth    uint8
		NegLinkWidth    uint8
		USPFlag         uint8
		LinkRate        uint8 `bitfield:"7"`
		LinkUp          uint8 `bitfield:"1"`
		LTSSM           uint16
		LaneReversal    uint8
		FirstActiveLane uint8
	}

	type LinkStatRsp struct {
		Stats [maxPorts]portLinkStat
	}

	rsp := LinkStatRsp{}

	if err := dev.RunCommand(LinkStatCommand, cmd, &rsp); err != nil {
		return nil, err
	}

	stats := make([]PortLinkStat, maxPorts)

	var statIdx = 0
	for _, s := range rsp.Stats {
		if s.StackId > maxStacks {
			continue
		}

		if s.LinkRate >= MaxSupportedTransferRates {
			return nil, fmt.Errorf("Link Rate beyond supported definitions")
		}

		stats[statIdx] = PortLinkStat{
			Partition: s.PartitionId,
			Stack:     s.StackId,
			StackId:   s.PortStackId,
			Upstream:  s.USPFlag,

			PhysPortId: s.PhysPortId,
			LogPortId:  s.LogicalPortId,

			CfgLinkWidth: s.CfgLinkWidth,
			NegLinkWidth: s.NegLinkWidth,

			LinkUp:    s.LinkUp == 1,
			LinkGen:   s.LinkRate,
			LinkState: PortLinkState(s.LTSSM & linkStateMask),

			CurLinkRateGBps: GetDataRateGBps(s.LinkRate) * float64(s.NegLinkWidth),
		}

		statIdx++
	}

	return stats[:statIdx], nil
}
