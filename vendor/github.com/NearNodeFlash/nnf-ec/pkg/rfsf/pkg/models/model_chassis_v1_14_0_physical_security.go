/*
 * Swordfish API
 *
 * This contains the definition of the Swordfish extensions to a Redfish service.
 *
 * API version: v1.2.c
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

// ChassisV1140PhysicalSecurity - The state of the physical security sensor.
type ChassisV1140PhysicalSecurity struct {

	IntrusionSensor ChassisV1140IntrusionSensor `json:"IntrusionSensor,omitempty"`

	// A numerical identifier to represent the physical security sensor.
	IntrusionSensorNumber int64 `json:"IntrusionSensorNumber,omitempty"`

	IntrusionSensorReArm ChassisV1140IntrusionSensorReArm `json:"IntrusionSensorReArm,omitempty"`
}
