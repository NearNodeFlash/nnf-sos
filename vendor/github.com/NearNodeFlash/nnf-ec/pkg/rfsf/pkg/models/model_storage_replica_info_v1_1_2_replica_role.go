/*
 * Swordfish API
 *
 * This contains the definition of the Swordfish extensions to a Redfish service.
 *
 * API version: v1.2.c
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi
// StorageReplicaInfoV112ReplicaRole : Values of ReplicaRole specify whether the resource is a source of replication or the target of replication.
type StorageReplicaInfoV112ReplicaRole string

// List of StorageReplicaInfo_v1_1_2_ReplicaRole
const (
	SOURCE_SRIV112RR StorageReplicaInfoV112ReplicaRole = "Source"
	TARGET_SRIV112RR StorageReplicaInfoV112ReplicaRole = "Target"
)
