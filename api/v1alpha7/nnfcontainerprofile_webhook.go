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
	"os"
	"reflect"
	"regexp"
	"strings"

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
//+kubebuilder:webhook:path=/validate-nnf-cray-hpe-com-v1alpha7-nnfcontainerprofile,mutating=false,failurePolicy=fail,sideEffects=None,groups=nnf.cray.hpe.com,resources=nnfcontainerprofiles,verbs=create;update,versions=v1alpha7,name=vnnfcontainerprofile.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &NnfContainerProfile{}

// expected name of the service account used for copy offload containers
var copyOffloadServiceAccountName = "nnf-dm-copy-offload"

// look for the string "copy offload" in the container name, with or without a hyphen/space
var copyOffloadRegex = regexp.MustCompile(`(?i)\bcopy[- ]?offload\b`)

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
	mpiJob := r.Data.NNFMPISpec != nil
	nonmpiJob := r.Data.NNFSpec != nil

	// Either Spec or MPISpec must be set, but not both
	if mpiJob && nonmpiJob {
		return fmt.Errorf("both NNFSpec and NNFMPISpec are provided - only 1 can be set")
	}
	if !mpiJob && !nonmpiJob {
		return fmt.Errorf("either NNFSpec or NNFMPISpec must be provided")
	}

	isCopyOffloadContainer := func(name string) bool {
		return copyOffloadRegex.MatchString(name)
	}

	if mpiJob {
		launcher := r.Data.NNFMPISpec.Launcher
		if len(launcher.Containers) < 1 {
			return fmt.Errorf("NNFMPISpec.Launcher must be present with at least 1 container defined")
		}
		worker := r.Data.NNFMPISpec.Worker
		if len(worker.Containers) < 1 {
			return fmt.Errorf("NNFMPISpec.Worker must be present with at least 1 container defined")
		}

		// When it looks like we have a copy offload Launcher container, ensure the service account is correct
		// for _, c := range launcher.Containers {
		// 	if isCopyOffloadContainer(c.Name) {

		// 		if launcher.Template.Spec.ServiceAccountName != copyOffloadServiceAccountName {
		// 			return fmt.Errorf(
		// 				"the specified container name ('%s') suggests that this container profile is intended for use with the Copy Offload API. "+
		// 					"Launcher containers used for Copy Offload must use the service account name '%s'",
		// 				c.Name,
		// 				copyOffloadServiceAccountName,
		// 			)
		// 		}
		// 	} else {
		// 		if launcher.Template.Spec.ServiceAccountName == copyOffloadServiceAccountName {
		// 			return fmt.Errorf(
		// 				"the specified Launcher container name ('%s') suggests that this container profile is NOT intended for use with the Copy Offload API. "+
		// 					"but the Copy Offload service account name '%s' is being used",
		// 				c.Name,
		// 				copyOffloadServiceAccountName,
		// 			)
		// 		}

		// 	}
		// }

	} else {
		// PreRunTimeoutSeconds will update the Jobs' ActiveDeadlineSeconds once PreRun timeout occurs, so we can't set them both
		if len(r.Data.NNFSpec.Containers) < 1 {
			return fmt.Errorf("at least 1 container must be defined in Spec")
		}

		// When it looks like we have a copy offload container based on the service account name, let the user know an MPISpec is required
		// if r.Data.Spec.ServiceAccountName == copyOffloadServiceAccountName {
		// 	return fmt.Errorf(
		// 		"the specified service account name ('%s') suggests that this container profile is intended for use with the Copy Offload API. "+
		// 			"Container profiles used for Copy Offload must use the MPISpec to define the Launcher and Worker containers",
		// 		copyOffloadServiceAccountName,
		// 	)
		// }
		// When it looks like we have a copy offload container, let the user know an MPISpec is required
		for _, c := range r.Data.NNFSpec.Containers {
			if isCopyOffloadContainer(c.Name) {
				return fmt.Errorf(
					"the specified container name ('%s') suggests that this container profile is intended for use with the Copy Offload API. "+
						"Container profiles used for Copy Offload must use the MPISpec to define the Launcher and Worker containers",
					c.Name,
				)
			}
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
