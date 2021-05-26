/*
 * Swordfish API
 *
 * This contains the definition of the Swordfish extensions to a Redfish service.
 *
 * API version: v1.2.c
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

// StorageReplicaInfoReplicaInfo - Defines the characteristics of a replica of a source.
type StorageReplicaInfoReplicaInfo struct {

	// True if consistency is enabled.
	ConsistencyEnabled bool `json:"ConsistencyEnabled,omitempty"`

	ConsistencyState StorageReplicaInfoV130ConsistencyState `json:"ConsistencyState,omitempty"`

	ConsistencyStatus StorageReplicaInfoV130ConsistencyStatus `json:"ConsistencyStatus,omitempty"`

	ConsistencyType StorageReplicaInfoV130ConsistencyType `json:"ConsistencyType,omitempty"`

	// If true, the storage array tells host to stop sending data to source element if copying to a remote element fails.
	FailedCopyStopsHostIO bool `json:"FailedCopyStopsHostIO,omitempty"`

	// Specifies the percent of the work completed to reach synchronization.
	PercentSynced int64 `json:"PercentSynced,omitempty"`

	Replica OdataV4IdRef `json:"Replica,omitempty"`

	ReplicaPriority StorageReplicaInfoV130ReplicaPriority `json:"ReplicaPriority,omitempty"`

	ReplicaProgressStatus StorageReplicaInfoV130ReplicaProgressStatus `json:"ReplicaProgressStatus,omitempty"`

	ReplicaReadOnlyAccess StorageReplicaInfoV130ReplicaReadOnlyAccess `json:"ReplicaReadOnlyAccess,omitempty"`

	ReplicaRecoveryMode StorageReplicaInfoV130ReplicaRecoveryMode `json:"ReplicaRecoveryMode,omitempty"`

	ReplicaRole StorageReplicaInfoV130ReplicaRole `json:"ReplicaRole,omitempty"`

	// Applies to Adaptive mode and it describes maximum number of bytes the SyncedElement (target) can be out of sync.
	ReplicaSkewBytes int64 `json:"ReplicaSkewBytes,omitempty"`

	ReplicaState StorageReplicaInfoV130ReplicaState `json:"ReplicaState,omitempty"`

	ReplicaType StorageReplicaInfoReplicaType `json:"ReplicaType,omitempty"`

	ReplicaUpdateMode StorageReplicaInfoReplicaUpdateMode `json:"ReplicaUpdateMode,omitempty"`

	RequestedReplicaState StorageReplicaInfoV130ReplicaState `json:"RequestedReplicaState,omitempty"`

	// Synchronization is maintained.
	SyncMaintained bool `json:"SyncMaintained,omitempty"`

	UndiscoveredElement StorageReplicaInfoV130UndiscoveredElement `json:"UndiscoveredElement,omitempty"`

	// Specifies when point-in-time copy was taken or when the replication relationship is activated, reactivated, resumed or re-established.
	WhenActivated string `json:"WhenActivated,omitempty"`

	// Specifies when the replication relationship is deactivated.
	WhenDeactivated string `json:"WhenDeactivated,omitempty"`

	// Specifies when the replication relationship is established.
	WhenEstablished string `json:"WhenEstablished,omitempty"`

	// Specifies when the replication relationship is suspended.
	WhenSuspended string `json:"WhenSuspended,omitempty"`

	// The point in time that the Elements were synchronized.
	WhenSynced string `json:"WhenSynced,omitempty"`

	// Specifies when the replication relationship is synchronized.
	WhenSynchronized string `json:"WhenSynchronized,omitempty"`

	DataProtectionLineOfService OdataV4IdRef `json:"DataProtectionLineOfService,omitempty"`

	SourceReplica OdataV4IdRef `json:"SourceReplica,omitempty"`

	ReplicaFaultDomain StorageReplicaInfoReplicaFaultDomain `json:"ReplicaFaultDomain,omitempty"`
}
