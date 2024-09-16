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

	nnfv1alpha2 "github.com/NearNodeFlash/nnf-sos/api/v1alpha2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	oldNnfAccessGVK = schema.GroupVersionKind{
		Group:   nnfv1alpha2.GroupVersion.Group,
		Version: "v1old",
		Kind:    "NnfAccess",
	}

	oldNnfContainerProfileGVK = schema.GroupVersionKind{
		Group:   nnfv1alpha2.GroupVersion.Group,
		Version: "v1old",
		Kind:    "NnfContainerProfile",
	}

	oldNnfDataMovementGVK = schema.GroupVersionKind{
		Group:   nnfv1alpha2.GroupVersion.Group,
		Version: "v1old",
		Kind:    "NnfDataMovement",
	}

	oldNnfDataMovementManagerGVK = schema.GroupVersionKind{
		Group:   nnfv1alpha2.GroupVersion.Group,
		Version: "v1old",
		Kind:    "NnfDataMovementManager",
	}

	oldNnfDataMovementProfileGVK = schema.GroupVersionKind{
		Group:   nnfv1alpha2.GroupVersion.Group,
		Version: "v1old",
		Kind:    "NnfDataMovementProfile",
	}

	oldNnfLustreMGTGVK = schema.GroupVersionKind{
		Group:   nnfv1alpha2.GroupVersion.Group,
		Version: "v1old",
		Kind:    "NnfLustreMGT",
	}

	oldNnfNodeGVK = schema.GroupVersionKind{
		Group:   nnfv1alpha2.GroupVersion.Group,
		Version: "v1old",
		Kind:    "NnfNode",
	}

	oldNnfNodeBlockStorageGVK = schema.GroupVersionKind{
		Group:   nnfv1alpha2.GroupVersion.Group,
		Version: "v1old",
		Kind:    "NnfNodeBlockStorage",
	}

	oldNnfNodeECDataGVK = schema.GroupVersionKind{
		Group:   nnfv1alpha2.GroupVersion.Group,
		Version: "v1old",
		Kind:    "NnfNodeECData",
	}

	oldNnfNodeStorageGVK = schema.GroupVersionKind{
		Group:   nnfv1alpha2.GroupVersion.Group,
		Version: "v1old",
		Kind:    "NnfNodeStorage",
	}

	oldNnfPortManagerGVK = schema.GroupVersionKind{
		Group:   nnfv1alpha2.GroupVersion.Group,
		Version: "v1old",
		Kind:    "NnfPortManager",
	}

	oldNnfStorageGVK = schema.GroupVersionKind{
		Group:   nnfv1alpha2.GroupVersion.Group,
		Version: "v1old",
		Kind:    "NnfStorage",
	}

	oldNnfStorageProfileGVK = schema.GroupVersionKind{
		Group:   nnfv1alpha2.GroupVersion.Group,
		Version: "v1old",
		Kind:    "NnfStorageProfile",
	}

	oldNnfSystemStorageGVK = schema.GroupVersionKind{
		Group:   nnfv1alpha2.GroupVersion.Group,
		Version: "v1old",
		Kind:    "NnfSystemStorage",
	}

// +crdbumper:scaffold:gvk
)

func TestMarshalData(t *testing.T) {
	g := NewWithT(t)

	t.Run("NnfAccess should write source object to destination", func(*testing.T) {
		src := &nnfv1alpha2.NnfAccess{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
				Labels: map[string]string{
					"label1": "",
				},
			},
			Spec: nnfv1alpha2.NnfAccessSpec{
				DesiredState: "mounted",
				UserID:       1551,
				GroupID:      2442,
			},
		}

		dst := &unstructured.Unstructured{}
		dst.SetGroupVersionKind(oldNnfAccessGVK)
		dst.SetName("test-1")

		g.Expect(MarshalData(src, dst)).To(Succeed())
		// ensure the src object is not modified
		g.Expect(src.GetLabels()).ToNot(BeEmpty())

		g.Expect(dst.GetAnnotations()[DataAnnotation]).ToNot(BeEmpty())

		g.Expect(dst.GetAnnotations()[DataAnnotation]).To(ContainSubstring("mounted"))
		g.Expect(dst.GetAnnotations()[DataAnnotation]).To(ContainSubstring("2442"))
		g.Expect(dst.GetAnnotations()[DataAnnotation]).To(ContainSubstring("1551"))
	})

	t.Run("NnfAccess should append the annotation", func(*testing.T) {
		src := &nnfv1alpha2.NnfAccess{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
			},
		}
		dst := &unstructured.Unstructured{}
		dst.SetGroupVersionKind(nnfv1alpha2.GroupVersion.WithKind("NnfAccess"))
		dst.SetName("test-1")
		dst.SetAnnotations(map[string]string{
			"annotation": "1",
		})

		g.Expect(MarshalData(src, dst)).To(Succeed())
		g.Expect(dst.GetAnnotations()).To(HaveLen(2))
	})

	t.Run("NnfContainerProfile should write source object to destination", func(*testing.T) {
		prerun := int64(345)
		userid := uint32(7667)
		groupid := uint32(8448)
		src := &nnfv1alpha2.NnfContainerProfile{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
				Labels: map[string]string{
					"label1": "",
				},
			},
			Data: nnfv1alpha2.NnfContainerProfileData{
				PreRunTimeoutSeconds: &prerun,
				UserID:               &userid,
				GroupID:              &groupid,
			},
		}

		dst := &unstructured.Unstructured{}
		dst.SetGroupVersionKind(oldNnfContainerProfileGVK)
		dst.SetName("test-1")

		g.Expect(MarshalData(src, dst)).To(Succeed())
		// ensure the src object is not modified
		g.Expect(src.GetLabels()).ToNot(BeEmpty())

		g.Expect(dst.GetAnnotations()[DataAnnotation]).ToNot(BeEmpty())

		g.Expect(dst.GetAnnotations()[DataAnnotation]).To(ContainSubstring("345"))
		g.Expect(dst.GetAnnotations()[DataAnnotation]).To(ContainSubstring("7667"))
		g.Expect(dst.GetAnnotations()[DataAnnotation]).To(ContainSubstring("8448"))
	})

	t.Run("NnfContainerProfile should append the annotation", func(*testing.T) {
		src := &nnfv1alpha2.NnfContainerProfile{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
			},
		}
		dst := &unstructured.Unstructured{}
		dst.SetGroupVersionKind(nnfv1alpha2.GroupVersion.WithKind("NnfContainerProfile"))
		dst.SetName("test-1")
		dst.SetAnnotations(map[string]string{
			"annotation": "1",
		})

		g.Expect(MarshalData(src, dst)).To(Succeed())
		g.Expect(dst.GetAnnotations()).To(HaveLen(2))
	})

	t.Run("NnfDataMovement should write source object to destination", func(*testing.T) {
		destpath := &nnfv1alpha2.NnfDataMovementSpecSourceDestination{
			Path: "little/red",
		}
		srcpath := &nnfv1alpha2.NnfDataMovementSpecSourceDestination{
			Path: "/dev/null",
		}
		src := &nnfv1alpha2.NnfDataMovement{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
				Labels: map[string]string{
					"label1": "",
				},
			},
			Spec: nnfv1alpha2.NnfDataMovementSpec{
				Destination: destpath,
				Source:      srcpath,
			},
		}

		dst := &unstructured.Unstructured{}
		dst.SetGroupVersionKind(oldNnfDataMovementGVK)
		dst.SetName("test-1")

		g.Expect(MarshalData(src, dst)).To(Succeed())
		// ensure the src object is not modified
		g.Expect(src.GetLabels()).ToNot(BeEmpty())

		g.Expect(dst.GetAnnotations()[DataAnnotation]).ToNot(BeEmpty())

		g.Expect(dst.GetAnnotations()[DataAnnotation]).To(ContainSubstring("little/red"))
		g.Expect(dst.GetAnnotations()[DataAnnotation]).To(ContainSubstring("/dev/null"))
	})

	t.Run("NnfDataMovement should append the annotation", func(*testing.T) {
		src := &nnfv1alpha2.NnfDataMovement{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
			},
		}
		dst := &unstructured.Unstructured{}
		dst.SetGroupVersionKind(nnfv1alpha2.GroupVersion.WithKind("NnfDataMovement"))
		dst.SetName("test-1")
		dst.SetAnnotations(map[string]string{
			"annotation": "1",
		})

		g.Expect(MarshalData(src, dst)).To(Succeed())
		g.Expect(dst.GetAnnotations()).To(HaveLen(2))
	})

	t.Run("NnfDataMovementManager should write source object to destination", func(*testing.T) {
		src := &nnfv1alpha2.NnfDataMovementManager{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
				Labels: map[string]string{
					"label1": "",
				},
			},
			Spec: nnfv1alpha2.NnfDataMovementManagerSpec{
				HostPath:  "/this/dir",
				MountPath: "/mnts",
			},
		}

		dst := &unstructured.Unstructured{}
		dst.SetGroupVersionKind(oldNnfDataMovementManagerGVK)
		dst.SetName("test-1")

		g.Expect(MarshalData(src, dst)).To(Succeed())
		// ensure the src object is not modified
		g.Expect(src.GetLabels()).ToNot(BeEmpty())

		g.Expect(dst.GetAnnotations()[DataAnnotation]).ToNot(BeEmpty())

		g.Expect(dst.GetAnnotations()[DataAnnotation]).To(ContainSubstring("/this/dir"))
		g.Expect(dst.GetAnnotations()[DataAnnotation]).To(ContainSubstring("/mnts"))
	})

	t.Run("NnfDataMovementManager should append the annotation", func(*testing.T) {
		src := &nnfv1alpha2.NnfDataMovementManager{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
			},
		}
		dst := &unstructured.Unstructured{}
		dst.SetGroupVersionKind(nnfv1alpha2.GroupVersion.WithKind("NnfDataMovementManager"))
		dst.SetName("test-1")
		dst.SetAnnotations(map[string]string{
			"annotation": "1",
		})

		g.Expect(MarshalData(src, dst)).To(Succeed())
		g.Expect(dst.GetAnnotations()).To(HaveLen(2))
	})

	t.Run("NnfDataMovementProfile should write source object to destination", func(*testing.T) {
		src := &nnfv1alpha2.NnfDataMovementProfile{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
				Labels: map[string]string{
					"label1": "",
				},
			},
			Data: nnfv1alpha2.NnfDataMovementProfileData{
				Command:     "mpirun is cool",
				StatCommand: "stat --something",
			},
		}

		dst := &unstructured.Unstructured{}
		dst.SetGroupVersionKind(oldNnfDataMovementProfileGVK)
		dst.SetName("test-1")

		g.Expect(MarshalData(src, dst)).To(Succeed())
		// ensure the src object is not modified
		g.Expect(src.GetLabels()).ToNot(BeEmpty())

		g.Expect(dst.GetAnnotations()[DataAnnotation]).ToNot(BeEmpty())

		g.Expect(dst.GetAnnotations()[DataAnnotation]).To(ContainSubstring("mpirun is cool"))
		g.Expect(dst.GetAnnotations()[DataAnnotation]).To(ContainSubstring("stat --something"))
	})

	t.Run("NnfDataMovementProfile should append the annotation", func(*testing.T) {
		src := &nnfv1alpha2.NnfDataMovementProfile{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
			},
		}
		dst := &unstructured.Unstructured{}
		dst.SetGroupVersionKind(nnfv1alpha2.GroupVersion.WithKind("NnfDataMovementProfile"))
		dst.SetName("test-1")
		dst.SetAnnotations(map[string]string{
			"annotation": "1",
		})

		g.Expect(MarshalData(src, dst)).To(Succeed())
		g.Expect(dst.GetAnnotations()).To(HaveLen(2))
	})

	t.Run("NnfLustreMGT should write source object to destination", func(*testing.T) {
		blacklist := []string{"black-fly", "black bird"}
		src := &nnfv1alpha2.NnfLustreMGT{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
				Labels: map[string]string{
					"label1": "",
				},
			},
			Spec: nnfv1alpha2.NnfLustreMGTSpec{
				FsNameStart:     "aaaa-pizza",
				FsNameBlackList: blacklist,
			},
		}

		dst := &unstructured.Unstructured{}
		dst.SetGroupVersionKind(oldNnfLustreMGTGVK)
		dst.SetName("test-1")

		g.Expect(MarshalData(src, dst)).To(Succeed())
		// ensure the src object is not modified
		g.Expect(src.GetLabels()).ToNot(BeEmpty())

		g.Expect(dst.GetAnnotations()[DataAnnotation]).ToNot(BeEmpty())

		g.Expect(dst.GetAnnotations()[DataAnnotation]).To(ContainSubstring("aaaa-pizza"))
		g.Expect(dst.GetAnnotations()[DataAnnotation]).To(ContainSubstring("black-fly"))
		g.Expect(dst.GetAnnotations()[DataAnnotation]).To(ContainSubstring("black bird"))
	})

	t.Run("NnfLustreMGT should append the annotation", func(*testing.T) {
		src := &nnfv1alpha2.NnfLustreMGT{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
			},
		}
		dst := &unstructured.Unstructured{}
		dst.SetGroupVersionKind(nnfv1alpha2.GroupVersion.WithKind("NnfLustreMGT"))
		dst.SetName("test-1")
		dst.SetAnnotations(map[string]string{
			"annotation": "1",
		})

		g.Expect(MarshalData(src, dst)).To(Succeed())
		g.Expect(dst.GetAnnotations()).To(HaveLen(2))
	})

	t.Run("NnfNode should write source object to destination", func(*testing.T) {
		src := &nnfv1alpha2.NnfNode{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
				Labels: map[string]string{
					"label1": "",
				},
			},
			Spec: nnfv1alpha2.NnfNodeSpec{
				Name: "rabbit-1",
				Pod:  "nnf-thingy-122",
			},
		}

		dst := &unstructured.Unstructured{}
		dst.SetGroupVersionKind(oldNnfNodeGVK)
		dst.SetName("test-1")

		g.Expect(MarshalData(src, dst)).To(Succeed())
		// ensure the src object is not modified
		g.Expect(src.GetLabels()).ToNot(BeEmpty())

		g.Expect(dst.GetAnnotations()[DataAnnotation]).ToNot(BeEmpty())

		g.Expect(dst.GetAnnotations()[DataAnnotation]).To(ContainSubstring("rabbit-1"))
		g.Expect(dst.GetAnnotations()[DataAnnotation]).To(ContainSubstring("nnf-thingy-122"))
	})

	t.Run("NnfNode should append the annotation", func(*testing.T) {
		src := &nnfv1alpha2.NnfNode{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
			},
		}
		dst := &unstructured.Unstructured{}
		dst.SetGroupVersionKind(nnfv1alpha2.GroupVersion.WithKind("NnfNode"))
		dst.SetName("test-1")
		dst.SetAnnotations(map[string]string{
			"annotation": "1",
		})

		g.Expect(MarshalData(src, dst)).To(Succeed())
		g.Expect(dst.GetAnnotations()).To(HaveLen(2))
	})

	t.Run("NnfNodeBlockStorage should write source object to destination", func(*testing.T) {
		alloc := []nnfv1alpha2.NnfNodeBlockStorageAllocationSpec{
			{Access: []string{"rabbit-44", "rabbit-10002"}},
		}
		src := &nnfv1alpha2.NnfNodeBlockStorage{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
				Labels: map[string]string{
					"label1": "",
				},
			},
			Spec: nnfv1alpha2.NnfNodeBlockStorageSpec{
				Allocations: alloc,
			},
		}

		dst := &unstructured.Unstructured{}
		dst.SetGroupVersionKind(oldNnfNodeBlockStorageGVK)
		dst.SetName("test-1")

		g.Expect(MarshalData(src, dst)).To(Succeed())
		// ensure the src object is not modified
		g.Expect(src.GetLabels()).ToNot(BeEmpty())

		g.Expect(dst.GetAnnotations()[DataAnnotation]).ToNot(BeEmpty())

		g.Expect(dst.GetAnnotations()[DataAnnotation]).To(ContainSubstring("rabbit-44"))
		g.Expect(dst.GetAnnotations()[DataAnnotation]).To(ContainSubstring("rabbit-10002"))
	})

	t.Run("NnfNodeBlockStorage should append the annotation", func(*testing.T) {
		src := &nnfv1alpha2.NnfNodeBlockStorage{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
			},
		}
		dst := &unstructured.Unstructured{}
		dst.SetGroupVersionKind(nnfv1alpha2.GroupVersion.WithKind("NnfNodeBlockStorage"))
		dst.SetName("test-1")
		dst.SetAnnotations(map[string]string{
			"annotation": "1",
		})

		g.Expect(MarshalData(src, dst)).To(Succeed())
		g.Expect(dst.GetAnnotations()).To(HaveLen(2))
	})

	t.Run("NnfNodeECData should write source object to destination", func(*testing.T) {
		elem1 := nnfv1alpha2.NnfNodeECPrivateData{"element1": "the world"}
		priv := map[string]nnfv1alpha2.NnfNodeECPrivateData{
			"thing1": elem1,
		}
		src := &nnfv1alpha2.NnfNodeECData{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
				Labels: map[string]string{
					"label1": "",
				},
			},
			Status: nnfv1alpha2.NnfNodeECDataStatus{
				Data: priv,
			},
		}

		dst := &unstructured.Unstructured{}
		dst.SetGroupVersionKind(oldNnfNodeECDataGVK)
		dst.SetName("test-1")

		g.Expect(MarshalData(src, dst)).To(Succeed())
		// ensure the src object is not modified
		g.Expect(src.GetLabels()).ToNot(BeEmpty())

		g.Expect(dst.GetAnnotations()[DataAnnotation]).ToNot(BeEmpty())

		g.Expect(dst.GetAnnotations()[DataAnnotation]).To(ContainSubstring("thing1"))
		g.Expect(dst.GetAnnotations()[DataAnnotation]).To(ContainSubstring("element1"))
		g.Expect(dst.GetAnnotations()[DataAnnotation]).To(ContainSubstring("the world"))
	})

	t.Run("NnfNodeECData should append the annotation", func(*testing.T) {
		src := &nnfv1alpha2.NnfNodeECData{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
			},
		}
		dst := &unstructured.Unstructured{}
		dst.SetGroupVersionKind(nnfv1alpha2.GroupVersion.WithKind("NnfNodeECData"))
		dst.SetName("test-1")
		dst.SetAnnotations(map[string]string{
			"annotation": "1",
		})

		g.Expect(MarshalData(src, dst)).To(Succeed())
		g.Expect(dst.GetAnnotations()).To(HaveLen(2))
	})

	t.Run("NnfNodeStorage should write source object to destination", func(*testing.T) {
		src := &nnfv1alpha2.NnfNodeStorage{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
				Labels: map[string]string{
					"label1": "",
				},
			},
			Spec: nnfv1alpha2.NnfNodeStorageSpec{
				UserID:         4997,
				GroupID:        2112,
				FileSystemType: "gfs2",
			},
		}

		dst := &unstructured.Unstructured{}
		dst.SetGroupVersionKind(oldNnfNodeStorageGVK)
		dst.SetName("test-1")

		g.Expect(MarshalData(src, dst)).To(Succeed())
		// ensure the src object is not modified
		g.Expect(src.GetLabels()).ToNot(BeEmpty())

		g.Expect(dst.GetAnnotations()[DataAnnotation]).ToNot(BeEmpty())

		g.Expect(dst.GetAnnotations()[DataAnnotation]).To(ContainSubstring("4997"))
		g.Expect(dst.GetAnnotations()[DataAnnotation]).To(ContainSubstring("2112"))
		g.Expect(dst.GetAnnotations()[DataAnnotation]).To(ContainSubstring("gfs2"))
	})

	t.Run("NnfNodeStorage should append the annotation", func(*testing.T) {
		src := &nnfv1alpha2.NnfNodeStorage{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
			},
		}
		dst := &unstructured.Unstructured{}
		dst.SetGroupVersionKind(nnfv1alpha2.GroupVersion.WithKind("NnfNodeStorage"))
		dst.SetName("test-1")
		dst.SetAnnotations(map[string]string{
			"annotation": "1",
		})

		g.Expect(MarshalData(src, dst)).To(Succeed())
		g.Expect(dst.GetAnnotations()).To(HaveLen(2))
	})

	t.Run("NnfPortManager should write source object to destination", func(*testing.T) {
		src := &nnfv1alpha2.NnfPortManager{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
				Labels: map[string]string{
					"label1": "",
				},
			},
			Spec: nnfv1alpha2.NnfPortManagerSpec{
				SystemConfiguration: corev1.ObjectReference{
					Namespace: "willy-wonka",
					Name:      "candy-land",
				},
			},
		}

		dst := &unstructured.Unstructured{}
		dst.SetGroupVersionKind(oldNnfPortManagerGVK)
		dst.SetName("test-1")

		g.Expect(MarshalData(src, dst)).To(Succeed())
		// ensure the src object is not modified
		g.Expect(src.GetLabels()).ToNot(BeEmpty())

		g.Expect(dst.GetAnnotations()[DataAnnotation]).ToNot(BeEmpty())

		g.Expect(dst.GetAnnotations()[DataAnnotation]).To(ContainSubstring("willy-wonka"))
		g.Expect(dst.GetAnnotations()[DataAnnotation]).To(ContainSubstring("candy-land"))
	})

	t.Run("NnfPortManager should append the annotation", func(*testing.T) {
		src := &nnfv1alpha2.NnfPortManager{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
			},
		}
		dst := &unstructured.Unstructured{}
		dst.SetGroupVersionKind(nnfv1alpha2.GroupVersion.WithKind("NnfPortManager"))
		dst.SetName("test-1")
		dst.SetAnnotations(map[string]string{
			"annotation": "1",
		})

		g.Expect(MarshalData(src, dst)).To(Succeed())
		g.Expect(dst.GetAnnotations()).To(HaveLen(2))
	})

	t.Run("NnfStorage should write source object to destination", func(*testing.T) {
		src := &nnfv1alpha2.NnfStorage{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
				Labels: map[string]string{
					"label1": "",
				},
			},
			Spec: nnfv1alpha2.NnfStorageSpec{
				FileSystemType: "gfs2",
				UserID:         4004,
				GroupID:        2992,
			},
		}

		dst := &unstructured.Unstructured{}
		dst.SetGroupVersionKind(oldNnfStorageGVK)
		dst.SetName("test-1")

		g.Expect(MarshalData(src, dst)).To(Succeed())
		// ensure the src object is not modified
		g.Expect(src.GetLabels()).ToNot(BeEmpty())

		g.Expect(dst.GetAnnotations()[DataAnnotation]).ToNot(BeEmpty())

		g.Expect(dst.GetAnnotations()[DataAnnotation]).To(ContainSubstring("gfs2"))
		g.Expect(dst.GetAnnotations()[DataAnnotation]).To(ContainSubstring("4004"))
		g.Expect(dst.GetAnnotations()[DataAnnotation]).To(ContainSubstring("2992"))
	})

	t.Run("NnfStorage should append the annotation", func(*testing.T) {
		src := &nnfv1alpha2.NnfStorage{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
			},
		}
		dst := &unstructured.Unstructured{}
		dst.SetGroupVersionKind(nnfv1alpha2.GroupVersion.WithKind("NnfStorage"))
		dst.SetName("test-1")
		dst.SetAnnotations(map[string]string{
			"annotation": "1",
		})

		g.Expect(MarshalData(src, dst)).To(Succeed())
		g.Expect(dst.GetAnnotations()).To(HaveLen(2))
	})

	t.Run("NnfStorageProfile should write source object to destination", func(*testing.T) {
		src := &nnfv1alpha2.NnfStorageProfile{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
				Labels: map[string]string{
					"label1": "",
				},
			},
			Data: nnfv1alpha2.NnfStorageProfileData{
				LustreStorage: nnfv1alpha2.NnfStorageProfileLustreData{
					ExternalMGS: "kfi@1:this@that",
				},
			},
		}

		dst := &unstructured.Unstructured{}
		dst.SetGroupVersionKind(oldNnfStorageProfileGVK)
		dst.SetName("test-1")

		g.Expect(MarshalData(src, dst)).To(Succeed())
		// ensure the src object is not modified
		g.Expect(src.GetLabels()).ToNot(BeEmpty())

		g.Expect(dst.GetAnnotations()[DataAnnotation]).ToNot(BeEmpty())

		g.Expect(dst.GetAnnotations()[DataAnnotation]).To(ContainSubstring("kfi@1:this@that"))
	})

	t.Run("NnfStorageProfile should append the annotation", func(*testing.T) {
		src := &nnfv1alpha2.NnfStorageProfile{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
			},
		}
		dst := &unstructured.Unstructured{}
		dst.SetGroupVersionKind(nnfv1alpha2.GroupVersion.WithKind("NnfStorageProfile"))
		dst.SetName("test-1")
		dst.SetAnnotations(map[string]string{
			"annotation": "1",
		})

		g.Expect(MarshalData(src, dst)).To(Succeed())
		g.Expect(dst.GetAnnotations()).To(HaveLen(2))
	})

	t.Run("NnfSystemStorage should write source object to destination", func(*testing.T) {
		src := &nnfv1alpha2.NnfSystemStorage{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
				Labels: map[string]string{
					"label1": "",
				},
			},
			Spec: nnfv1alpha2.NnfSystemStorageSpec{
				ClientMountPath: "/on/this",
			},
		}

		dst := &unstructured.Unstructured{}
		dst.SetGroupVersionKind(oldNnfSystemStorageGVK)
		dst.SetName("test-1")

		g.Expect(MarshalData(src, dst)).To(Succeed())
		// ensure the src object is not modified
		g.Expect(src.GetLabels()).ToNot(BeEmpty())

		g.Expect(dst.GetAnnotations()[DataAnnotation]).ToNot(BeEmpty())

		g.Expect(dst.GetAnnotations()[DataAnnotation]).To(ContainSubstring("/on/this"))
	})

	t.Run("NnfSystemStorage should append the annotation", func(*testing.T) {
		src := &nnfv1alpha2.NnfSystemStorage{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
			},
		}
		dst := &unstructured.Unstructured{}
		dst.SetGroupVersionKind(nnfv1alpha2.GroupVersion.WithKind("NnfSystemStorage"))
		dst.SetName("test-1")
		dst.SetAnnotations(map[string]string{
			"annotation": "1",
		})

		g.Expect(MarshalData(src, dst)).To(Succeed())
		g.Expect(dst.GetAnnotations()).To(HaveLen(2))
	})

	// +crdbumper:scaffold:marshaldata
}

func TestUnmarshalData(t *testing.T) {
	g := NewWithT(t)

	t.Run("NnfAccess should return false without errors if annotation doesn't exist", func(*testing.T) {
		src := &nnfv1alpha2.NnfAccess{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
			},
		}
		dst := &unstructured.Unstructured{}
		dst.SetGroupVersionKind(oldNnfAccessGVK)
		dst.SetName("test-1")

		ok, err := UnmarshalData(src, dst)
		g.Expect(ok).To(BeFalse())
		g.Expect(err).ToNot(HaveOccurred())
	})

	t.Run("NnfAccess should return true when a valid annotation with data exists", func(*testing.T) {
		src := &unstructured.Unstructured{}
		src.SetGroupVersionKind(oldNnfAccessGVK)
		src.SetName("test-1")
		src.SetAnnotations(map[string]string{
			DataAnnotation: "{\"metadata\":{\"name\":\"test-1\",\"creationTimestamp\":null,\"labels\":{\"label1\":\"\"}},\"spec\":{},\"status\":{}}",
		})

		dst := &nnfv1alpha2.NnfAccess{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
			},
		}

		ok, err := UnmarshalData(src, dst)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(ok).To(BeTrue())

		g.Expect(dst.GetLabels()).To(HaveLen(1))
		g.Expect(dst.GetName()).To(Equal("test-1"))
		g.Expect(dst.GetLabels()).To(HaveKeyWithValue("label1", ""))
		g.Expect(dst.GetAnnotations()).To(BeEmpty())
	})

	t.Run("NnfAccess should clean the annotation on successful unmarshal", func(*testing.T) {
		src := &unstructured.Unstructured{}
		src.SetGroupVersionKind(oldNnfAccessGVK)
		src.SetName("test-1")
		src.SetAnnotations(map[string]string{
			"annotation-1": "",
			DataAnnotation: "{\"metadata\":{\"name\":\"test-1\",\"creationTimestamp\":null,\"labels\":{\"label1\":\"\"}},\"spec\":{},\"status\":{}}",
		})

		dst := &nnfv1alpha2.NnfAccess{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
			},
		}

		ok, err := UnmarshalData(src, dst)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(ok).To(BeTrue())

		g.Expect(src.GetAnnotations()).ToNot(HaveKey(DataAnnotation))
		g.Expect(src.GetAnnotations()).To(HaveLen(1))
	})

	t.Run("NnfContainerProfile should return false without errors if annotation doesn't exist", func(*testing.T) {
		src := &nnfv1alpha2.NnfContainerProfile{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
			},
		}
		dst := &unstructured.Unstructured{}
		dst.SetGroupVersionKind(oldNnfContainerProfileGVK)
		dst.SetName("test-1")

		ok, err := UnmarshalData(src, dst)
		g.Expect(ok).To(BeFalse())
		g.Expect(err).ToNot(HaveOccurred())
	})

	t.Run("NnfContainerProfile should return true when a valid annotation with data exists", func(*testing.T) {
		src := &unstructured.Unstructured{}
		src.SetGroupVersionKind(oldNnfContainerProfileGVK)
		src.SetName("test-1")
		src.SetAnnotations(map[string]string{
			DataAnnotation: "{\"metadata\":{\"name\":\"test-1\",\"creationTimestamp\":null,\"labels\":{\"label1\":\"\"}},\"spec\":{},\"status\":{}}",
		})

		dst := &nnfv1alpha2.NnfContainerProfile{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
			},
		}

		ok, err := UnmarshalData(src, dst)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(ok).To(BeTrue())

		g.Expect(dst.GetLabels()).To(HaveLen(1))
		g.Expect(dst.GetName()).To(Equal("test-1"))
		g.Expect(dst.GetLabels()).To(HaveKeyWithValue("label1", ""))
		g.Expect(dst.GetAnnotations()).To(BeEmpty())
	})

	t.Run("NnfContainerProfile should clean the annotation on successful unmarshal", func(*testing.T) {
		src := &unstructured.Unstructured{}
		src.SetGroupVersionKind(oldNnfContainerProfileGVK)
		src.SetName("test-1")
		src.SetAnnotations(map[string]string{
			"annotation-1": "",
			DataAnnotation: "{\"metadata\":{\"name\":\"test-1\",\"creationTimestamp\":null,\"labels\":{\"label1\":\"\"}},\"spec\":{},\"status\":{}}",
		})

		dst := &nnfv1alpha2.NnfContainerProfile{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
			},
		}

		ok, err := UnmarshalData(src, dst)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(ok).To(BeTrue())

		g.Expect(src.GetAnnotations()).ToNot(HaveKey(DataAnnotation))
		g.Expect(src.GetAnnotations()).To(HaveLen(1))
	})

	t.Run("NnfDataMovement should return false without errors if annotation doesn't exist", func(*testing.T) {
		src := &nnfv1alpha2.NnfDataMovement{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
			},
		}
		dst := &unstructured.Unstructured{}
		dst.SetGroupVersionKind(oldNnfDataMovementGVK)
		dst.SetName("test-1")

		ok, err := UnmarshalData(src, dst)
		g.Expect(ok).To(BeFalse())
		g.Expect(err).ToNot(HaveOccurred())
	})

	t.Run("NnfDataMovement should return true when a valid annotation with data exists", func(*testing.T) {
		src := &unstructured.Unstructured{}
		src.SetGroupVersionKind(oldNnfDataMovementGVK)
		src.SetName("test-1")
		src.SetAnnotations(map[string]string{
			DataAnnotation: "{\"metadata\":{\"name\":\"test-1\",\"creationTimestamp\":null,\"labels\":{\"label1\":\"\"}},\"spec\":{},\"status\":{}}",
		})

		dst := &nnfv1alpha2.NnfDataMovement{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
			},
		}

		ok, err := UnmarshalData(src, dst)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(ok).To(BeTrue())

		g.Expect(dst.GetLabels()).To(HaveLen(1))
		g.Expect(dst.GetName()).To(Equal("test-1"))
		g.Expect(dst.GetLabels()).To(HaveKeyWithValue("label1", ""))
		g.Expect(dst.GetAnnotations()).To(BeEmpty())
	})

	t.Run("NnfDataMovement should clean the annotation on successful unmarshal", func(*testing.T) {
		src := &unstructured.Unstructured{}
		src.SetGroupVersionKind(oldNnfDataMovementGVK)
		src.SetName("test-1")
		src.SetAnnotations(map[string]string{
			"annotation-1": "",
			DataAnnotation: "{\"metadata\":{\"name\":\"test-1\",\"creationTimestamp\":null,\"labels\":{\"label1\":\"\"}},\"spec\":{},\"status\":{}}",
		})

		dst := &nnfv1alpha2.NnfDataMovement{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
			},
		}

		ok, err := UnmarshalData(src, dst)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(ok).To(BeTrue())

		g.Expect(src.GetAnnotations()).ToNot(HaveKey(DataAnnotation))
		g.Expect(src.GetAnnotations()).To(HaveLen(1))
	})

	t.Run("NnfDataMovementManager should return false without errors if annotation doesn't exist", func(*testing.T) {
		src := &nnfv1alpha2.NnfDataMovementManager{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
			},
		}
		dst := &unstructured.Unstructured{}
		dst.SetGroupVersionKind(oldNnfDataMovementManagerGVK)
		dst.SetName("test-1")

		ok, err := UnmarshalData(src, dst)
		g.Expect(ok).To(BeFalse())
		g.Expect(err).ToNot(HaveOccurred())
	})

	t.Run("NnfDataMovementManager should return true when a valid annotation with data exists", func(*testing.T) {
		src := &unstructured.Unstructured{}
		src.SetGroupVersionKind(oldNnfDataMovementManagerGVK)
		src.SetName("test-1")
		src.SetAnnotations(map[string]string{
			DataAnnotation: "{\"metadata\":{\"name\":\"test-1\",\"creationTimestamp\":null,\"labels\":{\"label1\":\"\"}},\"spec\":{},\"status\":{}}",
		})

		dst := &nnfv1alpha2.NnfDataMovementManager{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
			},
		}

		ok, err := UnmarshalData(src, dst)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(ok).To(BeTrue())

		g.Expect(dst.GetLabels()).To(HaveLen(1))
		g.Expect(dst.GetName()).To(Equal("test-1"))
		g.Expect(dst.GetLabels()).To(HaveKeyWithValue("label1", ""))
		g.Expect(dst.GetAnnotations()).To(BeEmpty())
	})

	t.Run("NnfDataMovementManager should clean the annotation on successful unmarshal", func(*testing.T) {
		src := &unstructured.Unstructured{}
		src.SetGroupVersionKind(oldNnfDataMovementManagerGVK)
		src.SetName("test-1")
		src.SetAnnotations(map[string]string{
			"annotation-1": "",
			DataAnnotation: "{\"metadata\":{\"name\":\"test-1\",\"creationTimestamp\":null,\"labels\":{\"label1\":\"\"}},\"spec\":{},\"status\":{}}",
		})

		dst := &nnfv1alpha2.NnfDataMovementManager{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
			},
		}

		ok, err := UnmarshalData(src, dst)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(ok).To(BeTrue())

		g.Expect(src.GetAnnotations()).ToNot(HaveKey(DataAnnotation))
		g.Expect(src.GetAnnotations()).To(HaveLen(1))
	})

	t.Run("NnfDataMovementProfile should return false without errors if annotation doesn't exist", func(*testing.T) {
		src := &nnfv1alpha2.NnfDataMovementProfile{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
			},
		}
		dst := &unstructured.Unstructured{}
		dst.SetGroupVersionKind(oldNnfDataMovementProfileGVK)
		dst.SetName("test-1")

		ok, err := UnmarshalData(src, dst)
		g.Expect(ok).To(BeFalse())
		g.Expect(err).ToNot(HaveOccurred())
	})

	t.Run("NnfDataMovementProfile should return true when a valid annotation with data exists", func(*testing.T) {
		src := &unstructured.Unstructured{}
		src.SetGroupVersionKind(oldNnfDataMovementProfileGVK)
		src.SetName("test-1")
		src.SetAnnotations(map[string]string{
			DataAnnotation: "{\"metadata\":{\"name\":\"test-1\",\"creationTimestamp\":null,\"labels\":{\"label1\":\"\"}},\"spec\":{},\"status\":{}}",
		})

		dst := &nnfv1alpha2.NnfDataMovementProfile{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
			},
		}

		ok, err := UnmarshalData(src, dst)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(ok).To(BeTrue())

		g.Expect(dst.GetLabels()).To(HaveLen(1))
		g.Expect(dst.GetName()).To(Equal("test-1"))
		g.Expect(dst.GetLabels()).To(HaveKeyWithValue("label1", ""))
		g.Expect(dst.GetAnnotations()).To(BeEmpty())
	})

	t.Run("NnfDataMovementProfile should clean the annotation on successful unmarshal", func(*testing.T) {
		src := &unstructured.Unstructured{}
		src.SetGroupVersionKind(oldNnfDataMovementProfileGVK)
		src.SetName("test-1")
		src.SetAnnotations(map[string]string{
			"annotation-1": "",
			DataAnnotation: "{\"metadata\":{\"name\":\"test-1\",\"creationTimestamp\":null,\"labels\":{\"label1\":\"\"}},\"spec\":{},\"status\":{}}",
		})

		dst := &nnfv1alpha2.NnfDataMovementProfile{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
			},
		}

		ok, err := UnmarshalData(src, dst)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(ok).To(BeTrue())

		g.Expect(src.GetAnnotations()).ToNot(HaveKey(DataAnnotation))
		g.Expect(src.GetAnnotations()).To(HaveLen(1))
	})

	t.Run("NnfLustreMGT should return false without errors if annotation doesn't exist", func(*testing.T) {
		src := &nnfv1alpha2.NnfLustreMGT{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
			},
		}
		dst := &unstructured.Unstructured{}
		dst.SetGroupVersionKind(oldNnfLustreMGTGVK)
		dst.SetName("test-1")

		ok, err := UnmarshalData(src, dst)
		g.Expect(ok).To(BeFalse())
		g.Expect(err).ToNot(HaveOccurred())
	})

	t.Run("NnfLustreMGT should return true when a valid annotation with data exists", func(*testing.T) {
		src := &unstructured.Unstructured{}
		src.SetGroupVersionKind(oldNnfLustreMGTGVK)
		src.SetName("test-1")
		src.SetAnnotations(map[string]string{
			DataAnnotation: "{\"metadata\":{\"name\":\"test-1\",\"creationTimestamp\":null,\"labels\":{\"label1\":\"\"}},\"spec\":{},\"status\":{}}",
		})

		dst := &nnfv1alpha2.NnfLustreMGT{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
			},
		}

		ok, err := UnmarshalData(src, dst)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(ok).To(BeTrue())

		g.Expect(dst.GetLabels()).To(HaveLen(1))
		g.Expect(dst.GetName()).To(Equal("test-1"))
		g.Expect(dst.GetLabels()).To(HaveKeyWithValue("label1", ""))
		g.Expect(dst.GetAnnotations()).To(BeEmpty())
	})

	t.Run("NnfLustreMGT should clean the annotation on successful unmarshal", func(*testing.T) {
		src := &unstructured.Unstructured{}
		src.SetGroupVersionKind(oldNnfLustreMGTGVK)
		src.SetName("test-1")
		src.SetAnnotations(map[string]string{
			"annotation-1": "",
			DataAnnotation: "{\"metadata\":{\"name\":\"test-1\",\"creationTimestamp\":null,\"labels\":{\"label1\":\"\"}},\"spec\":{},\"status\":{}}",
		})

		dst := &nnfv1alpha2.NnfLustreMGT{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
			},
		}

		ok, err := UnmarshalData(src, dst)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(ok).To(BeTrue())

		g.Expect(src.GetAnnotations()).ToNot(HaveKey(DataAnnotation))
		g.Expect(src.GetAnnotations()).To(HaveLen(1))
	})

	t.Run("NnfNode should return false without errors if annotation doesn't exist", func(*testing.T) {
		src := &nnfv1alpha2.NnfNode{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
			},
		}
		dst := &unstructured.Unstructured{}
		dst.SetGroupVersionKind(oldNnfNodeGVK)
		dst.SetName("test-1")

		ok, err := UnmarshalData(src, dst)
		g.Expect(ok).To(BeFalse())
		g.Expect(err).ToNot(HaveOccurred())
	})

	t.Run("NnfNode should return true when a valid annotation with data exists", func(*testing.T) {
		src := &unstructured.Unstructured{}
		src.SetGroupVersionKind(oldNnfNodeGVK)
		src.SetName("test-1")
		src.SetAnnotations(map[string]string{
			DataAnnotation: "{\"metadata\":{\"name\":\"test-1\",\"creationTimestamp\":null,\"labels\":{\"label1\":\"\"}},\"spec\":{},\"status\":{}}",
		})

		dst := &nnfv1alpha2.NnfNode{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
			},
		}

		ok, err := UnmarshalData(src, dst)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(ok).To(BeTrue())

		g.Expect(dst.GetLabels()).To(HaveLen(1))
		g.Expect(dst.GetName()).To(Equal("test-1"))
		g.Expect(dst.GetLabels()).To(HaveKeyWithValue("label1", ""))
		g.Expect(dst.GetAnnotations()).To(BeEmpty())
	})

	t.Run("NnfNode should clean the annotation on successful unmarshal", func(*testing.T) {
		src := &unstructured.Unstructured{}
		src.SetGroupVersionKind(oldNnfNodeGVK)
		src.SetName("test-1")
		src.SetAnnotations(map[string]string{
			"annotation-1": "",
			DataAnnotation: "{\"metadata\":{\"name\":\"test-1\",\"creationTimestamp\":null,\"labels\":{\"label1\":\"\"}},\"spec\":{},\"status\":{}}",
		})

		dst := &nnfv1alpha2.NnfNode{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
			},
		}

		ok, err := UnmarshalData(src, dst)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(ok).To(BeTrue())

		g.Expect(src.GetAnnotations()).ToNot(HaveKey(DataAnnotation))
		g.Expect(src.GetAnnotations()).To(HaveLen(1))
	})

	t.Run("NnfNodeBlockStorage should return false without errors if annotation doesn't exist", func(*testing.T) {
		src := &nnfv1alpha2.NnfNodeBlockStorage{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
			},
		}
		dst := &unstructured.Unstructured{}
		dst.SetGroupVersionKind(oldNnfNodeBlockStorageGVK)
		dst.SetName("test-1")

		ok, err := UnmarshalData(src, dst)
		g.Expect(ok).To(BeFalse())
		g.Expect(err).ToNot(HaveOccurred())
	})

	t.Run("NnfNodeBlockStorage should return true when a valid annotation with data exists", func(*testing.T) {
		src := &unstructured.Unstructured{}
		src.SetGroupVersionKind(oldNnfNodeBlockStorageGVK)
		src.SetName("test-1")
		src.SetAnnotations(map[string]string{
			DataAnnotation: "{\"metadata\":{\"name\":\"test-1\",\"creationTimestamp\":null,\"labels\":{\"label1\":\"\"}},\"spec\":{},\"status\":{}}",
		})

		dst := &nnfv1alpha2.NnfNodeBlockStorage{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
			},
		}

		ok, err := UnmarshalData(src, dst)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(ok).To(BeTrue())

		g.Expect(dst.GetLabels()).To(HaveLen(1))
		g.Expect(dst.GetName()).To(Equal("test-1"))
		g.Expect(dst.GetLabels()).To(HaveKeyWithValue("label1", ""))
		g.Expect(dst.GetAnnotations()).To(BeEmpty())
	})

	t.Run("NnfNodeBlockStorage should clean the annotation on successful unmarshal", func(*testing.T) {
		src := &unstructured.Unstructured{}
		src.SetGroupVersionKind(oldNnfNodeBlockStorageGVK)
		src.SetName("test-1")
		src.SetAnnotations(map[string]string{
			"annotation-1": "",
			DataAnnotation: "{\"metadata\":{\"name\":\"test-1\",\"creationTimestamp\":null,\"labels\":{\"label1\":\"\"}},\"spec\":{},\"status\":{}}",
		})

		dst := &nnfv1alpha2.NnfNodeBlockStorage{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
			},
		}

		ok, err := UnmarshalData(src, dst)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(ok).To(BeTrue())

		g.Expect(src.GetAnnotations()).ToNot(HaveKey(DataAnnotation))
		g.Expect(src.GetAnnotations()).To(HaveLen(1))
	})

	t.Run("NnfNodeECData should return false without errors if annotation doesn't exist", func(*testing.T) {
		src := &nnfv1alpha2.NnfNodeECData{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
			},
		}
		dst := &unstructured.Unstructured{}
		dst.SetGroupVersionKind(oldNnfNodeECDataGVK)
		dst.SetName("test-1")

		ok, err := UnmarshalData(src, dst)
		g.Expect(ok).To(BeFalse())
		g.Expect(err).ToNot(HaveOccurred())
	})

	t.Run("NnfNodeECData should return true when a valid annotation with data exists", func(*testing.T) {
		src := &unstructured.Unstructured{}
		src.SetGroupVersionKind(oldNnfNodeECDataGVK)
		src.SetName("test-1")
		src.SetAnnotations(map[string]string{
			DataAnnotation: "{\"metadata\":{\"name\":\"test-1\",\"creationTimestamp\":null,\"labels\":{\"label1\":\"\"}},\"spec\":{},\"status\":{}}",
		})

		dst := &nnfv1alpha2.NnfNodeECData{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
			},
		}

		ok, err := UnmarshalData(src, dst)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(ok).To(BeTrue())

		g.Expect(dst.GetLabels()).To(HaveLen(1))
		g.Expect(dst.GetName()).To(Equal("test-1"))
		g.Expect(dst.GetLabels()).To(HaveKeyWithValue("label1", ""))
		g.Expect(dst.GetAnnotations()).To(BeEmpty())
	})

	t.Run("NnfNodeECData should clean the annotation on successful unmarshal", func(*testing.T) {
		src := &unstructured.Unstructured{}
		src.SetGroupVersionKind(oldNnfNodeECDataGVK)
		src.SetName("test-1")
		src.SetAnnotations(map[string]string{
			"annotation-1": "",
			DataAnnotation: "{\"metadata\":{\"name\":\"test-1\",\"creationTimestamp\":null,\"labels\":{\"label1\":\"\"}},\"spec\":{},\"status\":{}}",
		})

		dst := &nnfv1alpha2.NnfNodeECData{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
			},
		}

		ok, err := UnmarshalData(src, dst)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(ok).To(BeTrue())

		g.Expect(src.GetAnnotations()).ToNot(HaveKey(DataAnnotation))
		g.Expect(src.GetAnnotations()).To(HaveLen(1))
	})

	t.Run("NnfNodeStorage should return false without errors if annotation doesn't exist", func(*testing.T) {
		src := &nnfv1alpha2.NnfNodeStorage{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
			},
		}
		dst := &unstructured.Unstructured{}
		dst.SetGroupVersionKind(oldNnfNodeStorageGVK)
		dst.SetName("test-1")

		ok, err := UnmarshalData(src, dst)
		g.Expect(ok).To(BeFalse())
		g.Expect(err).ToNot(HaveOccurred())
	})

	t.Run("NnfNodeStorage should return true when a valid annotation with data exists", func(*testing.T) {
		src := &unstructured.Unstructured{}
		src.SetGroupVersionKind(oldNnfNodeStorageGVK)
		src.SetName("test-1")
		src.SetAnnotations(map[string]string{
			DataAnnotation: "{\"metadata\":{\"name\":\"test-1\",\"creationTimestamp\":null,\"labels\":{\"label1\":\"\"}},\"spec\":{},\"status\":{}}",
		})

		dst := &nnfv1alpha2.NnfNodeStorage{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
			},
		}

		ok, err := UnmarshalData(src, dst)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(ok).To(BeTrue())

		g.Expect(dst.GetLabels()).To(HaveLen(1))
		g.Expect(dst.GetName()).To(Equal("test-1"))
		g.Expect(dst.GetLabels()).To(HaveKeyWithValue("label1", ""))
		g.Expect(dst.GetAnnotations()).To(BeEmpty())
	})

	t.Run("NnfNodeStorage should clean the annotation on successful unmarshal", func(*testing.T) {
		src := &unstructured.Unstructured{}
		src.SetGroupVersionKind(oldNnfNodeStorageGVK)
		src.SetName("test-1")
		src.SetAnnotations(map[string]string{
			"annotation-1": "",
			DataAnnotation: "{\"metadata\":{\"name\":\"test-1\",\"creationTimestamp\":null,\"labels\":{\"label1\":\"\"}},\"spec\":{},\"status\":{}}",
		})

		dst := &nnfv1alpha2.NnfNodeStorage{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
			},
		}

		ok, err := UnmarshalData(src, dst)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(ok).To(BeTrue())

		g.Expect(src.GetAnnotations()).ToNot(HaveKey(DataAnnotation))
		g.Expect(src.GetAnnotations()).To(HaveLen(1))
	})

	t.Run("NnfPortManager should return false without errors if annotation doesn't exist", func(*testing.T) {
		src := &nnfv1alpha2.NnfPortManager{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
			},
		}
		dst := &unstructured.Unstructured{}
		dst.SetGroupVersionKind(oldNnfPortManagerGVK)
		dst.SetName("test-1")

		ok, err := UnmarshalData(src, dst)
		g.Expect(ok).To(BeFalse())
		g.Expect(err).ToNot(HaveOccurred())
	})

	t.Run("NnfPortManager should return true when a valid annotation with data exists", func(*testing.T) {
		src := &unstructured.Unstructured{}
		src.SetGroupVersionKind(oldNnfPortManagerGVK)
		src.SetName("test-1")
		src.SetAnnotations(map[string]string{
			DataAnnotation: "{\"metadata\":{\"name\":\"test-1\",\"creationTimestamp\":null,\"labels\":{\"label1\":\"\"}},\"spec\":{},\"status\":{}}",
		})

		dst := &nnfv1alpha2.NnfPortManager{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
			},
		}

		ok, err := UnmarshalData(src, dst)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(ok).To(BeTrue())

		g.Expect(dst.GetLabels()).To(HaveLen(1))
		g.Expect(dst.GetName()).To(Equal("test-1"))
		g.Expect(dst.GetLabels()).To(HaveKeyWithValue("label1", ""))
		g.Expect(dst.GetAnnotations()).To(BeEmpty())
	})

	t.Run("NnfPortManager should clean the annotation on successful unmarshal", func(*testing.T) {
		src := &unstructured.Unstructured{}
		src.SetGroupVersionKind(oldNnfPortManagerGVK)
		src.SetName("test-1")
		src.SetAnnotations(map[string]string{
			"annotation-1": "",
			DataAnnotation: "{\"metadata\":{\"name\":\"test-1\",\"creationTimestamp\":null,\"labels\":{\"label1\":\"\"}},\"spec\":{},\"status\":{}}",
		})

		dst := &nnfv1alpha2.NnfPortManager{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
			},
		}

		ok, err := UnmarshalData(src, dst)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(ok).To(BeTrue())

		g.Expect(src.GetAnnotations()).ToNot(HaveKey(DataAnnotation))
		g.Expect(src.GetAnnotations()).To(HaveLen(1))
	})

	t.Run("NnfStorage should return false without errors if annotation doesn't exist", func(*testing.T) {
		src := &nnfv1alpha2.NnfStorage{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
			},
		}
		dst := &unstructured.Unstructured{}
		dst.SetGroupVersionKind(oldNnfStorageGVK)
		dst.SetName("test-1")

		ok, err := UnmarshalData(src, dst)
		g.Expect(ok).To(BeFalse())
		g.Expect(err).ToNot(HaveOccurred())
	})

	t.Run("NnfStorage should return true when a valid annotation with data exists", func(*testing.T) {
		src := &unstructured.Unstructured{}
		src.SetGroupVersionKind(oldNnfStorageGVK)
		src.SetName("test-1")
		src.SetAnnotations(map[string]string{
			DataAnnotation: "{\"metadata\":{\"name\":\"test-1\",\"creationTimestamp\":null,\"labels\":{\"label1\":\"\"}},\"spec\":{},\"status\":{}}",
		})

		dst := &nnfv1alpha2.NnfStorage{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
			},
		}

		ok, err := UnmarshalData(src, dst)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(ok).To(BeTrue())

		g.Expect(dst.GetLabels()).To(HaveLen(1))
		g.Expect(dst.GetName()).To(Equal("test-1"))
		g.Expect(dst.GetLabels()).To(HaveKeyWithValue("label1", ""))
		g.Expect(dst.GetAnnotations()).To(BeEmpty())
	})

	t.Run("NnfStorage should clean the annotation on successful unmarshal", func(*testing.T) {
		src := &unstructured.Unstructured{}
		src.SetGroupVersionKind(oldNnfStorageGVK)
		src.SetName("test-1")
		src.SetAnnotations(map[string]string{
			"annotation-1": "",
			DataAnnotation: "{\"metadata\":{\"name\":\"test-1\",\"creationTimestamp\":null,\"labels\":{\"label1\":\"\"}},\"spec\":{},\"status\":{}}",
		})

		dst := &nnfv1alpha2.NnfStorage{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
			},
		}

		ok, err := UnmarshalData(src, dst)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(ok).To(BeTrue())

		g.Expect(src.GetAnnotations()).ToNot(HaveKey(DataAnnotation))
		g.Expect(src.GetAnnotations()).To(HaveLen(1))
	})

	t.Run("NnfStorageProfile should return false without errors if annotation doesn't exist", func(*testing.T) {
		src := &nnfv1alpha2.NnfStorageProfile{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
			},
		}
		dst := &unstructured.Unstructured{}
		dst.SetGroupVersionKind(oldNnfStorageProfileGVK)
		dst.SetName("test-1")

		ok, err := UnmarshalData(src, dst)
		g.Expect(ok).To(BeFalse())
		g.Expect(err).ToNot(HaveOccurred())
	})

	t.Run("NnfStorageProfile should return true when a valid annotation with data exists", func(*testing.T) {
		src := &unstructured.Unstructured{}
		src.SetGroupVersionKind(oldNnfStorageProfileGVK)
		src.SetName("test-1")
		src.SetAnnotations(map[string]string{
			DataAnnotation: "{\"metadata\":{\"name\":\"test-1\",\"creationTimestamp\":null,\"labels\":{\"label1\":\"\"}},\"spec\":{},\"status\":{}}",
		})

		dst := &nnfv1alpha2.NnfStorageProfile{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
			},
		}

		ok, err := UnmarshalData(src, dst)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(ok).To(BeTrue())

		g.Expect(dst.GetLabels()).To(HaveLen(1))
		g.Expect(dst.GetName()).To(Equal("test-1"))
		g.Expect(dst.GetLabels()).To(HaveKeyWithValue("label1", ""))
		g.Expect(dst.GetAnnotations()).To(BeEmpty())
	})

	t.Run("NnfStorageProfile should clean the annotation on successful unmarshal", func(*testing.T) {
		src := &unstructured.Unstructured{}
		src.SetGroupVersionKind(oldNnfStorageProfileGVK)
		src.SetName("test-1")
		src.SetAnnotations(map[string]string{
			"annotation-1": "",
			DataAnnotation: "{\"metadata\":{\"name\":\"test-1\",\"creationTimestamp\":null,\"labels\":{\"label1\":\"\"}},\"spec\":{},\"status\":{}}",
		})

		dst := &nnfv1alpha2.NnfStorageProfile{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
			},
		}

		ok, err := UnmarshalData(src, dst)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(ok).To(BeTrue())

		g.Expect(src.GetAnnotations()).ToNot(HaveKey(DataAnnotation))
		g.Expect(src.GetAnnotations()).To(HaveLen(1))
	})

	t.Run("NnfSystemStorage should return false without errors if annotation doesn't exist", func(*testing.T) {
		src := &nnfv1alpha2.NnfSystemStorage{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
			},
		}
		dst := &unstructured.Unstructured{}
		dst.SetGroupVersionKind(oldNnfSystemStorageGVK)
		dst.SetName("test-1")

		ok, err := UnmarshalData(src, dst)
		g.Expect(ok).To(BeFalse())
		g.Expect(err).ToNot(HaveOccurred())
	})

	t.Run("NnfSystemStorage should return true when a valid annotation with data exists", func(*testing.T) {
		src := &unstructured.Unstructured{}
		src.SetGroupVersionKind(oldNnfSystemStorageGVK)
		src.SetName("test-1")
		src.SetAnnotations(map[string]string{
			DataAnnotation: "{\"metadata\":{\"name\":\"test-1\",\"creationTimestamp\":null,\"labels\":{\"label1\":\"\"}},\"spec\":{},\"status\":{}}",
		})

		dst := &nnfv1alpha2.NnfSystemStorage{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
			},
		}

		ok, err := UnmarshalData(src, dst)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(ok).To(BeTrue())

		g.Expect(dst.GetLabels()).To(HaveLen(1))
		g.Expect(dst.GetName()).To(Equal("test-1"))
		g.Expect(dst.GetLabels()).To(HaveKeyWithValue("label1", ""))
		g.Expect(dst.GetAnnotations()).To(BeEmpty())
	})

	t.Run("NnfSystemStorage should clean the annotation on successful unmarshal", func(*testing.T) {
		src := &unstructured.Unstructured{}
		src.SetGroupVersionKind(oldNnfSystemStorageGVK)
		src.SetName("test-1")
		src.SetAnnotations(map[string]string{
			"annotation-1": "",
			DataAnnotation: "{\"metadata\":{\"name\":\"test-1\",\"creationTimestamp\":null,\"labels\":{\"label1\":\"\"}},\"spec\":{},\"status\":{}}",
		})

		dst := &nnfv1alpha2.NnfSystemStorage{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
			},
		}

		ok, err := UnmarshalData(src, dst)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(ok).To(BeTrue())

		g.Expect(src.GetAnnotations()).ToNot(HaveKey(DataAnnotation))
		g.Expect(src.GetAnnotations()).To(HaveLen(1))
	})

	// +crdbumper:scaffold:unmarshaldata
}

// Just touch ginkgo, so it's here to interpret any ginkgo args from
// "make test", so that doesn't fail on this test file.
var _ = BeforeSuite(func() {})
