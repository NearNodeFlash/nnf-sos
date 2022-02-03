//go:build !ignore_autogenerated
// +build !ignore_autogenerated

/*
Copyright 2021 Hewlett Packard Enterprise Development LP
*/

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LustreStorageSpec) DeepCopyInto(out *LustreStorageSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LustreStorageSpec.
func (in *LustreStorageSpec) DeepCopy() *LustreStorageSpec {
	if in == nil {
		return nil
	}
	out := new(LustreStorageSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LustreStorageStatus) DeepCopyInto(out *LustreStorageStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LustreStorageStatus.
func (in *LustreStorageStatus) DeepCopy() *LustreStorageStatus {
	if in == nil {
		return nil
	}
	out := new(LustreStorageStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NnfDataMovement) DeepCopyInto(out *NnfDataMovement) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NnfDataMovement.
func (in *NnfDataMovement) DeepCopy() *NnfDataMovement {
	if in == nil {
		return nil
	}
	out := new(NnfDataMovement)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *NnfDataMovement) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NnfDataMovementList) DeepCopyInto(out *NnfDataMovementList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]NnfDataMovement, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NnfDataMovementList.
func (in *NnfDataMovementList) DeepCopy() *NnfDataMovementList {
	if in == nil {
		return nil
	}
	out := new(NnfDataMovementList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *NnfDataMovementList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NnfDataMovementNodeStatus) DeepCopyInto(out *NnfDataMovementNodeStatus) {
	*out = *in
	if in.Status != nil {
		in, out := &in.Status, &out.Status
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Messages != nil {
		in, out := &in.Messages, &out.Messages
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NnfDataMovementNodeStatus.
func (in *NnfDataMovementNodeStatus) DeepCopy() *NnfDataMovementNodeStatus {
	if in == nil {
		return nil
	}
	out := new(NnfDataMovementNodeStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NnfDataMovementSpec) DeepCopyInto(out *NnfDataMovementSpec) {
	*out = *in
	in.Source.DeepCopyInto(&out.Source)
	in.Destination.DeepCopyInto(&out.Destination)
	out.Storage = in.Storage
	out.Access = in.Access
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NnfDataMovementSpec.
func (in *NnfDataMovementSpec) DeepCopy() *NnfDataMovementSpec {
	if in == nil {
		return nil
	}
	out := new(NnfDataMovementSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NnfDataMovementSpecSourceDestination) DeepCopyInto(out *NnfDataMovementSpecSourceDestination) {
	*out = *in
	if in.StorageInstance != nil {
		in, out := &in.StorageInstance, &out.StorageInstance
		*out = new(v1.ObjectReference)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NnfDataMovementSpecSourceDestination.
func (in *NnfDataMovementSpecSourceDestination) DeepCopy() *NnfDataMovementSpecSourceDestination {
	if in == nil {
		return nil
	}
	out := new(NnfDataMovementSpecSourceDestination)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NnfDataMovementStatus) DeepCopyInto(out *NnfDataMovementStatus) {
	*out = *in
	out.Job = in.Job
	if in.NodeStatus != nil {
		in, out := &in.NodeStatus, &out.NodeStatus
		*out = make([]NnfDataMovementNodeStatus, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]metav1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NnfDataMovementStatus.
func (in *NnfDataMovementStatus) DeepCopy() *NnfDataMovementStatus {
	if in == nil {
		return nil
	}
	out := new(NnfDataMovementStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NnfDriveStatus) DeepCopyInto(out *NnfDriveStatus) {
	*out = *in
	out.NnfResourceStatus = in.NnfResourceStatus
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NnfDriveStatus.
func (in *NnfDriveStatus) DeepCopy() *NnfDriveStatus {
	if in == nil {
		return nil
	}
	out := new(NnfDriveStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NnfJobStorageInstance) DeepCopyInto(out *NnfJobStorageInstance) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NnfJobStorageInstance.
func (in *NnfJobStorageInstance) DeepCopy() *NnfJobStorageInstance {
	if in == nil {
		return nil
	}
	out := new(NnfJobStorageInstance)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *NnfJobStorageInstance) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NnfJobStorageInstanceList) DeepCopyInto(out *NnfJobStorageInstanceList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]NnfJobStorageInstance, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NnfJobStorageInstanceList.
func (in *NnfJobStorageInstanceList) DeepCopy() *NnfJobStorageInstanceList {
	if in == nil {
		return nil
	}
	out := new(NnfJobStorageInstanceList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *NnfJobStorageInstanceList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NnfJobStorageInstanceSpec) DeepCopyInto(out *NnfJobStorageInstanceSpec) {
	*out = *in
	out.Servers = in.Servers
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NnfJobStorageInstanceSpec.
func (in *NnfJobStorageInstanceSpec) DeepCopy() *NnfJobStorageInstanceSpec {
	if in == nil {
		return nil
	}
	out := new(NnfJobStorageInstanceSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NnfJobStorageInstanceStatus) DeepCopyInto(out *NnfJobStorageInstanceStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NnfJobStorageInstanceStatus.
func (in *NnfJobStorageInstanceStatus) DeepCopy() *NnfJobStorageInstanceStatus {
	if in == nil {
		return nil
	}
	out := new(NnfJobStorageInstanceStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NnfNode) DeepCopyInto(out *NnfNode) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NnfNode.
func (in *NnfNode) DeepCopy() *NnfNode {
	if in == nil {
		return nil
	}
	out := new(NnfNode)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *NnfNode) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NnfNodeList) DeepCopyInto(out *NnfNodeList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]NnfNode, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NnfNodeList.
func (in *NnfNodeList) DeepCopy() *NnfNodeList {
	if in == nil {
		return nil
	}
	out := new(NnfNodeList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *NnfNodeList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NnfNodeSpec) DeepCopyInto(out *NnfNodeSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NnfNodeSpec.
func (in *NnfNodeSpec) DeepCopy() *NnfNodeSpec {
	if in == nil {
		return nil
	}
	out := new(NnfNodeSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NnfNodeStatus) DeepCopyInto(out *NnfNodeStatus) {
	*out = *in
	if in.Servers != nil {
		in, out := &in.Servers, &out.Servers
		*out = make([]NnfServerStatus, len(*in))
		copy(*out, *in)
	}
	if in.Drives != nil {
		in, out := &in.Drives, &out.Drives
		*out = make([]NnfDriveStatus, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NnfNodeStatus.
func (in *NnfNodeStatus) DeepCopy() *NnfNodeStatus {
	if in == nil {
		return nil
	}
	out := new(NnfNodeStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NnfNodeStorage) DeepCopyInto(out *NnfNodeStorage) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NnfNodeStorage.
func (in *NnfNodeStorage) DeepCopy() *NnfNodeStorage {
	if in == nil {
		return nil
	}
	out := new(NnfNodeStorage)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *NnfNodeStorage) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NnfNodeStorageAllocationStatus) DeepCopyInto(out *NnfNodeStorageAllocationStatus) {
	*out = *in
	if in.CreationTime != nil {
		in, out := &in.CreationTime, &out.CreationTime
		*out = (*in).DeepCopy()
	}
	if in.DeletionTime != nil {
		in, out := &in.DeletionTime, &out.DeletionTime
		*out = (*in).DeepCopy()
	}
	out.StorageGroup = in.StorageGroup
	out.FileShare = in.FileShare
	out.StoragePool = in.StoragePool
	out.FileSystem = in.FileSystem
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]metav1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NnfNodeStorageAllocationStatus.
func (in *NnfNodeStorageAllocationStatus) DeepCopy() *NnfNodeStorageAllocationStatus {
	if in == nil {
		return nil
	}
	out := new(NnfNodeStorageAllocationStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NnfNodeStorageList) DeepCopyInto(out *NnfNodeStorageList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]NnfNodeStorage, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NnfNodeStorageList.
func (in *NnfNodeStorageList) DeepCopy() *NnfNodeStorageList {
	if in == nil {
		return nil
	}
	out := new(NnfNodeStorageList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *NnfNodeStorageList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NnfNodeStorageSpec) DeepCopyInto(out *NnfNodeStorageSpec) {
	*out = *in
	out.LustreStorage = in.LustreStorage
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NnfNodeStorageSpec.
func (in *NnfNodeStorageSpec) DeepCopy() *NnfNodeStorageSpec {
	if in == nil {
		return nil
	}
	out := new(NnfNodeStorageSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NnfNodeStorageStatus) DeepCopyInto(out *NnfNodeStorageStatus) {
	*out = *in
	if in.Allocations != nil {
		in, out := &in.Allocations, &out.Allocations
		*out = make([]NnfNodeStorageAllocationStatus, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	out.LustreStorage = in.LustreStorage
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NnfNodeStorageStatus.
func (in *NnfNodeStorageStatus) DeepCopy() *NnfNodeStorageStatus {
	if in == nil {
		return nil
	}
	out := new(NnfNodeStorageStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NnfResourceStatus) DeepCopyInto(out *NnfResourceStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NnfResourceStatus.
func (in *NnfResourceStatus) DeepCopy() *NnfResourceStatus {
	if in == nil {
		return nil
	}
	out := new(NnfResourceStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NnfServerStatus) DeepCopyInto(out *NnfServerStatus) {
	*out = *in
	out.NnfResourceStatus = in.NnfResourceStatus
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NnfServerStatus.
func (in *NnfServerStatus) DeepCopy() *NnfServerStatus {
	if in == nil {
		return nil
	}
	out := new(NnfServerStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NnfStorage) DeepCopyInto(out *NnfStorage) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NnfStorage.
func (in *NnfStorage) DeepCopy() *NnfStorage {
	if in == nil {
		return nil
	}
	out := new(NnfStorage)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *NnfStorage) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NnfStorageAllocationNodes) DeepCopyInto(out *NnfStorageAllocationNodes) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NnfStorageAllocationNodes.
func (in *NnfStorageAllocationNodes) DeepCopy() *NnfStorageAllocationNodes {
	if in == nil {
		return nil
	}
	out := new(NnfStorageAllocationNodes)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NnfStorageAllocationSetSpec) DeepCopyInto(out *NnfStorageAllocationSetSpec) {
	*out = *in
	out.NnfStorageLustreSpec = in.NnfStorageLustreSpec
	if in.Nodes != nil {
		in, out := &in.Nodes, &out.Nodes
		*out = make([]NnfStorageAllocationNodes, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NnfStorageAllocationSetSpec.
func (in *NnfStorageAllocationSetSpec) DeepCopy() *NnfStorageAllocationSetSpec {
	if in == nil {
		return nil
	}
	out := new(NnfStorageAllocationSetSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NnfStorageAllocationSetStatus) DeepCopyInto(out *NnfStorageAllocationSetStatus) {
	*out = *in
	if in.NodeStorageReferences != nil {
		in, out := &in.NodeStorageReferences, &out.NodeStorageReferences
		*out = make([]v1.ObjectReference, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NnfStorageAllocationSetStatus.
func (in *NnfStorageAllocationSetStatus) DeepCopy() *NnfStorageAllocationSetStatus {
	if in == nil {
		return nil
	}
	out := new(NnfStorageAllocationSetStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NnfStorageList) DeepCopyInto(out *NnfStorageList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]NnfStorage, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NnfStorageList.
func (in *NnfStorageList) DeepCopy() *NnfStorageList {
	if in == nil {
		return nil
	}
	out := new(NnfStorageList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *NnfStorageList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NnfStorageLustreSpec) DeepCopyInto(out *NnfStorageLustreSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NnfStorageLustreSpec.
func (in *NnfStorageLustreSpec) DeepCopy() *NnfStorageLustreSpec {
	if in == nil {
		return nil
	}
	out := new(NnfStorageLustreSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NnfStorageSpec) DeepCopyInto(out *NnfStorageSpec) {
	*out = *in
	if in.AllocationSets != nil {
		in, out := &in.AllocationSets, &out.AllocationSets
		*out = make([]NnfStorageAllocationSetSpec, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NnfStorageSpec.
func (in *NnfStorageSpec) DeepCopy() *NnfStorageSpec {
	if in == nil {
		return nil
	}
	out := new(NnfStorageSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NnfStorageStatus) DeepCopyInto(out *NnfStorageStatus) {
	*out = *in
	if in.AllocationSets != nil {
		in, out := &in.AllocationSets, &out.AllocationSets
		*out = make([]NnfStorageAllocationSetStatus, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NnfStorageStatus.
func (in *NnfStorageStatus) DeepCopy() *NnfStorageStatus {
	if in == nil {
		return nil
	}
	out := new(NnfStorageStatus)
	in.DeepCopyInto(out)
	return out
}
