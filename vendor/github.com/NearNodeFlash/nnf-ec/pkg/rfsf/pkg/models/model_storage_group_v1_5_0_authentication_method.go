/*
 * Swordfish API
 *
 * This contains the definition of the Swordfish extensions to a Redfish service.
 *
 * API version: v1.2.c
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

type StorageGroupV150AuthenticationMethod string

// List of StorageGroup_v1_5_0_AuthenticationMethod
const (
	NONE_SGV150AM StorageGroupV150AuthenticationMethod = "None"
	CHAP_SGV150AM StorageGroupV150AuthenticationMethod = "CHAP"
	MUTUAL_CHAP_SGV150AM StorageGroupV150AuthenticationMethod = "MutualCHAP"
	DHCHAP_SGV150AM StorageGroupV150AuthenticationMethod = "DHCHAP"
)
