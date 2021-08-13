#!/bin/bash

CMD=$1
if [ $CMD == "kind-destroy" ]; then
    kind delete cluster
fi

# Create a kind-cluster consisting of 
if [ $CMD == "kind-create" ]; then
    CONFIG=kind-config.yaml
    if [ ! -f "$CONFIG" ]; then 
        cat > $CONFIG << EOF
# three node (two workers) cluster config
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
networking:
  apiServerAddress: "127.0.0.1"
nodes:
- role: control-plane
- role: worker
- role: worker
EOF
    fi

    kind create cluster --wait 60s --config kind-config.yaml

    # Let our first kind-worker double as a generic worker for the SLC and as a
    # rabbit for the NLC.
    kubectl label node kind-worker cray.nnf.manager=true

    for NODE in $(kubectl get nodes --no-headers | awk '{print $1;}' | grep --invert-match "control-plane"); do
        kubectl label node $NODE cray.nnf.node=true
        kubectl label node $NODE cray.nnf.x-name=$NODE
    done

    #Required for webhooks
    kubectl apply -f https://github.com/jetstack/cert-manager/releases/download/v1.4.0/cert-manager.yaml
fi

if [ $CMD == "kind-reset" ]; then
    ./playground.sh kind-destroy
    ./playground.sh kind-create
fi

if [ $CMD == "busybox" ]; then
    kubectl run -it --rm --restart=Never busybox --image=radial/busyboxplus:curl sh
fi

# Other useful commands are in the Makefile. Noteably
# make docker-build kind-push
# make deploy
# make undeploy

if [ $CMD == "slc-create" ]; then
    kubectl apply -f config/samples/nnf_v1alpha1_storage.yaml
fi

if [ $CMD == "slc-describe" ]; then
    STORAGES=$(kubectl get nnfstorages -n nnf-system --no-headers | awk '{print $1}')
    for STORAGE in $STORAGES; do
        kubectl describe nnfstorages/$STORAGE -n nnf-system
    done
fi

if [ $CMD == "slc-log" ]; then
    kubectl logs $(kubectl get pods -n nnf-system | grep nnf-controller | awk '{print $1}') -n nnf-system -c manager
fi

if [ $CMD == "nlc-log" ]; then
    NS="$2"
    if [ -z "$NS" ]; then
        echo "Node name required. Should be one of..."
        kubectl get nodes --selector=cray.nnf.node=true --no-headers | awk '{print $1}'
        exit 1
    fi

    POD=$(kubectl get nnfnode/nnf-nlc -n $NS -o jsonpath='{.spec.pod}')
    kubectl logs $POD -n nnf-system
fi
