/*
 * Copyright 2025 Hewlett Packard Enterprise Development LP
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

package v1alpha5

import (
	"testing"

	fuzz "github.com/google/gofuzz"
	. "github.com/onsi/ginkgo/v2"
	"k8s.io/apimachinery/pkg/api/apitesting/fuzzer"
	runtimeserializer "k8s.io/apimachinery/pkg/runtime/serializer"

	nnfv1alpha6 "github.com/NearNodeFlash/nnf-sos/api/v1alpha6"
	utilconversion "github.com/NearNodeFlash/nnf-sos/github/cluster-api/util/conversion"
)

func TestFuzzyConversion(t *testing.T) {

	t.Run("for NnfAccess", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Hub:   &nnfv1alpha6.NnfAccess{},
		Spoke: &NnfAccess{},
	}))

	t.Run("for NnfContainerProfile", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Hub:   &nnfv1alpha6.NnfContainerProfile{},
		Spoke: &NnfContainerProfile{},
	}))

	t.Run("for NnfDataMovement", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Hub:   &nnfv1alpha6.NnfDataMovement{},
		Spoke: &NnfDataMovement{},
	}))

	t.Run("for NnfDataMovementManager", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Hub:   &nnfv1alpha6.NnfDataMovementManager{},
		Spoke: &NnfDataMovementManager{},
	}))

	t.Run("for NnfDataMovementProfile", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Hub:   &nnfv1alpha6.NnfDataMovementProfile{},
		Spoke: &NnfDataMovementProfile{},
	}))

	t.Run("for NnfLustreMGT", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Hub:   &nnfv1alpha6.NnfLustreMGT{},
		Spoke: &NnfLustreMGT{},
	}))

	t.Run("for NnfNode", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Hub:   &nnfv1alpha6.NnfNode{},
		Spoke: &NnfNode{},
	}))

	t.Run("for NnfNodeBlockStorage", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Hub:   &nnfv1alpha6.NnfNodeBlockStorage{},
		Spoke: &NnfNodeBlockStorage{},
	}))

	t.Run("for NnfNodeECData", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Hub:   &nnfv1alpha6.NnfNodeECData{},
		Spoke: &NnfNodeECData{},
	}))

	t.Run("for NnfNodeStorage", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Hub:   &nnfv1alpha6.NnfNodeStorage{},
		Spoke: &NnfNodeStorage{},
	}))

	t.Run("for NnfPortManager", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Hub:   &nnfv1alpha6.NnfPortManager{},
		Spoke: &NnfPortManager{},
	}))

	t.Run("for NnfStorage", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Hub:   &nnfv1alpha6.NnfStorage{},
		Spoke: &NnfStorage{},
	}))

	t.Run("for NnfStorageProfile", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Hub:         &nnfv1alpha6.NnfStorageProfile{},
		Spoke:       &NnfStorageProfile{},
		FuzzerFuncs: []fuzzer.FuzzerFuncs{NnfStorageProfileFuzzFunc},
	}))

	t.Run("for NnfSystemStorage", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Hub:   &nnfv1alpha6.NnfSystemStorage{},
		Spoke: &NnfSystemStorage{},
	}))

}

func NnfStorageProfileFuzzFunc(_ runtimeserializer.CodecFactory) []interface{} {
	return []interface{}{
		NnfStorageProfilev5Fuzzer,
		NnfStorageProfilev6Fuzzer,
	}
}

func NnfStorageProfilev5Fuzzer(in *NnfStorageProfileData, c fuzz.Continue) {
	// Tell the fuzzer to begin by fuzzing everything in the object.
	c.FuzzNoCustom(in)

	// Remove any fuzz from the PostMount and PreUnmount fields in the MDT, MGT, and MGT/MDT
	// command lines. They aren't used, and they're removed starting in v1alpha6
	in.LustreStorage.MdtCmdLines.PostMount = []string{}
	in.LustreStorage.MdtCmdLines.PreUnmount = []string{}
	in.LustreStorage.MgtCmdLines.PostMount = []string{}
	in.LustreStorage.MgtCmdLines.PreUnmount = []string{}
	in.LustreStorage.MgtMdtCmdLines.PostMount = []string{}
	in.LustreStorage.MgtMdtCmdLines.PreUnmount = []string{}
}

func NnfStorageProfilev6Fuzzer(in *nnfv1alpha6.NnfStorageProfileData, c fuzz.Continue) {
	// Tell the fuzzer to begin by fuzzing everything in the object.
	c.FuzzNoCustom(in)
}

// Just touch ginkgo, so it's here to interpret any ginkgo args from
// "make test", so that doesn't fail on this test file.
var _ = BeforeSuite(func() {})
