package fabric

import (
	"stash.us.cray.com/rabsw/nnf-ec/internal/switchtec/pkg/switchtec"
)

type SwitchtecControllerInterface interface {
	Open(path string) (SwitchtecDeviceInterface, error)
}

type SwitchtecDeviceInterface interface {
	// these are ugly getters for the nvme devices to get attributes
	// about the device interface. Not sure how to make this less crap
	// without adding an unnecessarily large abstraction. I'll keep it
	// for now until something better is found.
	Device() *switchtec.Device
	Path() *string

	Close()

	Identify() (int32, error)

	GetFirmwareVersion() (string, error)
	GetModel() (string, error)
	GetManufacturer() (string, error)
	GetSerialNumber() (string, error)

	GetPortStatus() ([]switchtec.PortLinkStat, error)
	GetPortMetrics() (PortMetrics, error)
	GetEvents() ([]switchtec.GfmsEvent, error)

	EnumerateEndpoint(uint8, func(epPort *switchtec.DumpEpPortDevice) error) error

	Bind(uint8, uint8, uint16) error
}

type PortMetric struct {
	PhysPortId uint8

	// NOTE: This is a terrible definition in the switchtec package - these are raw Tx/Rx counter values
	// that can be _used_ for calculating the bandwidth. Think about renaming this in the switchtec package
	// to something like "PortMetrics"
	switchtec.BandwidthCounter
}

type PortMetrics []PortMetric
