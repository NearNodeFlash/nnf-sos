/*
 * Copyright 2021, 2022 Hewlett Packard Enterprise Development LP
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
	"net"
	"net/url"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var lustrefilesystemlog = logf.Log.WithName("lustrefilesystem-resource")

func (r *LustreFileSystem) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-cray-hpe-com-v1alpha1-lustrefilesystem,mutating=false,failurePolicy=fail,sideEffects=None,groups=cray.hpe.com,resources=lustrefilesystems,verbs=create;update,versions=v1alpha1,name=vlustrefilesystem.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &LustreFileSystem{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *LustreFileSystem) ValidateCreate() error {
	lustrefilesystemlog.Info("validate create", "name", r.Name)

	return r.validateLustreFileSystem()
}

func (r *LustreFileSystem) validateLustreFileSystem() error {
	var errList field.ErrorList
	if err := r.validateMgsNids(); err != nil {
		errList = append(errList, err)
	}
	if err := r.validateMountRoot(); err != nil {
		errList = append(errList, err)
	}

	if len(errList) != 0 {
		return errors.NewInvalid(
			schema.GroupKind{Group: "", Kind: "LustreFileSystem"},
			r.Name,
			errList,
		)
	}

	return nil
}

func (r *LustreFileSystem) validateMgsNids() *field.Error {

	re := regexp.MustCompile(`[:,]`)
	f := field.NewPath("spec").Child("mgsNids")

	for _, nid := range re.Split(r.Spec.MgsNids, -1) {
		if !strings.Contains(nid, "@") {
			return field.Invalid(f, nid, "must be a valid mgsNid format [HOST]@[INTERFACE]")
		}

		hostname := strings.SplitN(nid, "@", 2)[0]
		if net.ParseIP(hostname) != nil {
			// Valid IP
			return nil
		}

		_, err := url.Parse(hostname)
		if err != nil {
			return field.Invalid(f, nid, "invalid hostname format")
		}
	}

	return nil
}

func (r *LustreFileSystem) validateMountRoot() *field.Error {
	f := field.NewPath("spec").Child("mountRoot")
	mount := r.Spec.MountRoot
	if !filepath.IsAbs(mount) {
		return field.Invalid(f, mount, "not an absolute file path")
	}

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *LustreFileSystem) ValidateUpdate(old runtime.Object) error {
	lustrefilesystemlog.Info("validate update", "name", r.Name)

	obj := old.(*LustreFileSystem)
	// Allow metadata to be updated, for things like finalizers,
	// ownerReferences, and labels, but do not allow Spec to be updated.
	if !reflect.DeepEqual(r.Spec, obj.Spec) {
		return field.Invalid(field.NewPath("spec"), r.Spec, "specification is immutable")
	}

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *LustreFileSystem) ValidateDelete() error {
	lustrefilesystemlog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}
