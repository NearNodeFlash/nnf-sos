#!/bin/bash

CMD=$1

SUB_PKGS=(
    ".dws-operator"
    "."
)

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

    kind create cluster --wait 60s --image=kindest/node:v1.20.0 --config kind-config.yaml

    # Let our first kind-worker double as a generic worker for the SLC and as a
    # rabbit for the NLC.
    kubectl label node kind-worker cray.nnf.manager=true
    kubectl label node kind-worker cray.wlm.manager=true

    for NODE in $(kubectl get nodes --no-headers | grep --invert-match "control-plane" | awk '{print $1}'); do
        kubectl label node $NODE cray.nnf.node=true
        kubectl label node $NODE cray.nnf.x-name=$NODE
    done

    #Required for webhooks
    kubectl apply -f https://github.com/jetstack/cert-manager/releases/download/v1.4.0/cert-manager.yaml
fi

if [ $CMD == "kind-destroy" ]; then
    kind delete cluster
fi

if [ $CMD == "kind-reset" ]; then
    ./playground.sh kind-destroy
    ./playground.sh kind-create
fi

if [[ $CMD = restart-pods ]]; then
    ./playground.sh pause all
    while :
    do
        cnt=$(kubectl get pods -A | grep -E 'dws|nnf' | wc -l)
        (( cnt == 0 )) && break
    done
    ./playground.sh resume all
    while :
    do
        waitfor=$(kubectl get pods -A | grep -E 'dws|nnf' | grep " 0/")
        [[ -z $waitfor ]] && break
    done
fi

if [ $CMD == "busybox" ]; then
    kubectl run -it --rm --restart=Never busybox --image=radial/busyboxplus:curl sh
fi

if [ $CMD == "docker-build" ] || [ $CMD == "kind-push" ] || [ $CMD == "deploy" ]; then
    for PKG in ${SUB_PKGS[@]}; do
        ( cd $PKG; make $CMD )
    done
fi

if [ $CMD == "undeploy" ]; then
    # Reverse the list.
    for PKG in $(echo ${SUB_PKGS[@]} | tr ' ' '\n' | tac); do
        ( cd $PKG; make $CMD )
    done
fi

if [[ $CMD == pause || $CMD == resume ]]; then
    COMP="$2"

    do_node_manager=no
    do_controller_manager=no
    do_dws_operator=no

    case "$COMP" in
    node-manager)
        do_node_manager=yes
        ;;
    controller-manager)
        do_controller_manager=yes
        ;;
    dws-operator)
        do_dws_operator=yes
        ;;
    all)
        do_controller_manager=yes
        do_node_manager=yes
        do_dws_operator=yes
        ;;
    *)
        echo "Specify: node-manager, controller-manager, dws-operator, or all"
        exit 1
        ;;
    esac

    if [[ $CMD == pause ]]; then
        [[ $do_node_manager == yes ]] && kubectl patch daemonset -n nnf-system nnf-node-manager -p '{"spec":{"template":{"spec":{"nodeSelector":{"cray.nnf.node":"false"}}}}}'
        [[ $do_controller_manager == yes ]] && kubectl scale deployment --replicas=0 -n nnf-system nnf-controller-manager
        [[ $do_dws_operator == yes ]] && kubectl scale deployment --replicas=0 -n dws-operator-system dws-operator-controller-manager
    fi
    if [[ $CMD == resume ]]; then
        [[ $do_node_manager == yes ]] && kubectl patch daemonset -n nnf-system nnf-node-manager -p '{"spec":{"template":{"spec":{"nodeSelector":{"cray.nnf.node":"true"}}}}}'
        [[ $do_controller_manager == yes ]] && kubectl scale deployment --replicas=1 -n nnf-system nnf-controller-manager
        [[ $do_dws_operator == yes ]] && kubectl scale deployment --replicas=1 -n dws-operator-system dws-operator-controller-manager
    fi
fi

if [[ $CMD == list ]]; then
    kinds_dws=$(kubectl api-resources | grep -i dws | awk '{print $1}' | sort)
    kinds_nnf=$(kubectl api-resources | grep -i nnf | awk '{print $1}' | sort)
    for x in $kinds_dws $kinds_nnf; do
        echo "=== $x"
        kubectl get -A $x
        echo
    done
fi

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
