/*
 * Swordfish API
 *
 * This contains the definition of the Swordfish extensions to a Redfish service.
 *
 * API version: v1.2.c
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

// PcIeFunctionV123Links - The links to other Resources that are related to this Resource.
type PcIeFunctionV123Links struct {

	// An array of links to the drives that the PCIe device produces.
	Drives []OdataV4IdRef `json:"Drives,omitempty"`

	// The number of items in a collection.
	DrivesodataCount int64 `json:"Drives@odata.count,omitempty"`

	// An array of links to the Ethernet interfaces that the PCIe device produces.
	EthernetInterfaces []OdataV4IdRef `json:"EthernetInterfaces,omitempty"`

	// The number of items in a collection.
	EthernetInterfacesodataCount int64 `json:"EthernetInterfaces@odata.count,omitempty"`

	// An array of links to the network device functions that the PCIe device produces.
	NetworkDeviceFunctions []OdataV4IdRef `json:"NetworkDeviceFunctions,omitempty"`

	// The number of items in a collection.
	NetworkDeviceFunctionsodataCount int64 `json:"NetworkDeviceFunctions@odata.count,omitempty"`

	// The OEM extension.
	Oem map[string]interface{} `json:"Oem,omitempty"`

	PCIeDevice OdataV4IdRef `json:"PCIeDevice,omitempty"`

	// An array of links to the storage controllers that the PCIe device produces.
	StorageControllers []StorageStorageController `json:"StorageControllers,omitempty"`

	// The number of items in a collection.
	StorageControllersodataCount int64 `json:"StorageControllers@odata.count,omitempty"`
}
