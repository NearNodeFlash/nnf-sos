/*
 * Copyright 2021-2025 Hewlett Packard Enterprise Development LP
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
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/DataWorkflowServices/dws/utils/dwdparse"
)

//+kubebuilder:rbac:groups=dataworkflowservices.github.io,resources=dwdirectiverules,verbs=get;list;watch

// log is for logging in this package.
var workflowlog = logf.Log.WithName("workflow-resource")

var c client.Client

// SetupWebhookWithManager will setup the manager to manage the webhooks
func (w *Workflow) SetupWebhookWithManager(mgr ctrl.Manager) error {
	c = mgr.GetClient()
	return ctrl.NewWebhookManagedBy(mgr).
		For(w).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-dataworkflowservices-github-io-v1alpha3-workflow,mutating=true,failurePolicy=fail,sideEffects=None,groups=dataworkflowservices.github.io,resources=workflows,verbs=create,versions=v1alpha3,name=mworkflow.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Defaulter = &Workflow{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (w *Workflow) Default() {
	workflowlog.Info("default", "name", w.Name)

	_ = checkDirectives(w, &MutatingRuleParser{})

	if w.Status.Env == nil {
		w.Status.Env = make(map[string]string)
	}

	w.Status.Env["DW_WORKFLOW_NAME"] = w.Name
	w.Status.Env["DW_WORKFLOW_NAMESPACE"] = w.Namespace
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
//+kubebuilder:webhook:path=/validate-dataworkflowservices-github-io-v1alpha3-workflow,mutating=false,failurePolicy=fail,sideEffects=None,groups=dataworkflowservices.github.io,resources=workflows,verbs=create;update,versions=v1alpha3,name=vworkflow.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Validator = &Workflow{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (w *Workflow) ValidateCreate() (admission.Warnings, error) {
	workflowlog.Info("validate-create", "name", w.Name)

	specPath := field.NewPath("Spec")

	if w.Spec.DesiredState != StateProposal {
		s := fmt.Sprintf("desired state must start in %s", StateProposal)
		return nil, field.Invalid(specPath.Child("DesiredState"), w.Spec.DesiredState, s)
	}
	if w.Spec.Hurry {
		return nil, field.Forbidden(specPath.Child("Hurry"), "the hurry flag may not be set on creation")
	}
	if w.Status.State != "" {
		return nil, field.Forbidden(field.NewPath("Status").Child("State"), "the status state may not be set on creation")
	}

	return nil, checkDirectives(w, &ValidatingRuleParser{})
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (w *Workflow) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	// workflowlog.Info("validate-update", "name", w.Name)  // Too chatty.

	oldWorkflow, ok := old.(*Workflow)
	if !ok {
		err := fmt.Errorf("invalid Workflow resource")
		workflowlog.Error(err, "old runtime.Object is not a Workflow resource")

		return nil, err
	}

	if w.Spec.Hurry && w.Spec.DesiredState != StateTeardown {
		s := fmt.Sprintf("the hurry flag may be set only in %s", StateTeardown)
		return nil, field.Invalid(field.NewPath("Spec").Child("Hurry"), w.Spec.Hurry, s)
	}

	// Check that immutable fields haven't changed.
	err := validateWorkflowImmutable(w, oldWorkflow)
	if err != nil {
		return nil, err
	}

	// Initial setup of the Workflow by the dws controller requires setting the status
	// state to proposal and adding a finalizer.
	if oldWorkflow.Status.State == "" && w.Spec.DesiredState == StateProposal {
		return nil, nil
	}

	// Validate the elements in the Drivers array
	for i, driverStatus := range w.Status.Drivers {

		driverError := func(errString string) error {
			return field.InternalError(field.NewPath("Status").Child("Drivers").Index(i), errors.New(errString))
		}

		// Elements with watchStates not equal to the current state should not change
		if driverStatus.WatchState != oldWorkflow.Status.State {
			if !reflect.DeepEqual(oldWorkflow.Status.Drivers[i], driverStatus) {
				return nil, driverError("driver entry for non-current state cannot be changed")
			}
			continue
		}

		if driverStatus.Completed {
			if driverStatus.Status != StatusCompleted {
				return nil, driverError("driver cannot be completed without status=Completed")
			}

			if driverStatus.Error != "" {
				return nil, driverError("driver cannot be completed when error is present")
			}
		} else {
			if oldWorkflow.Status.Drivers[i].Completed {
				return nil, driverError("driver cannot change from completed state")
			}
		}
	}

	oldState := oldWorkflow.Status.State
	newState := w.Spec.DesiredState

	// Progressing to teardown is allowed at any time, and changes to the
	// Workflow that don't change the state are fine too (immutable fields were
	// already checked)
	if newState == StateTeardown || newState == oldState {
		return nil, nil
	}

	// Error checks
	if oldState.after(newState) {
		return nil, field.Invalid(field.NewPath("Spec").Child("DesiredState"), w.Spec.DesiredState, "DesiredState cannot progress backwards")
	}

	if oldState.next() != newState {
		return nil, field.Invalid(field.NewPath("Spec").Child("DesiredState"), w.Spec.DesiredState, "states cannot be skipped")
	}

	if !oldWorkflow.Status.Ready {
		return nil, field.Invalid(field.NewPath("Status").Child("State"), oldWorkflow.Status.State, "current desired state not yet achieved")
	}

	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (w *Workflow) ValidateDelete() (admission.Warnings, error) {
	return nil, nil
}

func validateWorkflowImmutable(newWorkflow *Workflow, oldWorkflow *Workflow) error {

	immutableError := func(childField string) error {
		return field.Forbidden(field.NewPath("Spec").Child(childField), "field is immutable")
	}

	if newWorkflow.Spec.WLMID != oldWorkflow.Spec.WLMID {
		return immutableError("WLMID")
	}

	if newWorkflow.Spec.JobID != oldWorkflow.Spec.JobID {
		return immutableError("JobID")
	}

	if newWorkflow.Spec.UserID != oldWorkflow.Spec.UserID {
		return immutableError("UserID")
	}

	if newWorkflow.Spec.GroupID != oldWorkflow.Spec.GroupID {
		return immutableError("GroupID")
	}

	if !reflect.DeepEqual(newWorkflow.Spec.DWDirectives, oldWorkflow.Spec.DWDirectives) {
		return immutableError("DWDirectives")
	}

	return nil
}

func checkDirectives(workflow *Workflow, ruleParser RuleParser) error {
	// Ok if we don't have any DW directives, stop parsing.
	if len(workflow.Spec.DWDirectives) == 0 {
		return nil
	}

	if err := ruleParser.ReadRules(); err != nil {
		return err
	}

	// Forward the rule and directive index to the rule parsers matched directive handling
	onValidDirectiveFunc := func(index int, rule dwdparse.DWDirectiveRuleSpec) {
		ruleParser.MatchedDirective(workflow, rule.WatchStates, index, rule.DriverLabel)
	}

	return dwdparse.Validate(ruleParser.GetRuleList(), workflow.Spec.DWDirectives, onValidDirectiveFunc)
}

// RuleParser defines the interface a rule parser must provide
// +kubebuilder:object:generate=false
type RuleParser interface {
	ReadRules() error
	GetRuleList() []dwdparse.DWDirectiveRuleSpec
	MatchedDirective(*Workflow, string, int, string)
}

// RuleList contains the rules to be applied for a particular driver
// +kubebuilder:object:generate=false
type RuleList struct {
	rules []dwdparse.DWDirectiveRuleSpec
}

// ReadRules imports the RulesList into usable go structures.
func (r *RuleList) ReadRules() error {
	ruleSetList := &DWDirectiveRuleList{}
	ns := client.InNamespace(os.Getenv("POD_NAMESPACE"))
	listOpts := []client.ListOption{
		// Use all the DWDirectiveRules in the namespace we're running in
		ns,
	}

	if err := c.List(context.TODO(), ruleSetList, listOpts...); err != nil {
		return err
	}

	if len(ruleSetList.Items) == 0 {
		return fmt.Errorf("unable to find ruleset in namespace: %s", ns)
	}

	r.rules = []dwdparse.DWDirectiveRuleSpec{}
	for _, ruleSet := range ruleSetList.Items {
		for _, rule := range ruleSet.Spec {
			if rule.DriverLabel == "" {
				rule.DriverLabel = ruleSet.Name
			}
			r.rules = append(r.rules, rule)
		}
	}

	return nil
}

// GetRuleList returns the current rules
func (r *RuleList) GetRuleList() []dwdparse.DWDirectiveRuleSpec {
	return r.rules
}

// MutatingRuleParser implements the RuleParser interface.
var _ RuleParser = &MutatingRuleParser{}

// MutatingRuleParser provides the rulelist
// +kubebuilder:object:generate=false
type MutatingRuleParser struct {
	RuleList
}

// MatchedDirective updates the driver status entries to indicate driver availability
func (r *MutatingRuleParser) MatchedDirective(workflow *Workflow, watchStates string, index int, label string) {
	if len(watchStates) == 0 {
		// Nothing to do
		return
	}

	registrationMap := make(map[WorkflowState]bool)
	// Build a map of previously registered states for this driver
	for _, s := range workflow.Status.Drivers {
		if s.DWDIndex != index {
			continue
		}

		if s.DriverID != label {
			continue
		}

		registrationMap[s.WatchState] = true
	}

	// Update driver status entries to indicate driver availability
	for _, state := range strings.Split(watchStates, ",") {
		state := WorkflowState(state)

		// If this driver is already registered for this directive, skip it
		if registrationMap[state] {
			continue
		}

		// Register states for this driver
		driverStatus := WorkflowDriverStatus{
			DriverID:   label,
			DWDIndex:   index,
			WatchState: state,
			Status:     StatusPending,
		}
		workflow.Status.Drivers = append(workflow.Status.Drivers, driverStatus)
		workflowlog.Info("Registering driver", "Driver", driverStatus.DriverID, "Watch state", state)
	}
}

// ValidatingRuleParser implements the RuleParser interface.
var _ RuleParser = &ValidatingRuleParser{}

// ValidatingRuleParser provides the rulelist
// +kubebuilder:object:generate=false
type ValidatingRuleParser struct {
	RuleList
}

// MatchedDirective provides the interface function for the validating webhook
func (r *ValidatingRuleParser) MatchedDirective(workflow *Workflow, watchStates string, index int, label string) {
}
