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

package v1alpha1

import (
	"fmt"
	"reflect"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var nnfcontainerprofilelog = logf.Log.WithName("nnfcontainerprofile-resource")

func (r *NnfContainerProfile) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-nnf-cray-hpe-com-v1alpha1-nnfcontainerprofile,mutating=false,failurePolicy=fail,sideEffects=None,groups=nnf.cray.hpe.com,resources=nnfcontainerprofiles,verbs=create;update,versions=v1alpha1,name=vnnfcontainerprofile.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &NnfContainerProfile{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *NnfContainerProfile) ValidateCreate() error {
	nnfcontainerprofilelog.Info("validate create", "name", r.Name)

	return r.validateContent()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *NnfContainerProfile) ValidateUpdate(old runtime.Object) error {
	nnfcontainerprofilelog.Info("validate update", "name", r.Name)

	obj := old.(*NnfContainerProfile)
	if obj.Data.Pinned {
		// Allow metadata to be updated, for things like finalizers,
		// ownerReferences, and labels, but do not allow Data to be
		// updated.
		if !reflect.DeepEqual(r.Data, obj.Data) {
			err := fmt.Errorf("update on pinned resource not allowed")
			nnfcontainerprofilelog.Error(err, "invalid")
			return err
		}
	}

	return r.validateContent()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *NnfContainerProfile) ValidateDelete() error {
	nnfcontainerprofilelog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}

func (r *NnfContainerProfile) validateContent() error {
	if r.Data.RetryLimit < 0 {
		return fmt.Errorf("retryLimit cannot be less than 0")
	}

	if r.Data.PostRunTimeoutSeconds < 0 {
		return fmt.Errorf("postRunTimeoutSeconds cannot be less than 0")
	}

	return nil
}
