/*
 * Swordfish API
 *
 * This contains the definition of the Swordfish extensions to a Redfish service.
 *
 * API version: v1.2.c
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

// VolumeV161CreateReplicaTarget - This action is used to create a new volume resource to provide expanded data protection through a replica relationship with the specified source volume.
type VolumeV161CreateReplicaTarget struct {

	// Link to invoke action
	Target string `json:"target,omitempty"`

	// Friendly action name
	Title string `json:"title,omitempty"`
}