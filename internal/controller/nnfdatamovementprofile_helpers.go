/*
 * Copyright 2024-2025 Hewlett Packard Enterprise Development LP
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

package controller

import (
	"context"
	"os"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	dwsv1alpha6 "github.com/DataWorkflowServices/dws/api/v1alpha6"
	nnfv1alpha8 "github.com/NearNodeFlash/nnf-sos/api/v1alpha8"
)

// findProfileToUse verifies a NnfDataMovementProfile named in the directive or verifies that a default can be found.
func findDMProfileToUse(ctx context.Context, clnt client.Client, args map[string]string) (*nnfv1alpha8.NnfDataMovementProfile, error) {
	var profileName string

	NnfDataMovementProfile := &nnfv1alpha8.NnfDataMovementProfile{}

	profileNamespace := os.Getenv("NNF_DM_PROFILE_NAMESPACE")

	// If a profile is named then verify that it exists.  Otherwise, verify
	// that a default profile can be found.
	profileName, present := args["profile"]
	if present == false {
		NnfDataMovementProfiles := &nnfv1alpha8.NnfDataMovementProfileList{}
		if err := clnt.List(ctx, NnfDataMovementProfiles, &client.ListOptions{Namespace: profileNamespace}); err != nil {
			return nil, err
		}
		profilesFound := make([]string, 0, len(NnfDataMovementProfiles.Items))
		for _, profile := range NnfDataMovementProfiles.Items {
			if profile.Data.Default {
				objkey := client.ObjectKeyFromObject(&profile)
				profilesFound = append(profilesFound, objkey.Name)
			}
		}
		// Require that there be one and only one default.
		if len(profilesFound) == 0 {
			return nil, dwsv1alpha6.NewResourceError("").WithUserMessage("Unable to find a default NnfDataMovementProfile to use").WithFatal()
		} else if len(profilesFound) > 1 {
			return nil, dwsv1alpha6.NewResourceError("").WithUserMessage("More than one default NnfDataMovementProfile found; unable to pick one: %v", profilesFound).WithFatal()
		}
		profileName = profilesFound[0]
	}
	if len(profileName) == 0 {
		return nil, dwsv1alpha6.NewResourceError("").WithUserMessage("Unable to find an NnfDataMovementProfile name").WithUser().WithFatal()
	}
	err := clnt.Get(ctx, types.NamespacedName{Namespace: profileNamespace, Name: profileName}, NnfDataMovementProfile)
	if err != nil {
		return nil, dwsv1alpha6.NewResourceError("").WithUserMessage("Unable to find NnfDataMovementProfile: %s", profileName).WithUser().WithFatal()
	}

	return NnfDataMovementProfile, nil
}

// findPinnedProfile finds the specified pinned profile.
func findPinnedDMProfile(ctx context.Context, clnt client.Client, namespace string, pinnedName string) (*nnfv1alpha8.NnfDataMovementProfile, error) {

	NnfDataMovementProfile := &nnfv1alpha8.NnfDataMovementProfile{}
	err := clnt.Get(ctx, types.NamespacedName{Namespace: namespace, Name: pinnedName}, NnfDataMovementProfile)
	if err != nil {
		return nil, err
	}
	if !NnfDataMovementProfile.Data.Pinned {
		return nil, dwsv1alpha6.NewResourceError("Expected pinned NnfDataMovementProfile, but it was not pinned: %s", pinnedName).WithFatal()
	}
	return NnfDataMovementProfile, nil
}

// createPinnedProfile finds the specified profile and makes a pinned copy of it.
func createPinnedDMProfile(ctx context.Context, clnt client.Client, clntScheme *runtime.Scheme, args map[string]string, owner metav1.Object, pinnedName string) (*nnfv1alpha8.NnfDataMovementProfile, error) {

	// If we've already pinned a profile, then we're done and
	// we no longer have a use for the original profile.
	NnfDataMovementProfile, err := findPinnedDMProfile(ctx, clnt, owner.GetNamespace(), pinnedName)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, err
		}
	} else {
		// The pinned profile already exists.
		return NnfDataMovementProfile, nil
	}

	// Find the original profile so we can pin a copy of it.
	NnfDataMovementProfile, err = findDMProfileToUse(ctx, clnt, args)
	if err != nil {
		return nil, err
	}

	newProfile := NnfDataMovementProfile.DeepCopy()
	newProfile.ObjectMeta = metav1.ObjectMeta{
		Name:      pinnedName,
		Namespace: owner.GetNamespace(),
	}
	newProfile.Data.Pinned = true
	newProfile.Data.Default = false
	controllerutil.SetControllerReference(owner, newProfile, clntScheme)

	dwsv1alpha6.AddOwnerLabels(newProfile, owner)
	err = clnt.Create(ctx, newProfile)
	if err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return nil, err
		}
	}

	return newProfile, nil
}
