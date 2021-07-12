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

    kind create cluster --config kind-config.yaml

    for NODE in $(kubectl get nodes --no-headers | awk '{print $1;}' | grep --invert-match "control-plane"); do
        kubectl label node $NODE cray.nnf.node=true
        kubectl label node $NODE cray.nnf.x-name=$NODE
    done
fi

if [ $CMD == "busybox" ]; then
    kubectl run -it --rm --restart=Never busybox --image=radial/busyboxplus:curl sh
fi

# Other useful commands are in the Makefile. Noteably
# make docker-build kind-push
# make deploy
# make undeploy

if [ $CMD == "nlc-create" ]; then
    # TODO: Create a sample per storage allocation type, and with various servers attached
    kubectl apply -f config/samples/nnf_v1alpha1_node.yaml
fi

if [ $CMD == "nlc-log" ]; then
    NS="$2"
    if [ -z "$NS" ]; then
        echo "Node name required. should be one of..."
        kubectl get nodes --selector=cray.nnf.node=true --no-headers | awk '{print $1}'
        exit 1
    fi

    POD=$(kubectl get nnfnode/nnf-nlc -n $NS -o jsonpath='{.spec.pod}')
    kubectl logs $POD -n nnf-system
fi