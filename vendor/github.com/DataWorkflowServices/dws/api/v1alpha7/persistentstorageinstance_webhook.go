/*
 * Copyright 2023-2025 Hewlett Packard Enterprise Development LP
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

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var persistentstorageinstancelog = logf.Log.WithName("persistentstorageinstance-resource")

// SetupWebhookWithManager will setup the manager to manage the webhooks
func (r *PersistentStorageInstance) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
//+kubebuilder:webhook:path=/validate-dataworkflowservices-github-io-v1alpha7-persistentstorageinstance,mutating=false,failurePolicy=fail,sideEffects=None,groups=dataworkflowservices.github.io,resources=persistentstorageinstances,verbs=create;update,versions=v1alpha7,name=vpersistentstorageinstance.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Validator = &PersistentStorageInstance{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *PersistentStorageInstance) ValidateCreate() (admission.Warnings, error) {
	persistentstorageinstancelog.Info("validate-create", "name", r.Name)

	if r.Spec.State != PSIStateActive {
		s := fmt.Sprintf("spec.state must be %s on creation", PSIStateActive)
		return nil, field.Invalid(field.NewPath("Spec").Child("State"), r.Spec.State, s)
	}

	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *PersistentStorageInstance) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	persistentstorageinstancelog.Info("validate-update", "name", r.Name)

	oldPSI, ok := old.(*PersistentStorageInstance)
	if !ok {
		err := fmt.Errorf("invalid PersistentStorageInstance resource")
		persistentstorageinstancelog.Error(err, "old runtime.Object is not a PersistentStorageInstance resource")
		return nil, err
	}

	immutableError := func(childField string) error {
		return field.Forbidden(field.NewPath("Spec").Child(childField), "field is immutable")
	}

	if r.Spec.Name != oldPSI.Spec.Name {
		return nil, immutableError("Name")
	}

	if r.Spec.FsType != oldPSI.Spec.FsType {
		return nil, immutableError("FsType")
	}

	if r.Spec.DWDirective != oldPSI.Spec.DWDirective {
		return nil, immutableError("DWDirective")
	}

	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *PersistentStorageInstance) ValidateDelete() (admission.Warnings, error) {
	return nil, nil
}
