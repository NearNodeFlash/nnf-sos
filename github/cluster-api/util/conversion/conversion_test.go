/*
Copyright 2019 The Kubernetes Authors.

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

package conversion

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	//metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	//"k8s.io/apimachinery/pkg/runtime/schema"
	//nnfv1alpha1 "github.com/NearNodeFlash/nnf-sos/api/v1alpha1"
)

var (

// +crdbumper:scaffold:gvk
)

func TestMarshalData(t *testing.T) {
	_ = NewWithT(t)

	// +crdbumper:scaffold:marshaldata
}

func TestUnmarshalData(t *testing.T) {
	_ = NewWithT(t)

	// +crdbumper:scaffold:unmarshaldata
}

// Just touch ginkgo, so it's here to interpret any ginkgo args from
// "make test", so that doesn't fail on this test file.
var _ = BeforeSuite(func() {})
