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

	dwsv1alpha3 "github.com/DataWorkflowServices/dws/api/v1alpha3"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var computeRabbitMap map[string]string

func initializeComputeRabbitMap(ctx context.Context, c client.Client) error {
	systemConfiguration := &dwsv1alpha3.SystemConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default",
			Namespace: corev1.NamespaceDefault,
		},
	}

	if err := c.Get(ctx, client.ObjectKeyFromObject(systemConfiguration), systemConfiguration); err != nil {
		return dwsv1alpha3.NewResourceError("could not get SystemConfiguration %v", client.ObjectKeyFromObject(systemConfiguration)).WithError(err).WithMajor()
	}

	computeRabbitMap = make(map[string]string)

	// Walk through the SystemConfiguration and build a map of compute to Rabbit names
	for _, storageNode := range systemConfiguration.Spec.StorageNodes {
		for _, computeAccess := range storageNode.ComputesAccess {
			computeRabbitMap[computeAccess.Name] = storageNode.Name
		}
	}

	return nil
}

// getRabbitFromCompute finds the name of the Rabbit that is physically connected to the compute node
func GetRabbitFromCompute(ctx context.Context, c client.Client, compute string) (string, error) {
	var log logr.Logger
	log.Info("Rabbit from compute", "compute", compute)
	if len(computeRabbitMap) == 0 {
		log.Info("empty map")
		err := initializeComputeRabbitMap(ctx, c)
		if err != nil {
			log.Info("error initializing")
			return "", err
		}
	}

	rabbit, found := computeRabbitMap[compute]
	if !found {
		log.Info("could not find Rabbit for compute", "compute", compute)
		return "", dwsv1alpha3.NewResourceError("could not find compute in SystemConfiguration: %s", compute).WithMajor()
	}

	log.Info("found", "compute", compute, "rabbit", rabbit)
	return rabbit, nil
}
