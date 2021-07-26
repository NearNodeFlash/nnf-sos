/*
Copyright 2021 Hewlett Packard Enterprise Development LP
*/

package v1alpha1

// NNF Resource Status provides common fields that are included in all NNF Resources
type NnfResourceStatus struct {
	// Id reflects the NNF Node unique identifier for this NNF Server resource.
	Id string `json:"id,omitempty"`

	// Name reflects the common name of this NNF Server resource.
	Name string `json:"name,omitempty"`

	Status NnfResourceStatusType `json:"status,omitempty"`

	Health NnfResourceHealthType `json:"health,omitempty"`
}
