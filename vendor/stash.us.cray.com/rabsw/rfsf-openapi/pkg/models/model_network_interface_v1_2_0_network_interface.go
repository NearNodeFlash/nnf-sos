/*
 * Swordfish API
 *
 * This contains the definition of the Swordfish extensions to a Redfish service.
 *
 * API version: v1.2.c
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

// NetworkInterfaceV120NetworkInterface - The NetworkInterface schema describes links to the network adapters, network ports, and network device functions, and represents the functionality available to the containing system.
type NetworkInterfaceV120NetworkInterface struct {

	// The OData description of a payload.
	OdataContext string `json:"@odata.context,omitempty"`

	// The current ETag of the resource.
	OdataEtag string `json:"@odata.etag,omitempty"`

	// The unique identifier for a resource.
	OdataId string `json:"@odata.id"`

	// The type of a resource.
	OdataType string `json:"@odata.type"`

	Actions NetworkInterfaceV120Actions `json:"Actions,omitempty"`

	// The description of this resource.  Used for commonality in the schema definitions.
	Description string `json:"Description,omitempty"`

	// The identifier that uniquely identifies the resource within the collection of similar resources.
	Id string `json:"Id"`

	Links NetworkInterfaceV120Links `json:"Links,omitempty"`

	// The name of the resource or array member.
	Name string `json:"Name"`

	NetworkDeviceFunctions OdataV4IdRef `json:"NetworkDeviceFunctions,omitempty"`

	NetworkPorts OdataV4IdRef `json:"NetworkPorts,omitempty"`

	// The OEM extension.
	Oem map[string]interface{} `json:"Oem,omitempty"`

	Ports OdataV4IdRef `json:"Ports,omitempty"`

	Status ResourceStatus `json:"Status,omitempty"`
}
