package nnf

import (
	"fmt"
	"sort"

	"github.com/google/uuid"

	nvme "stash.us.cray.com/rabsw/nnf-ec/pkg/manager-nvme"

	openapi "stash.us.cray.com/rabsw/nnf-ec/pkg/rfsf/pkg/common"
)

// AllocationPolicy -
type AllocationPolicy interface {
	Initialize(capacityBytes uint64) error
	CheckCapacity() error
	Allocate(guid uuid.UUID) ([]ProvidingVolume, error)
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
	DefaultAlloctionPolicy     = SpareAllocationPolicyType
	DefaultAlloctionCompliance = StrictAllocationComplianceType
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

	policy := DefaultAlloctionPolicy
	compliance := DefaultAlloctionCompliance

	if oem != nil {
		overrides := AllocationPolicyOem{
			Policy:     DefaultAlloctionPolicy,
			Compliance: DefaultAlloctionCompliance,
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

type SpareAllocationPolicy struct {
	compliance     AllocationComplianceType
	storage        []*nvme.Storage
	capacityBytes  uint64
	allocatedBytes uint64
}

func (p *SpareAllocationPolicy) Initialize(capacityBytes uint64) error {

	storage := []*nvme.Storage{}
	for _, s := range nvme.GetStorage() {
		if s.IsEnabled() {
			storage = append(storage, s)
		}
	}

	// Sort the drives in decreasing order of unallocated bytes
	sort.Slice(storage, func(i, j int) bool {
		return !!!(storage[i].UnallocatedBytes() < storage[j].UnallocatedBytes())
	})

	count := 16
	if len(storage) < count {
		count = len(storage)
	}

	p.storage = storage[:count]
	p.capacityBytes = capacityBytes

	return nil
}

func (p *SpareAllocationPolicy) CheckCapacity() error {
	if p.capacityBytes == 0 {
		return fmt.Errorf("Requested capacity %#x if invalid", p.capacityBytes)
	}

	var availableBytes = uint64(0)
	for _, s := range p.storage {
		availableBytes += s.UnallocatedBytes()
	}

	if availableBytes < p.capacityBytes {
		return fmt.Errorf("Insufficient capacity available. Requested: %#x Available: %#x", p.capacityBytes, availableBytes)
	}

	if p.compliance != RelaxedAllocationComplianceType {

		if len(p.storage) != 16 {
			return fmt.Errorf("Insufficient drive count. Available %d", len(p.storage))
		}

		if p.capacityBytes%uint64(len(p.storage)) != 0 {
			return fmt.Errorf("Requested capacity is a non-multiple of available storage")
		}
	}

	return nil
}

func (p *SpareAllocationPolicy) Allocate(pid uuid.UUID) ([]ProvidingVolume, error) {

	perStorageCapacityBytes := p.capacityBytes / uint64(len(p.storage))
	remainingCapacityBytes := p.capacityBytes

	volumes := []ProvidingVolume{}
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

		remainingCapacityBytes = remainingCapacityBytes - volume.GetCapaityBytes()
		volumes = append(volumes, ProvidingVolume{storage: storage, volume: volume})
	}

	return volumes, nil
}

func createVolume(storage *nvme.Storage, capacityBytes uint64, pid uuid.UUID, idx, count int) (*nvme.Volume, error) {
	return nvme.CreateVolume(storage, capacityBytes)
}
