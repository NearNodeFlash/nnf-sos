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
IMG=$3
OVERLAY=$4

if [[ $CMD == 'deploy' ]]; then
    echo "Waiting for the dws webhook to become ready..."
    while :; do
        ready=$(kubectl get pods -n dws-operator-system -l control-plane=webhook --no-headers | awk '{print $2}')
        [[ $ready == "1/1" ]] && break
        sleep 1
    done

    $(cd config/manager && $KUSTOMIZE edit set image controller=$IMG)

    # Use server-side apply to deploy nnfcontainerprofiles successfully since they include
    # MPIJobSpec (with large annotations).
    $KUSTOMIZE build config/$OVERLAY | kubectl apply --server-side=true --force-conflicts -f -

    echo "Waiting for the nnf-sos webhook to become ready..."
    while :; do
        ready=$(kubectl get pods -n nnf-system -l control-plane=controller-manager --no-headers | awk '{print $2}')
        [[ $ready == "2/2" ]] && break
        sleep 1
    done

    # Use server-side apply to deploy nnfcontainerprofiles successfully since they include
    # MPIJobSpec (with large annotations).
    $KUSTOMIZE build config/examples | kubectl apply --server-side=true --force-conflicts -f -
fi

if [[ $CMD == 'undeploy' ]]; then
    $KUSTOMIZE build config/examples | kubectl delete --ignore-not-found -f -
    $KUSTOMIZE build config/$OVERLAY | kubectl delete --ignore-not-found -f -
fi
