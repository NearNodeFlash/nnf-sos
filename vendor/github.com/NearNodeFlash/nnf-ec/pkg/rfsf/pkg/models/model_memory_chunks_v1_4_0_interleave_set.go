/*
 * Swordfish API
 *
 * This contains the definition of the Swordfish extensions to a Redfish service.
 *
 * API version: v1.2.c
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

// MemoryChunksV140InterleaveSet - This an interleave set for a memory chunk.
type MemoryChunksV140InterleaveSet struct {

	Memory OdataV4IdRef `json:"Memory,omitempty"`

	// Level of the interleave set for multi-level tiered memory.
	MemoryLevel int64 `json:"MemoryLevel,omitempty"`

	// Offset within the DIMM that corresponds to the start of this memory region, measured in mebibytes (MiB).
	OffsetMiB int64 `json:"OffsetMiB,omitempty"`

	// DIMM region identifier.
	RegionId string `json:"RegionId,omitempty"`

	// Size of this memory region measured in mebibytes (MiB).
	SizeMiB int64 `json:"SizeMiB,omitempty"`
}