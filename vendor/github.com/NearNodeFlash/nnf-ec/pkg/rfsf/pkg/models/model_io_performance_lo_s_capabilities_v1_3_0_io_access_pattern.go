/*
 * Swordfish API
 *
 * This contains the definition of the Swordfish extensions to a Redfish service.
 *
 * API version: v1.2.c
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi
// IOPerformanceLoSCapabilitiesV130IOAccessPattern : Typical IO access patterns.
type IOPerformanceLoSCapabilitiesV130IOAccessPattern string

// List of IOPerformanceLoSCapabilities_v1_3_0_IOAccessPattern
const (
	READ_WRITE_IOPLSCV130IOAP IOPerformanceLoSCapabilitiesV130IOAccessPattern = "ReadWrite"
	SEQUENTIAL_READ_IOPLSCV130IOAP IOPerformanceLoSCapabilitiesV130IOAccessPattern = "SequentialRead"
	SEQUENTIAL_WRITE_IOPLSCV130IOAP IOPerformanceLoSCapabilitiesV130IOAccessPattern = "SequentialWrite"
	RANDOM_READ_NEW_IOPLSCV130IOAP IOPerformanceLoSCapabilitiesV130IOAccessPattern = "RandomReadNew"
	RANDOM_READ_AGAIN_IOPLSCV130IOAP IOPerformanceLoSCapabilitiesV130IOAccessPattern = "RandomReadAgain"
)
