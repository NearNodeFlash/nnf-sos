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

package fence

import "time"

// FenceRequest represents a fencing request written by a fence agent
type FenceRequest struct {
	RequestID    string    `json:"request_id"`
	Timestamp    string    `json:"timestamp"`
	Action       string    `json:"action"`
	TargetNode   string    `json:"target_node"`
	RecorderNode string    `json:"recorder_node"`
	FilePath     string    `json:"-"` // Not from JSON, added when parsing
	ReceivedAt   time.Time `json:"-"` // When we received this request, for stale cleanup
}

// FenceResponse represents the response written by nnf-sos after processing a fence request
type FenceResponse struct {
	RequestID       string `json:"request_id"`
	Status          string `json:"status,omitempty"` // "success" or "error"
	Success         bool   `json:"success"`
	Message         string `json:"message"`
	Timestamp       string `json:"timestamp"` // ISO 8601 timestamp
	TargetNode      string `json:"target_node"`
	Action          string `json:"action,omitempty"`
	RecorderNode    string `json:"recorder_node,omitempty"`
	ActionPerformed string `json:"action_performed,omitempty"`
}
