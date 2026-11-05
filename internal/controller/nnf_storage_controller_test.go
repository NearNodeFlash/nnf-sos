/*
 * Copyright 2024-2026 Hewlett Packard Enterprise Development LP
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

package controller

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nnfv1alpha11 "github.com/NearNodeFlash/nnf-sos/api/v1alpha11"
)

// nnfNodeStorageListStub serves a fixed set of NnfNodeStorages from List so the
// live node controllers in the test suite cannot change the children mid-test.
type nnfNodeStorageListStub struct {
	client.Client
	items []nnfv1alpha11.NnfNodeStorage
}

func (c *nnfNodeStorageListStub) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	if l, ok := list.(*nnfv1alpha11.NnfNodeStorageList); ok {
		l.Items = c.items
		return nil
	}
	return c.Client.List(ctx, list, opts...)
}

var _ = Describe("NNFStorage Controller Test", func() {

	It("It should correctly create a human-readable lustre mapping for NnfStorage", func() {
		s := nnfv1alpha11.NnfStorage{
			Spec: nnfv1alpha11.NnfStorageSpec{
				AllocationSets: []nnfv1alpha11.NnfStorageAllocationSetSpec{
					{Name: "ost", Nodes: []nnfv1alpha11.NnfStorageAllocationNodes{
						{Name: "rabbit-node-1", Count: 2},
						{Name: "rabbit-node-2", Count: 1}},
					},
					// throw another OST on rabbit-node-2
					{Name: "ost", Nodes: []nnfv1alpha11.NnfStorageAllocationNodes{
						{Name: "rabbit-node-2", Count: 1}},
					},
					{Name: "mdt", Nodes: []nnfv1alpha11.NnfStorageAllocationNodes{
						{Name: "rabbit-node-3", Count: 1},
						{Name: "rabbit-node-4", Count: 1},
						{Name: "rabbit-node-8", Count: 1}},
					},
					{Name: "mgt", Nodes: []nnfv1alpha11.NnfStorageAllocationNodes{
						{Name: "rabbit-node-3", Count: 1}},
					},
					{Name: "mgtmdt", Nodes: []nnfv1alpha11.NnfStorageAllocationNodes{
						{Name: "rabbit-node-4", Count: 1}},
					},
				},
			},
		}

		Expect(s.Spec.AllocationSets).To(HaveLen(5))
		m := getLustreMappingFromStorage(&s)
		Expect(m).To(HaveLen(5)) // should have keys for 4 lustre components (i.e. ost, mdt, mgt, mgtmdt) + rabbits

		Expect(m["ost"]).To(HaveLen(4))
		Expect(m["ost"]).Should(ContainElements("rabbit-node-1", "rabbit-node-1", "rabbit-node-2", "rabbit-node-2"))

		Expect(m["mdt"]).To(HaveLen(3))
		Expect(m["mdt"]).Should(ContainElements("rabbit-node-3", "rabbit-node-4", "rabbit-node-8"))

		Expect(m["mgt"]).To(HaveLen(1))
		Expect(m["mgt"]).Should(ContainElements("rabbit-node-3"))

		Expect(m["mgtmdt"]).To(HaveLen(1))
		Expect(m["mgtmdt"]).Should(ContainElements("rabbit-node-4"))

		Expect(m["nnfNode"]).To(HaveLen(5))
		Expect(m["nnfNode"]).Should(ContainElements("rabbit-node-1", "rabbit-node-2", "rabbit-node-3", "rabbit-node-4", "rabbit-node-8"))
	})

	Describe("aggregateNodeStorageStatus skipOST0 probe", func() {
		// The probe peeks at OST readiness with an offset expected count; it must
		// never write status. Without the !skipOST0 guards, an in-memory Ready
		// downgrade on every healthy-lustre pass reaches the API on any mid-pass
		// requeue.
		newStorage := func() *nnfv1alpha11.NnfStorage {
			return &nnfv1alpha11.NnfStorage{
				ObjectMeta: metav1.ObjectMeta{Name: "probe-test", Namespace: "default"},
				Spec: nnfv1alpha11.NnfStorageSpec{
					FileSystemType: "lustre",
					AllocationSets: []nnfv1alpha11.NnfStorageAllocationSetSpec{{
						Name:                 "ost",
						NnfStorageLustreSpec: nnfv1alpha11.NnfStorageLustreSpec{TargetType: "ost"},
						Nodes: []nnfv1alpha11.NnfStorageAllocationNodes{
							{Name: "rabbit-probe-node-1", Count: 1},
							{Name: "rabbit-probe-node-2", Count: 1},
						},
					}},
				},
				Status: nnfv1alpha11.NnfStorageStatus{
					Ready: true,
					AllocationSets: []nnfv1alpha11.NnfStorageAllocationSetStatus{
						{Ready: true, AllocationCount: 2},
					},
				},
			}
		}

		newChild := func(namespace string, ready bool) nnfv1alpha11.NnfNodeStorage {
			return nnfv1alpha11.NnfNodeStorage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "probe-child",
					Namespace: namespace,
					// A finalizer keeps the child-timeout branch out of play
					// regardless of NNF_CHILD_RESOURCE_TIMEOUT_SECONDS.
					Finalizers: []string{finalizerNnfNodeStorage},
				},
				Status: nnfv1alpha11.NnfNodeStorageStatus{Ready: ready},
			}
		}

		newReconciler := func(children ...nnfv1alpha11.NnfNodeStorage) *NnfStorageReconciler {
			return &NnfStorageReconciler{
				Client: &nnfNodeStorageListStub{Client: k8sClient, items: children},
				Log:    ctrl.Log.WithName("probe-test"),
				Scheme: testEnv.Scheme,
			}
		}

		It("does not write status when a child is not ready", func() {
			storage := newStorage()
			before := *storage.Status.DeepCopy()
			r := newReconciler(newChild("rabbit-probe-node-1", true), newChild("rabbit-probe-node-2", false))

			res, err := r.aggregateNodeStorageStatus(context.TODO(), storage, 0, false, true)
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(Equal(&ctrl.Result{}))
			Expect(storage.Status).To(Equal(before))
		})

		It("does not write status when a child is missing", func() {
			storage := newStorage()
			before := *storage.Status.DeepCopy()
			// No children: found=0 vs expected=1 (one less than the node count).
			r := newReconciler()

			res, err := r.aggregateNodeStorageStatus(context.TODO(), storage, 0, false, true)
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(Equal(&ctrl.Result{}))
			Expect(storage.Status).To(Equal(before))
		})

		It("still downgrades Ready outside the probe", func() {
			storage := newStorage()
			r := newReconciler(newChild("rabbit-probe-node-1", true), newChild("rabbit-probe-node-2", false))

			_, err := r.aggregateNodeStorageStatus(context.TODO(), storage, 0, false, false)
			Expect(err).ToNot(HaveOccurred())
			Expect(storage.Status.AllocationSets[0].Ready).To(BeFalse())
			Expect(storage.Status.Ready).To(BeFalse())
		})

		It("still downgrades Ready outside the probe when a child is missing", func() {
			storage := newStorage()
			r := newReconciler(newChild("rabbit-probe-node-1", true))

			_, err := r.aggregateNodeStorageStatus(context.TODO(), storage, 0, false, false)
			Expect(err).ToNot(HaveOccurred())
			Expect(storage.Status.AllocationSets[0].Ready).To(BeFalse())
			Expect(storage.Status.Ready).To(BeFalse())
		})
	})
})
