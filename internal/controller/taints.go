/*
 * Copyright 2022-2026 Hewlett Packard Enterprise Development LP
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

// Taint helper functions extracted from k8s.io/kubernetes/pkg/util/taints to
// eliminate the heavyweight k8s.io/kubernetes dependency from go.mod.
// Only the three functions actually used by this controller are included.

import (
	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
)

// taintExists checks if the given taint exists in a list of taints.
func taintExists(taints []corev1.Taint, taintToFind *corev1.Taint) bool {
	for _, taint := range taints {
		if taint.MatchTaint(taintToFind) {
			return true
		}
	}
	return false
}

// deleteTaint removes all taints that have the same key and effect as taintToDelete.
func deleteTaint(taints []corev1.Taint, taintToDelete *corev1.Taint) ([]corev1.Taint, bool) {
	var newTaints []corev1.Taint
	deleted := false
	for i := range taints {
		if taintToDelete.MatchTaint(&taints[i]) {
			deleted = true
			continue
		}
		newTaints = append(newTaints, taints[i])
	}
	return newTaints, deleted
}

// removeTaint tries to remove a taint from a Node. Returns a new copy of
// the updated Node and true if something was updated, false otherwise.
func removeTaint(node *corev1.Node, taint *corev1.Taint) (*corev1.Node, bool, error) {
	newNode := node.DeepCopy()
	nodeTaints := newNode.Spec.Taints
	if len(nodeTaints) == 0 {
		return newNode, false, nil
	}

	if !taintExists(nodeTaints, taint) {
		return newNode, false, nil
	}

	newTaints, _ := deleteTaint(nodeTaints, taint)
	newNode.Spec.Taints = newTaints
	return newNode, true, nil
}

// addOrUpdateTaint tries to add a taint to a Node. Returns a new copy of
// the updated Node and true if something was updated, false otherwise.
func addOrUpdateTaint(node *corev1.Node, taint *corev1.Taint) (*corev1.Node, bool, error) {
	newNode := node.DeepCopy()
	nodeTaints := newNode.Spec.Taints

	var newTaints []corev1.Taint
	updated := false
	for i := range nodeTaints {
		if taint.MatchTaint(&nodeTaints[i]) {
			if apiequality.Semantic.DeepEqual(*taint, nodeTaints[i]) {
				return newNode, false, nil
			}
			newTaints = append(newTaints, *taint)
			updated = true
			continue
		}

		newTaints = append(newTaints, nodeTaints[i])
	}

	if !updated {
		newTaints = append(newTaints, *taint)
	}

	newNode.Spec.Taints = newTaints
	return newNode, true, nil
}
