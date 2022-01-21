/*
Copyright 2021 Hewlett Packard Enterprise Development LP
*/

package v1alpha1

import (
	sf "github.hpe.com/hpe/hpc-rabsw-nnf-ec/pkg/rfsf/pkg/models"
)

// NnfResourceHealthType defines the health of an NNF resource.
type NnfResourceHealthType string

const (
	// ResourceOkay is SF health OK
	ResourceOkay NnfResourceHealthType = NnfResourceHealthType(sf.OK_RH)

	// ResourceWarning is SF health WARNING
	ResourceWarning = NnfResourceHealthType(sf.WARNING_RH)

	// ResourceCritical is SF health CRITICAL
	ResourceCritical = NnfResourceHealthType(sf.CRITICAL_RH)
)

// ResourceHealth maps a SF ResourceStatus to an NNFResourceHealthType
func ResourceHealth(s sf.ResourceStatus) NnfResourceHealthType {
	switch s.Health {
	case sf.OK_RH:
		return ResourceOkay
	case sf.WARNING_RH:
		return ResourceWarning
	case sf.CRITICAL_RH:
		return ResourceCritical
	}

	panic("Unknown Resource Health " + string(s.Health))
}

// UpdateIfWorseThan examines the input health type and update the health if it is worse
// than the stored value
func (rht NnfResourceHealthType) UpdateIfWorseThan(health *NnfResourceHealthType) {
	switch rht {
	case ResourceWarning:
		if *health == ResourceOkay {
			*health = ResourceWarning
		}
	case ResourceCritical:
		if *health != ResourceCritical {
			*health = ResourceCritical
		}
	default:
	}
}
