/*
 * Swordfish API
 *
 * This contains the definition of the Swordfish extensions to a Redfish service.
 *
 * API version: v1.2.c
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

type MemoryV1100ErrorCorrection string

// List of Memory_v1_10_0_ErrorCorrection
const (
	NO_ECC_MV1100EC MemoryV1100ErrorCorrection = "NoECC"
	SINGLE_BIT_ECC_MV1100EC MemoryV1100ErrorCorrection = "SingleBitECC"
	MULTI_BIT_ECC_MV1100EC MemoryV1100ErrorCorrection = "MultiBitECC"
	ADDRESS_PARITY_MV1100EC MemoryV1100ErrorCorrection = "AddressParity"
)
