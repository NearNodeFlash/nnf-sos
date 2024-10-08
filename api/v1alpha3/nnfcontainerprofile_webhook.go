/*
 * Copyright 2023-2024 Hewlett Packard Enterprise Development LP
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

package v1alpha3

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/kubeflow/mpi-operator/pkg/apis/kubeflow/v2beta1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var nnfcontainerprofilelog = logf.Log.WithName("nnfcontainerprofile")

// SetupWebhookWithManager will setup the manager to manage the webhooks
func (r *NnfContainerProfile) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
//+kubebuilder:webhook:path=/validate-nnf-cray-hpe-com-v1alpha2-nnfcontainerprofile,mutating=false,failurePolicy=fail,sideEffects=None,groups=nnf.cray.hpe.com,resources=nnfcontainerprofiles,verbs=create;update,versions=v1alpha2,name=vnnfcontainerprofile.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &NnfContainerProfile{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *NnfContainerProfile) ValidateCreate() (admission.Warnings, error) {
	nnfcontainerprofilelog.Info("validate create", "name", r.Name)

	// If it's not pinned, then it's being made available for users to select
	// and it must be in the correct namespace.
	profileNamespace := os.Getenv("NNF_CONTAINER_PROFILE_NAMESPACE")
	if !r.Data.Pinned && r.GetNamespace() != profileNamespace {
		err := fmt.Errorf("incorrect namespace for profile that is intended to be selected by users; the namespace should be '%s'", profileNamespace)
		nnfstorageprofilelog.Error(err, "invalid")
		return nil, err
	}

	if err := r.validateContent(); err != nil {
		nnfcontainerprofilelog.Error(err, "invalid NnfContainerProfile resource")
		return nil, err
	}

	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *NnfContainerProfile) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	nnfcontainerprofilelog.Info("validate update", "name", r.Name)

	obj := old.(*NnfContainerProfile)

	if obj.Data.Pinned != r.Data.Pinned {
		err := fmt.Errorf("the pinned flag is immutable")
		nnfcontainerprofilelog.Error(err, "invalid")
		return nil, err
	}

	if obj.Data.Pinned {
		// Allow metadata to be updated, for things like finalizers,
		// ownerReferences, and labels, but do not allow Data to be
		// updated.
		if !reflect.DeepEqual(r.Data, obj.Data) {
			err := fmt.Errorf("update on pinned resource not allowed")
			nnfcontainerprofilelog.Error(err, "invalid")
			return nil, err
		}
	}

	if err := r.validateContent(); err != nil {
		nnfcontainerprofilelog.Error(err, "invalid NnfContainerProfile resource")
		return nil, err
	}

	return nil, nil
}

func (r *NnfContainerProfile) validateContent() error {
	mpiJob := r.Data.MPISpec != nil
	nonmpiJob := r.Data.Spec != nil

	// Either Spec or MPISpec must be set, but not both
	if mpiJob && nonmpiJob {
		return fmt.Errorf("both Spec and MPISpec are provided - only 1 can be set")
	}
	if !mpiJob && !nonmpiJob {
		return fmt.Errorf("either Spec or MPISpec must be provided")
	}

	if mpiJob {
		// PreRunTimeoutSeconds will update the Jobs' ActiveDeadlineSeconds once PreRun timeout occurs, so we can't set them both
		if r.Data.MPISpec.RunPolicy.ActiveDeadlineSeconds != nil && r.Data.PreRunTimeoutSeconds != nil && *r.Data.PreRunTimeoutSeconds > 0 {
			return fmt.Errorf("both PreRunTimeoutSeconds and MPISpec.RunPolicy.ActiveDeadlineSeconds are provided - only 1 can be set")
		}
		// PostRunTimeoutSeconds will update the Jobs' ActiveDeadlineSeconds once PostRun starts, so we can't set them both
		if r.Data.MPISpec.RunPolicy.ActiveDeadlineSeconds != nil && r.Data.PostRunTimeoutSeconds != nil && *r.Data.PostRunTimeoutSeconds > 0 {
			return fmt.Errorf("both PostRunTimeoutSeconds and MPISpec.RunPolicy.ActiveDeadlineSeconds are provided - only 1 can be set")
		}
		// Don't allow users to set the backoff limit directly
		if r.Data.MPISpec.RunPolicy.BackoffLimit != nil && r.Data.RetryLimit > 0 {
			return fmt.Errorf("MPISpec.RunPolicy.BackoffLimit is set. Use RetryLimit instead")
		}

		launcher, launcherOk := r.Data.MPISpec.MPIReplicaSpecs[v2beta1.MPIReplicaTypeLauncher]
		if !launcherOk || len(launcher.Template.Spec.Containers) < 1 {
			return fmt.Errorf("MPISpec.MPIReplicaSpecs.Launcher must be present with at least 1 container defined")
		}
		worker, workerOk := r.Data.MPISpec.MPIReplicaSpecs[v2beta1.MPIReplicaTypeWorker]
		if !workerOk || len(worker.Template.Spec.Containers) < 1 {
			return fmt.Errorf("MPISpec.MPIReplicaSpecs.Worker must be present with at least 1 container defined")
		}
	} else {
		// PreRunTimeoutSeconds will update the Jobs' ActiveDeadlineSeconds once PreRun timeout occurs, so we can't set them both
		if r.Data.Spec.ActiveDeadlineSeconds != nil && r.Data.PreRunTimeoutSeconds != nil && *r.Data.PreRunTimeoutSeconds > 0 {
			return fmt.Errorf("both PreRunTimeoutSeconds and Spec.ActiveDeadlineSeconds are provided - only 1 can be set")
		}
		// PostRunTimeoutSeconds will update the Jobs' ActiveDeadlineSeconds once PostRun starts, so we can't set them both
		if r.Data.Spec.ActiveDeadlineSeconds != nil && r.Data.PostRunTimeoutSeconds != nil && *r.Data.PostRunTimeoutSeconds > 0 {
			return fmt.Errorf("both PostRunTimeoutSeconds and Spec.ActiveDeadlineSeconds are provided - only 1 can be set")
		}

		if len(r.Data.Spec.Containers) < 1 {
			return fmt.Errorf("at least 1 container must be defined in Spec")
		}
	}

	// Ensure only DW_GLOBAL_ storages have PVCMode
	for _, storage := range r.Data.Storages {
		if !strings.HasPrefix(storage.Name, "DW_GLOBAL_") {
			if storage.PVCMode != "" {
				return fmt.Errorf("PVCMode is only supported for global lustre storages (DW_GLOBAL_)")
			}
		}
	}

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *NnfContainerProfile) ValidateDelete() (admission.Warnings, error) {
	return nil, nil
}
