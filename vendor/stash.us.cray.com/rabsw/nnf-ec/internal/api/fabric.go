package api

import (
	"stash.us.cray.com/rabsw/nnf-ec/internal/events"
)

// FabricApi - Presents an API into the fabric outside of the fabric manager
// TODO: This should be obsolete - the NVMe Namespace Manager can
//       include the Fabric Manager (but NOT the other way around!!! Go doesn't
//       support circular bindings.
type FabricControllerApi interface {
	// Takes a PortEvent and converts it to the realtive port index within the Fabric Controller.
	// For example, if the fabric consists of a single switch USP and 4 DSPs labeled 0,1,2,3  then
	// a port event of type DSP with event attributes: <FabricId = 0, SwitchId = 0, PortId = 2>
	// would return 2 as the DSP index is 2 is the second DSP type within the fabric.
	ConvertPortEventToRelativePortIndex(events.PortEvent) (int, error)

	FindDownstreamEndpoint(portId, functionId string) (string, error)
}

// FabricDeviceControllerApi defines the interface for controlling a device on the fabric.
type FabricDeviceControllerApi interface {
}

var FabricController FabricControllerApi

func RegisterFabricController(f FabricControllerApi) {
	FabricController = f
}
