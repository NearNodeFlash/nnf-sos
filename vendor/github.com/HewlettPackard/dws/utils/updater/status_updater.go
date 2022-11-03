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

package updater

import (
	"context"
	"reflect"

	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Status provides an interface for copying the status T
type Status[T any] interface {
	DeepCopy() T
}

type resource[T any] interface {
	client.Object
	GetStatus() Status[T]
}

type statusUpdater[T any] struct {
	resource resource[T]
	status   T
}

// NewStatusUpdater returns a status updater meant for updating the status of the
// supplied resource when the updater is closed. Typically users will want to
// create a status updater early on in a controller's Reconcile() method and add a
// deferred method to close the updater when returning from reconcile.
//
// i.e.
//
//	func (r *myController) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
//		rsrc := &MyResource{}
//		if err := r.Get(ctx, req.NamespacedName, rsrc); err != nil {
//			return err
//		}
//
//		updater := NewStatusUpdater[*MyResourceStatus](rsrc)
//		defer func() {
//			if err == nil {
//				err = updater.Close(ctx, r)
//			}
//		}()
//
//		...
func NewStatusUpdater[S Status[S]](rsrc resource[S]) *statusUpdater[S] {
	return &statusUpdater[S]{
		resource: rsrc,
		status:   rsrc.GetStatus().DeepCopy(),
	}
}

// CloseWithUpdate will attempt to update the resource if any of the status fields have
// changed from the initially recorded status. CloseWithUpdate will NOT return an error
// if there is a resource conflict on this version of the resource. The reconciler will
// already have an event queued for the new version of the resource.
func (updater *statusUpdater[S]) CloseWithUpdate(ctx context.Context, c client.Writer, err error) error {
	return updater.close(ctx, c, err)
}

// CloseWithStatusUpdate will attempt to update the resource's status if any of the status
// fields have changed from the initially recorded status. CloseWithStatusUpdate will NOT
// return an error if there is a resource conflict on this version of the resource. The
// reconciler will already have an event queued for the new version of the resource.
func (updater *statusUpdater[S]) CloseWithStatusUpdate(ctx context.Context, c client.StatusClient, err error) error {
	return updater.close(ctx, c.Status(), err)
}

type clientUpdater interface {
	Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error
}

func (updater *statusUpdater[S]) close(ctx context.Context, c clientUpdater, err error) error {
	if !reflect.DeepEqual(updater.resource.GetStatus(), updater.status) {

		// Always attempt an update to the resource even in the presence of different error, but
		// do not override the original error if present.
		updateError := c.Update(ctx, updater.resource)

		if err == nil {
			// Do not return an error if there is a resource conflict on this version of the resource.
			// The reconciler will already have an event queued for the new version of the resource.
			if !errors.IsConflict(updateError) {
				return updateError
			}
		}

		return err

	}

	return err
}
