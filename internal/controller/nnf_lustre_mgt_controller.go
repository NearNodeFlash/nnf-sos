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

package controller

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	dwsv1alpha6 "github.com/DataWorkflowServices/dws/api/v1alpha6"
	"github.com/DataWorkflowServices/dws/utils/updater"
	nnfv1alpha9 "github.com/NearNodeFlash/nnf-sos/api/v1alpha9"
	"github.com/NearNodeFlash/nnf-sos/internal/controller/metrics"
	"github.com/NearNodeFlash/nnf-sos/pkg/command"
)

type ControllerType string

const (
	ControllerWorker ControllerType = "Worker"
	ControllerRabbit ControllerType = "Rabbit"
	ControllerTest   ControllerType = "Test"
)

const (
	// finalizerNnfLustreMGT defines the key used in identifying the
	// storage object as being owned by this NNF Storage Reconciler. This
	// prevents the system from deleting the custom resource until the
	// reconciler has finished using the resource.
	finalizerNnfLustreMGT = "nnf.cray.hpe.com/nnf_lustre_mgt"
)

// NnfLustreMGTReconciler contains the elements needed during reconciliation for NnfLustreMGT
type NnfLustreMGTReconciler struct {
	client.Client
	Log          logr.Logger
	Scheme       *kruntime.Scheme
	ChildObjects []dwsv1alpha6.ObjectList

	ControllerType ControllerType
}

//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnflustremgts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnflustremgts/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnflustremgts/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.2/pkg/reconcile
func (r *NnfLustreMGTReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
	log := r.Log.WithValues("NnfLustreMGT", req.NamespacedName)

	metrics.NnfLustreMGTReconcilesTotal.Inc()

	nnfLustreMgt := &nnfv1alpha9.NnfLustreMGT{}
	if err := r.Get(ctx, req.NamespacedName, nnfLustreMgt); err != nil {
		// ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	statusUpdater := updater.NewStatusUpdater[*nnfv1alpha9.NnfLustreMGTStatus](nnfLustreMgt)
	defer func() { err = statusUpdater.CloseWithStatusUpdate(ctx, r.Client.Status(), err) }()
	defer func() { nnfLustreMgt.Status.SetResourceErrorAndLog(err, log) }()

	// Check if the object is being deleted.
	if !nnfLustreMgt.GetDeletionTimestamp().IsZero() {
		if !controllerutil.ContainsFinalizer(nnfLustreMgt, finalizerNnfLustreMGT) {
			return ctrl.Result{}, nil
		}

		controllerutil.RemoveFinalizer(nnfLustreMgt, finalizerNnfLustreMGT)
		if err := r.Update(ctx, nnfLustreMgt); err != nil {
			if !apierrors.IsConflict(err) {
				return ctrl.Result{}, err
			}

			return ctrl.Result{Requeue: true}, nil
		}

		return ctrl.Result{}, nil
	}

	// Add the finalizer
	if !controllerutil.ContainsFinalizer(nnfLustreMgt, finalizerNnfLustreMGT) {
		controllerutil.AddFinalizer(nnfLustreMgt, finalizerNnfLustreMGT)
		if err := r.Update(ctx, nnfLustreMgt); err != nil {
			if !apierrors.IsConflict(err) {

				return ctrl.Result{}, err
			}

			return ctrl.Result{Requeue: true}, nil
		}

		return ctrl.Result{}, nil
	}

	// Set the next fsname in the status section if it's empty
	if nnfLustreMgt.Status.FsNameNext == "" {
		fsnameNext := nnfLustreMgt.Spec.FsNameStart

		// If the object reference for the configmap is set and non-empty, then that overrides
		// the FsNameStart field in the spec.
		if nnfLustreMgt.Spec.FsNameStartReference != (corev1.ObjectReference{}) {
			if nnfLustreMgt.Spec.FsNameStartReference.Kind != reflect.TypeOf(corev1.ConfigMap{}).Name() {
				return ctrl.Result{}, dwsv1alpha6.NewResourceError("lustre MGT start reference does not have kind '%s'", reflect.TypeOf(corev1.ConfigMap{}).Name()).WithFatal().WithUser()
			}

			configMap := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      nnfLustreMgt.Spec.FsNameStartReference.Name,
					Namespace: nnfLustreMgt.Spec.FsNameStartReference.Namespace,
				},
			}

			if err := r.Get(ctx, client.ObjectKeyFromObject(configMap), configMap); err != nil {
				return ctrl.Result{}, dwsv1alpha6.NewResourceError("could not get Lustre MGT start fsname config map: %v", client.ObjectKeyFromObject(configMap)).WithError(err).WithMajor()
			}

			if configMap.Data != nil {
				if _, exists := configMap.Data["NextFsName"]; exists {
					if len(configMap.Data["NextFsName"]) != 8 {
						return ctrl.Result{}, dwsv1alpha6.NewResourceError("starting fsname from config map: %v was not 8 characters", client.ObjectKeyFromObject(configMap)).WithError(err).WithFatal()
					}

					fsnameNext = configMap.Data["NextFsName"]
				}
			}
		}

		// If the fsname isn't blacklisted, set it as the next fsname
		if blackListed := isFsNameBlackListed(nnfLustreMgt, nnfLustreMgt.Spec.FsNameStart); !blackListed {
			nnfLustreMgt.Status.FsNameNext = fsnameNext
			return ctrl.Result{}, nil
		}

		// Otherwise, increment the fsname until we find one that isn't blacklisted
		result, err := r.SetFsNameNext(ctx, nnfLustreMgt, nnfLustreMgt.Spec.FsNameStart)
		if err != nil {
			return ctrl.Result{}, err
		}
		if result != nil {
			return *result, nil
		}

		return ctrl.Result{}, nil
	}

	result, err := r.HandleNewClaims(ctx, nnfLustreMgt)
	if err != nil {
		return ctrl.Result{}, err
	}

	if result != nil {
		return *result, nil
	}

	if err := r.RemoveOldClaims(ctx, nnfLustreMgt); err != nil {
		return ctrl.Result{}, err
	}

	result, err = r.HandleNewCommands(ctx, nnfLustreMgt)
	if err != nil {
		return ctrl.Result{}, err
	}

	if result != nil {
		return *result, nil
	}

	if err := r.RemoveOldCommands(ctx, nnfLustreMgt); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// incrementRuneList increments a list of runes where start is the starting
// rune and end is the ending rune.
// For example, incrementRuneList([]rune{'a','z','z'}, 'a', 'z') returns []rune{'b', 'a', 'a'}
func incrementRuneList(digits []rune, start rune, end rune) []rune {
	if len(digits) == 0 {
		return []rune{}
	}

	digit := digits[len(digits)-1] + 1
	if digit > end {
		digit = start
		return append(incrementRuneList(digits[:len(digits)-1], start, end), digit)
	}
	digits[len(digits)-1] = digit

	return digits
}

func incrementFsName(fsname string) string {
	var runeList []rune
	for _, runeDigit := range fsname {
		runeList = append(runeList, runeDigit)
	}

	return string(incrementRuneList(runeList, 'a', 'z'))
}

func isFsNameBlackListed(nnfLustreMgt *nnfv1alpha9.NnfLustreMGT, fsname string) bool {
	// Check the blacklist
	for _, blackListedFsName := range nnfLustreMgt.Spec.FsNameBlackList {
		if fsname == blackListedFsName {
			return true
		}
	}

	return false
}

// SetFsNameNext sets the Status.FsNameNext field to the next available fsname. It also
// updates the configmap the FsNameStartReference field if needed.
func (r *NnfLustreMGTReconciler) SetFsNameNext(ctx context.Context, nnfLustreMgt *nnfv1alpha9.NnfLustreMGT, fsname string) (*ctrl.Result, error) {
	// Find the next available fsname that isn't blacklisted
	for {
		fsname = incrementFsName(fsname)
		if blackListed := isFsNameBlackListed(nnfLustreMgt, fsname); !blackListed {
			break
		}
	}

	// If a configmap is specified in the FsNameStartReference, update it to hold the value
	// of the next fsname
	if nnfLustreMgt.Spec.FsNameStartReference != (corev1.ObjectReference{}) {
		if nnfLustreMgt.Spec.FsNameStartReference.Kind != reflect.TypeOf(corev1.ConfigMap{}).Name() {
			return nil, dwsv1alpha6.NewResourceError("lustre MGT start reference does not have kind '%s'", reflect.TypeOf(corev1.ConfigMap{}).Name()).WithFatal().WithUser()
		}

		// Get used fsname Config map
		configMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      nnfLustreMgt.Spec.FsNameStartReference.Name,
				Namespace: nnfLustreMgt.Spec.FsNameStartReference.Namespace,
			},
		}

		if err := r.Get(ctx, client.ObjectKeyFromObject(configMap), configMap); err != nil {
			return nil, dwsv1alpha6.NewResourceError("could not get Lustre MGT start fsname config map: %v", client.ObjectKeyFromObject(configMap)).WithError(err).WithMajor()
		}

		if configMap.Data == nil {
			configMap.Data = make(map[string]string)
		}
		configMap.Data["NextFsName"] = fsname

		if err := r.Update(ctx, configMap); err != nil {
			if apierrors.IsConflict(err) {
				return &ctrl.Result{Requeue: true}, nil
			}
			return nil, dwsv1alpha6.NewResourceError("could not update Lustre MGT used fsname config map:  %v", client.ObjectKeyFromObject(configMap)).WithError(err).WithMajor()
		}
	}

	// Don't set the next fsname in the status section until after the configmap is updated
	nnfLustreMgt.Status.FsNameNext = fsname

	return nil, nil
}

// HandleNewClaims looks for any new claims in Spec.ClaimList and assigns them
// an fsname
func (r *NnfLustreMGTReconciler) HandleNewClaims(ctx context.Context, nnfLustreMgt *nnfv1alpha9.NnfLustreMGT) (*ctrl.Result, error) {
	claimMap := map[corev1.ObjectReference]string{}
	for _, claim := range nnfLustreMgt.Status.ClaimList {
		claimMap[claim.Reference] = claim.FsName
	}

	for _, reference := range nnfLustreMgt.Spec.ClaimList {
		if _, exists := claimMap[reference]; !exists {
			fsnameNext := nnfLustreMgt.Status.FsNameNext

			result, err := r.SetFsNameNext(ctx, nnfLustreMgt, fsnameNext)
			if err != nil {
				return nil, err
			}
			if result != nil {
				return result, nil
			}

			newClaim := nnfv1alpha9.NnfLustreMGTStatusClaim{
				Reference: reference,
				FsName:    fsnameNext,
			}

			nnfLustreMgt.Status.ClaimList = append(nnfLustreMgt.Status.ClaimList, newClaim)

			return nil, nil
		}
	}

	return nil, nil
}

// RemoveOldClaims removes any old entries from the Status.ClaimList and erases the fsname from
// the MGT if necessary.
func (r *NnfLustreMGTReconciler) RemoveOldClaims(ctx context.Context, nnfLustreMgt *nnfv1alpha9.NnfLustreMGT) error {
	claimMap := map[corev1.ObjectReference]bool{}
	for _, reference := range nnfLustreMgt.Spec.ClaimList {
		claimMap[reference] = true
	}

	for i, claim := range nnfLustreMgt.Status.ClaimList {
		if _, exists := claimMap[claim.Reference]; !exists {
			if err := r.EraseOldFsName(nnfLustreMgt, claim.FsName); err != nil {
				return err
			}

			nnfLustreMgt.Status.ClaimList = append(nnfLustreMgt.Status.ClaimList[:i], nnfLustreMgt.Status.ClaimList[i+1:]...)

			return nil
		}
	}

	return nil
}

func (r *NnfLustreMGTReconciler) EraseOldFsName(nnfLustreMgt *nnfv1alpha9.NnfLustreMGT, fsname string) error {
	log := r.Log.WithValues("NnfLustreMGT", client.ObjectKeyFromObject(nnfLustreMgt))

	if os.Getenv("ENVIRONMENT") == "kind" {
		return nil
	}
	if r.ControllerType == ControllerRabbit {
		if cmd, found := os.LookupEnv("NNF_LUSTRE_FSNAME_ERASE_COMMAND"); found {
			log.Info("Erasing fsname from MGT", "fsname", fsname)
			if _, err := command.Run(fmt.Sprintf("%s %s", cmd, fsname), log); err != nil {
				// Ignore errors from when the fsname has already been erased
				if strings.Contains(err.Error(), "OBD_IOC_LCFG_ERASE failed: No such file or directory") {
					return nil
				}

				return dwsv1alpha6.NewResourceError("unable to remove fsname '%s' from MGT", fsname).WithError(err).WithMajor()
			}
		}
	}

	return nil
}

// HandleNewCommands looks for new commands in the Spec section and runs them
func (r *NnfLustreMGTReconciler) HandleNewCommands(ctx context.Context, nnfLustreMgt *nnfv1alpha9.NnfLustreMGT) (*ctrl.Result, error) {
	commandMap := map[corev1.ObjectReference]*nnfv1alpha9.NnfLustreMGTStatusCommand{}
	for _, command := range nnfLustreMgt.Status.CommandList {
		commandMap[command.Reference] = &command
	}

	for _, command := range nnfLustreMgt.Spec.CommandList {
		if commandStatus, exists := commandMap[command.Reference]; !exists || !commandStatus.Ready {
			err := r.RunCommands(nnfLustreMgt, command.Commands)

			if !exists {
				commandStatus = &nnfv1alpha9.NnfLustreMGTStatusCommand{
					Reference: command.Reference,
				}
			}

			if err != nil {
				commandStatus.Error = dwsv1alpha6.NewResourceError("error running MGT command").WithError(err)
				commandStatus.Ready = false
			} else {
				commandStatus.Error = nil
				commandStatus.Ready = true
			}

			if !exists {
				nnfLustreMgt.Status.CommandList = append(nnfLustreMgt.Status.CommandList, *commandStatus)
			}

			return nil, nil
		}
	}

	return nil, nil
}

// RemoveOldCommands removes any old entries from the Status.CommandList
func (r *NnfLustreMGTReconciler) RemoveOldCommands(ctx context.Context, nnfLustreMgt *nnfv1alpha9.NnfLustreMGT) error {
	commandMap := map[corev1.ObjectReference]bool{}
	for _, command := range nnfLustreMgt.Spec.CommandList {
		commandMap[command.Reference] = true
	}

	for i, command := range nnfLustreMgt.Status.CommandList {
		if _, exists := commandMap[command.Reference]; !exists {
			nnfLustreMgt.Status.CommandList = append(nnfLustreMgt.Status.CommandList[:i], nnfLustreMgt.Status.CommandList[i+1:]...)

			return nil
		}
	}

	return nil
}

func (r *NnfLustreMGTReconciler) RunCommands(nnfLustreMgt *nnfv1alpha9.NnfLustreMGT, commandLines []string) error {
	log := r.Log.WithValues("NnfLustreMGT", client.ObjectKeyFromObject(nnfLustreMgt))
	if r.ControllerType != ControllerRabbit {
		return nil
	}

	if os.Getenv("ENVIRONMENT") != "production" {
		return nil
	}

	for _, commandLine := range commandLines {
		if _, err := command.Run(commandLine, log); err != nil {
			return dwsv1alpha6.NewResourceError("unable to run MGT command: %s", commandLine).WithError(err).WithMajor()
		}
	}

	return nil
}

// MGTs on the Rabbits have their NnfLustreMGT resource created in the Rabbit
// node namespace. The NnfLustreMGT controller runs in the nnf-node-controller
// so it can run the lctl command to erase the old fsname
func filterByRabbitNamespace() predicate.Predicate {
	return predicate.NewPredicateFuncs(func(object client.Object) bool {
		return object.GetNamespace() == os.Getenv("NNF_NODE_NAME")
	})
}

// MGTs on an external node have their NnfLustreMGT resource created in the
// nnf-system namespace. A NnfLustreMGT controller runs on a worker node to
// reconcile these resources since they don't need to run any lctl commands.
func filterByNnfSystemNamespace() predicate.Predicate {
	return predicate.NewPredicateFuncs(func(object client.Object) bool {
		return object.GetNamespace() == "nnf-system"
	})
}

// SetupWithManager sets up the controller with the Manager.
func (r *NnfLustreMGTReconciler) SetupWithManager(mgr ctrl.Manager) error {
	builder := ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		For(&nnfv1alpha9.NnfLustreMGT{})

	switch r.ControllerType {
	case ControllerRabbit:
		builder = builder.WithEventFilter(filterByRabbitNamespace())
	case ControllerWorker:
		builder = builder.WithEventFilter(filterByNnfSystemNamespace())
	default:
	}

	return builder.Complete(r)
}
