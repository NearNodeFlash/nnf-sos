#!/bin/bash

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
