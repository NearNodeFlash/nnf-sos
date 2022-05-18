/*
 * Swordfish API
 *
 * This contains the definition of the Swordfish extensions to a Redfish service.
 *
 * API version: v1.2.c
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi
// StorageReplicaInfoV120ReplicaRole : Values of ReplicaRole specify whether the resource is a source of replication or the target of replication.
type StorageReplicaInfoV120ReplicaRole string

// List of StorageReplicaInfo_v1_2_0_ReplicaRole
const (
	SOURCE_SRIV120RR StorageReplicaInfoV120ReplicaRole = "Source"
	TARGET_SRIV120RR StorageReplicaInfoV120ReplicaRole = "Target"
)