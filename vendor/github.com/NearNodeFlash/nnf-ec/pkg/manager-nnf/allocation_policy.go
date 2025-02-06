/*
 * Copyright 2020-2024 Hewlett Packard Enterprise Development LP
 * Other additional copyright holders may be indicated within.
 *
 * The entirety of this work is licensed under the Apache License,
 * Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License.
 *
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package nnf

import (
	"fmt"
	"sort"

	"github.com/google/uuid"

	nvme "github.com/NearNodeFlash/nnf-ec/pkg/manager-nvme"

	openapi "github.com/NearNodeFlash/nnf-ec/pkg/rfsf/pkg/common"
)

// AllocationPolicy -
type AllocationPolicy interface {
	Initialize(capacityBytes uint64) error
	CheckAndAdjustCapacity() error
	Allocate(guid uuid.UUID) ([]nvme.ProvidingVolume, error)
}

// AllocationPolicyType -
type AllocationPolicyType string

const (
	SpareAllocationPolicyType        AllocationPolicyType = "spare"
	GlobalAllocationPolicyType                            = "global"
	SwitchLocalAllocationPolicyType                       = "switch-local"
	ComputeLocalAllocationPolicyType                      = "compute-local"
)

// AllocationComplianceType -
type AllocationComplianceType string

const (
	StrictAllocationComplianceType  AllocationComplianceType = "strict"
	RelaxedAllocationComplianceType                          = "relaxed"
)

// Default AllocationPolicy and AllocationCompliance
const (
	DefaultAllocationPolicy     = SpareAllocationPolicyType
	DefaultAllocationCompliance = StrictAllocationComplianceType
)

// AllocationPolicyOem -
type AllocationPolicyOem struct {
	Policy     AllocationPolicyType     `json:"Policy,omitempty"`
	Compliance AllocationComplianceType `json:"Compliance,omitempty"`

	// This is a hint to the allocation policy on which server endpoint
	// will be receiving the pool. This is designed for switch-local and
	// compute-local where placement matters.
	ServerEndpointId string `json:"ServerEndpoint,omitempty"`
}

// NewAllocationPolicy - Allocates a new Allocation Policy with the desired parameters.
// The provided config is the defaults as defined by the NNF Config (see config.yaml);
// Knowledgable users have the option to specify overrides if the default configuration
// is not as desired.
func NewAllocationPolicy(config AllocationConfig, oem map[string]interface{}) AllocationPolicy {

	policy := DefaultAllocationPolicy
	compliance := DefaultAllocationCompliance

	if oem != nil {
		overrides := AllocationPolicyOem{
			Policy:     DefaultAllocationPolicy,
			Compliance: DefaultAllocationCompliance,
		}

		if err := openapi.UnmarshalOem(oem, &overrides); err == nil {
			if overrides.Policy != "default" {
				policy = overrides.Policy
			}

			if overrides.Compliance != "default" {
				compliance = overrides.Compliance
			}
		}
	}

	switch policy {
	case SpareAllocationPolicyType:
		return &SpareAllocationPolicy{compliance: compliance}
	case GlobalAllocationPolicyType:
		return nil // TODO?
	case SwitchLocalAllocationPolicyType:
		return nil // TODO?
	case ComputeLocalAllocationPolicyType:
		return nil // TODO?
	}

	return nil
}

/* ------------------------------ Spare Allocation Policy --------------------- */

// Required number of drives for the Spare Allocation Policy
const SpareAllocationPolicyRequiredDriveCount = 16

type SpareAllocationPolicy struct {
	compliance     AllocationComplianceType
	storage        []*nvme.Storage
	capacityBytes  uint64
	allocatedBytes uint64
}

// Initialize the policy
func (p *SpareAllocationPolicy) Initialize(capacityBytes uint64) error {

	storage := []*nvme.Storage{}
	for _, s := range nvme.GetStorage() {
		if s.IsEnabled() && s.UnallocatedBytes() > 0 {
			storage = append(storage, s)
		}
	}

	// Sort the drives in decreasing order of unallocated bytes
	sort.Slice(storage, func(i, j int) bool {
		return !!!(storage[i].UnallocatedBytes() < storage[j].UnallocatedBytes())
	})

	count := SpareAllocationPolicyRequiredDriveCount
	if len(storage) < count {
		count = len(storage)
	}

	p.storage = storage[:count]
	p.capacityBytes = capacityBytes

	return nil
}

// CheckAndAdjustCapacity - check the policy and adjust capacity to match policy if possible
func (p *SpareAllocationPolicy) CheckAndAdjustCapacity() error {
	if p.capacityBytes == 0 {
		return fmt.Errorf("Requested capacity must be non-zero")
	}

	var availableBytes = uint64(0)
	for _, s := range p.storage {
		availableBytes += s.UnallocatedBytes()
	}

	if p.compliance != RelaxedAllocationComplianceType {

		if len(p.storage) != SpareAllocationPolicyRequiredDriveCount {
			return fmt.Errorf("Insufficient drive count. Required: %d Available: %d", SpareAllocationPolicyRequiredDriveCount, len(p.storage))
		}

		roundUpToMultiple := func(n, m uint64) uint64 { // Round 'n' up to a multiple of 'm'
			return ((n + m - 1) / m) * m
		}

		// Validate each drive can contribute sufficient capacity towards the entire pool.
		poolCapacityBytes := roundUpToMultiple(p.capacityBytes, SpareAllocationPolicyRequiredDriveCount)
		driveCapacityBytes := roundUpToMultiple(poolCapacityBytes/SpareAllocationPolicyRequiredDriveCount, 4096)

		for _, s := range p.storage {
			if driveCapacityBytes > s.UnallocatedBytes() {
				return fmt.Errorf("Insufficient drive capacity available. Requested: %d Available: %d", driveCapacityBytes, s.UnallocatedBytes())
			}
		}

		// Adjust the pool's capacity such that it is a multiple of the number drives.
		p.capacityBytes = driveCapacityBytes * uint64(len(p.storage))
	}

	if availableBytes < p.capacityBytes {
		return fmt.Errorf("Insufficient capacity available. Requested: %d Available: %d", p.capacityBytes, availableBytes)
	}

	return nil
}

// Allocate - allocate the storage
func (p *SpareAllocationPolicy) Allocate(pid uuid.UUID) ([]nvme.ProvidingVolume, error) {

	perStorageCapacityBytes := p.capacityBytes / uint64(len(p.storage))
	remainingCapacityBytes := p.capacityBytes

	volumes := []nvme.ProvidingVolume{}
	for idx, storage := range p.storage {

		capacityBytes := perStorageCapacityBytes

		// Leftover bytes are placed on trailing volume; note that this
		// is never the case for strict allocation in which the requested
		// allocation must be a multiple of the storage size.
		if idx == len(p.storage)-1 {
			capacityBytes = remainingCapacityBytes
		}

		volume, err := createVolume(storage, capacityBytes, pid, idx, len(p.storage))

		if err != nil {
			return volumes, fmt.Errorf("Create Volume Failure: %s", err)
		}

		remainingCapacityBytes = remainingCapacityBytes - volume.GetCapacityBytes()
		volumes = append(volumes, nvme.ProvidingVolume{Storage: storage, VolumeId: volume.Id()})
	}

	return volumes, nil
}

func createVolume(storage *nvme.Storage, capacityBytes uint64, pid uuid.UUID, idx, count int) (*nvme.Volume, error) {
	return nvme.CreateVolume(storage, capacityBytes)
}
