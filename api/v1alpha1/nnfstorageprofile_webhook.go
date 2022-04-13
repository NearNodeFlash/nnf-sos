/*
Copyright 2022 Hewlett Packard Enterprise Development LP
*/

package v1alpha1

import (
	"fmt"

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

	if err := r.validateContent(); err != nil {
		nnfstorageprofilelog.Error(err, "invalid NnfStorageProfile resource")
		return err
	}
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *NnfStorageProfile) ValidateUpdate(old runtime.Object) error {
	nnfstorageprofilelog.V(1).Info("validate update", "name", r.Name)

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

	if err := r.validateContentLustre(); err != nil {
		return err
	}
	return nil
}

func (r *NnfStorageProfile) validateContentLustre() error {
	if r.Data.LustreStorage != nil && (r.Data.LustreStorage.CombinedMGTMDT && len(r.Data.LustreStorage.ExternalMGS) > 0) {
		return fmt.Errorf("Cannot set both combinedMgtMdt and externalMgs")
	}

	return nil
}
