/*
 * Copyright 2022-2025 Hewlett Packard Enterprise Development LP
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

package v1alpha5

import (
	"fmt"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var nnfstorageprofilelog = logf.Log.WithName("nnfstorageprofile")

// SetupWebhookWithManager will setup the manager to manage the webhooks
func (r *NnfStorageProfile) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
//+kubebuilder:webhook:path=/validate-nnf-cray-hpe-com-v1alpha5-nnfstorageprofile,mutating=false,failurePolicy=fail,sideEffects=None,groups=nnf.cray.hpe.com,resources=nnfstorageprofiles,verbs=create;update,versions=v1alpha5,name=vnnfstorageprofile.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &NnfStorageProfile{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *NnfStorageProfile) ValidateCreate() (admission.Warnings, error) {
	nnfstorageprofilelog.V(1).Info("validate create", "name", r.Name)

	// If it's not pinned, then it's being made available for users to select
	// and it must be in the correct namespace.
	profileNamespace := os.Getenv("NNF_STORAGE_PROFILE_NAMESPACE")
	if !r.Data.Pinned && r.GetNamespace() != profileNamespace {
		err := fmt.Errorf("incorrect namespace for profile that is intended to be selected by users; the namespace should be '%s'", profileNamespace)
		nnfstorageprofilelog.Error(err, "invalid")
		return nil, err
	}
	if err := r.validateContent(); err != nil {
		nnfstorageprofilelog.Error(err, "invalid NnfStorageProfile resource")
		return nil, err
	}
	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *NnfStorageProfile) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	nnfstorageprofilelog.V(1).Info("validate update", "name", r.Name)

	obj := old.(*NnfStorageProfile)
	if obj.Data.Pinned != r.Data.Pinned {
		err := fmt.Errorf("the pinned flag is immutable")
		nnfstorageprofilelog.Error(err, "invalid")
		return nil, err
	}

	// WARNING: NnfStorageProfile allows the obj.Data section to be modified.
	// This is the place in the webhook where our other profile types, such as
	// NnfContainerProfile or NnfDataMovementProfile, would verify that their
	// obj.Data has not been modified.

	if err := r.validateContent(); err != nil {
		nnfstorageprofilelog.Error(err, "invalid NnfStorageProfile resource")
		return nil, err
	}
	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *NnfStorageProfile) ValidateDelete() (admission.Warnings, error) {
	return nil, nil
}

func (r *NnfStorageProfile) validateContent() error {

	if r.Data.Default && r.Data.Pinned {
		return fmt.Errorf("the NnfStorageProfile cannot be both default and pinned")
	}
	if err := r.validateContentLustre(); err != nil {
		return err
	}
	return nil
}

func (r *NnfStorageProfile) validateContentLustre() error {
	if r.Data.LustreStorage.CombinedMGTMDT && len(r.Data.LustreStorage.ExternalMGS) > 0 {
		return fmt.Errorf("cannot set both combinedMgtMdt and externalMgs")
	}

	if len(r.Data.LustreStorage.StandaloneMGTPoolName) > 0 && len(r.Data.LustreStorage.ExternalMGS) > 0 {
		return fmt.Errorf("cannot set both standaloneMgtPoolName and externalMgs")
	}

	if len(r.Data.LustreStorage.StandaloneMGTPoolName) > 0 && r.Data.LustreStorage.CombinedMGTMDT {
		return fmt.Errorf("cannot set standaloneMgtPoolName and combinedMgtMdt")
	}

	for _, target := range []string{"mgt", "mdt", "mgtmdt", "ost"} {
		targetMiscOptions := r.GetLustreMiscOptions(target)
		err := r.validateLustreTargetMiscOptions(targetMiscOptions)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *NnfStorageProfile) validateLustreTargetMiscOptions(targetMiscOptions NnfStorageProfileLustreMiscOptions) error {
	if targetMiscOptions.Count > 0 && targetMiscOptions.Scale > 0 {
		return fmt.Errorf("count and scale cannot both be specified in Lustre target options")
	}

	return nil
}
