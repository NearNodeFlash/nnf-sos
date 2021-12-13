package api

// FabricApi - Presents an API into the fabric outside of the fabric manager
// TODO: This should be obsolete - the NVMe Namespace Manager can
//       include the Fabric Manager (but NOT the other way around!!! Go doesn't
//       support circular bindings.
type FabricControllerApi interface {
	// Locates the index of the Downstream Port DSP within the Fabric Controller's list of all Downstream Endpoints, regardless of current endpoint status.
	GetDownstreamPortRelativePortIndex(switchId, portId string) (int, error)

	FindDownstreamEndpoint(portId, functionId string) (string, error)
}

// FabricDeviceControllerApi defines the interface for controlling a device on the fabric.
type FabricDeviceControllerApi interface {
}

var FabricController FabricControllerApi

func RegisterFabricController(f FabricControllerApi) {
	FabricController = f
}
