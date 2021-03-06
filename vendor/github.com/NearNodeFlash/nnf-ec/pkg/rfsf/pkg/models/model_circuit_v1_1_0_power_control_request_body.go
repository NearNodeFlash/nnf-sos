/*
 * Swordfish API
 *
 * This contains the definition of the Swordfish extensions to a Redfish service.
 *
 * API version: v1.2.c
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

// CircuitV110PowerControlRequestBody - This action turns the circuit on or off.
type CircuitV110PowerControlRequestBody struct {

	PowerState ResourcePowerState `json:"PowerState,omitempty"`
}
