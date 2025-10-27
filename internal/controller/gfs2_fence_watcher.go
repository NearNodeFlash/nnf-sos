/*
 * Copyright 2021-2025 Hewlett Packard Enterprise Development LP
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
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/NearNodeFlash/nnf-sos/internal/controller/metrics"
)

const (
	gfs2FenceRequestDir = "/localdisk/gfs2-fencing/requests"
)

// GFS2FenceWatcher watches for new files in the GFS2 fencing request directory
// and triggers reconciliation of the NnfNodeStorage controller
type GFS2FenceWatcher struct {
	log           logr.Logger
	watcher       *fsnotify.Watcher
	reconcileChan chan reconcile.Request
	nodeNamespace string
	nodeName      string
}

// NewGFS2FenceWatcher creates a new file system watcher for GFS2 fence requests
func NewGFS2FenceWatcher(log logr.Logger, nodeNamespace, nodeName string) (*GFS2FenceWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &GFS2FenceWatcher{
		log:           log,
		watcher:       watcher,
		reconcileChan: make(chan reconcile.Request, 100),
		nodeNamespace: nodeNamespace,
		nodeName:      nodeName,
	}, nil
}

// Start begins watching the GFS2 fence request directory
func (w *GFS2FenceWatcher) Start(ctx context.Context) error {
	log := w.log.WithValues("watchDir", gfs2FenceRequestDir)

	// Create the directory if it doesn't exist
	if err := os.MkdirAll(gfs2FenceRequestDir, 0755); err != nil {
		log.Error(err, "Failed to create GFS2 fence request directory")
		return err
	}

	// Add the directory to the watcher
	if err := w.watcher.Add(gfs2FenceRequestDir); err != nil {
		log.Error(err, "Failed to add directory to watcher")
		return err
	}

	log.Info("Started watching for GFS2 fence requests")

	// Start the event processing goroutine
	go w.processEvents(ctx)

	return nil
}

// processEvents handles file system events
func (w *GFS2FenceWatcher) processEvents(ctx context.Context) {
	log := w.log.WithValues("watchDir", gfs2FenceRequestDir)

	for {
		select {
		case <-ctx.Done():
			log.Info("Stopping GFS2 fence watcher")
			w.watcher.Close()
			return

		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}

			// Only process file creation events
			if event.Op&fsnotify.Create == fsnotify.Create {
				filename := filepath.Base(event.Name)
				log.Info("New fence request file detected", "file", event.Name, "filename", filename)

				// Increment the metric counter
				metrics.Gfs2FenceRequestsTotal.Inc()

				// Trigger reconciliation - you can parse the filename to determine
				// which NnfNodeStorage resource to reconcile
				w.reconcileChan <- reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      w.nodeName,
						Namespace: w.nodeNamespace,
					},
				}
			}

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			log.Error(err, "File watcher error")
		}
	}
}

// GetReconcileChannel returns the channel that receives reconcile requests
func (w *GFS2FenceWatcher) GetReconcileChannel() <-chan reconcile.Request {
	return w.reconcileChan
}

// Stop gracefully stops the watcher
func (w *GFS2FenceWatcher) Stop() error {
	return w.watcher.Close()
}
