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

package v1alpha7

import (
	"fmt"
	"os"
	"reflect"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var nnfdatamovementprofilelog = logf.Log.WithName("nnfdatamovementprofile")

// SetupWebhookWithManager will setup the manager to manage the webhooks
func (r *NnfDataMovementProfile) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-nnf-cray-hpe-com-v1alpha7-nnfdatamovementprofile,mutating=false,failurePolicy=fail,sideEffects=None,groups=nnf.cray.hpe.com,resources=nnfdatamovementprofiles,verbs=create;update,versions=v1alpha7,name=vnnfdatamovementprofile.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &NnfDataMovementProfile{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *NnfDataMovementProfile) ValidateCreate() (admission.Warnings, error) {
	nnfdatamovementprofilelog.Info("validate create", "name", r.Name)

	// If it's not pinned, then it's being made available for users to select
	// and it must be in the correct namespace.
	profileNamespace := os.Getenv("NNF_DM_PROFILE_NAMESPACE")
	if !r.Data.Pinned && r.GetNamespace() != profileNamespace {
		err := fmt.Errorf("incorrect namespace for profile that is intended to be selected by users; the namespace should be '%s'", profileNamespace)
		nnfdatamovementprofilelog.Error(err, "invalid")
		return nil, err
	}

	if err := r.validateContent(); err != nil {
		nnfdatamovementprofilelog.Error(err, "invalid NnfDataMovementProfile resource")
		return nil, err
	}

	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *NnfDataMovementProfile) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	nnfdatamovementprofilelog.Info("validate update", "name", r.Name)

	obj := old.(*NnfDataMovementProfile)
	if obj.Data.Pinned != r.Data.Pinned {
		err := fmt.Errorf("the pinned flag is immutable")
		nnfdatamovementprofilelog.Error(err, "invalid")
		return nil, err
	}
	if obj.Data.Pinned {
		// Allow metadata to be updated, for things like finalizers,
		// ownerReferences, and labels, but do not allow Data to be
		// updated.
		if !reflect.DeepEqual(r.Data, obj.Data) {
			msg := "update on pinned resource not allowed"
			err := fmt.Errorf(msg)
			nnfdatamovementprofilelog.Error(err, "invalid")
			return nil, err
		}
	}

	if err := r.validateContent(); err != nil {
		nnfdatamovementprofilelog.Error(err, "invalid NnfDataMovementProfile resource")
		return nil, err
	}
	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *NnfDataMovementProfile) ValidateDelete() (admission.Warnings, error) {
	return nil, nil
}

func (r *NnfDataMovementProfile) validateContent() error {

	if r.Data.Default && r.Data.Pinned {
		return fmt.Errorf("the NnfDataMovementProfile cannot be both default and pinned")
	}

	if r.Data.Slots > 0 && r.Data.MaxSlots > 0 {
		if r.Data.Slots > r.Data.MaxSlots {
			return fmt.Errorf("both Slots and MaxSlots are provided and Slots (%d) is more than MaxSlots (%d)", r.Data.Slots, r.Data.MaxSlots)
		}
	}

	return nil
}
