/*
 * Swordfish API
 *
 * This contains the definition of the Swordfish extensions to a Redfish service.
 *
 * API version: v1.2.c
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

// StorageControllerV100NvMeControllerProperties - NVMe related properties for a storage controller.
type StorageControllerV100NvMeControllerProperties struct {

	// The ANA characteristics and volume information.
	ANACharacteristics []StorageControllerV100AnaCharacteristics `json:"ANACharacteristics,omitempty"`

	ControllerType StorageControllerV100NVMeControllerType `json:"ControllerType,omitempty"`

	// The maximum individual queue size that an NVMe IO controller supports.
	MaxQueueSize int64 `json:"MaxQueueSize,omitempty"`

	NVMeControllerAttributes StorageControllerV100NvMeControllerAttributes `json:"NVMeControllerAttributes,omitempty"`

	NVMeSMARTCriticalWarnings StorageControllerV100NvMeSmartCriticalWarnings `json:"NVMeSMARTCriticalWarnings,omitempty"`

	// The version of the NVMe Base Specification supported.
	NVMeVersion string `json:"NVMeVersion,omitempty"`
}
