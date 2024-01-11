/*
 * Copyright 2022 Hewlett Packard Enterprise Development LP
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

package var_handler

import (
	"fmt"
	"strings"
)

type VarHandler struct {
	VarMap map[string]string
}

func NewVarHandler(vars map[string]string) *VarHandler {
	v := &VarHandler{}
	v.VarMap = vars
	return v
}

func (v *VarHandler) AddVar(name string, value string) {
	v.VarMap[name] = value
}

// ListToVars splits the value of one of its variables, and creates a new
// indexed variable for each of the items in the split.
func (v *VarHandler) ListToVars(listVarName, newVarPrefix string) error {
	theList, ok := v.VarMap[listVarName]
	if !ok {
		return fmt.Errorf("Unable to find the variable named %s", listVarName)
	}

	for i, val := range strings.Split(theList, " ") {
		v.VarMap[fmt.Sprintf("%s%d", newVarPrefix, i+1)] = val
	}
	return nil
}

func (v *VarHandler) ReplaceAll(s string) string {
	for key, value := range v.VarMap {
		s = strings.ReplaceAll(s, key, value)
	}
	return s
}
