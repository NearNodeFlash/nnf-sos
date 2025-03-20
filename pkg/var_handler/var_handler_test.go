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
