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

package v1alpha1

import (
	"context"
	"reflect"

	"golang.org/x/sync/errgroup"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	OwnerKindLabel      = "dws.cray.hpe.com/owner.kind"
	OwnerNameLabel      = "dws.cray.hpe.com/owner.name"
	OwnerNamespaceLabel = "dws.cray.hpe.com/owner.namespace"
)

type DeleteStatus int

const (
	DeleteRetry DeleteStatus = iota
	DeleteComplete
)

//+kubebuilder:object:generate=false
type ObjectList interface {
	GetObjectList() []client.Object
}

// AddOwnerLabels adds labels to a child resource that identifies the owner
func AddOwnerLabels(child metav1.Object, owner metav1.Object) {
	labels := child.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}

	labels[OwnerKindLabel] = reflect.Indirect(reflect.ValueOf(owner)).Type().Name()
	labels[OwnerNameLabel] = owner.GetName()
	labels[OwnerNamespaceLabel] = owner.GetNamespace()

	child.SetLabels(labels)
}

// MatchingOwner returns the MatchingLabels to match the owner labels
func MatchingOwner(owner metav1.Object) client.MatchingLabels {
	return client.MatchingLabels(map[string]string{
		OwnerKindLabel:      reflect.Indirect(reflect.ValueOf(owner)).Type().Name(),
		OwnerNameLabel:      owner.GetName(),
		OwnerNamespaceLabel: owner.GetNamespace(),
	})
}

// AddWorkflowLabels adds labels to a resource to indicate which workflow it belongs to
func AddWorkflowLabels(child metav1.Object, workflow *Workflow) {
	labels := child.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}

	labels[WorkflowNameLabel] = workflow.Name
	labels[WorkflowNamespaceLabel] = workflow.Namespace

	child.SetLabels(labels)
}

// MatchingWorkflow returns the MatchingLabels to match the workflow labels
func MatchingWorkflow(workflow *Workflow) client.MatchingLabels {
	return client.MatchingLabels(map[string]string{
		WorkflowNameLabel:      workflow.Name,
		WorkflowNamespaceLabel: workflow.Namespace,
	})
}

// AddPersistentStorageLabels adds labels to a resource to indicate which persistent storage instance it belongs to
func AddPersistentStorageLabels(child metav1.Object, persistentStorage *PersistentStorageInstance) {
	labels := child.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}

	labels[PersistentStorageNameLabel] = persistentStorage.Name
	labels[PersistentStorageNamespaceLabel] = persistentStorage.Namespace

	child.SetLabels(labels)
}

// MatchingPersistentStorage returns the MatchingLabels to match the persistent storage labels
func MatchingPersistentStorage(persistentStorage *PersistentStorageInstance) client.MatchingLabels {
	return client.MatchingLabels(map[string]string{
		PersistentStorageNameLabel:      persistentStorage.Name,
		PersistentStorageNamespaceLabel: persistentStorage.Namespace,
	})
}

// InheritParentLabels adds all labels from a parent resource to a child resource, excluding
// the owner labels
func InheritParentLabels(child metav1.Object, owner metav1.Object) {
	labels := child.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}

	for key, value := range owner.GetLabels() {
		// don't inherit owner labels
		if key == OwnerNameLabel || key == OwnerNamespaceLabel || key == OwnerKindLabel {
			continue
		}

		labels[key] = value
	}

	child.SetLabels(labels)
}

// deleteChildrenSingle deletes all the children of a single type. Children are found using
// the owner labels
func deleteChildrenSingle(ctx context.Context, c client.Client, childObjectList ObjectList, parent metav1.Object) (DeleteStatus, error) {
	// List all the children and filter by the owner labels
	err := c.List(ctx, childObjectList.(client.ObjectList), MatchingOwner(parent))
	if err != nil {
		return DeleteRetry, err
	}

	objectList := childObjectList.GetObjectList()
	if len(objectList) == 0 {
		return DeleteComplete, nil
	}

	// Check whether the child objects span multiple namespaces.
	multipleNamespaces := false
	namespace := ""
	for _, obj := range objectList {
		if obj.GetNamespace() != namespace && namespace != "" {
			multipleNamespaces = true
		}

		namespace = obj.GetNamespace()

		// Wait for any deletes to finish if the resource is already
		// marked for deletion
		if !obj.GetDeletionTimestamp().IsZero() {
			return DeleteRetry, nil
		}
	}

	// If all the child resources are in a single namespace then we can use DeleteAllOf
	if !multipleNamespaces {
		err = c.DeleteAllOf(ctx, objectList[0], client.InNamespace(namespace), MatchingOwner(parent))
		if err != nil {
			return DeleteRetry, err
		}

		return DeleteRetry, nil
	}

	// If the child resources span multiple namespaces, then we have to delete them
	// each individually.
	g := new(errgroup.Group)
	for _, obj := range objectList {
		obj := obj

		// Start a goroutine for each object to delete
		g.Go(func() error {
			return c.Delete(ctx, obj)
		})
	}

	return DeleteRetry, g.Wait()
}

// DeleteChildren deletes all the children of a parent with the resource types defined
// in a list of ObjectList types. All children of a single type will be fully deleted
// before starting to delete any children of the next type.
func DeleteChildren(ctx context.Context, c client.Client, childObjectLists []ObjectList, parent metav1.Object) (DeleteStatus, error) {
	for _, childObjectList := range childObjectLists {
		deleteStatus, err := deleteChildrenSingle(ctx, c, childObjectList, parent)
		if err != nil {
			return DeleteRetry, err
		}

		if deleteStatus == DeleteRetry {
			return DeleteRetry, nil
		}
	}

	return DeleteComplete, nil
}

func OwnerLabelMapFunc(o client.Object) []reconcile.Request {
	labels := o.GetLabels()

	ownerName, exists := labels[OwnerNameLabel]
	if exists == false {
		return []reconcile.Request{}
	}

	ownerNamespace, exists := labels[OwnerNamespaceLabel]
	if exists == false {
		return []reconcile.Request{}
	}

	return []reconcile.Request{
		{NamespacedName: types.NamespacedName{
			Name:      ownerName,
			Namespace: ownerNamespace,
		}},
	}
}
