/*
 * Copyright 2022, 2025 Hewlett Packard Enterprise Development LP
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
	"testing"

	. "github.com/onsi/ginkgo/v2"
)

func TestVarHandler(t *testing.T) {
	varMap := map[string]string{
		"$BIGDOG":  "Jules",
		"$GREYDOG": "Henri",
	}

	v := NewVarHandler(varMap)
	// Add a late-arriving variable.
	v.VarMap["$FAVTOY"] = "rope"

	in1 := "The big dog is $BIGDOG, the little dog is $GREYDOG. $BIGDOG and $GREYDOG are best friends and their favorite toy is the $FAVTOY."
	want1 := "The big dog is Jules, the little dog is Henri. Jules and Henri are best friends and their favorite toy is the rope."
	out1 := v.ReplaceAll(in1)
	if out1 != want1 {
		t.Errorf("Did not get the desired result.  Got (%s)", out1)
	}

	// Change a variable.
	v.VarMap["$FAVTOY"] = "ball"
	in2 := "$BIGDOG likes the $FAVTOY."
	want2 := "Jules likes the ball."
	out2 := v.ReplaceAll(in2)
	if out2 != want2 {
		t.Errorf("Did not get desired result.  Got (%s)", out2)
	}

	// Delete a variable.
	delete(v.VarMap, "$FAVTOY")
	in3 := "$GREYDOG's favorite toy was the $FAVTOY."
	want3 := "Henri's favorite toy was the $FAVTOY."
	out3 := v.ReplaceAll(in3)
	if out3 != want3 {
		t.Errorf("Did not get desired result.  Got (%s)", out3)
	}

	// Add a list to turn into variables.
	v.VarMap["$DEVICE_LIST"] = "/dev/nvme0n1 /dev/nvme1n1 /dev/nvme0n2 /dev/nvme1n2"
	if err := v.ListToVars("$DEVICE_LIST", "$DEVICE"); err != nil {
		t.Errorf("Did not split list: %v", err)
	} else {
		in4 := "zpool mirror $DEVICE1 $DEVICE2 mirror $DEVICE3 $DEVICE4"
		want4 := "zpool mirror /dev/nvme0n1 /dev/nvme1n1 mirror /dev/nvme0n2 /dev/nvme1n2"
		out4 := v.ReplaceAll(in4)
		if out4 != want4 {
			t.Errorf("Did not get desired result. Got (%s)", out4)
		}
	}
}

// Just touch ginkgo, so it's here to interpret any ginkgo args from
// "make test", so that doesn't fail on this test file.
var _ = BeforeSuite(func() {})

func TestReplaceAllNested(t *testing.T) {
	tests := []struct {
		name     string
		varMap   map[string]string
		input    string
		expected string
	}{
		{
			name: "simple replacement",
			varMap: map[string]string{
				"$VAR1": "value1",
			},
			input:    "This is $VAR1",
			expected: "This is value1",
		},
		{
			name: "nested variable replacement",
			varMap: map[string]string{
				"$OUTER": "$INNER",
				"$INNER": "final_value",
			},
			input:    "Result: $OUTER",
			expected: "Result: final_value",
		},
		{
			name: "double nested replacement",
			varMap: map[string]string{
				"$LEVEL1": "$LEVEL2",
				"$LEVEL2": "$LEVEL3",
				"$LEVEL3": "deep_value",
			},
			input:    "Deep: $LEVEL1",
			expected: "Deep: deep_value",
		},
		{
			name: "multiple variables in one string",
			varMap: map[string]string{
				"$A": "alpha",
				"$B": "beta",
				"$C": "gamma",
			},
			input:    "$A-$B-$C",
			expected: "alpha-beta-gamma",
		},
		{
			name: "nested with multiple variables",
			varMap: map[string]string{
				"$PATH": "/mnt/$DIR/$FILE",
				"$DIR":  "data",
				"$FILE": "output.txt",
			},
			input:    "Writing to $PATH",
			expected: "Writing to /mnt/data/output.txt",
		},
		{
			name: "depth limit protection - deep nesting within limit",
			varMap: map[string]string{
				"$L1": "$L2",
				"$L2": "$L3",
				"$L3": "resolved",
			},
			input:    "Deep: $L1",
			expected: "Deep: resolved", // Within depth limit, fully resolves
		},
		{
			name: "no variables to replace",
			varMap: map[string]string{
				"$VAR": "value",
			},
			input:    "No variables here",
			expected: "No variables here",
		},
		{
			name: "variable not in map",
			varMap: map[string]string{
				"$KNOWN": "known_value",
			},
			input:    "$UNKNOWN stays as is",
			expected: "$UNKNOWN stays as is",
		},
		{
			name:     "empty var map",
			varMap:   map[string]string{},
			input:    "$VAR should stay",
			expected: "$VAR should stay",
		},
		{
			name: "variable value contains partial variable name",
			varMap: map[string]string{
				"$PREFIX":    "/mnt/nnf",
				"$FULL_PATH": "$PREFIX/subdir",
			},
			input:    "Path is $FULL_PATH",
			expected: "Path is /mnt/nnf/subdir",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewVarHandler(tt.varMap)
			result := v.ReplaceAll(tt.input)
			if result != tt.expected {
				t.Errorf("ReplaceAll() = %q, want %q", result, tt.expected)
			}
		})
	}
}
