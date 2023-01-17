/*
 * Copyright 2023 Hewlett Packard Enterprise Development LP
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

// TODO: This file has duplicate functionality with nnfstorageprofile_helpers.go. These two files
// should be combined and made generic.

package controllers

import (
	"context"
	"fmt"
	"os"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	dwsv1alpha1 "github.com/HewlettPackard/dws/api/v1alpha1"
	nnfv1alpha1 "github.com/NearNodeFlash/nnf-sos/api/v1alpha1"
)

// findProfileToUse verifies a NnfContainerProfile named in the directive or verifies that a default can be found.
func findContainerProfileToUse(ctx context.Context, clnt client.Client, args map[string]string) (*nnfv1alpha1.NnfContainerProfile, error) {
	var profileName string

	NnfContainerProfile := &nnfv1alpha1.NnfContainerProfile{}

	profileNamespace := os.Getenv("NNF_CONTAINER_PROFILE_NAMESPACE")

	// If a profile is named then verify that it exists.
	profileName, present := args["profile"]
	if !present {
		return nil, fmt.Errorf("no profile argument supplied")
	}

	err := clnt.Get(ctx, types.NamespacedName{Namespace: profileNamespace, Name: profileName}, NnfContainerProfile)
	if err != nil {
		return nil, err
	}
	return NnfContainerProfile, nil
}

// findPinnedProfile finds the specified pinned profile.
func findPinnedContainerProfile(ctx context.Context, clnt client.Client, namespace string, pinnedName string) (*nnfv1alpha1.NnfContainerProfile, error) {

	NnfContainerProfile := &nnfv1alpha1.NnfContainerProfile{}
	err := clnt.Get(ctx, types.NamespacedName{Namespace: namespace, Name: pinnedName}, NnfContainerProfile)
	if err != nil {
		return nil, err
	}
	if !NnfContainerProfile.Data.Pinned {
		return nil, fmt.Errorf("Expected pinned NnfContainerProfile, but it was not pinned: %s", pinnedName)
	}
	return NnfContainerProfile, nil
}

// createPinnedProfile finds the specified profile and makes a pinned copy of it.
func createPinnedContainerProfile(ctx context.Context, clnt client.Client, clntScheme *runtime.Scheme, args map[string]string, owner metav1.Object, pinnedName string) (*nnfv1alpha1.NnfContainerProfile, error) {

	// If we've already pinned a profile, then we're done and
	// we no longer have a use for the original profile.
	NnfContainerProfile, err := findPinnedContainerProfile(ctx, clnt, owner.GetNamespace(), pinnedName)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, err
		}
	} else {
		// The pinned profile already exists.
		return NnfContainerProfile, nil
	}

	// Find the original profile so we can pin a copy of it.
	NnfContainerProfile, err = findContainerProfileToUse(ctx, clnt, args)
	if err != nil {
		return nil, err
	}

	newProfile := NnfContainerProfile.DeepCopy()
	newProfile.ObjectMeta = metav1.ObjectMeta{
		Name:      pinnedName,
		Namespace: owner.GetNamespace(),
	}
	newProfile.Data.Pinned = true
	controllerutil.SetControllerReference(owner, newProfile, clntScheme)

	dwsv1alpha1.AddOwnerLabels(newProfile, owner)
	err = clnt.Create(ctx, newProfile)
	if err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return nil, err
		}
	}

	return newProfile, nil
}

// addPinnedContainerProfileLabel adds name/namespace labels to a resource to indicate
// which pinned storage profile is being used with that resource.
func addPinnedContainerProfileLabel(object metav1.Object, NnfContainerProfile *nnfv1alpha1.NnfContainerProfile) {
	labels := object.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}

	labels[nnfv1alpha1.PinnedContainerProfileLabelName] = NnfContainerProfile.GetName()
	labels[nnfv1alpha1.PinnedContainerProfileLabelNameSpace] = NnfContainerProfile.GetNamespace()
	object.SetLabels(labels)
}

// getPinnedContainerProfileFromLabel finds the pinned storage profile via the labels on the
// specified resource.
func getPinnedContainerProfileFromLabel(ctx context.Context, clnt client.Client, object metav1.Object) (*nnfv1alpha1.NnfContainerProfile, error) {
	labels := object.GetLabels()
	if labels == nil {
		return nil, fmt.Errorf("unable to find labels")
	}

	pinnedName, okName := labels[nnfv1alpha1.PinnedContainerProfileLabelName]
	if !okName {
		return nil, fmt.Errorf("unable to find %s label", nnfv1alpha1.PinnedContainerProfileLabelName)
	}
	pinnedNamespace, okNamespace := labels[nnfv1alpha1.PinnedContainerProfileLabelNameSpace]
	if !okNamespace {
		return nil, fmt.Errorf("unable to find %s label", nnfv1alpha1.PinnedContainerProfileLabelNameSpace)
	}

	return findPinnedContainerProfile(ctx, clnt, pinnedNamespace, pinnedName)
}
