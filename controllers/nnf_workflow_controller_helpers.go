package controllers

import (
	"strings"

	dwsv1alpha1 "github.hpe.com/hpe/hpc-dpm-dws-operator/api/v1alpha1"
	"github.hpe.com/hpe/hpc-dpm-dws-operator/utils/dwdparse"
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

// Returns a <name, path> pair for the given staging argument (typically source or destination)
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
