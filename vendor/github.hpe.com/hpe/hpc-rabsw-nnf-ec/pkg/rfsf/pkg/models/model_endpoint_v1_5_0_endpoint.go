/*
 * Swordfish API
 *
 * This contains the definition of the Swordfish extensions to a Redfish service.
 *
 * API version: v1.2.c
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

// EndpointV150Endpoint - The Endpoint schema contains the properties of an endpoint resource that represents the properties of an entity that sends or receives protocol-defined messages over a transport.
type EndpointV150Endpoint struct {

	// The OData description of a payload.
	OdataContext string `json:"@odata.context,omitempty"`

	// The current ETag of the resource.
	OdataEtag string `json:"@odata.etag,omitempty"`

	// The unique identifier for a resource.
	OdataId string `json:"@odata.id"`

	// The type of a resource.
	OdataType string `json:"@odata.type"`

	Actions EndpointV150Actions `json:"Actions,omitempty"`

	// All the entities connected to this endpoint.
	ConnectedEntities []EndpointV150ConnectedEntity `json:"ConnectedEntities,omitempty"`

	// The description of this resource.  Used for commonality in the schema definitions.
	Description string `json:"Description,omitempty"`

	EndpointProtocol ProtocolProtocol `json:"EndpointProtocol,omitempty"`

	// The amount of memory in bytes that the host should allocate to connect to this endpoint.
	HostReservationMemoryBytes int64 `json:"HostReservationMemoryBytes,omitempty"`

	// An array of details for each IP transport supported by this endpoint.  The array structure can model multiple IP addresses for this endpoint.
	IPTransportDetails []EndpointV150IpTransportDetails `json:"IPTransportDetails,omitempty"`

	// The identifier that uniquely identifies the resource within the collection of similar resources.
	Id string `json:"Id"`

	// Identifiers for this endpoint.
	Identifiers []ResourceIdentifier `json:"Identifiers,omitempty"`

	Links EndpointV150Links `json:"Links,omitempty"`

	// The name of the resource or array member.
	Name string `json:"Name"`

	// The OEM extension.
	Oem map[string]interface{} `json:"Oem,omitempty"`

	PciId EndpointV150PciId `json:"PciId,omitempty"`

	// Redundancy information for the lower-level endpoints supporting this endpoint.
	Redundancy []RedundancyRedundancy `json:"Redundancy,omitempty"`

	// The number of items in a collection.
	RedundancyodataCount int64 `json:"Redundancy@odata.count,omitempty"`

	Status ResourceStatus `json:"Status,omitempty"`
}