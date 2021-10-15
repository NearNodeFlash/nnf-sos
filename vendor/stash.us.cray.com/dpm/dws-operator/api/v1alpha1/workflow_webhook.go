/*
Copyright 2021 Hewlett Packard Enterprise Development LP
*/

package v1alpha1

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

// SetupWebhookWithManager connects the webhook with the manager
func (w *Workflow) SetupWebhookWithManager(mgr ctrl.Manager) error {
	c = mgr.GetClient()
	return ctrl.NewWebhookManagedBy(mgr).
		For(w).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-dws-cray-hpe-com-v1alpha1-workflow,mutating=true,failurePolicy=fail,sideEffects=None,groups=dws.cray.hpe.com,resources=workflows,verbs=create;update,versions=v1alpha1,name=mworkflow.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Defaulter = &Workflow{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (w *Workflow) Default() {
	workflowlog.Info("default", "name", w.Name)

	// TODO: Find a better way to initialize these. open-api doesn't like NULL for an initial value
	w.Status.DesiredStateChange = metav1.Now()
	w.Status.ReadyChange = metav1.Now()

	_ = checkDirectives(w, &MutatingRuleParser{})
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-dws-cray-hpe-com-v1alpha1-workflow,mutating=false,failurePolicy=fail,sideEffects=None,groups=dws.cray.hpe.com,resources=workflows,verbs=create;update,versions=v1alpha1,name=vworkflow.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Validator = &Workflow{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (w *Workflow) ValidateCreate() error {
	workflowlog.Info("validate create", "name", w.Name)

	if w.Spec.DesiredState != "proposal" {
		return fmt.Errorf("desired state must start in 'proposal'")
	}

	return checkDirectives(w, &ValidatingRuleParser{})
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (w *Workflow) ValidateUpdate(old runtime.Object) error {
	workflowlog.Info("validate update", "name", w.Name)

	oldWorkflow, ok := old.(*Workflow)
	if !ok {
		err := fmt.Errorf("invalid Workflow resource")
		workflowlog.Error(err, "old runtime.Object is not a Workflow resource")

		return err
	}

	// Check that immutable fields haven't changed.
	err := validateWorkflowImmutable(w, oldWorkflow)
	if err != nil {
		return err
	}

	// Initial setup of the Workflow by the dws controller requires setting the status
	// state to proposal
	if oldWorkflow.Status.State == "" && w.Status.State == "proposal" && w.Spec.DesiredState == "proposal" {
		return nil
	}

	// New state is the desired state in the Spec
	newState, err := GetWorkflowState(w.Spec.DesiredState)
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
		return fmt.Errorf("state cannot progress backwards")
	}

	if newState > oldState+1 {
		return fmt.Errorf("states cannot be skipped")
	}

	if !oldWorkflow.Status.Ready {
		return fmt.Errorf("current desired state not yet achieved")
	}

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (w *Workflow) ValidateDelete() error {
	workflowlog.Info("validate delete", "name", w.Name)
	return nil
}

func validateWorkflowImmutable(a *Workflow, b *Workflow) error {
	if a.Spec.WLMID != b.Spec.WLMID {
		return fmt.Errorf("cannot change immutable field WLMID")
	}

	if a.Spec.JobID != b.Spec.JobID {
		return fmt.Errorf("cannot change immutable field JobID")
	}

	if a.Spec.UserID != b.Spec.UserID {
		return fmt.Errorf("cannot change immutable field UserID")
	}

	if !reflect.DeepEqual(a.Spec.DWDirectives, b.Spec.DWDirectives) {
		return fmt.Errorf("cannot change immutable field DWDirectives")
	}

	return nil
}

func checkDirectives(workflow *Workflow, ruleParser RuleParser) error {
	// Ok if we don't have any DW directives, stop parsing.
	if len(workflow.Spec.DWDirectives) == 0 {
		return nil
	}

	err := ruleParser.ReadRules()
	if err != nil {
		return err
	}

	for i, directive := range workflow.Spec.DWDirectives {
		validDirective := false
		for _, rule := range ruleParser.GetRuleList() {
			// validate #DW syntax
			const rejectUnsupportedCommands bool = true

			valid, err := dwdparse.ValidateDWDirective(rule, directive, rejectUnsupportedCommands)
			if err != nil {
				// #DW parser validation failed
				workflowlog.Info("dwDirective validation failed", "Error", err)
				return err
			}

			if valid {
				validDirective = true
				ruleParser.MatchedDirective(workflow, rule.WatchStates, i, rule.DriverLabel)
			}
		}

		if !validDirective {
			return fmt.Errorf("invalid directive found: '%s'", directive)
		}
	}

	return nil
}

// RuleParser defines the interface a rule parser must provide
//+kubebuilder:object:generate=false
type RuleParser interface {
	ReadRules() error
	GetRuleList() []dwdparse.DWDirectiveRuleSpec
	MatchedDirective(*Workflow, string, int, string)
}

// RuleList contains the rules to be applied for a particular driver
//+kubebuilder:object:generate=false
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
		return fmt.Errorf("Unable to find ruleset in namespace: %s", ns)
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
//+kubebuilder:object:generate=false
type MutatingRuleParser struct {
	RuleList
}

// MatchedDirective updates the driver status entries to indicate driver availability
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
		driverStatus.CompleteTime = metav1.Now()
		workflow.Status.Drivers = append(workflow.Status.Drivers, driverStatus)
		workflowlog.Info("Registering driver", "Driver", driverStatus.DriverID, "Watch state", driverStatus.WatchState)
	}
}

// ValidatingRuleParser implements the RuleParser interface.
var _ RuleParser = &ValidatingRuleParser{}

// ValidatingRuleParser provides the rulelist
//+kubebuilder:object:generate=false
type ValidatingRuleParser struct {
	RuleList
}

// MatchedDirective provides the interface function for the validating webhook
func (r *ValidatingRuleParser) MatchedDirective(workflow *Workflow, watchStates string, index int, label string) {
}
