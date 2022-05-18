/*
 * Swordfish API
 *
 * This contains the definition of the Swordfish extensions to a Redfish service.
 *
 * API version: v1.2.c
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi
// StorageReplicaInfoV130ConsistencyType : The values of ConsistencyType indicates the consistency type used by the source and its associated target group.
type StorageReplicaInfoV130ConsistencyType string

// List of StorageReplicaInfo_v1_3_0_ConsistencyType
const (
	SEQUENTIALLY_CONSISTENT_SRIV130CT StorageReplicaInfoV130ConsistencyType = "SequentiallyConsistent"
)