/*
Copyright 2021 Hewlett Packard Enterprise Development LP
*/

package v1alpha1

// NnfResourceStateType defines valid states that a user can configure an NNF resource
type NnfResourceStateType string

const (
	//
	// Below reflects the current status of a static resource
	//

	// Enable means this static NNF resource should be enabled.
	ResourceEnable NnfResourceStateType = "Enable"

	// Disable means this static NNF resource should be disabled. Not all static resources can be disabled.
	ResourceDisable = "Disable"

	//
	// Below reflects the current status of a managed (user created) resource
	//

	// Create means the resource should be created and enabled for operation. For a newly
	// created resource, the default state is create.
	ResourceCreate NnfResourceStateType = "Create"

	// Destroy means the resource should be released from the allocated resource pool, and
	// this resource and all child resources will be released to the free resource pools
	// managed by the system.
	ResourceDestroy = "Destroy"
)
