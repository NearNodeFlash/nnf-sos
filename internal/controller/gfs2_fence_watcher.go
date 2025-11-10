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

	"github.com/NearNodeFlash/nnf-sos/internal/controller/metrics"
)

// FenceRequest represents the structure of a fence request file written by the fence agent
type FenceRequest struct {
	RequestID    string    `json:"request_id"`
	Timestamp    string    `json:"timestamp"`
	Action       string    `json:"action"`
	TargetNode   string    `json:"target_node"`
	RecorderNode string    `json:"recorder_node"`
	FilePath     string    `json:"-"` // Not from JSON, added when parsing
	ReceivedAt   time.Time `json:"-"` // When we received this request, for stale cleanup
}

// FenceEvent represents a fence event detected by the watcher
type FenceEvent struct {
	TargetNode   string
	IsUnfence    bool // true if this is an unfence event (file removal)
	FenceRequest *FenceRequest
}

// GFS2FenceWatcher watches for new files in a GFS2 fencing directory
// and emits events when fence requests/responses are detected
type GFS2FenceWatcher struct {
	log             logr.Logger
	watcher         *fsnotify.Watcher
	eventChan       chan FenceEvent
	watchDir        string                     // Directory to watch (request or response)
	pendingRequests map[string][]*FenceRequest // key: targetNode, value: pending fence requests
	requestsMutex   sync.Mutex
}

const (
	// pendingRequestTimeout is how long we keep pending requests before cleaning them up
	pendingRequestTimeout = 5 * time.Minute
)

// NewGFS2FenceWatcher creates a new file system watcher for GFS2 fence files
// watchDir specifies the directory to monitor (e.g., fence.RequestDir or fence.ResponseDir)
func NewGFS2FenceWatcher(log logr.Logger, watchDir string) (*GFS2FenceWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &GFS2FenceWatcher{
		log:             log,
		watcher:         watcher,
		eventChan:       make(chan FenceEvent, 100),
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

	// Start the cleanup goroutine to remove stale pending requests
	go w.cleanupStaleRequests(ctx)

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
				log.Info("Fence file written (ready to process)", "file", event.Name, "filename", filename)

				// Brief delay to ensure file is fully flushed to disk
				time.Sleep(50 * time.Millisecond)

				// Increment the metric counter
				metrics.Gfs2FenceRequestsTotal.Inc()

				// Parse the fence request file to get the target node
				fenceRequest, err := w.ParseFenceRequest(event.Name)
				if err != nil {
					log.Error(err, "Failed to parse fence file", "file", event.Name)
					// Skip this request if we can't parse it
					continue
				}

				log.Info("Detected fence event for compute node", "targetNode", fenceRequest.TargetNode)

				// Store the parsed fence request in the pending requests map with current time
				fenceRequest.ReceivedAt = time.Now()
				w.requestsMutex.Lock()
				if w.pendingRequests[fenceRequest.TargetNode] == nil {
					w.pendingRequests[fenceRequest.TargetNode] = []*FenceRequest{}
				}
				w.pendingRequests[fenceRequest.TargetNode] = append(w.pendingRequests[fenceRequest.TargetNode], fenceRequest)
				w.requestsMutex.Unlock()

				// Emit fence event
				w.eventChan <- FenceEvent{
					TargetNode:   fenceRequest.TargetNode,
					IsUnfence:    false,
					FenceRequest: fenceRequest,
				}
			} else if event.Op&fsnotify.Remove == fsnotify.Remove {
				// Process Remove events for fence file deletion (unfencing)
				filename := filepath.Base(event.Name)
				log.Info("Fence file removed (unfencing)", "file", event.Name, "filename", filename)

				// Try to parse the filename to extract target node
				// The filename should be a UUID with .json extension
				// We'll try to read any cached info, but emit event anyway
				w.eventChan <- FenceEvent{
					TargetNode: "", // Unknown target node for removal events
					IsUnfence:  true,
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

// GetEventChannel returns the channel that receives fence events
func (w *GFS2FenceWatcher) GetEventChannel() <-chan FenceEvent {
	return w.eventChan
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

// cleanupStaleRequests periodically removes old pending requests that were never processed
func (w *GFS2FenceWatcher) cleanupStaleRequests(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.requestsMutex.Lock()
			now := time.Now()
			for targetNode, requests := range w.pendingRequests {
				// Filter out stale requests
				fresh := make([]*FenceRequest, 0, len(requests))
				staleCount := 0
				for _, req := range requests {
					if now.Sub(req.ReceivedAt) < pendingRequestTimeout {
						fresh = append(fresh, req)
					} else {
						staleCount++
					}
				}

				if staleCount > 0 {
					w.log.Info("Cleaned up stale fence requests",
						"targetNode", targetNode,
						"staleCount", staleCount,
						"remainingCount", len(fresh))
				}

				if len(fresh) > 0 {
					w.pendingRequests[targetNode] = fresh
				} else {
					delete(w.pendingRequests, targetNode)
				}
			}
			w.requestsMutex.Unlock()
		}
	}
}

// Stop gracefully stops the watcher
func (w *GFS2FenceWatcher) Stop() error {
	return w.watcher.Close()
}
