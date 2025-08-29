/*
 * Copyright 2025 Hewlett Packard Enterprise Development LP
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

package helpers

import (
	"context"

	dwsv1alpha6 "github.com/DataWorkflowServices/dws/api/v1alpha6"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// getRabbitFromCompute finds the name of the Rabbit that is physically connected to the compute node
func GetRabbitFromCompute(ctx context.Context, c client.Client, compute string) (string, error) {
	// To find the Rabbit associated with this compute, we have to search the SystemConfiguration resource
	systemConfiguration := &dwsv1alpha6.SystemConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default",
			Namespace: corev1.NamespaceDefault,
		},
	}

	if err := c.Get(ctx, client.ObjectKeyFromObject(systemConfiguration), systemConfiguration); err != nil {
		return "", dwsv1alpha6.NewResourceError("could not get SystemConfiguration %v", client.ObjectKeyFromObject(systemConfiguration)).WithError(err).WithMajor()
	}

	// Search through the SystemConfiguration to find the Compute node name
	for _, storageNode := range systemConfiguration.Spec.StorageNodes {
		for _, computeAccess := range storageNode.ComputesAccess {
			if computeAccess.Name == compute {
				return storageNode.Name, nil
			}
		}
	}

	return "", dwsv1alpha6.NewResourceError("could not find compute in SystemConfiguration: %s", compute).WithMajor()
}
