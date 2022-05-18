package controllers

import (
	"fmt"
	"strings"

	dwsv1alpha1 "github.com/HewlettPackard/dws/api/v1alpha1"
	"github.com/HewlettPackard/dws/utils/dwdparse"
)

// Returns the directive index with the 'name' argument matching name, or -1 if not found
func findDirectiveIndexByName(workflow *dwsv1alpha1.Workflow, name string) int {
	for idx, directive := range workflow.Spec.DWDirectives {
		parameters, _ := dwdparse.BuildArgsMap(directive)
		if parameters["name"] == name {
			return idx
		}
	}
	return -1
}

// Returns the directive index matching the copy_out directive whose source field references
// the provided name argument, or -1 if not found.
func findCopyOutDirectiveIndexByName(workflow *dwsv1alpha1.Workflow, name string) int {
	for idx, directive := range workflow.Spec.DWDirectives {
		if strings.HasPrefix(directive, "#DW copy_out") {
			parameters, _ := dwdparse.BuildArgsMap(directive) // ignore error, directives are validated in proposal

			srcName, _ := splitStagingArgumentIntoNameAndPath(parameters["source"]) // i.e. source=$JOB_DW_[name]
			if srcName == name {
				return idx
			}
		}
	}

	return -1
}

// Returns a <name, path> pair for the given staging argument (typically source or destination)
// i.e. $JOB_DW_my-file-system-name/path/to/a/file into "my-file-system-name" and "/path/to/a/file"
func splitStagingArgumentIntoNameAndPath(arg string) (string, string) {

	var name = ""
	if strings.HasPrefix(arg, "$JOB_DW_") {
		name = strings.SplitN(strings.Replace(arg, "$JOB_DW_", "", 1), "/", 2)[0]
	} else if strings.HasPrefix(arg, "$PERSISTENT_DW_") {
		name = strings.SplitN(strings.Replace(arg, "$PERSISTENT_DW_", "", 1), "/", 2)[0]
	}
	var path = "/"
	if strings.Count(arg, "/") >= 1 {
		path = "/" + strings.SplitN(arg, "/", 2)[1]
	}
	return name, path

}

// createDirectiveBreakdownName returns a DBD name for the #DW directive at the specified index
func createDirectiveBreakdownName(workflow *dwsv1alpha1.Workflow, dwIndex int) string {
	return fmt.Sprintf("%s-%d", workflow.Name, dwIndex)
}

// Returns the <name, namespace> pair for the #DW directive at the specified index
func getStorageReferenceNameFromWorkflowActual(workflow *dwsv1alpha1.Workflow, dwdIndex int) (string, string) {

	directive := workflow.Spec.DWDirectives[dwdIndex]
	p, _ := dwdparse.BuildArgsMap(directive) // ignore error, directives were validated in proposal

	var name, namespace string

	switch p["command"] {
	case "persistentdw", "create_persistent", "delete_persistent":
		name = p["name"]
		namespace = workflow.Namespace
	default:
		name = workflow.Status.DirectiveBreakdowns[dwdIndex].Name
		namespace = workflow.Status.DirectiveBreakdowns[dwdIndex].Namespace
	}

	return name, namespace
}

// Returns the intended <name, namespace> pair for the #DW directive at the specified index
func getStorageReferenceNameFromWorkflowIntended(workflow *dwsv1alpha1.Workflow, dwdIndex int) (string, string) {

	directive := workflow.Spec.DWDirectives[dwdIndex]
	p, _ := dwdparse.BuildArgsMap(directive) // ignore error, directives were validated in proposal

	var name string
	namespace := workflow.Namespace
	switch p["command"] {
	case "persistentdw", "create_persistent", "delete_persistent":
		name = p["name"]
	default:
		name = createDirectiveBreakdownName(workflow, dwdIndex)
	}

	return name, namespace
}

// Returns the <name, namespace> pair for the #DW directive in the given DirectiveBreakdown
func getStorageReferenceNameFromDBD(dbd *dwsv1alpha1.DirectiveBreakdown) (string, string) {

	var name string
	namespace := dbd.Namespace
	if dbd.Spec.Lifetime == dwsv1alpha1.DirectiveLifetimePersistent {
		name = dbd.Spec.Name
	} else {
		name = dbd.Name
	}
	return name, namespace
}
