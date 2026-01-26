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

package v1alpha10

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
//+kubebuilder:webhook:path=/validate-nnf-cray-hpe-com-v1alpha10-nnfstorageprofile,mutating=false,failurePolicy=fail,sideEffects=None,groups=nnf.cray.hpe.com,resources=nnfstorageprofiles,verbs=create;update,versions=v1alpha10,name=vnnfstorageprofile.kb.io,admissionReviewVersions=v1

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
	if err := r.validateContentShared(); err != nil {
		return err
	}
	return nil
}

func (r *NnfStorageProfile) validateContentShared() error {
	// Validate XFS storage variable overrides
	if err := validateVariableOverrideKeys(r.Data.XFSStorage.VariableOverride); err != nil {
		return fmt.Errorf("xfsStorage: %w", err)
	}

	// Validate GFS2 storage variable overrides
	if err := validateVariableOverrideKeys(r.Data.GFS2Storage.VariableOverride); err != nil {
		return fmt.Errorf("gfs2Storage: %w", err)
	}

	// Validate Raw storage variable overrides
	if err := validateVariableOverrideKeys(r.Data.RawStorage.VariableOverride); err != nil {
		return fmt.Errorf("rawStorage: %w", err)
	}

	return nil
}

func (r *NnfStorageProfile) validateContentLustre() error {
	if r.Data.LustreStorage.CombinedMGTMDT && len(r.Data.LustreStorage.MgtOptions.ExternalMGS) > 0 {
		return fmt.Errorf("cannot set both combinedMgtMdt and externalMgs")
	}

	if len(r.Data.LustreStorage.MgtOptions.StandaloneMGTPoolName) > 0 && len(r.Data.LustreStorage.MgtOptions.ExternalMGS) > 0 {
		return fmt.Errorf("cannot set both standaloneMgtPoolName and externalMgs")
	}

	if len(r.Data.LustreStorage.MgtOptions.StandaloneMGTPoolName) > 0 && r.Data.LustreStorage.CombinedMGTMDT {
		return fmt.Errorf("cannot set standaloneMgtPoolName and combinedMgtMdt")
	}

	for _, target := range []string{"mgt", "mdt", "mgtmdt", "ost"} {
		targetOptions := r.GetLustreTargetOptions(target)
		err := r.validateLustreTargetOptions(targetOptions)
		if err != nil {
			return err
		}
	}

	// Validate client options variable overrides
	if err := validateVariableOverrideKeys(r.Data.LustreStorage.ClientOptions.VariableOverride); err != nil {
		return err
	}

	return nil
}

func (r *NnfStorageProfile) validateLustreTargetOptions(targetOptions NnfStorageProfileLustreTargetOptions) error {
	if targetOptions.Count > 0 && targetOptions.Scale > 0 {
		return fmt.Errorf("count and scale cannot both be specified in Lustre target options")
	}

	if err := validateVariableOverrideKeys(targetOptions.VariableOverride); err != nil {
		return err
	}

	return nil
}

// validateVariableOverrideKeys checks that variable override keys are valid.
// Keys must start with '$' and be known variable names.
func validateVariableOverrideKeys(overrides map[string]string) error {
	validKeys := map[string]struct{}{
		"$FS_NAME":        {},
		"$MGS_NID":        {},
		"$INDEX":          {},
		"$BACKFS":         {},
		"$POOL_NAME":      {},
		"$ZPOOL_NAME":     {},
		"$ZPOOL_DATA_SET": {},
		"$ZVOL_NAME":      {},
		"$DEVICE":         {},
		"$DEVICE_LIST":    {},
		"$DEVICE_NUM":     {},
		"$MOUNT_PATH":     {},
		"$TARGET_TYPE":    {},
		"$TARGET_PATH":    {},
		"$VG_NAME":        {},
		"$LV_NAME":        {},
		"$LV_SIZE":        {},
		"$LV_INDEX":       {},
		"$PERCENT_VG":     {},
		"$FSTYPE":         {},
		"$MOUNT_TARGET":   {},
		"$TEMP_DIR":       {},
		"$JOBID":          {},
		"$USERID":         {},
		"$GROUPID":        {},
		"$CLUSTER_NAME":   {},
		"$LOCK_SPACE":     {},
		"$PROTOCOL":       {},
		"$NUM_OSTS":       {},
		"$NUM_NNFNODES":   {},
	}

	for key := range overrides {
		if len(key) == 0 {
			return fmt.Errorf("variableOverride key cannot be empty")
		}
		if key[0] != '$' {
			return fmt.Errorf("variableOverride key '%s' must start with '$'", key)
		}
		if _, ok := validKeys[key]; !ok {
			return fmt.Errorf("variableOverride key '%s' is not a recognized variable name", key)
		}
	}

	return nil
}
