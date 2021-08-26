/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"stash.us.cray.com/dpm/dws-operator/utils/dwdparse"
)

//+kubebuilder:rbac:groups=dws.cray.hpe.com,resources=dwdirectiverules,verbs=get;list;watch

// log is for logging in this package.
var workflowlog = logf.Log.WithName("workflow-resource")

var c client.Client

func (r *Workflow) SetupWebhookWithManager(mgr ctrl.Manager) error {
	c = mgr.GetClient()
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-dws-cray-hpe-com-v1alpha1-workflow,mutating=true,failurePolicy=fail,sideEffects=None,groups=dws.cray.hpe.com,resources=workflows,verbs=create;update,versions=v1alpha1,name=mworkflow.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Defaulter = &Workflow{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (workflow *Workflow) Default() {
	workflowlog.Info("default", "name", workflow.Name)

	_ = checkDirectives(workflow, &MutatingRuleParser{})

	return
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-dws-cray-hpe-com-v1alpha1-workflow,mutating=false,failurePolicy=fail,sideEffects=None,groups=dws.cray.hpe.com,resources=workflows,verbs=create;update,versions=v1alpha1,name=vworkflow.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Validator = &Workflow{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Workflow) ValidateCreate() error {
	workflowlog.Info("validate create", "name", r.Name)

	if r.Spec.DesiredState != "proposal" {
		return fmt.Errorf("Desired state must start in 'proposal'")
	}

	return checkDirectives(r, &ValidatingRuleParser{})
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Workflow) ValidateUpdate(old runtime.Object) error {
	workflowlog.Info("validate update", "name", r.Name)

	oldWorkflow, ok := old.(*Workflow)
	if !ok {
		err := fmt.Errorf("Invalid Workflow resource")
		workflowlog.Error(err, "Old runtime.Object is not a Workflow resource")

		return err
	}

	// Check that immutable fields haven't changed.
	err := validateWorkflowImmutable(r, oldWorkflow)
	if err != nil {
		return err
	}

	// Initial setup of the Workflow by the dws controller requires setting the status
	// state to proposal
	if oldWorkflow.Status.State == "" && r.Status.State == "proposal" && r.Spec.DesiredState == "proposal" {
		return nil
	}

	// New state is the desired state in the Spec
	newState, err := GetWorkflowState(r.Spec.DesiredState)
	if err != nil {
		return err
	}

	// Old state is the current state we're on. This is found in the status
	oldState, err := GetWorkflowState(oldWorkflow.Status.State)
	if err != nil {
		return err
	}

	// Progressing to teardown is allowed at any time, and changes to the
	// Workflow that don't change the state are fine too (immutable fields were
	// already checked)
	if newState == StateTeardown || newState == oldState {
		return nil
	}

	if newState < oldState {
		return fmt.Errorf("State can not progress backwards")
	}

	if newState > oldState+1 {
		return fmt.Errorf("States can not be skipped")
	}

	if oldWorkflow.Status.Ready == false {
		return fmt.Errorf("Current desired state not yet achieved")
	}

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Workflow) ValidateDelete() error {
	workflowlog.Info("validate delete", "name", r.Name)
	return nil
}

func validateWorkflowImmutable(a *Workflow, b *Workflow) error {
	if a.Spec.WLMID != b.Spec.WLMID {
		return fmt.Errorf("Can not change immutable field WLMID")
	}

	if a.Spec.JobID != b.Spec.JobID {
		return fmt.Errorf("Can not change immutable field JobID")
	}

	if a.Spec.UserID != b.Spec.UserID {
		return fmt.Errorf("Can not change immutable field UserID")
	}

	if reflect.DeepEqual(a.Spec.DWDirectives, b.Spec.DWDirectives) == false {
		return fmt.Errorf("Can not change immutable field DWDirectives")
	}

	return nil
}

func checkDirectives(workflow *Workflow, ruleParser RuleParser) error {
	if len(workflow.Spec.DWDirectives) == 0 {
		return errors.New("no #DW directives found")
	}

	err := ruleParser.ReadRules()
	if err != nil {
		return nil
	}

	for i, directive := range workflow.Spec.DWDirectives {
		invalidDirective := true
		for _, rule := range ruleParser.GetRuleList() {
			// validate syntax #DW syntax
			const rejectUnsupportedCommands bool = true

			valid, err := dwdparse.ValidateDWDirective(rule, directive, rejectUnsupportedCommands)
			if err != nil {
				// #DW parser validation failed
				workflowlog.Info("dwDirective validation failed", "Error", err)
				return err
			}

			if valid == true {
				invalidDirective = false
				ruleParser.MatchedDirective(workflow, rule.WatchStates, i, rule.DriverLabel)
			}
		}

		if invalidDirective == true {
			return fmt.Errorf("Invalid directive found: '%s'", directive)
		}
	}

	return nil
}

//+kubebuilder:object:generate=false
type RuleParser interface {
	ReadRules() error
	GetRuleList() []dwdparse.DWDirectiveRuleSpec
	MatchedDirective(*Workflow, string, int, string)
}

//+kubebuilder:object:generate=false
type RuleList struct {
	rules []dwdparse.DWDirectiveRuleSpec
}

func (r *RuleList) ReadRules() error {
	ruleSetList := &DWDirectiveRuleList{}
	listOpts := []client.ListOption{
		// Use all the DWDirectiveRules in the namespace we're running in
		client.InNamespace(os.Getenv("POD_NAMESPACE")),
	}

	if err := c.List(context.TODO(), ruleSetList, listOpts...); err != nil {
		return err
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

func (r *RuleList) GetRuleList() []dwdparse.DWDirectiveRuleSpec {
	return r.rules
}

// MutatingRuleParser implements the RuleParser interface.
var _ RuleParser = &MutatingRuleParser{}

//+kubebuilder:object:generate=false
type MutatingRuleParser struct {
	RuleList
}

func (r *MutatingRuleParser) MatchedDirective(workflow *Workflow, watchStates string, index int, label string) {
	if watchStates == "" {
		// Nothing to do
		return
	}

	registrationMap := make(map[string]bool)
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
		// If this driver is already registered for this directive, skip it
		if registrationMap[state] {
			continue
		}

		driverStatus := WorkflowDriverStatus{}
		driverStatus.DriverID = label
		// Register states for this driver
		driverStatus.DWDIndex = index
		driverStatus.WatchState = state
		workflow.Status.Drivers = append(workflow.Status.Drivers, driverStatus)
		workflowlog.Info("Registering driver", "Driver", driverStatus.DriverID, "Watch state", driverStatus.WatchState)
	}
	return
}

// ValidatingRuleParser implements the RuleParser interface.
var _ RuleParser = &ValidatingRuleParser{}

//+kubebuilder:object:generate=false
type ValidatingRuleParser struct {
	RuleList
}

func (r *ValidatingRuleParser) MatchedDirective(workflow *Workflow, watchStates string, index int, label string) {
	return
}
