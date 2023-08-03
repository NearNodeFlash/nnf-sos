/*
 * Copyright 2022-2023 Hewlett Packard Enterprise Development LP
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

package v1alpha1

import (
	"fmt"
	"os"
	"reflect"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var nnfstorageprofilelog = logf.Log.WithName("nnfstorageprofile-resource")

func (r *NnfStorageProfile) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-nnf-cray-hpe-com-v1alpha1-nnfstorageprofile,mutating=false,failurePolicy=fail,sideEffects=None,groups=nnf.cray.hpe.com,resources=nnfstorageprofiles,verbs=create;update,versions=v1alpha1,name=vnnfstorageprofile.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &NnfStorageProfile{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *NnfStorageProfile) ValidateCreate() error {
	nnfstorageprofilelog.V(1).Info("validate create", "name", r.Name)

	// If it's not pinned, then it's being made available for users to select
	// and it must be in the correct namespace.
	profileNamespace := os.Getenv("NNF_STORAGE_PROFILE_NAMESPACE")
	if !r.Data.Pinned && r.GetNamespace() != profileNamespace {
		err := fmt.Errorf("incorrect namespace for profile that is intended to be selected by users; the namespace should be '%s'", profileNamespace)
		nnfstorageprofilelog.Error(err, "invalid")
		return err
	}
	if err := r.validateContent(); err != nil {
		nnfstorageprofilelog.Error(err, "invalid NnfStorageProfile resource")
		return err
	}
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *NnfStorageProfile) ValidateUpdate(old runtime.Object) error {
	nnfstorageprofilelog.V(1).Info("validate update", "name", r.Name)

	obj := old.(*NnfStorageProfile)
	if obj.Data.Pinned != r.Data.Pinned {
		err := fmt.Errorf("the pinned flag is immutable")
		nnfcontainerprofilelog.Error(err, "invalid")
		return err
	}
	if obj.Data.Pinned {
		// Allow metadata to be updated, for things like finalizers,
		// ownerReferences, and labels, but do not allow Data to be
		// updated.
		if !reflect.DeepEqual(r.Data, obj.Data) {
			msg := "update on pinned resource not allowed"
			err := fmt.Errorf(msg)
			nnfstorageprofilelog.Error(err, "invalid")
			return err
		}
	}

	if err := r.validateContent(); err != nil {
		nnfstorageprofilelog.Error(err, "invalid NnfStorageProfile resource")
		return err
	}
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *NnfStorageProfile) ValidateDelete() error {
	//nnfstorageprofilelog.V(1).Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
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

	if r.Data.LustreStorage.StandaloneMGT && len(r.Data.LustreStorage.ExternalMGS) > 0 {
		return fmt.Errorf("cannot set both standaloneMgt and externalMgs")
	}

	if r.Data.LustreStorage.StandaloneMGT && r.Data.LustreStorage.CombinedMGTMDT {
		return fmt.Errorf("cannot set standaloneMgt and combinedMgtMdt")
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
