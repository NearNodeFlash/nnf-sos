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
	"os"
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

	// Acknowledge the fence in DLM.
	// Use nsenter to enter the host network namespace because dlm_controld
	// listens on an abstract Unix socket which is tied to the network namespace.
	// The container runs in its own network namespace and cannot reach dlm_controld directly.
	log.Info("Acknowledging DLM fence", "nodeName", nodeName, "nodeID", nodeID)
	cmd := exec.Command("nsenter", "-t", "1", "-n", "--", "dlm_tool", "fence_ack", strconv.Itoa(nodeID))
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

// getCorosyncNodeID looks up the corosync node ID for a given hostname by
// parsing /etc/corosync/corosync.conf.
//
// We cannot use corosync-cmapctl because corosync listens on an abstract Unix
// socket (visible in the host network namespace only). The nnf-node-manager
// container runs in its own network namespace and cannot reach it.
//
// Returns 0 if the node is not found in the corosync configuration.
func getCorosyncNodeID(nodeName string) (int, error) {
	const corosyncConf = "/etc/corosync/corosync.conf"

	data, err := os.ReadFile(corosyncConf)
	if err != nil {
		return 0, fmt.Errorf("failed to read %s: %w", corosyncConf, err)
	}

	// Parse the corosync.conf nodelist section.
	// Format:
	//   nodelist {
	//       node {
	//           ring0_addr: 10.1.1.5
	//           name: rabbit-node-1
	//           nodeid: 1
	//           quorum_votes: 3
	//       }
	//       node {
	//           name: rabbit-compute-2
	//           nodeid: 2
	//       }
	//   }
	scanner := bufio.NewScanner(strings.NewReader(string(data)))

	targetShort := strings.Split(nodeName, ".")[0]

	inNodelist := false
	inNode := false
	braceDepth := 0
	currentName := ""
	currentID := 0

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Track entry into the nodelist block
		if !inNodelist && strings.HasPrefix(line, "nodelist") && strings.Contains(line, "{") {
			inNodelist = true
			braceDepth = 1
			continue
		}

		if !inNodelist {
			continue
		}

		// Track braces
		if strings.Contains(line, "{") {
			braceDepth++
			if braceDepth == 2 {
				// Entering a node block
				inNode = true
				currentName = ""
				currentID = 0
			}
			continue
		}
		if strings.Contains(line, "}") {
			if inNode && braceDepth == 2 {
				// Leaving a node block — check if this was our target
				shortName := strings.Split(currentName, ".")[0]
				if currentID != 0 && (currentName == nodeName || shortName == targetShort) {
					return currentID, nil
				}
				inNode = false
			}
			braceDepth--
			if braceDepth <= 0 {
				// Left the nodelist block
				break
			}
			continue
		}

		if !inNode {
			continue
		}

		// Parse key: value lines inside a node block
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		switch key {
		case "name":
			currentName = val
		case "nodeid":
			if id, err := strconv.Atoi(val); err == nil {
				currentID = id
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
