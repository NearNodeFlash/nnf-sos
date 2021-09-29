package nvme

import (
	"encoding/binary"
	"fmt"
	"path"
	"regexp"
	"strconv"
	"strings"
	"unsafe"

	"github.com/HewlettPackard/structex"

	"stash.us.cray.com/rabsw/nnf-ec/internal/switchtec/pkg/switchtec"
)

// Device describes the NVMe Device and its attributes
type Device struct {
	Path string
	ops  ops
}

// NVMe User I/O Request
// This is a copy of the C definition in '/include/linux/nvme_ioctl.h' struct nvme_user_io{}

type UserIoCmd struct {
	Opcode   uint8
	Flags    uint8
	Control  uint16
	NBlocks  uint16
	Reserved uint16
	Metadata uint64
	Addr     uint64
	StartLBA uint64
	DsMgmt   uint32
	RefTag   uint32
	AppTag   uint16
	AppMask  uint16
}

// PassthruCmd is used for NVMe Passthrough from user space to the NVMe Device Driver
// This is a copy of the C definition in 'include/linux/nvme_ioctl.h' struct nvme_passthru_cmd{}
type PassthruCmd struct {
	Opcode      uint8  // Byte 0
	Flags       uint8  // Byte 1
	Reserved1   uint16 // Bytes 2-3
	NSID        uint32 // Bytes 4-7
	Cdw2        uint32 // Bytes 8-11
	Cdw3        uint32 // Bytes 12-15
	Metadata    uint64 // Bytes 16-23
	Addr        uint64 // Bytes 24-31
	MetadataLen uint32 // Bytes 32-35
	DataLen     uint32 // Bytes 36-39
	Cdw10       uint32 // Bytes 40-43
	Cdw11       uint32 // Bytes 44-47
	Cdw12       uint32 // Bytes 48-51
	Cdw13       uint32 // Bytes 52-55
	Cdw14       uint32 // Bytes 56-60
	Cdw15       uint32 // Bytes 60-63
	TimeoutMs   uint32 // Bytes 64-67
	Result      uint32 // Bytes 68-71
}

type AdminCmd = PassthruCmd

type AdminCommandOpCode uint8

// Admin Command Op Codes. These are from the NVMe Specification
const (
	IdentifyOpCode            AdminCommandOpCode = 0x6
	SetFeatures               AdminCommandOpCode = 0x09
	GetFeatures               AdminCommandOpCode = 0x0A
	NamespaceManagementOpCode AdminCommandOpCode = 0x0D
	NamespaceAttachOpCode     AdminCommandOpCode = 0x15
	VirtualMgmtOpCode         AdminCommandOpCode = 0x1c
	NvmeMiSend                AdminCommandOpCode = 0x1D
	NvmeMiRecv                AdminCommandOpCode = 0x1E
)

func Open(devPath string) (*Device, error) {

	if strings.HasPrefix(path.Base(devPath), "nvme") {
		return nil, fmt.Errorf("direct nvme device not suppported yet")
	}

	ops := switchOps{}

	deviceExp := regexp.MustCompile("(?P<pdfid>[^@]*)?@(?P<device>.*)")
	if m := deviceExp.FindStringSubmatch(devPath); m != nil {
		pdfidStr := m[1]
		deviceStr := m[2]

		pdfid, err := strconv.ParseUint(pdfidStr, 0, 16)
		if err != nil {
			return nil, err
		}

		ops.pdfid = uint16(pdfid)
		ops.devPath = deviceStr

		dev, err := switchtec.Open(ops.devPath)
		if err != nil {
			return nil, err
		}

		if err := connectDevice(dev, &ops); err != nil {
			dev.Close()
			return nil, err
		}

		ops.dev = dev

	} else {
		return nil, fmt.Errorf("Unknown device path")
	}

	return &Device{
		Path: ops.devPath,
		ops:  &ops}, nil
}

// Connect - Connect will connect a PDFID to an existing switchtec device
func Connect(dev *switchtec.Device, pdfid uint16) (*Device, error) {
	ops := switchOps{
		dev:   dev,
		pdfid: pdfid,
	}

	if err := connectDevice(dev, &ops); err != nil {
		return nil, err
	}

	return &Device{ops: &ops}, nil
}

func connectDevice(dev *switchtec.Device, ops *switchOps) (err error) {
	if ops.tunnelStatus, err = dev.EndPointTunnelStatus(ops.pdfid); err != nil {
		return err
	}

	if ops.tunnelStatus == switchtec.DisabledEpTunnelStatus {
		if err = dev.EndPointTunnelEnable(ops.pdfid); err != nil {
			return err
		}
		ops.tunnelStatus = switchtec.EnabledEpTunnelStatus
	}

	return nil
}

func (dev *Device) Close() {
	if dev.ops != nil {
		dev.ops.close()
	}

	dev.ops = nil
}

const (
	IdentifyDataSize uint32 = 4096
)

type IdentifyControllerOrNamespaceType int // CNS

const (
	Namespace_CNS                     IdentifyControllerOrNamespaceType = 0x00
	Controller_CNS                    IdentifyControllerOrNamespaceType = 0x01
	NamespaceList_CNS                 IdentifyControllerOrNamespaceType = 0x02
	NamespaceDescriptorList_CNS       IdentifyControllerOrNamespaceType = 0x03
	NVMSetList_CNS                    IdentifyControllerOrNamespaceType = 0x04
	NamespacePresentList_CNS          IdentifyControllerOrNamespaceType = 0x10
	NamespacePresent_CNS              IdentifyControllerOrNamespaceType = 0x11
	ControllerNamespaceList_CNS       IdentifyControllerOrNamespaceType = 0x12
	ControllerList_CNS                IdentifyControllerOrNamespaceType = 0x13
	PrimaryControllerCapabilities_CNS IdentifyControllerOrNamespaceType = 0x14
	SecondaryControllerList_CNS       IdentifyControllerOrNamespaceType = 0x15
	NamespaceGranularityList_CNS      IdentifyControllerOrNamespaceType = 0x16
	UUIDList_CNS                      IdentifyControllerOrNamespaceType = 0x17
)

type NamespaceIdentifier uint32

const (
	// The capabilities and settings that are common to all namespaces are reported
	// by the Identify Namespace data structure for namespace ID FFFFFFFFh.
	COMMON_NAMESPACE_IDENTIFIER NamespaceIdentifier = 0xFFFFFFFF
)

// IdentifyNamespace -
func (dev *Device) IdentifyNamespace(namespaceId uint32, present bool) (*IdNs, error) {

	ns := new(IdNs)

	cns := Namespace_CNS
	if present {
		cns = NamespacePresent_CNS
	}

	buf := structex.NewBuffer(ns)

	if err := dev.Identify(namespaceId, cns, buf.Bytes()); err != nil {
		return nil, err
	}

	if err := structex.Decode(buf, ns); err != nil {
		return nil, err
	}

	return ns, nil
}

// IdentifyNamespaceList -
func (dev *Device) IdentifyNamespaceList(namespaceId uint32, all bool) ([1024]NamespaceIdentifier, error) {

	var data struct {
		Identifiers [1024]NamespaceIdentifier
	}

	cns := NamespaceList_CNS // Display only controller active NS'
	if all {
		cns = NamespacePresentList_CNS // Display all namespaces present on the device
	}

	buf := structex.NewBuffer(data)

	if err := dev.Identify(namespaceId, cns, buf.Bytes()); err != nil {
		return data.Identifiers, err
	}

	if err := structex.Decode(buf, &data); err != nil {
		return data.Identifiers, err
	}

	return data.Identifiers, nil
}

// IdentifyController -
func (dev *Device) IdentifyController() (*IdCtrl, error) {
	id := new(IdCtrl)

	buf := structex.NewBuffer(id)

	if err := dev.Identify(0, Controller_CNS, buf.Bytes()); err != nil {
		return nil, err
	}

	if err := structex.Decode(buf, id); err != nil {
		return nil, err
	}

	return id, nil
}

// IdentifyNamespaceControllerList - Identifies the list of controllers attached to the provided namespace
func (dev *Device) IdentifyNamespaceControllerList(namespaceId uint32) (*CtrlList, error) {

	ctrlList := new(CtrlList)
	buf := structex.NewBuffer(ctrlList)

	if err := dev.Identify(0, ControllerNamespaceList_CNS, buf.Bytes()); err != nil {
		return nil, err
	}

	if err := structex.Decode(buf, ctrlList); err != nil {
		return nil, err
	}

	return ctrlList, nil
}

// Identify -
func (dev *Device) Identify(namespaceId uint32, cns IdentifyControllerOrNamespaceType, data []byte) error {
	return dev.IdentifyRaw(namespaceId, uint32(cns), 0, data)
}

// IdentifyRaw -
func (dev *Device) IdentifyRaw(namespaceId uint32, cdw10 uint32, cdw11 uint32, data []byte) error {
	var addr uint64 = 0
	if data != nil {
		addr = uint64(uintptr(unsafe.Pointer(&data[0])))
	}

	cmd := AdminCmd{
		Opcode:  uint8(IdentifyOpCode),
		NSID:    namespaceId,
		Addr:    addr,
		DataLen: IdentifyDataSize,
		Cdw10:   cdw10,
		Cdw11:   cdw11,
	}

	return dev.ops.submitAdminPassthru(dev, &cmd, data)
}

type OptionalControllerCapabilities uint16

const (
	SecuritySendReceiveCapabilities OptionalControllerCapabilities = (1 << 0)
	FormatNVMCommandSupport                                        = (1 << 1)
	FirmwareCommitImageDownload                                    = (1 << 2)
	NamespaceManagementCapability                                  = (1 << 3)
	DeviceSelfTestCommand                                          = (1 << 4)
	DirectivesSupport                                              = (1 << 5)
	NVMeMISendReceiveSupport                                       = (1 << 6)
	VirtualiztionManagementSupport                                 = (1 << 7)
	DoorbellBufferConfigCommand                                    = (1 << 8)
	GetLBAStatusCapability                                         = (1 << 9)
)

// Identify Controller Data Structure
// Figure 249 from NVM-Express 1_4a-2020.03.09-Ratified specification
type id_ctrl struct {
	PCIVendorId                 uint16   // VID
	PCISubsystemVendorId        uint16   // SSVID
	SerialNumber                [20]byte // SN
	ModelNumber                 [40]byte // MN
	FirmwareRevision            [8]byte  // FR
	RecommendedArbitrationBurst uint8    // RAB
	IEEOUIIdentifier            [3]byte  // IEEE
	ControllerCapabilities      struct {
		MultiPort                 uint8 `bitfield:"1"` // Bit 0: NVM subsystem may contain more than one NVM subsystem port
		MultiController           uint8 `bitfield:"1"` // Bit 1: NVM subsystem may contain two or more controllers
		SRIOVVirtualFunction      uint8 `bitfield:"1"` // Bit 2: The controller is associated with an SR-IOV Virtual Function.
		AsymmetricNamespaceAccess uint8 `bitfield:"1"` // Bit 3: NVM subsystem supports Asymmetric Namespace Access Reporting.
		Reserved                  uint8 `bitfield:"4"`
	} // CMIC
	MaximumDataTransferSize                   uint8  // MDTS
	ControllerId                              uint16 // CNTLID
	Version                                   uint32 // VER
	RTD3ResumeLatency                         uint32 // RTD3R
	RTD3EntryLatency                          uint32 // RTD3E
	AsynchronousEventsSupport                 uint32 // OAES
	ControllerAttributes                      uint32 // CTRATT
	ReadRecoveryLevelsSupported               uint16 // RRLS
	Reserved102                               [9]byte
	ControllerType                            uint8    // CNTRLTYPE
	FRUGloballyUniqueIdentifier               [16]byte // FGUID
	CommandRetryDelayTime1                    uint16   // CRDT1
	CommandRetryDelayTime2                    uint16   // CRDT2
	CommandRetryDelayTime3                    uint16   // CRDT3
	Reserved134                               [122]byte
	OptionalAdminCommandSupport               uint16   // OACS
	AbortCommandLimit                         uint8    // ACL
	AsynchronousEventRequestLimit             uint8    // AERL
	FirmwareUpdates                           uint8    // FRMW
	LogPageAttributes                         uint8    // LPA
	ErrorLogPageEntries                       uint8    // ELPE
	NumberPowerStatesSupport                  uint8    // NPSS
	AdminVendorSpecificCommandConfiguration   uint8    // AVSCC
	AutonomousPowerStateTranstionAttributes   uint8    // APSTA
	WarningCompositeTemperatureThreshold      uint16   // WCTEMP
	CriticalCompositeTemperatureThreashold    uint16   // CCTEMP
	MaximumTimeForFirmwareActivation          uint16   // MTFA 100ms unites
	HostMemoryBufferPreferredSize             uint32   // HMPRE 4 KiB Units
	HostMemoryBufferMinimumSize               uint32   // HMMIN 4KiB Units
	TotalNVMCapacity                          [16]byte // TNVMCAP
	UnallocatedNVMCapacity                    [16]byte // UNVMCAP
	ReplayProtectedMemoryBlockSupport         uint32   // RPMBS
	ExtendedDriveSelfTestTime                 uint16   // EDSTT
	DeviceSelfTestOptions                     uint8    // DSTO
	FirmwareUpdateGranularity                 uint8    // FWUG
	KeepAliveSupport                          uint16   // KAS
	HostControllerThermalManagementAttributes uint16   // HCTMA
	MinimumThermalManagementTemperature       uint16   // MNTMT
	MaximumThermalManagementTemperature       uint16   // MXTMT
	SanitizeCapabilities                      uint32   // SNAICAP
	HostMemoryBufferMinimDescriptorEntrySize  uint32   // HMMINDS
	HostMemoryMaximumDescriptorEntries        uint16   // HMMAXD
	NVMSetIdentifierMaximum                   uint16   // NSETIDMAX
	EnduraceGroupIdentifierMaximum            uint16   // ENDGIDMAX
	ANATranstitionTime                        uint8    // ANATT
	AsymentricNamespaceAccessCapabilities     uint8    // ANACAP
	ANAGroupIdenfierMaximum                   uint32   // ANAGRPMAX
	NumberOfANAGroupIdentifiers               uint32   // NANAGRPID
	PersistentEventLogSize                    uint32   // PELS
	Reserved356                               [156]byte
	CommandSetAttributes                      struct {
		SubmissionQueueEntrySize              uint8  // SQES
		CompletionQueueEntrySize              uint8  // CQES
		MaximumOutstandingCommands            uint16 // MAXCMD
		NumberOfNamespaces                    uint32 // NN
		OptionalNVMCommandSupport             uint16 // ONCS
		FuesedOperationSupport                uint16 // FUSES
		FormatNVMAttributes                   uint8  // FNA
		VolatileWriteCache                    uint8  // VWC
		AtomicWriteUnitNormal                 uint16 // AWUN
		AtomicWriteUnitPowerFail              uint16 // AWUPF
		NVMVendorSpecificCommandConfiguration uint8  // NVSCC
		NamespaceWriteProtectionCapabilities  uint8  // NWPC
		AtomicCompareAndWriteUnit             uint16 // ACWU
	}
	Reserved534                      [2]byte
	SGLSupport                       uint32 // SGLS
	MaximumNumberOfAllowedNamespaces uint32 // MNAN
	Reserved544                      [224]byte
	NVMSubsystemNVMeQualifiedName    [256]byte // SUBNQN
	Reserved1024                     [768]byte
	IOCCSZ                           uint32
	IORCSZ                           uint32
	ICDOFF                           uint16
	CTRATTR                          uint8
	MSBDB                            uint8
	Reserved1804                     [244]byte
	PowerStateDescriptors            [32]id_power_state
	VendorSpecific                   [1024]byte
}

type id_power_state struct {
	MaxPowerInCentiwatts    uint16
	Reserved2               uint8
	Flags                   uint8
	EntryLatency            uint32 // EXLAT microseconds
	ExitLatency             uint32 // EXLAT microseconds
	RelativeReadThroughpout uint8  // RRT
	RelativeReadLatency     uint8  // RRL
	RelativeWriteThroughput uint8  // RWT
	RelativeWriteLatency    uint8  // RWL
	IdlePower               uint16 // IDLP
	IdlePowerScale          uint8  // IPS
	Reserved19              uint8
	ActivePower             uint16 // ACTP
	ActivePowerScale        uint8  // APS
	Reserved23              [9]byte
}

type IdCtrl id_ctrl

// GetCapability -
func (id *IdCtrl) GetCapability(cap OptionalControllerCapabilities) bool {
	return id.OptionalAdminCommandSupport&uint16(cap) != 0
}

// Identify Namespace Data Structure
// Figure 247 from NVM-Express 1_4a-2020.03.09-Ratified specification
//
// All the parameters listed are the long-form name with the 'Namespace'
// prefix dropped, if it exists. structex annotations where applicable.
type IdNs struct {
	Size        uint64 // NSZE
	Capacity    uint64 // NCAP
	Utilization uint64 // NUSE
	Features    struct {
		Thinp    uint8 `bitfield:"1"` // Bit 0: Thin provisioning supported for this namespace
		Nsabp    uint8 `bitfield:"1"` // Bit 1: NAWUN, NAWUPF, NACWU are defined for this namespace (TODO: Wtf is this?)
		Dae      uint8 `bitfield:"1"` // Bit 2: Deallocated or Unwritten Logical Block error support for this namespace
		Uidreuse uint8 `bitfield:"1"` // Bit 3: NGUID field for this namespace, if non-zero, is never reused by the controller.
		OptPerf  uint8 `bitfield:"1"` // Bit 4: If 1, indicates NPWG, NPWA, NPDG, NPDA, NOWS are defined for this namespace and should be used for host I/O optimization.
		Reserved uint8 `bitfield:"3"` // Bits 7:5 are reserved
	} // NSFEAT
	NumberOfLBAFormats   uint8            // NLBAF
	FormattedLBASize     FormattedLBASize // FLABS
	MetadataCapabilities struct {
		ExtendedSupport uint8 `bitfield:"1"` // Bit 0: If 1, indicates support metadata transfer as part of extended data LBA
		PointerSupport  uint8 `bitfield:"1"` // Bit 1: If 1, indicates support for metadata transfer in a separate buffer specified by Metadata Pointer in SQE
		Reserved        uint8 `bitfield:"6"` // Bits 7:2 are reserved
	} // MC
	EndToEndDataProtectionCaps         uint8                 // DPC
	EndToEndDataProtectionTypeSettings uint8                 // DPS
	MultiPathIOSharingCapabilities     NamespaceCapabilities // NMIC
	ReservationCapabilities            struct {
		PersistThroughPowerLoss        uint8 `bitfield:"1"` // Bit 0: If 1 indicates namespace supports the Persist Through Power Loss capability
		WriteExclusiveReservation      uint8 `bitfield:"1"` // Bit 1: If 1 indicates namespace supports the Write Exclusive reservation type
		ExclusiveAccessReservation     uint8 `bitfield:"1"` // Bit 2: If 1 indicates namespace supports Exclusive Access reservation type
		WriteExclusiveRegistrantsOnly  uint8 `bitfield:"1"` // Bit 3:
		ExclusiveAccessRegistrantsOnly uint8 `bitfield:"1"` // Bit 4:
		WriteExclusiveAllRegistrants   uint8 `bitfield:"1"` // Bit 5:
		ExclusiveAllRegistrants        uint8 `bitfield:"1"` // Bit 6:
		IgnoreExistingKey              uint8 `bitfield:"1"` // Bit 7: If 1 indicates that the Ingore Existing Key is used as defined in 1.3 or later specification. If 0, 1.2.1 or earlier definition
	} // RESCAP
	FormatProgressIndicator struct {
		PercentageRemaining   uint8 `bitfield:"7"` // Bits 6:0 indicates percentage of the Format NVM command that remains completed. If bit 7 is set to '1' then a value of 0 indicates the namespace is formatted with FLBAs and DPS fields and no format is in progress
		FormatProgressSupport uint8 `bitfield:"1"` // Bit 7 if 1 indicates the namespace supports the Format Progress Indicator field
	} // FPI
	DeallocateLogiclBlockFeatures  uint8     // DLFEAT
	AtomicWriteUnitNormal          uint16    // NAWUN
	AtomicWriteUnitPowerFail       uint16    // NAWUPF
	AtomicCompareAndWriteUnit      uint16    // NACWU
	AtomicBoundarySizeNormal       uint16    // NABSN
	AtomicBoundaryOffset           uint16    // NABO
	AtomicBoundarySizePowerFail    uint16    // NABSPF
	OptimalIOBoundary              uint16    // NOIOB
	NVMCapacity                    [16]uint8 // NVMCAP Total size of the NVM allocated to this namespace, in bytes.
	PreferredWriteGranularity      uint16    // NPWG
	PreferredWriteAlignment        uint16    // NPWA
	PreferredDeallocateGranularity uint16    // NPDG
	PreferredDeallocateAlignment   uint16    // NPDA
	OptimalWriteSize               uint16    // NOWS The size in logical block for optimal write performance for this namespace. Should be a multiple of NPWG. Refer to 8.25 for details
	Reserved74                     [18]uint8
	AnaGroupIdentifier             uint32 // ANAGRPID indicates the ANA Group Identifier for thsi ANA group (refer to section 8.20.2) of which the namespace is a member.
	Reserved96                     [3]uint8
	Attributes                     struct {
		WriteProtected uint8 `bitfield:"1"` // Bit 0: indicates namespace is currently write protected due to any error condition
		Reserved       uint8 `bitfield:"7"` // Bits 7:1 are reserved
	} // NSATTR
	NvmSetIdentifier             uint16    // NVMSETID
	EnduranceGroupIdentifier     uint16    // ENDGID
	GloballyUniqueIdentifier     [16]uint8 // NGUID
	IeeeExtendedUniqueIdentifier [8]uint8  // EUI64
	LBAFormats                   [16]struct {
		MetadataSize        uint16 // MS
		LBADataSize         uint8  // LBADS Indicates the LBA data size supported in terms of a power-of-two value. If the value is 0, the the LBA format is not supported
		RelativePerformance uint8  // RP indicates the relative performance of this LBA formt relative to other LBA formats
	}
	Reserved192    [192]uint8
	VendorSpecific [3712]uint8
}

type FormattedLBASize struct {
	Format   uint8 `bitfield:"4"` // Bits 3:0: Indicates one of the 16 supported LBA Formats
	Metadata uint8 `bitfield:"1"` // Bit 4: If 1, indicates metadata is transfer at the end of the data LBA
	Reserved uint8 `bitfield:"3"` // Bits 7:5 are reserved
} // FLBAS

type NamespaceCapabilities struct {
	Sharing  uint8 `bitfield:"1"` // Bit 0: If 1, namespace may be attached to two ro more controllers in the NVM subsystem concurrently
	Reserved uint8 `bitfield:"7"` // Bits 7:1 are reserved
} // NMIC

type CtrlList struct {
	Num         uint16 `countOf:"Identifiers"`
	Identifiers [2047]uint16
}

func (dev *Device) CreateNamespace(size uint64, capacity uint64, format uint8, dps uint8, sharing uint8, anagrpid uint32, nvmsetid uint16, timeout uint32) (uint32, error) {

	ns := IdNs{
		Size:                               size,
		Capacity:                           capacity,
		FormattedLBASize:                   FormattedLBASize{Format: format},
		EndToEndDataProtectionTypeSettings: dps,
		MultiPathIOSharingCapabilities:     NamespaceCapabilities{Sharing: sharing},
		AnaGroupIdentifier:                 anagrpid,
		NvmSetIdentifier:                   nvmsetid,
	}

	buf, err := structex.EncodeByteBuffer(ns)
	if err != nil {
		return 0, err
	}

	cmd := AdminCmd{
		Opcode:    uint8(NamespaceManagementOpCode),
		Addr:      uint64(uintptr(unsafe.Pointer(&ns))),
		Cdw10:     0,
		DataLen:   uint32(len(buf)),
		TimeoutMs: timeout,
	}

	if err := dev.ops.submitAdminPassthru(dev, &cmd, buf); err != nil {
		return 0, err
	}

	return cmd.Result, nil
}

func (dev *Device) DeleteNamespace(namespaceID uint32) error {
	cmd := AdminCmd{
		Opcode: uint8(NamespaceManagementOpCode),
		NSID:   namespaceID,
		Cdw10:  1,
	}

	return dev.ops.submitAdminPassthru(dev, &cmd, nil)
}

func (dev *Device) manageNamespace(namespaceID uint32, controllers []uint16, attach bool) error {

	list := CtrlList{
		Num: uint16(len(controllers)),
	}

	copy(list.Identifiers[:], controllers[:])

	buf := structex.NewBuffer(list)
	if buf == nil {
		return fmt.Errorf("Buffer allocation failed")
	}

	if err := structex.Encode(buf, list); err != nil {
		return fmt.Errorf("Encoding error")
	}

	cmd := AdminCmd{
		Opcode:  uint8(NamespaceAttachOpCode),
		NSID:    namespaceID,
		Addr:    uint64(uintptr(unsafe.Pointer(&list))),
		Cdw10:   map[bool]uint32{true: 0, false: 1}[attach],
		DataLen: 0x1000,
	}

	return dev.ops.submitAdminPassthru(dev, &cmd, buf.Bytes())
}

func (dev *Device) AttachNamespace(namespaceID uint32, controllers []uint16) error {
	return dev.manageNamespace(namespaceID, controllers, true)
}

func (dev *Device) DetachNamespace(namespaceID uint32, controllers []uint16) error {
	return dev.manageNamespace(namespaceID, controllers, false)
}

type VirtualManagementResourceType uint32

const (
	VQResourceType VirtualManagementResourceType = 0
	VIResourceType VirtualManagementResourceType = 1
)

var (
	VirtualManagementResourceTypeName = map[VirtualManagementResourceType]string{
		VQResourceType: "VQ",
		VIResourceType: "VI",
	}
)

type VirtualManagementAction uint32

const (
	PrimaryFlexibleAction  VirtualManagementAction = 0x1
	SecondaryOfflineAction VirtualManagementAction = 0x7
	SecondaryAssignAction  VirtualManagementAction = 0x8
	SecondaryOnlineAction  VirtualManagementAction = 0x9
)

func (dev *Device) VirtualMgmt(ctrlId uint16, action VirtualManagementAction, resourceType VirtualManagementResourceType, numResources uint32) error {

	var cdw10 uint32
	cdw10 = uint32(ctrlId) << 16
	cdw10 = cdw10 | uint32(resourceType<<8)
	cdw10 = cdw10 | uint32(action<<0)

	cmd := AdminCmd{
		Opcode: uint8(VirtualMgmtOpCode),
		Cdw10:  cdw10,
		Cdw11:  numResources,
	}

	return dev.ops.submitAdminPassthru(dev, &cmd, nil)

}

type SecondaryControllerEntry struct { // TODO: Add structex annotations
	SecondaryControllerID       uint16
	PrimaryControllerID         uint16
	SecondaryControllerState    uint8
	Reserved5                   [3]uint8
	VirtualFunctionNumber       uint16
	VQFlexibleResourcesAssigned uint16
	VIFlexibleResourcesAssigned uint16
	Reserved14                  [18]uint8
}

type SecondaryControllerList struct {
	Count    uint8 `countOf:"Entries"`
	Reserved [31]uint8
	Entries  [127]SecondaryControllerEntry
}

// ListSecondary retries the secondary controller list associated with the
// primary controller of the given device. This differs from the C implementation
// in that num-entries is not an option. The maximum number of entries is
// always returned. It is up to the caller to trim down the response.
func (dev *Device) ListSecondary(startingCtrlId uint32, namespaceId uint32) (*SecondaryControllerList, error) {

	list := new(SecondaryControllerList)

	buf := structex.NewBuffer(list)
	if buf == nil {
		return nil, fmt.Errorf("Cannot allocate bufffer")
	}

	if err := dev.IdentifyRaw(namespaceId, (startingCtrlId<<16)|uint32(SecondaryControllerList_CNS), 0, buf.Bytes()); err != nil {
		return nil, err
	}

	if err := structex.Decode(buf, &list); err != nil {
		return nil, err
	}

	return list, nil
}

type Feature uint8

const (
	NoneFeature                  Feature = 0x00
	ArbitrationFeature                   = 0x01
	PowerManagmentFeature                = 0x02
	LBARangeFeature                      = 0x03
	TemperatureThresholdFeature          = 0x04
	ErrorRecoveryFeature                 = 0x05
	VolatileWriteCacheFeature            = 0x06
	NumQueuesFeature                     = 0x07
	IRQCoalesceFeature                   = 0x08
	IRQConfigFeature                     = 0x09
	WriteAtomicFeature                   = 0x0a
	AsyncEventFeature                    = 0x0b
	AutoPSTFeature                       = 0x0c
	HostMemoryBufferFeature              = 0x0d
	TimestampFeature                     = 0x0e
	KATOFeature                          = 0x0f
	HCTMFeature                          = 0x10
	NoPSCFeature                         = 0x11
	RRLFeature                           = 0x12
	PLMConfigFeature                     = 0x13
	PLMWindowFeature                     = 0x14
	LBAStatusInfoFeature                 = 0x15
	HostBehaviorFeature                  = 0x16
	SanitizeFeature                      = 0x17
	EnduranceFeature                     = 0x18
	IOCSProfileFeature                   = 0x19
	SWProgressFeature                    = 0x80
	HostIDFeature                        = 0x81
	ReservationMaskFeature               = 0x82
	ReservationPersistentFeature         = 0x83
	WriteProtectFeature                  = 0x84
	MiControllerMetadata                 = 0x7E
	MiNamespaceMetadata                  = 0x7F
)

var FeatureBufferLength = [256]uint32{
	LBARangeFeature:         4096,
	AutoPSTFeature:          256,
	HostMemoryBufferFeature: 256,
	TimestampFeature:        8,
	PLMConfigFeature:        512,
	HostBehaviorFeature:     512,
	HostIDFeature:           8,
	MiControllerMetadata:    4096,
	MiNamespaceMetadata:     4096,
}

func (dev *Device) GetFeature(nsid uint32, fid Feature, sel int, cdw11 uint32, len uint32, buf []byte) error {
	cdw10 := uint32(fid) | uint32(sel)<<8

	return dev.feature(GetFeatures, nsid, cdw10, cdw11, 0, len, buf)
}

func (dev *Device) SetFeature(nsid uint32, fid Feature, cdw12 uint32, save bool, len uint32, buf []byte) error {
	cdw10 := uint32(fid)
	if save {
		cdw10 = cdw10 | (1 << 31)
	}

	return dev.feature(SetFeatures, nsid, cdw10, 0, cdw12, len, buf)
}

func (dev *Device) feature(opCode AdminCommandOpCode, nsid uint32, cdw10, cdw11, cdw12 uint32, len uint32, buf []byte) error {

	cmd := AdminCmd{
		Opcode:  uint8(opCode),
		NSID:    nsid,
		Cdw10:   cdw10,
		Cdw11:   cdw11,
		Cdw12:   cdw12,
		Addr:    uint64(uintptr(unsafe.Pointer(&buf[0]))),
		DataLen: len,
	}

	return dev.ops.submitAdminPassthru(dev, &cmd, buf)
}

type MIHostMetadata struct {
	NumDescriptors uint8
	Reserved1      uint8
	DescriptorData []byte
}

type MIHostMetadataElementDescriptor struct {
	Type uint8
	Rev  uint8
	Len  uint16
	Val  []byte
}

type HostMetadataElementType uint8

const (
	OsCtrlNameElementType           HostMetadataElementType = 0x01
	OsDriverNameElementType                                 = 0x02
	OsDriverVersionElementType                              = 0x03
	PreBootCtrlNameElementType                              = 0x04
	PreBootDriverNameElementType                            = 0x05
	PreBootDriverVersionElementType                         = 0x06

	OsNamespaceNameElementType      HostMetadataElementType = 0x01
	PreBootNamespaceNameElementType                         = 0x02
)

type MiMeatadataFeatureBuilder struct {
	data [4096]byte

	offset int
}

func NewMiFeatureBuilder() *MiMeatadataFeatureBuilder {
	return &MiMeatadataFeatureBuilder{offset: 2}
}

func (builder *MiMeatadataFeatureBuilder) AddElement(typ uint8, rev uint8, data []byte) *MiMeatadataFeatureBuilder {

	// Increment number of elements
	builder.data[0] = builder.data[0] + 1

	offset := builder.offset
	builder.data[offset] = typ
	offset++
	builder.data[offset] = rev
	offset++

	binary.LittleEndian.PutUint16(builder.data[offset:], uint16(len(data)))
	offset += 2

	copy(builder.data[offset:], data)
	offset += len(data)

	builder.offset = offset

	return builder
}

func (builder *MiMeatadataFeatureBuilder) Bytes() []byte {
	return builder.data[:builder.offset]
}
