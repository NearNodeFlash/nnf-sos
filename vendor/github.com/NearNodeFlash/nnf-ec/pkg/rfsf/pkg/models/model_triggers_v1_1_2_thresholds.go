/*
 * Swordfish API
 *
 * This contains the definition of the Swordfish extensions to a Redfish service.
 *
 * API version: v1.2.c
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

// TriggersV112Thresholds - The set of thresholds for a sensor.
type TriggersV112Thresholds struct {

	LowerCritical TriggersV112Threshold `json:"LowerCritical,omitempty"`

	LowerWarning TriggersV112Threshold `json:"LowerWarning,omitempty"`

	UpperCritical TriggersV112Threshold `json:"UpperCritical,omitempty"`

	UpperWarning TriggersV112Threshold `json:"UpperWarning,omitempty"`
}
