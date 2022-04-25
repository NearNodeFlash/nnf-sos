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

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nnfv1alpha1 "github.hpe.com/hpe/hpc-rabsw-nnf-sos/api/v1alpha1"
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
		if err := clnt.List(ctx, nnfStorageProfiles); err != nil {
			return nil, err
		}
		numDefaults := 0
		for _, profile := range nnfStorageProfiles.Items {
			if profile.Data.Default {
				objkey := client.ObjectKeyFromObject(&profile)
				profileName = objkey.Name
				numDefaults++
			}
		}
		// Require that there be one and only one default.
		if numDefaults == 0 {
			return nil, fmt.Errorf("Unable to find a default NnfStorageProfile to use")
		} else if numDefaults > 1 {
			return nil, fmt.Errorf("More than one default NnfStorageProfile found; unable to pick one")
		}
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
	} else if nnfStorageProfile != nil && nnfStorageProfile.Data.LustreStorage != nil {
		if nnfStorageProfile.Data.LustreStorage.CombinedMGTMDT {
			lustreData.CombinedMGTMDT = true
		} else if len(nnfStorageProfile.Data.LustreStorage.ExternalMGS) > 0 {
			lustreData.ExternalMGS = make([]string, len(nnfStorageProfile.Data.LustreStorage.ExternalMGS))
			copy(lustreData.ExternalMGS, nnfStorageProfile.Data.LustreStorage.ExternalMGS)
		}
	}

	return lustreData
}
