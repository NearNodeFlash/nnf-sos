/*
 * Copyright 2022-2023 Hewlett Packard Enterprise Development LP
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

package v1alpha2

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
	OwnerKindLabel      = "dataworkflowservices.github.io/owner.kind"
	OwnerNameLabel      = "dataworkflowservices.github.io/owner.name"
	OwnerNamespaceLabel = "dataworkflowservices.github.io/owner.namespace"
)

// +kubebuilder:object:generate=false
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

func RemoveOwnerLabels(child metav1.Object) {
	labels := child.GetLabels()
	if labels == nil {
		return
	}

	delete(labels, OwnerKindLabel)
	delete(labels, OwnerNameLabel)
	delete(labels, OwnerNamespaceLabel)

	child.SetLabels(labels)
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

// DeleteStatus provides information about the status of DeleteChildren* operation
// +kubebuilder:object:generate=false
// +k8s:conversion-gen=false
type DeleteStatus struct {
	complete bool
	objects  []client.Object
}

// Complete returns true if the delete is complete, and false otherwise
func (d *DeleteStatus) Complete() bool { return d.complete }

// Info returns key/value pairs that describe the delete status operation; the returned array
// must alternate string keys and arbitrary values so it can be passed to logr.Logging.Info()
func (d *DeleteStatus) Info() []interface{} {
	args := make([]interface{}, 0)
	args = append(args, "complete", d.complete)

	if len(d.objects) >= 1 {
		args = append(args, "object", client.ObjectKeyFromObject(d.objects[0]).String())
	}
	if len(d.objects) > 1 {
		args = append(args, "count", len(d.objects))
	}

	return args
}

var deleteRetry = DeleteStatus{complete: false}
var deleteComplete = DeleteStatus{complete: true}

func (d DeleteStatus) withObject(obj client.Object) DeleteStatus {
	d.objects = []client.Object{obj}
	return d
}

func (d DeleteStatus) withObjectList(objs []client.Object) DeleteStatus {
	d.objects = objs
	return d
}

// deleteChildrenSingle deletes all the children of a single type. Children are found using
// the owner labels
func deleteChildrenSingle(ctx context.Context, c client.Client, childObjectList ObjectList, parent metav1.Object, matchingLabels client.MatchingLabels) (DeleteStatus, error) {
	// List all the children and filter by the owner labels
	err := c.List(ctx, childObjectList.(client.ObjectList), matchingLabels)
	if err != nil {
		return deleteRetry, err
	}

	objectList := childObjectList.GetObjectList()
	if len(objectList) == 0 {
		return deleteComplete, nil
	}

	// Check whether the child objects span multiple namespaces.
	multipleNamespaces := false
	namespace := ""
	for _, obj := range objectList {
		if obj.GetNamespace() != namespace && namespace != "" {
			multipleNamespaces = true
		}

		namespace = obj.GetNamespace()

		// Wait for any deletes to finish if the resource is already marked for deletion
		if !obj.GetDeletionTimestamp().IsZero() {
			return deleteRetry.withObject(obj), nil
		}
	}

	// If all the child resources are in a single namespace then we can use DeleteAllOf
	if !multipleNamespaces {
		err = c.DeleteAllOf(ctx, objectList[0], client.InNamespace(namespace), matchingLabels)
		if err != nil {
			return deleteRetry, err
		}

		return deleteRetry.withObjectList(objectList), nil
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

	return deleteRetry.withObjectList(objectList), g.Wait()
}

// DeleteChildrenWithLabels deletes all the children of a parent with the resource types defined
// in a list of ObjectList types and the labels defined in matchingLabels. All children of a
// single type will be fully deleted before starting to delete any children of the next type.
func DeleteChildrenWithLabels(ctx context.Context, c client.Client, childObjectLists []ObjectList, parent metav1.Object, matchingLabels client.MatchingLabels) (DeleteStatus, error) {
	for label, value := range MatchingOwner(parent) {
		matchingLabels[label] = value
	}

	for _, childObjectList := range childObjectLists {
		deleteStatus, err := deleteChildrenSingle(ctx, c, childObjectList, parent, matchingLabels)
		if err != nil {
			return deleteRetry, err
		}

		if !deleteStatus.Complete() {
			return deleteStatus, nil
		}
	}

	return deleteComplete, nil
}

// DeleteChildren deletes all the children of a parent with the resource types defined
// in a list of ObjectList types. All children of a single type will be fully deleted
// before starting to delete any children of the next type.
func DeleteChildren(ctx context.Context, c client.Client, childObjectLists []ObjectList, parent metav1.Object) (DeleteStatus, error) {
	return DeleteChildrenWithLabels(ctx, c, childObjectLists, parent, client.MatchingLabels(map[string]string{}))
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
