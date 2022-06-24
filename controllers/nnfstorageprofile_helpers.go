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

// findProfileToUse verifies a NnfStorageProfile named in the directive or verifies that a default can be found.
func findProfileToUse(ctx context.Context, clnt client.Client, args map[string]string) (*nnfv1alpha1.NnfStorageProfile, error) {
	var profileName string

	nnfStorageProfile := &nnfv1alpha1.NnfStorageProfile{}

	profileNamespace := os.Getenv("NNF_STORAGE_PROFILE_NAMESPACE")

	// If a profile is named then verify that it exists.  Otherwise, verify
	// that a default profile can be found.
	profileName, present := args["profile"]
	if present == false {
		nnfStorageProfiles := &nnfv1alpha1.NnfStorageProfileList{}
		if err := clnt.List(ctx, nnfStorageProfiles, &client.ListOptions{Namespace: profileNamespace}); err != nil {
			return nil, err
		}
		profilesFound := make([]string, 0, len(nnfStorageProfiles.Items))
		for _, profile := range nnfStorageProfiles.Items {
			if profile.Data.Default {
				objkey := client.ObjectKeyFromObject(&profile)
				profilesFound = append(profilesFound, objkey.Name)
			}
		}
		// Require that there be one and only one default.
		if len(profilesFound) == 0 {
			return nil, fmt.Errorf("Unable to find a default NnfStorageProfile to use")
		} else if len(profilesFound) > 1 {
			return nil, fmt.Errorf("More than one default NnfStorageProfile found; unable to pick one: %v", profilesFound)
		}
		profileName = profilesFound[0]
	}
	if len(profileName) == 0 {
		return nil, fmt.Errorf("Unable to find an NnfStorageProfile name")
	}
	err := clnt.Get(ctx, types.NamespacedName{Namespace: profileNamespace, Name: profileName}, nnfStorageProfile)
	if err != nil {
		return nil, err
	}
	return nnfStorageProfile, nil
}

// findPinnedProfile finds the specified pinned profile.
func findPinnedProfile(ctx context.Context, clnt client.Client, namespace string, pinnedName string) (*nnfv1alpha1.NnfStorageProfile, error) {

	nnfStorageProfile := &nnfv1alpha1.NnfStorageProfile{}
	err := clnt.Get(ctx, types.NamespacedName{Namespace: namespace, Name: pinnedName}, nnfStorageProfile)
	if err != nil {
		return nil, err
	}
	if !nnfStorageProfile.Data.Pinned {
		return nil, fmt.Errorf("Expected pinned NnfStorageProfile, but it was not pinned: %s", pinnedName)
	}
	return nnfStorageProfile, nil
}

// createPinnedProfile finds the specified profile and makes a pinned copy of it.
func createPinnedProfile(ctx context.Context, clnt client.Client, clntScheme *runtime.Scheme, args map[string]string, owner metav1.Object, pinnedName string) (*nnfv1alpha1.NnfStorageProfile, error) {

	// If we've already pinned a profile, then we're done and
	// we no longer have a use for the original profile.
	nnfStorageProfile, err := findPinnedProfile(ctx, clnt, owner.GetNamespace(), pinnedName)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, err
		}
	} else {
		// The pinned profile already exists.
		return nnfStorageProfile, nil
	}

	// Find the original profile so we can pin a copy of it.
	nnfStorageProfile, err = findProfileToUse(ctx, clnt, args)
	if err != nil {
		return nil, err
	}

	newProfile := nnfStorageProfile.DeepCopy()
	newProfile.ObjectMeta = metav1.ObjectMeta{
		Name:      pinnedName,
		Namespace: owner.GetNamespace(),
	}
	newProfile.Data.Pinned = true
	newProfile.Data.Default = false
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

// mergeLustreStorageDirectiveAndProfile returns an object that merges Lustre options from the DW directive with lustre options from the NnfStorageProfile, with the proper precedence.
func mergeLustreStorageDirectiveAndProfile(dwArgs map[string]string, nnfStorageProfile *nnfv1alpha1.NnfStorageProfile) *nnfv1alpha1.NnfStorageProfileLustreData {
	lustreData := &nnfv1alpha1.NnfStorageProfileLustreData{}

	// The combined_mgtmdt and external_mgs args in the directive
	// take precedence over the storage profile.
	//
	// The directive may have only one of these specified; this is
	// enforced by a webhook.  Likewise, the profile may have only
	// one specified; this is also enforced by a webhook.

	if _, present := dwArgs["combined_mgtmdt"]; present {
		lustreData.CombinedMGTMDT = true
	} else if externalMgs, present := dwArgs["external_mgs"]; present {
		lustreData.ExternalMGS = append(lustreData.ExternalMGS, externalMgs)
	} else if nnfStorageProfile != nil {
		if nnfStorageProfile.Data.LustreStorage.CombinedMGTMDT {
			lustreData.CombinedMGTMDT = true
		} else if len(nnfStorageProfile.Data.LustreStorage.ExternalMGS) > 0 {
			lustreData.ExternalMGS = make([]string, len(nnfStorageProfile.Data.LustreStorage.ExternalMGS))
			copy(lustreData.ExternalMGS, nnfStorageProfile.Data.LustreStorage.ExternalMGS)
		}
	}

	return lustreData
}
