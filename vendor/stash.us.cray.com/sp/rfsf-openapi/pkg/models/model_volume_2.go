/*
 * Swordfish API
 *
 * This contains the definition of the Swordfish extensions to a Redfish service.
 *
 * API version: v1.0.7
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

// Volume2 - Volume contains properties used to describe a volume, virtual disk, LUN, or other logical storage entity for any system.
type Volume2 struct {
	OdataContext string `json:"@odata.context,omitempty"`

	OdataId string `json:"@odata.id,omitempty"`

	OdataType string `json:"@odata.type,omitempty"`

	Actions Volume2Actions `json:"Actions,omitempty"`

	// The size of the smallest addressible unit (Block) of this volume in bytes
	BlockSizeBytes float32 `json:"BlockSizeBytes,omitempty"`

	// The size in bytes of this Volume
	CapacityBytes float32 `json:"CapacityBytes,omitempty"`

	// An array of space allocations for this volume.
	CapacitySources CapacitySource `json:"CapacitySources,omitempty"`

	Description string `json:"Description,omitempty"`

	// Is this Volume encrypted
	Encrypted bool `json:"Encrypted,omitempty"`

	// The types of encryption used by this Volume
	EncryptionTypes []EncryptionTypes `json:"EncryptionTypes,omitempty"`

	Id string `json:"Id"`

	// The Durable names for the volume
	Identifiers []string `json:"Identifiers,omitempty"`

	Links Volume2Links `json:"Links,omitempty"`

	Name string `json:"Name"`

	Oem GridRaidOEM

	// The operations currently running on the Volume
	Operations []Operations2 `json:"Operations,omitempty"`

	// The size in bytes of this Volume's optimum IO size.
	OptimumIOSizeBytes float32 `json:"OptimumIOSizeBytes,omitempty"`

	Status map[string]interface{} `json:"Status,omitempty"`

	RAIDType string `json:"RAIDType,omitempty"`
}

type GridRaidOEM struct {
	UUID              string `json:"UUID,omitempty"`
	RaidLevel         string `json:"RaidLevel"` // Required
	BitMap            string `json:"BitMap,omitempty"`
	ChunkSize         int    `json:"ChunkSize,omitempty"`
	Size              int64  `json:"Size,omitempty"`
	Layout            string `json:"Layout,omitempty"`
	LayoutNear        int    `json:"LayoutNear,omitempty"`
	StripeWidth       int    `json:"StripeWidth,omitempty"`
	DistributedSpares int    `json:"DistributedSpares,omitempty"`
	WibInfoEntry      WibInfo
	JornalInfoEntry   JournalInfo
}
type WibInfo struct {
	ArrayName  string `json:"ArrayName"` //Required
	RaidLevel  string `json:"RaidLevel"` //Required
	ChunkSize  int    `json:"ChunkSize,omitempty"`
	LayoutNear int    `json:"LayoutNear,omitempty"`
}

type JournalInfo struct {
	ArrayName  string `json:"ArrayName"` //Required
	RaidLevel  string `json:"RaidLevel"` //Required
	ChunkSize  int    `json:"ChunkSize,omitempty"`
	LayoutNear int    `json:"LayoutNear,omitempty"`
}
