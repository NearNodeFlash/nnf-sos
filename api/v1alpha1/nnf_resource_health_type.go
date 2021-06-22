/*
Copyright 2021 Hewlett Packard Enterprise Development LP
*/

package v1alpha1

import (
	sf "stash.us.cray.com/rabsw/rfsf-openapi/pkg/models"
)

// NnfResourceHealthType defines the health of an NNF resource.
type NnfResourceHealthType string

const (
	ResourceOkay NnfResourceHealthType = NnfResourceHealthType(sf.OK_RH)

	ResourceWarning = NnfResourceHealthType(sf.WARNING_RH)

	ResourceCritical = NnfResourceHealthType(sf.CRITICAL_RH)
)

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
