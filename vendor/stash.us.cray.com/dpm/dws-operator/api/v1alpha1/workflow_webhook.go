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
	"os"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-dws-cray-hpe-com-v1alpha1-workflow,mutating=true,failurePolicy=fail,sideEffects=None,groups=dws.cray.hpe.com,resources=workflows,verbs=create;update,versions=v1alpha1,name=mworkflow.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Defaulter = &Workflow{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (workflow *Workflow) Default() {
	workflowlog.Info("default", "name", workflow.Name)

	webhookType := os.Getenv("WORKFLOW_WEBHOOK_TYPE")
	if strings.Contains(webhookType, "mutating") == false {
		return
	}

	driverID := os.Getenv("WORKFLOW_WEBHOOK_DRIVER_ID")
	if len(driverID) == 0 {
		return
	}

	// Check if there are any DW directives that we care about
	err := validate(workflow)
	if err != nil {
		return
	}

	// Apply driver status sections in the WFR
	applyDriverStatus(driverID, workflow.Spec.DWDirectives, workflow)

	return
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-dws-cray-hpe-com-v1alpha1-workflow,mutating=false,failurePolicy=fail,sideEffects=None,groups=dws.cray.hpe.com,resources=workflows,verbs=create;update,versions=v1alpha1,name=vworkflow.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Validator = &Workflow{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Workflow) ValidateCreate() error {
	workflowlog.Info("validate create", "name", r.Name)

	webhookType := os.Getenv("WORKFLOW_WEBHOOK_TYPE")
	if strings.Contains(webhookType, "validating") == false {
		return nil
	}
	return validate(r)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Workflow) ValidateUpdate(old runtime.Object) error {
	workflowlog.Info("validate update", "name", r.Name)

	webhookType := os.Getenv("WORKFLOW_WEBHOOK_TYPE")
	if strings.Contains(webhookType, "validating") == false {
		return nil
	}
	return validate(r)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Workflow) ValidateDelete() error {
	workflowlog.Info("validate delete", "name", r.Name)
	return nil
}

func validate(workflow *Workflow) error {

	// Retrieve the `dwDirectives` values.
	dwDirectives := workflow.Spec.DWDirectives
	if len(dwDirectives) == 0 {
		return errors.New("no #DW directives found")
	}

	// WORKFLOW_WEBHOOK_RULESET_LIST is a comma separated list of rule sets
	ruleSetList := strings.Split(os.Getenv("WORKFLOW_WEBHOOK_RULESET_LIST"), ",")

	foundValid := false
	for _, ruleSetName := range ruleSetList {
		if ruleSetName == "" {
			continue
		}
		// validate syntax #DW syntax
		const rejectUnsupportedCommands bool = true

		dwdRules := &DWDirectiveRule{}
		namespacedName := types.NamespacedName{
			Name:      ruleSetName,
			Namespace: "dws-operator-system",
		}

		if err := c.Get(context.TODO(), namespacedName, dwdRules); err != nil {
			return err
		}

		valid, err := dwdparse.ValidateDWDirectives(dwdRules.Spec, dwDirectives, rejectUnsupportedCommands)
		if err != nil {
			// #DW parser validation failed
			workflowlog.Info("dwDirective validation failed", "Error", err)
			return err
		}

		if valid == true {
			foundValid = true
		}
	}

	if foundValid == false {
		return errors.New("Invalid directive found")
	}

	return nil
}

var stateMap map[string][]string

func setStateMap() {
	// Check for state/command mapping overrides, otherwise use defaults
	setupCommands := strings.Split(os.Getenv("SETUP_COMMANDS"), ",")
	if len(setupCommands[0]) == 0 {
		setupCommands = []string{"jobdw", "create_persistent"}
	}
	teardownCommands := strings.Split(os.Getenv("TEARDOWN_COMMANDS"), ",")
	if len(teardownCommands[0]) == 0 {
		teardownCommands = []string{"jobdw", "destroy_persistent"}
	}
	datainCommands := strings.Split(os.Getenv("DATA_IN_COMMANDS"), ",")
	if len(datainCommands[0]) == 0 {
		datainCommands = []string{"copy_in", "copy-in", "tier_in", "tier-in"}
	}
	dataoutCommands := strings.Split(os.Getenv("DATA_OUT_COMMANDS"), ",")
	if len(dataoutCommands[0]) == 0 {
		dataoutCommands = []string{"copy_out", "copy-out", "tier_out", "tier-out"}
	}
	prerunCommands := strings.Split(os.Getenv("PRERUN_COMMANDS"), ",")
	if len(prerunCommands[0]) == 0 {
		prerunCommands = []string{"jobdw", "persistentdw"}
	}
	postrunCommands := strings.Split(os.Getenv("POSTRUN_COMMANDS"), ",")
	if len(postrunCommands[0]) == 0 {
		postrunCommands = []string{"jobdw", "persistentdw"}
	}

	stateMap = map[string][]string{
		"setup":    setupCommands,
		"teardown": teardownCommands,
		"data_out": dataoutCommands,
		"data_in":  datainCommands,
		"pre_run":  prerunCommands,
		"post_run": postrunCommands,
	}
}

func applyDriverStatus(driverID string, directives []string, workflow *Workflow) {

	// Calling this everytime picks up an changes that may have occurred in environment variables.
	setStateMap()

	// Update driver status entries to indicate driver availability
	watchStates := strings.Split(os.Getenv("WORKFLOW_WEBHOOK_WATCH_STATES"), ",")
	if len(watchStates) == 0 {
		// Nothing to do
		return
	}

	registrationMap := make(map[string]bool)
	// Build a map of previously registered states for this driver
	for _, s := range workflow.Status.Drivers {
		if s.DriverID == driverID {
			registrationMap[s.WatchState] = true
		}
	}

	for idx, dwd := range directives {
		dwdArgs := strings.Fields(dwd)

		for _, state := range watchStates {
			// If this driver is already registered for this directive, skip it
			if registrationMap[state] {
				continue
			}
			for _, cmd := range stateMap[state] {
				if cmd == dwdArgs[1] {
					driverStatus := WorkflowDriverStatus{}
					driverStatus.DriverID = driverID
					// Register states for this driver
					driverStatus.DWDIndex = idx
					driverStatus.WatchState = state
					workflow.Status.Drivers = append(workflow.Status.Drivers, driverStatus)
					workflowlog.Info("Registering driver", "Driver", driverStatus.DriverID, "Watch state", driverStatus.WatchState)
				}
			}
		}
	}
}
