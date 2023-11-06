#!/bin/bash

# Copyright 2021-2023 Hewlett Packard Enterprise Development LP
# Other additional copyright holders may be indicated within.
#
# The entirety of this work is licensed under the Apache License,
# Version 2.0 (the "License"); you may not use this file except
# in compliance with the License.
#
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -e

# Deploy/undeploy controller to the K8s cluster specified in ~/.kube/config.

CMD=$1
KUSTOMIZE=$2
OVERLAY_DIR=$3
OVERLAY_EXAMPLES_DIR=$4

if [[ $CMD == 'deploy' ]]; then
    echo "Waiting for the dws webhook to become ready..."
    while :; do
        ready=$(kubectl get deployments -n dws-system dws-webhook -o json | jq -Mr '.status.readyReplicas')
        [[ $ready -ge 1 ]] && break
        sleep 1
    done

    # Use server-side apply to deploy nnfcontainerprofiles successfully since they include
    # MPIJobSpec (with large annotations).
    $KUSTOMIZE build $OVERLAY_DIR | kubectl apply --server-side=true --force-conflicts -f -

    echo "Waiting for the nnf-sos webhook to become ready..."
    while :; do
        ready=$(kubectl get pods -n nnf-system -l control-plane=controller-manager --no-headers | awk '{print $2}')
        [[ $ready == "2/2" ]] && break
        sleep 1
    done

    # Use server-side apply to deploy nnfcontainerprofiles successfully since they include
    # MPIJobSpec (with large annotations).
    $KUSTOMIZE build $OVERLAY_EXAMPLES_DIR | kubectl apply --server-side=true --force-conflicts -f -

    # Deploy the nnfportmanager after everything else
    echo "Waiting for the nnfportmamanger CRD to become ready..."
    while :; do
        sleep 1
        kubectl get crds nnfportmanagers.nnf.cray.hpe.com && break
    done
    $KUSTOMIZE build config/ports| kubectl apply --server-side=true --force-conflicts -f -

    # Deploy the ServiceMonitor resource if its CRD is found. The CRD would
    # have been installed by a metrics service such as Prometheus.
    if kubectl get crd servicemonitors.monitoring.coreos.com > /dev/null 2>&1; then
        $KUSTOMIZE build config/prometheus | kubectl apply -f-
    fi
fi

if [[ $CMD == 'undeploy' ]]; then
    $KUSTOMIZE build config/prometheus | kubectl delete --ignore-not-found -f-
    $KUSTOMIZE build config/ports | kubectl delete --ignore-not-found -f -
    $KUSTOMIZE build $OVERLAY_EXAMPLES_DIR | kubectl delete --ignore-not-found -f -
    $KUSTOMIZE build $OVERLAY_DIR | kubectl delete --ignore-not-found -f -
fi
