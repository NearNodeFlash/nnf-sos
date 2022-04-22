#!/bin/bash

# Copyright 2021, 2022 Hewlett Packard Enterprise Development LP
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

if [[ $CMD == 'deploy' ]]
then
    $(cd config/manager && $KUSTOMIZE edit set image controller=$IMG)

    $KUSTOMIZE build config/$OVERLAY | kubectl apply -f -

    echo "Waiting for the NnfStorageProfile webhook to become ready..."
    while :
    do
        ready=$(kubectl get pods -n nnf-system -l control-plane=controller-manager --no-headers | awk '{print $2}')
        [[ $ready == "2/2" ]] && break
        sleep 1
    done
    kubectl apply -f config/samples/placeholder_nnfstorageprofile.yaml
fi

if [[ $CMD == 'undeploy' ]]
then
	kubectl delete -f config/samples/placeholder_nnfstorageprofile.yaml || true
	$KUSTOMIZE build config/$OVERLAY | kubectl delete -f -
fi
