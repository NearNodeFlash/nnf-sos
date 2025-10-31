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
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/NearNodeFlash/nnf-sos/internal/controller/metrics"
)

// FenceRequest represents the structure of a fence request file written by the fence agent
type FenceRequest struct {
	RequestID    string `json:"request_id"`
	Timestamp    string `json:"timestamp"`
	Action       string `json:"action"`
	TargetNode   string `json:"target_node"`
	RecorderNode string `json:"recorder_node"`
	FilePath     string `json:"-"` // Not from JSON, added when parsing
}

// GFS2FenceWatcher watches for new files in a GFS2 fencing directory
// and triggers reconciliation of controllers
type GFS2FenceWatcher struct {
	log             logr.Logger
	watcher         *fsnotify.Watcher
	reconcileChan   chan reconcile.Request
	nodeNamespace   string
	nodeName        string
	watchDir        string                     // Directory to watch (request or response)
	pendingRequests map[string][]*FenceRequest // key: targetNode, value: pending fence requests
	requestsMutex   sync.Mutex
}

// NewGFS2FenceWatcher creates a new file system watcher for GFS2 fence files
// watchDir specifies the directory to monitor (e.g., fence.RequestDir or fence.ResponseDir)
func NewGFS2FenceWatcher(log logr.Logger, nodeNamespace, nodeName, watchDir string) (*GFS2FenceWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &GFS2FenceWatcher{
		log:             log,
		watcher:         watcher,
		reconcileChan:   make(chan reconcile.Request, 100),
		nodeNamespace:   nodeNamespace,
		nodeName:        nodeName,
		watchDir:        watchDir,
		pendingRequests: make(map[string][]*FenceRequest),
	}, nil
}

// Start begins watching the GFS2 fence directory
func (w *GFS2FenceWatcher) Start(ctx context.Context) error {
	log := w.log.WithValues("watchDir", w.watchDir)

	// Create the directory if it doesn't exist
	if err := os.MkdirAll(w.watchDir, 0755); err != nil {
		log.Error(err, "Failed to create GFS2 fence directory")
		return err
	}

	// Add the directory to the watcher
	if err := w.watcher.Add(w.watchDir); err != nil {
		log.Error(err, "Failed to add directory to watcher")
		return err
	}

	log.Info("Started watching for GFS2 fence files")

	// Start the event processing goroutine
	go w.processEvents(ctx)

	return nil
}

// processEvents handles file system events
func (w *GFS2FenceWatcher) processEvents(ctx context.Context) {
	log := w.log.WithValues("watchDir", w.watchDir)

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

			// Process Write events for new fence requests/responses
			if event.Op&fsnotify.Write == fsnotify.Write {
				filename := filepath.Base(event.Name)
				log.Info("Fence request file written (ready to process)", "file", event.Name, "filename", filename)

				// Brief delay to ensure file is fully flushed to disk
				time.Sleep(50 * time.Millisecond)

				// Increment the metric counter
				metrics.Gfs2FenceRequestsTotal.Inc()

				// Parse the fence request file to get the target node
				fenceRequest, err := w.ParseFenceRequest(event.Name)
				if err != nil {
					log.Error(err, "Failed to parse fence request file", "file", event.Name)
					// Skip this request if we can't determine the target node
					continue
				}

				log.Info("Triggering fence action for compute node", "targetNode", fenceRequest.TargetNode)

				// Store the parsed fence request in the pending requests map
				w.requestsMutex.Lock()
				if w.pendingRequests[fenceRequest.TargetNode] == nil {
					w.pendingRequests[fenceRequest.TargetNode] = []*FenceRequest{}
				}
				w.pendingRequests[fenceRequest.TargetNode] = append(w.pendingRequests[fenceRequest.TargetNode], fenceRequest)
				w.requestsMutex.Unlock()

				// Trigger reconciliation for the target node's NnfNodeStorage.
				// The rabbit maintains NnfNodeStorage resources for storage it provides to compute nodes.
				// The reconciler will find GFS2 filesystems shared with this compute node,
				// locate the associated storage groups, and delete them to fence the node.
				w.reconcileChan <- reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      fenceRequest.TargetNode,
						Namespace: w.nodeNamespace,
					},
				}
			} else if event.Op&fsnotify.Remove == fsnotify.Remove {
				// Process Remove events for fence file deletion (unfencing)
				// For NnfNode controller watching response directory, this triggers status update
				filename := filepath.Base(event.Name)
				log.Info("Fence response file removed (unfencing)", "file", event.Name, "filename", filename)

				// Trigger reconciliation of the NnfNode to update status
				// The node name and namespace should match the reconciler's node
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

// GetPendingRequests retrieves and removes pending fence requests for a target node
func (w *GFS2FenceWatcher) GetPendingRequests(targetNode string) []*FenceRequest {
	w.requestsMutex.Lock()
	defer w.requestsMutex.Unlock()

	requests := w.pendingRequests[targetNode]
	delete(w.pendingRequests, targetNode)
	return requests
}

// ParseFenceRequest reads and parses a fence request JSON file
func (w *GFS2FenceWatcher) ParseFenceRequest(filePath string) (*FenceRequest, error) {
	log := w.log.WithValues("file", filePath)

	// Read the file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// Parse JSON
	var request FenceRequest
	if err := json.Unmarshal(data, &request); err != nil {
		return nil, err
	}

	// Store the file path for later cleanup
	request.FilePath = filePath

	log.Info("Parsed fence request",
		"requestID", request.RequestID,
		"targetNode", request.TargetNode,
		"action", request.Action,
		"recorderNode", request.RecorderNode)

	return &request, nil
}

// Stop gracefully stops the watcher
func (w *GFS2FenceWatcher) Stop() error {
	return w.watcher.Close()
}
