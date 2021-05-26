/*
 * Swordfish API
 *
 * This contains the definition of the Swordfish extensions to a Redfish service.
 *
 * API version: v1.0.7
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

type FlowControl string

// List of FlowControl
const (
	NONE_FLOW FlowControl = "None"
	TX FlowControl = "TX"
	RX FlowControl = "RX"
	TX_RX FlowControl = "TX_RX"
)
