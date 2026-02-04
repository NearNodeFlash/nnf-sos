/*
 * Copyright 2026 Hewlett Packard Enterprise Development LP
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

import (
	"bufio"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
)

// AcknowledgeDLMFence acknowledges that a node has been fenced in DLM.
// This is necessary to unblock any pending DLM lock operations that are
// waiting for fencing confirmation. Without this, operations like
// `vgchange --lock-start` and `lvchange --activate ys` will hang.
//
// The nodeName should be the short hostname of the fenced node (e.g., "rabbit-compute-3").
// This function looks up the corosync node ID for the given hostname and calls
// `dlm_tool fence_ack <nodeid>` to acknowledge the fence.
//
// This function handles all errors internally and logs them appropriately.
// It never fails because DLM fence acknowledgment is best-effort.
func AcknowledgeDLMFence(nodeName string, log logr.Logger) {
	// Get the corosync node ID for the given hostname
	nodeID, err := getCorosyncNodeID(nodeName)
	if err != nil {
		log.V(1).Info("Could not get corosync node ID, skipping DLM fence acknowledgment",
			"nodeName", nodeName, "error", err)
		// Not a fatal error - DLM might not be configured or the node might not be in the cluster
		return
	}

	if nodeID == 0 {
		log.V(1).Info("Node not found in corosync configuration, skipping DLM fence acknowledgment",
			"nodeName", nodeName)
		return
	}

	// Acknowledge the fence in DLM
	log.Info("Acknowledging DLM fence", "nodeName", nodeName, "nodeID", nodeID)
	cmd := exec.Command("dlm_tool", "fence_ack", strconv.Itoa(nodeID))
	output, err := cmd.CombinedOutput()
	if err != nil {
		// dlm_tool fence_ack can fail if the node was never in a DLM lockspace
		// or if DLM isn't running. This is not fatal.
		log.V(1).Info("dlm_tool fence_ack returned error (may be expected)",
			"nodeName", nodeName, "nodeID", nodeID, "output", string(output), "error", err)
		return
	}

	log.Info("Successfully acknowledged DLM fence", "nodeName", nodeName, "nodeID", nodeID)
}

// getCorosyncNodeID looks up the corosync node ID for a given hostname.
// Returns 0 if the node is not found in the corosync configuration.
func getCorosyncNodeID(nodeName string) (int, error) {
	// Use corosync-cmapctl to get the nodelist
	cmd := exec.Command("corosync-cmapctl", "-g", "nodelist")
	output, err := cmd.Output()
	if err != nil {
		// Corosync might not be running or not configured
		return 0, fmt.Errorf("corosync-cmapctl failed: %w", err)
	}

	// Parse the output looking for nodelist entries
	// Format is like:
	// nodelist.node.0.name (str) = rabbit-node-1
	// nodelist.node.0.nodeid (u32) = 1
	// nodelist.node.1.name (str) = rabbit-compute-2
	// nodelist.node.1.nodeid (u32) = 2
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	nodeNames := make(map[int]string) // index -> name
	nodeIDs := make(map[int]int)      // index -> nodeid

	for scanner.Scan() {
		line := scanner.Text()

		// Parse name entries
		if strings.Contains(line, ".name") && strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				name := strings.TrimSpace(parts[1])
				// Extract the index from the key (e.g., "nodelist.node.0.name")
				key := strings.TrimSpace(parts[0])
				keyParts := strings.Split(key, ".")
				if len(keyParts) >= 3 {
					if idx, err := strconv.Atoi(keyParts[2]); err == nil {
						nodeNames[idx] = name
					}
				}
			}
		}

		// Parse nodeid entries
		if strings.Contains(line, ".nodeid") && strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				idStr := strings.TrimSpace(parts[1])
				// Extract the index from the key (e.g., "nodelist.node.0.nodeid")
				key := strings.TrimSpace(parts[0])
				keyParts := strings.Split(key, ".")
				if len(keyParts) >= 3 {
					if idx, err := strconv.Atoi(keyParts[2]); err == nil {
						if id, err := strconv.Atoi(idStr); err == nil {
							nodeIDs[idx] = id
						}
					}
				}
			}
		}
	}

	// Find the node ID for the requested hostname
	for idx, name := range nodeNames {
		// Match either the full name or short hostname
		shortName := strings.Split(name, ".")[0]
		targetShort := strings.Split(nodeName, ".")[0]
		if name == nodeName || shortName == targetShort {
			if id, ok := nodeIDs[idx]; ok {
				return id, nil
			}
		}
	}

	return 0, nil
}

// ConfirmPacemakerFence confirms a node as fenced in Pacemaker.
// This removes the UNCLEAN status from the node, allowing cluster operations
// like `pcs cluster stop` to proceed without waiting.
func ConfirmPacemakerFence(nodeName string, log logr.Logger) error {
	log.Info("Confirming Pacemaker fence", "nodeName", nodeName)

	// Use echo 'y' to auto-confirm the fence operation
	cmd := exec.Command("bash", "-c", fmt.Sprintf("echo 'y' | pcs stonith confirm %s", nodeName))
	output, err := cmd.CombinedOutput()
	if err != nil {
		// pcs stonith confirm can fail if the node wasn't UNCLEAN
		// or if Pacemaker isn't running. Log but don't return error.
		log.V(1).Info("pcs stonith confirm returned error (may be expected)",
			"nodeName", nodeName, "output", string(output), "error", err)
		return nil
	}

	log.Info("Successfully confirmed Pacemaker fence", "nodeName", nodeName)
	return nil
}
