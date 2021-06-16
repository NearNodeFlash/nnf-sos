package v1alpha1

import (
	sf "stash.us.cray.com/rabsw/rfsf-openapi/pkg/models"
)

type NnfResourceStatusType string

const (
	//
	// Below reflects the current status of a static resource
	//

	// Enabled means the static NNF resource is enabled and ready to fullfil requests for
	// managed resources.
	ResourceEnabled NnfResourceStatusType = NnfResourceStatusType(sf.ENABLED_RST)

	// Disabled means the static NNF resource is present but disabled and not available for use
	ResourceDisabled = NnfResourceStatusType(sf.DISABLED_RST)

	// NotPresents means the static NNF resource is not found; likely because it is disconnected
	// or in a powered down state.
	ResourceNotPresent = "NotPresent"

	//
	// Below reflects the current status of a managed (user created) resource
	//

	// Starting means the NNF resource is currently in the process of starting - resources
	// are being prepared for transition to an Active state.
	ResourceStarting = NnfResourceStatusType(sf.STARTING_RST)

	// Ready means the NNF resource is ready for use.
	ResourceReady = "Ready"

	// Failed means the NNF resource has failed during startup or execution. A failed state is
	// an unrecoverable condition. Additional information about the Failed cause can be found by
	// looking at the owning resource's Conditions field. A failed resource can only be removed
	// by transition to a Delete state.
	ResourceFailed = "Failed"

	// Invalid means the NNF resource configuration is invalid due to an improper format or arrangement
	// of listed resource parameters.
	ResourceInvalid = "Invalid"
)

func StaticResourceStatus(s sf.ResourceStatus) NnfResourceStatusType {
	switch s.State {
	case sf.STARTING_RST:
		return ResourceStarting
	case sf.ENABLED_RST:
		return ResourceReady
	case sf.DISABLED_RST:
		return ResourceDisabled
	case sf.ABSENT_RST:
		return ResourceNotPresent
	}

	panic("Unknown Resource State " + string(s.State))
}

func ResourceStatus(s sf.ResourceStatus) NnfResourceStatusType {
	switch s.State {
	case sf.STARTING_RST:
		return ResourceStarting
	case sf.ENABLED_RST:
		return ResourceReady
	default:
		return ResourceFailed
	}
}
