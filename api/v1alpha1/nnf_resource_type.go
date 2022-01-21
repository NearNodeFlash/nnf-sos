/*
Copyright 2021 Hewlett Packard Enterprise Development LP
*/

package v1alpha1

// NnfResourceStatus provides common fields that are included in all NNF Resources
type NnfResourceStatus struct {
	// ID reflects the NNF Node unique identifier for this NNF Server resource.
	ID string `json:"id,omitempty"`

	// Name reflects the common name of this NNF Server resource.
	Name string `json:"name,omitempty"`

	Status NnfResourceStatusType `json:"status,omitempty"`

	Health NnfResourceHealthType `json:"health,omitempty"`
}
