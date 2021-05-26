/*
 * Swordfish API
 *
 * This contains the definition of the Swordfish extensions to a Redfish service.
 *
 * API version: v1.2.c
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

// OperatingConfigV101OperatingConfig - The OperatingConfig schema specifies a configuration that can be used when the processor is operational.
type OperatingConfigV101OperatingConfig struct {

	// The OData description of a payload.
	OdataContext string `json:"@odata.context,omitempty"`

	// The current ETag of the resource.
	OdataEtag string `json:"@odata.etag,omitempty"`

	// The unique identifier for a resource.
	OdataId string `json:"@odata.id"`

	// The type of a resource.
	OdataType string `json:"@odata.type"`

	Actions OperatingConfigV101Actions `json:"Actions,omitempty"`

	// The base (nominal) clock speed of the processor in MHz.
	BaseSpeedMHz int64 `json:"BaseSpeedMHz,omitempty"`

	// The clock speed for sets of cores when the configuration is operational.
	BaseSpeedPrioritySettings []OperatingConfigV101BaseSpeedPrioritySettings `json:"BaseSpeedPrioritySettings,omitempty"`

	// The description of this resource.  Used for commonality in the schema definitions.
	Description string `json:"Description,omitempty"`

	// The identifier that uniquely identifies the resource within the collection of similar resources.
	Id string `json:"Id"`

	// The maximum temperature of the junction in degrees Celsius.
	MaxJunctionTemperatureCelsius int64 `json:"MaxJunctionTemperatureCelsius,omitempty"`

	// The maximum clock speed to which the processor can be configured in MHz.
	MaxSpeedMHz int64 `json:"MaxSpeedMHz,omitempty"`

	// The name of the resource or array member.
	Name string `json:"Name"`

	// The OEM extension.
	Oem map[string]interface{} `json:"Oem,omitempty"`

	// The thermal design point of the processor in watts.
	TDPWatts int64 `json:"TDPWatts,omitempty"`

	// The number of cores in the processor that can be configured.
	TotalAvailableCoreCount int64 `json:"TotalAvailableCoreCount,omitempty"`

	// The turbo profiles for the processor.  A turbo profile is the maximum turbo clock speed as a function of the number of active cores.
	TurboProfile []OperatingConfigV101TurboProfileDatapoint `json:"TurboProfile,omitempty"`
}
