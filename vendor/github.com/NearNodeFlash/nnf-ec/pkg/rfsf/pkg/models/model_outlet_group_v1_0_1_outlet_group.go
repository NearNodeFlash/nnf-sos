/*
 * Swordfish API
 *
 * This contains the definition of the Swordfish extensions to a Redfish service.
 *
 * API version: v1.2.c
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

// OutletGroupV101OutletGroup - The OutletGroup schema contains definitions for an electrical outlet group.
type OutletGroupV101OutletGroup struct {

	// The OData description of a payload.
	OdataContext string `json:"@odata.context,omitempty"`

	// The current ETag of the resource.
	OdataEtag string `json:"@odata.etag,omitempty"`

	// The unique identifier for a resource.
	OdataId string `json:"@odata.id"`

	// The type of a resource.
	OdataType string `json:"@odata.type"`

	Actions OutletGroupV101Actions `json:"Actions,omitempty"`

	// The creator of this outlet group.
	CreatedBy string `json:"CreatedBy,omitempty"`

	// The description of this resource.  Used for commonality in the schema definitions.
	Description string `json:"Description,omitempty"`

	EnergykWh SensorSensorEnergykWhExcerpt `json:"EnergykWh,omitempty"`

	// The identifier that uniquely identifies the resource within the collection of similar resources.
	Id string `json:"Id"`

	Links OutletGroupV101Links `json:"Links,omitempty"`

	// The name of the resource or array member.
	Name string `json:"Name"`

	// The OEM extension.
	Oem map[string]interface{} `json:"Oem,omitempty"`

	// The number of seconds to delay power on after a PowerControl action to cycle power.  Zero seconds indicates no delay.
	PowerCycleDelaySeconds *float32 `json:"PowerCycleDelaySeconds,omitempty"`

	// Indicates if the outlet group can be powered.
	PowerEnabled bool `json:"PowerEnabled,omitempty"`

	// The number of seconds to delay power off after a PowerControl action.  Zero seconds indicates no delay to power off.
	PowerOffDelaySeconds *float32 `json:"PowerOffDelaySeconds,omitempty"`

	// The number of seconds to delay power up after a power cycle or a PowerControl action.  Zero seconds indicates no delay to power up.
	PowerOnDelaySeconds *float32 `json:"PowerOnDelaySeconds,omitempty"`

	// The number of seconds to delay power on after power has been restored.  Zero seconds indicates no delay.
	PowerRestoreDelaySeconds *float32 `json:"PowerRestoreDelaySeconds,omitempty"`

	PowerRestorePolicy CircuitPowerRestorePolicyTypes `json:"PowerRestorePolicy,omitempty"`

	PowerState ResourcePowerState `json:"PowerState,omitempty"`

	PowerWatts SensorSensorPowerExcerpt `json:"PowerWatts,omitempty"`

	Status ResourceStatus `json:"Status,omitempty"`
}
