//go:build !ignore_autogenerated
// +build !ignore_autogenerated

/*
 * Copyright 2022 Hewlett Packard Enterprise Development LP
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

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClientEndpointsSpec) DeepCopyInto(out *ClientEndpointsSpec) {
	*out = *in
	if in.NodeNames != nil {
		in, out := &in.NodeNames, &out.NodeNames
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClientEndpointsSpec.
func (in *ClientEndpointsSpec) DeepCopy() *ClientEndpointsSpec {
	if in == nil {
		return nil
	}
	out := new(ClientEndpointsSpec)
	in.DeepCopyInto(out)
	return out
}

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
func (in *NnfAccess) DeepCopyInto(out *NnfAccess) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NnfAccess.
func (in *NnfAccess) DeepCopy() *NnfAccess {
	if in == nil {
		return nil
	}
	out := new(NnfAccess)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *NnfAccess) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NnfAccessList) DeepCopyInto(out *NnfAccessList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]NnfAccess, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NnfAccessList.
func (in *NnfAccessList) DeepCopy() *NnfAccessList {
	if in == nil {
		return nil
	}
	out := new(NnfAccessList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *NnfAccessList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NnfAccessSpec) DeepCopyInto(out *NnfAccessSpec) {
	*out = *in
	out.ClientReference = in.ClientReference
	out.StorageReference = in.StorageReference
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NnfAccessSpec.
func (in *NnfAccessSpec) DeepCopy() *NnfAccessSpec {
	if in == nil {
		return nil
	}
	out := new(NnfAccessSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NnfAccessStatus) DeepCopyInto(out *NnfAccessStatus) {
	*out = *in
	in.ResourceError.DeepCopyInto(&out.ResourceError)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NnfAccessStatus.
func (in *NnfAccessStatus) DeepCopy() *NnfAccessStatus {
	if in == nil {
		return nil
	}
	out := new(NnfAccessStatus)
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
func (in *NnfDataMovementCommandStatus) DeepCopyInto(out *NnfDataMovementCommandStatus) {
	*out = *in
	in.ElapsedTime.DeepCopyInto(&out.ElapsedTime)
	if in.ProgressPercentage != nil {
		in, out := &in.ProgressPercentage, &out.ProgressPercentage
		*out = new(int32)
		**out = **in
	}
	in.LastMessageTime.DeepCopyInto(&out.LastMessageTime)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NnfDataMovementCommandStatus.
func (in *NnfDataMovementCommandStatus) DeepCopy() *NnfDataMovementCommandStatus {
	if in == nil {
		return nil
	}
	out := new(NnfDataMovementCommandStatus)
	in.DeepCopyInto(out)
	return out
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
func (in *NnfDataMovementSpec) DeepCopyInto(out *NnfDataMovementSpec) {
	*out = *in
	if in.Source != nil {
		in, out := &in.Source, &out.Source
		*out = new(NnfDataMovementSpecSourceDestination)
		**out = **in
	}
	if in.Destination != nil {
		in, out := &in.Destination, &out.Destination
		*out = new(NnfDataMovementSpecSourceDestination)
		**out = **in
	}
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
	out.StorageReference = in.StorageReference
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
	if in.StartTime != nil {
		in, out := &in.StartTime, &out.StartTime
		*out = (*in).DeepCopy()
	}
	if in.EndTime != nil {
		in, out := &in.EndTime, &out.EndTime
		*out = (*in).DeepCopy()
	}
	if in.CommandStatus != nil {
		in, out := &in.CommandStatus, &out.CommandStatus
		*out = new(NnfDataMovementCommandStatus)
		(*in).DeepCopyInto(*out)
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
func (in *NnfNodeECData) DeepCopyInto(out *NnfNodeECData) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NnfNodeECData.
func (in *NnfNodeECData) DeepCopy() *NnfNodeECData {
	if in == nil {
		return nil
	}
	out := new(NnfNodeECData)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *NnfNodeECData) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NnfNodeECDataList) DeepCopyInto(out *NnfNodeECDataList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]NnfNodeECData, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NnfNodeECDataList.
func (in *NnfNodeECDataList) DeepCopy() *NnfNodeECDataList {
	if in == nil {
		return nil
	}
	out := new(NnfNodeECDataList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *NnfNodeECDataList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NnfNodeECDataSpec) DeepCopyInto(out *NnfNodeECDataSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NnfNodeECDataSpec.
func (in *NnfNodeECDataSpec) DeepCopy() *NnfNodeECDataSpec {
	if in == nil {
		return nil
	}
	out := new(NnfNodeECDataSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NnfNodeECDataStatus) DeepCopyInto(out *NnfNodeECDataStatus) {
	*out = *in
	if in.Data != nil {
		in, out := &in.Data, &out.Data
		*out = make(map[string]NnfNodeECPrivateData, len(*in))
		for key, val := range *in {
			var outVal map[string]string
			if val == nil {
				(*out)[key] = nil
			} else {
				in, out := &val, &outVal
				*out = make(NnfNodeECPrivateData, len(*in))
				for key, val := range *in {
					(*out)[key] = val
				}
			}
			(*out)[key] = outVal
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NnfNodeECDataStatus.
func (in *NnfNodeECDataStatus) DeepCopy() *NnfNodeECDataStatus {
	if in == nil {
		return nil
	}
	out := new(NnfNodeECDataStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in NnfNodeECPrivateData) DeepCopyInto(out *NnfNodeECPrivateData) {
	{
		in := &in
		*out = make(NnfNodeECPrivateData, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NnfNodeECPrivateData.
func (in NnfNodeECPrivateData) DeepCopy() NnfNodeECPrivateData {
	if in == nil {
		return nil
	}
	out := new(NnfNodeECPrivateData)
	in.DeepCopyInto(out)
	return *out
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
	in.Spec.DeepCopyInto(&out.Spec)
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
	if in.NVMeList != nil {
		in, out := &in.NVMeList, &out.NVMeList
		*out = make([]NnfNodeStorageNVMeStatus, len(*in))
		copy(*out, *in)
	}
	out.FileShare = in.FileShare
	out.StoragePool = in.StoragePool
	out.FileSystem = in.FileSystem
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]v1.Condition, len(*in))
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
func (in *NnfNodeStorageNVMeStatus) DeepCopyInto(out *NnfNodeStorageNVMeStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NnfNodeStorageNVMeStatus.
func (in *NnfNodeStorageNVMeStatus) DeepCopy() *NnfNodeStorageNVMeStatus {
	if in == nil {
		return nil
	}
	out := new(NnfNodeStorageNVMeStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NnfNodeStorageSpec) DeepCopyInto(out *NnfNodeStorageSpec) {
	*out = *in
	out.LustreStorage = in.LustreStorage
	if in.ClientEndpoints != nil {
		in, out := &in.ClientEndpoints, &out.ClientEndpoints
		*out = make([]ClientEndpointsSpec, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
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
	in.ResourceError.DeepCopyInto(&out.ResourceError)
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
func (in *NnfStorageProfile) DeepCopyInto(out *NnfStorageProfile) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Data.DeepCopyInto(&out.Data)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NnfStorageProfile.
func (in *NnfStorageProfile) DeepCopy() *NnfStorageProfile {
	if in == nil {
		return nil
	}
	out := new(NnfStorageProfile)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *NnfStorageProfile) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NnfStorageProfileData) DeepCopyInto(out *NnfStorageProfileData) {
	*out = *in
	in.LustreStorage.DeepCopyInto(&out.LustreStorage)
	in.GFS2Storage.DeepCopyInto(&out.GFS2Storage)
	in.XFSStorage.DeepCopyInto(&out.XFSStorage)
	in.RawStorage.DeepCopyInto(&out.RawStorage)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NnfStorageProfileData.
func (in *NnfStorageProfileData) DeepCopy() *NnfStorageProfileData {
	if in == nil {
		return nil
	}
	out := new(NnfStorageProfileData)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NnfStorageProfileGFS2Data) DeepCopyInto(out *NnfStorageProfileGFS2Data) {
	*out = *in
	in.Options.DeepCopyInto(&out.Options)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NnfStorageProfileGFS2Data.
func (in *NnfStorageProfileGFS2Data) DeepCopy() *NnfStorageProfileGFS2Data {
	if in == nil {
		return nil
	}
	out := new(NnfStorageProfileGFS2Data)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NnfStorageProfileList) DeepCopyInto(out *NnfStorageProfileList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]NnfStorageProfile, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NnfStorageProfileList.
func (in *NnfStorageProfileList) DeepCopy() *NnfStorageProfileList {
	if in == nil {
		return nil
	}
	out := new(NnfStorageProfileList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *NnfStorageProfileList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NnfStorageProfileLustreData) DeepCopyInto(out *NnfStorageProfileLustreData) {
	*out = *in
	in.MgtOptions.DeepCopyInto(&out.MgtOptions)
	in.MdtOptions.DeepCopyInto(&out.MdtOptions)
	in.OstOptions.DeepCopyInto(&out.OstOptions)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NnfStorageProfileLustreData.
func (in *NnfStorageProfileLustreData) DeepCopy() *NnfStorageProfileLustreData {
	if in == nil {
		return nil
	}
	out := new(NnfStorageProfileLustreData)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NnfStorageProfileMiscData) DeepCopyInto(out *NnfStorageProfileMiscData) {
	*out = *in
	if in.AddMkfs != nil {
		in, out := &in.AddMkfs, &out.AddMkfs
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.AddMount != nil {
		in, out := &in.AddMount, &out.AddMount
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.AddPvCreate != nil {
		in, out := &in.AddPvCreate, &out.AddPvCreate
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.AddVgCreate != nil {
		in, out := &in.AddVgCreate, &out.AddVgCreate
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.AddLvCreate != nil {
		in, out := &in.AddLvCreate, &out.AddLvCreate
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NnfStorageProfileMiscData.
func (in *NnfStorageProfileMiscData) DeepCopy() *NnfStorageProfileMiscData {
	if in == nil {
		return nil
	}
	out := new(NnfStorageProfileMiscData)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NnfStorageProfileRawData) DeepCopyInto(out *NnfStorageProfileRawData) {
	*out = *in
	in.Options.DeepCopyInto(&out.Options)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NnfStorageProfileRawData.
func (in *NnfStorageProfileRawData) DeepCopy() *NnfStorageProfileRawData {
	if in == nil {
		return nil
	}
	out := new(NnfStorageProfileRawData)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NnfStorageProfileXFSData) DeepCopyInto(out *NnfStorageProfileXFSData) {
	*out = *in
	in.Options.DeepCopyInto(&out.Options)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NnfStorageProfileXFSData.
func (in *NnfStorageProfileXFSData) DeepCopy() *NnfStorageProfileXFSData {
	if in == nil {
		return nil
	}
	out := new(NnfStorageProfileXFSData)
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
		copy(*out, *in)
	}
	in.ResourceError.DeepCopyInto(&out.ResourceError)
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
