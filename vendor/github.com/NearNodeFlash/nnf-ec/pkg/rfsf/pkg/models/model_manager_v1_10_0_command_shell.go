/*
 * Swordfish API
 *
 * This contains the definition of the Swordfish extensions to a Redfish service.
 *
 * API version: v1.2.c
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

// ManagerV1100CommandShell - The information about a command shell service that this manager provides.
type ManagerV1100CommandShell struct {

	// This property enumerates the command shell connection types that the implementation allows.
	ConnectTypesSupported []ManagerV1100CommandConnectTypesSupported `json:"ConnectTypesSupported,omitempty"`

	// The maximum number of service sessions, regardless of protocol, that this manager can support.
	MaxConcurrentSessions int64 `json:"MaxConcurrentSessions,omitempty"`

	// An indication of whether the service is enabled for this manager.
	ServiceEnabled bool `json:"ServiceEnabled,omitempty"`
}