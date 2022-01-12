/*
 * Swordfish API
 *
 * This contains the definition of the Swordfish extensions to a Redfish service.
 *
 * API version: v1.2.c
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

type PortV130FlowControl string

// List of Port_v1_3_0_FlowControl
const (
	NONE_PV130FC PortV130FlowControl = "None"
	TX_PV130FC PortV130FlowControl = "TX"
	RX_PV130FC PortV130FlowControl = "RX"
	TX_RX_PV130FC PortV130FlowControl = "TX_RX"
)