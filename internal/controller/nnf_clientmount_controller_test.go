/*
 * Copyright 2024 Hewlett Packard Enterprise Development LP
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
	"sync"

	nnfv1alpha3 "github.com/NearNodeFlash/nnf-sos/api/v1alpha3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	dwsv1alpha2 "github.com/DataWorkflowServices/dws/api/v1alpha2"
)

var _ = Describe("Clientmount Controller Test", func() {

	var (
		r     *NnfClientMountReconciler
		ns    *corev1.Namespace
		setup sync.Once
	)

	BeforeEach(func() {
		setup.Do(func() {
			r = &NnfClientMountReconciler{
				Client: k8sClient,
				Scheme: testEnv.Scheme,
				Log:    ctrl.Log.WithName("controllers").WithName("ClientMountTest"),
			} // use this to access private reconciler methods

			ns = &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
				Name: "rabbit-node-2",
			}}
			Expect(k8sClient.Create(context.TODO(), ns)).To(Succeed())
			Eventually(func() error {
				return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(ns), ns)
			}).Should(Succeed())
		})
	})

	It("It should correctly create a human-readable lustre mapping for Servers ", func() {
		s := dwsv1alpha2.Servers{
			Status: dwsv1alpha2.ServersStatus{
				AllocationSets: []dwsv1alpha2.ServersStatusAllocationSet{
					{Label: "ost", Storage: map[string]dwsv1alpha2.ServersStatusStorage{
						"rabbit-node-1": dwsv1alpha2.ServersStatusStorage{
							AllocationSize: 123345,
						},
						"rabbit-node-2": dwsv1alpha2.ServersStatusStorage{
							AllocationSize: 123345,
						},
					}},
					{Label: "mdt", Storage: map[string]dwsv1alpha2.ServersStatusStorage{
						"rabbit-node-3": dwsv1alpha2.ServersStatusStorage{
							AllocationSize: 123345,
						},
						"rabbit-node-4": dwsv1alpha2.ServersStatusStorage{
							AllocationSize: 123345,
						},
						"rabbit-node-8": dwsv1alpha2.ServersStatusStorage{
							AllocationSize: 123345,
						},
					}},
				},
			},
		}

		m := createLustreMapping(&s)
		Expect(m).To(HaveLen(2))
		Expect(m["ost"]).To(HaveLen(2))
		Expect(m["ost"]).Should(ContainElements("rabbit-node-1", "rabbit-node-2"))
		Expect(m["mdt"]).To(HaveLen(3))
		Expect(m["mdt"]).Should(ContainElements("rabbit-node-3", "rabbit-node-4", "rabbit-node-8"))
	})

	XIt("", func() {
		wfName := "fluxjob-12345"

		cm := &dwsv1alpha2.ClientMount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      wfName + "-ownergroup",
				Namespace: ns.Name,
			},
			Spec: dwsv1alpha2.ClientMountSpec{
				DesiredState: dwsv1alpha2.ClientMountStateMounted,
				Mounts: []dwsv1alpha2.ClientMountInfo{
					{
						MountPath: "/tmp/",
						Device: dwsv1alpha2.ClientMountDevice{
							Type: "lustre",
							Lustre: &dwsv1alpha2.ClientMountDeviceLustre{
								FileSystemName: "fake",
								MgsAddresses:   "fake",
							},
							DeviceReference: &dwsv1alpha2.ClientMountDeviceReference{
								ObjectReference: corev1.ObjectReference{
									Name:      "fake",
									Namespace: corev1.NamespaceDefault,
								},
							}},
						TargetType: "directory", Type: "lustre"},
				},
			},
		}
		storage := &nnfv1alpha3.NnfStorage{ObjectMeta: metav1.ObjectMeta{
			Name:      "imstorage",
			Namespace: metav1.NamespaceDefault,
		}}
		per := &dwsv1alpha2.PersistentStorageInstance{ObjectMeta: metav1.ObjectMeta{
			Name:      "impersistent",
			Namespace: metav1.NamespaceDefault,
		}}
		server := &dwsv1alpha2.Servers{ObjectMeta: metav1.ObjectMeta{
			Name:      "imaserver",
			Namespace: metav1.NamespaceDefault,
		}}

		dwsv1alpha2.AddOwnerLabels(cm, storage)
		dwsv1alpha2.AddOwnerLabels(storage, per)
		dwsv1alpha2.AddOwnerLabels(server, per)

		idx := 7
		addDirectiveIndexLabel(cm, idx)
		addDirectiveIndexLabel(storage, idx)

		Expect(k8sClient.Create(context.TODO(), cm)).To(Succeed())
		Eventually(func() error {
			return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(cm), cm)
		}).Should(Succeed())

		Expect(k8sClient.Create(context.TODO(), storage)).To(Succeed())
		Eventually(func() error {
			return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(storage), storage)
		}).Should(Succeed())

		Expect(k8sClient.Create(context.TODO(), per)).To(Succeed())
		Expect(k8sClient.Create(context.TODO(), server)).To(Succeed())

		s, err := r.getServerForClientMount(context.TODO(), cm)
		Expect(s).ToNot(BeNil())
		Expect(err).To(BeNil())
		Expect(s.Name).To(Equal("imAServer"))
		Expect(s.Namespace).To(Equal(metav1.NamespaceDefault))

	})
})
