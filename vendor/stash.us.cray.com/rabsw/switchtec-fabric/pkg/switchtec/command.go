package switchtec

// A Command describes the executable command sent to the device.
type Command uint32

// The supported command set for a device.
const (
	// Global Address Space commands
	LinkStatCommand Command = 28
	GASReadCommand  Command = 0x29 // 41
	PartitionInfo   Command = 0x2B // 43
	GASWriteCommand Command = 0x34 // 52

	EchoCommand     Command = 0x41 // 65
	GetPaxIDCommand Command = 0x81 // 129

	// Global Fabric Management Service commands
	DumpCommand              Command = 0x83
	BindUnbindCommand        Command = 0x84
	DeviceManageCommand      Command = 0x85 // Obsolete - See NvmeAdminPassthru
	PortConfigCommand        Command = 0x88
	GfmsEventCommand         Command = 0x89
	PortControlCommand       Command = 0x8D
	EpResourceAccessCommand  Command = 0x8E
	EpTunnelConfigCommand    Command = 0x8F
	NvmeAdminPassthruCommand Command = 0x91

	GetDeviceInfo          Command = 0x100
	SerialNumberSecVersion Command = 0x109
)

func (cmd Command) String() string {
	switch cmd {
	case LinkStatCommand:
		return "LinkStat"
	case GASReadCommand:
		return "GAS Read"
	case PartitionInfo:
		return "Partition Info"
	case GASWriteCommand:
		return "GAS Write"
	case EchoCommand:
		return "Echo"
	case GetPaxIDCommand:
		return "Get PAX ID"
	case DumpCommand:
		return "Dump"
	case BindUnbindCommand:
		return "Bind-Unbind"
	case DeviceManageCommand:
		return "Device Manage"
	case PortConfigCommand:
		return "Port Config"
	case GfmsEventCommand:
		return "GFMS Event"
	case PortControlCommand:
		return "Port Control"
	case EpResourceAccessCommand:
		return "EP Resource Access"
	case EpTunnelConfigCommand:
		return "EP Tunnel Config"
	case NvmeAdminPassthruCommand:
		return "NVMe Admin Passthru"
	case GetDeviceInfo:
		return "Get Device Info"
	case SerialNumberSecVersion:
		return "Serial Number Security Version"
	default:
		return "Unknown"
	}
}

// SubCommand -
type SubCommand uint8

const (
	PortBindSubCommand   SubCommand = 1
	PortUnbindSubCommand SubCommand = 2

	PartitionInfoGetAllInfoSubCommand  SubCommand = 0
	PartitionInfoGetMetadataSubCommand SubCommand = 1

	TopologyInfoDumpStartSubCommand     SubCommand = 1
	TopologyInfoDumpStatusGetSubCommand SubCommand = 2
	TopologyInfoDumpDataGetSubCommand   SubCommand = 3
	TopologyInfoDumpFinishSubCommand    SubCommand = 4

	GfmsDumpStartSubCommand  = 1
	GfmsDumpGetSubCommand    = 2
	GfmsDumpFinishSubCommand = 3

	NvmeAdminPassthruStart SubCommand = 1
	NvmeAdminPassthruData  SubCommand = 2
	NvmeAdminPassthruEnd   SubCommand = 3

	ClearGfmsEventsSubCommand SubCommand = 0
	GetGfmsEventsSubCommand   SubCommand = 1
)

// Maximum length of any command, in bytes
const (
	maxDataLength = 1024
)
