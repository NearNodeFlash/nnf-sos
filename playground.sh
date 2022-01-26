#!/bin/bash
CMD=$1

reset_proxy ()
{
    # kill any running proxy
    pkill -f 'kubectl proxy --port 8080'

    # Nice article explaining the need for a proxy: <https://stackoverflow.com/questions/54332972/what-is-the-purpose-of-kubectl-proxy#54345488>
    # Also: <https://kubernetes.io/docs/tutorials/kubernetes-basics/expose/expose-intro/>
    # Proxy port 8080 to allow direct REST access
    kubectl proxy --port 8080 > /dev/null &

    # give the proxy a chance to get started
    sleep .1
}

SUB_PKGS=(
    ".dws-operator"
    "."
)

if [[ "$CMD" == "kind-create" ]]; then
    CONFIG=kind-config.yaml
    if [[ ! -f "$CONFIG" ]]; then
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

    # Use the kind-control-plane node for the SLCMs.  Remove its default taint
    # and label it for our use.
    kubectl taint node kind-control-plane node-role.kubernetes.io/master:NoSchedule-
    kubectl label node kind-control-plane cray.nnf.manager=true
    kubectl label node kind-control-plane cray.wlm.manager=true

    # Taint the kind workers as rabbit nodes for the NLCMs, to keep any
    # non-NLCM pods off of them.
    NODES=$(kubectl get nodes --no-headers -o custom-columns=:metadata.name | grep -v control-plane | paste -d" " -s -)
    kubectl taint nodes $NODES cray.nnf.node=true:NoSchedule

    # Label the kind-workers as rabbit nodes for the NLCMs.
    for NODE in $(kubectl get nodes --no-headers | grep --invert-match "control-plane" | awk '{print $1}'); do
        kubectl label node "$NODE" cray.nnf.node=true
        kubectl label node "$NODE" cray.nnf.x-name="$NODE"
    done

    #Required for webhooks
    kubectl apply -f https://github.com/jetstack/cert-manager/releases/download/v1.4.0/cert-manager.yaml
fi

if [[ "$CMD" == kind-destroy ]]; then
    kind delete cluster
fi

if [[ "$CMD" == kind-reset ]]; then
    ./playground.sh kind-destroy
    ./playground.sh kind-create
fi


# The following commands apply to initializing the current DP0 environment
# Nodes containing 'cn' are considered to be worker nodes for the time being.
if [[ "$CMD" == dp0-init ]]; then
    COMPUTE_NODES=$(kubectl get nodes --no-headers -o custom-columns=:metadata.name | grep cn | paste -d" " -s -)
    RABBIT_NODES=$(kubectl get nodes --no-headers -o custom-columns=:metadata.name | grep -v cn | grep -v master | paste -d" " -s -)
    MASTER_NODES=$(kubectl get nodes --no-headers -o custom-columns=:metadata.name | grep master | paste -d" " -s -)

    echo COMPUTE_NODES "$COMPUTE_NODES"
    echo RABBIT_NODES "$RABBIT_NODES"
    echo MASTER_NODES "$MASTER_NODES"

    # Label the COMPUTE_NODES to allow them to handle wlm and nnf-sos
    # We are using COMPUTE_NODES as generic k8s workers
    for NODE in $COMPUTE_NODES; do
        # Label them for SLCMs.
        kubectl label node "$NODE" cray.nnf.manager=true
        kubectl label node "$NODE" cray.wlm.manager=true
    done

    for NODE in $RABBIT_NODES; do
        # Taint the rabbit nodes for the NLCMs, to keep any
        # non-NLCM pods off of them.
        kubectl taint node "$NODE" cray.nnf.node=true:NoSchedule

        # Label the rabbit nodes for the NLCMs.
        kubectl label node "$NODE" cray.nnf.node=true
        kubectl label node "$NODE" cray.nnf.x-name="$NODE"
    done

    #Required for webhooks
    # kubectl delete -f https://github.com/jetstack/cert-manager/releases/download/v1.4.0/cert-manager.yaml
    kubectl apply -f https://github.com/jetstack/cert-manager/releases/download/v1.4.0/cert-manager.yaml
fi

# The following commands apply to initializing the current DP1 environment
# Nodes containing 'cn' are considered to be worker nodes for the time being.
if [[ "$CMD" == dp1-init ]]; then
    WORKER_NODES=$(kubectl get nodes --no-headers -o custom-columns=:metadata.name | grep -i 'worker' | paste -d" " -s -)
    RABBIT_NODES=$(kubectl get nodes --no-headers -o custom-columns=:metadata.name | grep -i 'node'   | grep -v master | paste -d" " -s -)
    MASTER_NODES=$(kubectl get nodes --no-headers -o custom-columns=:metadata.name | grep -i 'master' | paste -d" " -s -)

    echo WORKER_NODES "$WORKER_NODES"
    echo RABBIT_NODES "$RABBIT_NODES"
    echo MASTER_NODES "$MASTER_NODES"

    # Label the WORKER_NODES to allow them to handle wlm and nnf-sos
    for NODE in $WORKER_NODES; do
        # Label them for SLCMs.
        kubectl label node "$NODE" cray.nnf.manager=true
        kubectl label node "$NODE" cray.wlm.manager=true
    done

    for NODE in $RABBIT_NODES; do
        # Taint the rabbit nodes for the NLCMs, to keep any
        # non-NLCM pods off of them.
        kubectl taint node "$NODE" cray.nnf.node=true:NoSchedule

        # Label the rabbit nodes for the NLCMs.
        kubectl label node "$NODE" cray.nnf.node=true
        kubectl label node "$NODE" cray.nnf.x-name="$NODE"
    done

    #Required for webhooks
    # kubectl delete -f https://github.com/jetstack/cert-manager/releases/download/v1.4.0/cert-manager.yaml
    kubectl apply -f https://github.com/jetstack/cert-manager/releases/download/v1.4.0/cert-manager.yaml
fi

if [[ "$CMD" == restart-pods ]]; then
    ./playground.sh pause all
    while :
    do
        cnt=$(kubectl get pods -A | grep -cE 'dws|nnf')

        (( cnt == 0 )) && break
    done
    ./playground.sh resume all
    while :
    do
        waitfor=$(kubectl get pods -A | grep -E 'dws|nnf' | grep " 0/")
        [[ -z $waitfor ]] && break
    done
fi

if [[ "$CMD" == "busybox" ]]; then
    kubectl run -it --rm --restart=Never busybox --image=radial/busyboxplus:curl sh
fi

if [[ "$CMD" == "docker-build" ]] || [[ "$CMD" == "kind-push" ]] || [[ "$CMD" == "deploy" ]]; then
    for PKG in "${SUB_PKGS[@]}"; do
        ( cd "$PKG" && make "$CMD" )
    done
fi

if [[ "$CMD" == "undeploy" ]]; then
    # Remove the nnfnode resources created by the NLCs
    for NAMESPACE in $(kubectl get nnfnodes -A -o custom-columns=":metadata.namespace" --no-headers); do
        kubectl delete nnfnode -n "$NAMESPACE" nnf-nlc
    done

    # Reverse the list.
    for PKG in $(echo "${SUB_PKGS[@]}" | tr ' ' '\n' | tac); do
        ( cd "$PKG" && make "$CMD" )
    done
fi

if [[ "$CMD" == pause || "$CMD" == resume ]]; then
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

    if [[ "$CMD" == pause ]]; then
        [[ $do_node_manager == yes ]] && kubectl patch daemonset -n nnf-system nnf-node-manager -p '{"spec":{"template":{"spec":{"nodeSelector":{"cray.nnf.node":"false"}}}}}'
        [[ $do_controller_manager == yes ]] && kubectl scale deployment --replicas=0 -n nnf-system nnf-controller-manager
        [[ $do_dws_operator == yes ]] && kubectl scale deployment --replicas=0 -n dws-operator-system dws-operator-controller-manager
    fi
    if [[ "$CMD" == resume ]]; then
        [[ $do_node_manager == yes ]] && kubectl patch daemonset -n nnf-system nnf-node-manager -p '{"spec":{"template":{"spec":{"nodeSelector":{"cray.nnf.node":"true"}}}}}'
        [[ $do_controller_manager == yes ]] && kubectl scale deployment --replicas=1 -n nnf-system nnf-controller-manager
        [[ $do_dws_operator == yes ]] && kubectl scale deployment --replicas=1 -n dws-operator-system dws-operator-controller-manager
    fi
fi

if [[ "$CMD" == list ]]; then
     kinds_dws_nnf=$(kubectl api-resources | grep -iE 'dws|nnf' | awk '{print $1":"$2}' | sort)
     for want_kind in dws nnf
     do
         for x in $(echo "$kinds_dws_nnf" | grep ":$want_kind\." | awk -F: '{print $1}')
         do
             echo "=== $x"
             kubectl get -A "$x"
             echo
         done
    done
fi

if [[ "$CMD" == "slc-create" ]]; then
    kubectl apply -f config/samples/nnf_v1alpha1_storage.yaml
fi

if [[ "$CMD" == "slc-describe" ]]; then
    STORAGES=$(kubectl get nnfstorages -n nnf-system --no-headers | awk '{print $1}')
    for STORAGE in $STORAGES; do
        kubectl describe nnfstorages/"$STORAGE" -n nnf-system
    done
fi

if [[ "$CMD" == "dws-log" ]]; then
    kubectl logs "$(kubectl get pods -n dws-operator-system | grep manager | awk '{print $1}')" -n dws-operator-system -c manager
fi

if [[ "$CMD" == "slc-log" ]]; then
    kubectl logs "$(kubectl get pods -n nnf-system | grep nnf-controller | awk '{print $1}')" -n nnf-system -c manager
fi

if [[ "$CMD" == "nlc-log" ]]; then
    NS="$2"
    if [[ -z "$NS" ]]; then
        echo "Node name required. Should be one of..."
        kubectl get nodes --selector=cray.nnf.node=true --no-headers | awk '{print $1}'
        exit 1
    fi

    POD=$(kubectl get nnfnode/nnf-nlc -n "$NS" -o jsonpath='{.spec.pod}')
    kubectl logs "$POD" -n nnf-system
fi

if [[ "$CMD" == "nlc-sh" ]]; then
    NS="$2"
    if [[ -z "$NS" ]]; then
        echo "Node name required. Should be one of..."
        kubectl get nodes --selector=cray.nnf.node=true --no-headers | awk '{print $1}'
        exit 1
    fi

    POD=$(kubectl get pods -n nnf-system --no-headers -o wide --field-selector spec.nodeName="$NS" | awk '{print $1}')
    echo Shelling into "$NS": "$POD"
    kubectl exec --stdin --tty $POD -n nnf-system -- /bin/bash
fi

if [[ "$CMD" == "wfCreate" ]]; then
    if [[ -z "$2" ]]; then
        NWF=1
    else
        NWF=$2
    fi

    reset_proxy
    echo Creating "$NWF" workflows
    for ((i=0; i < NWF; ++i)) do
        # Create a new workflow
        echo w"$i"
        sh ./config/samples/scripts/wfrLustre w"$i"
    done

    kubectl get workflows.dws.cray.hpe.com -o yaml | grep ready; kubectl get workflows.dws.cray.hpe.com -o yaml | grep elaps
fi

if [[ "$CMD" == "wfDelete" ]]; then
    reset_proxy

    workflows=$(kubectl get workflows -A --no-headers 2> /dev/null | awk '{print "ns="$1":res="$2}')
    echo "$workflows"
    [[ -z $workflows ]] && exit 0
    for nsres in $workflows
    do
        eval "$(echo "$nsres" | tr ':' ' ')"
        #shellcheck disable=2154
        kubectl delete workflow -n "$ns" "$res"
    done
fi

if [[ "$CMD" == make-standalone-playground ]]
then
    PLAY=standalone-playground
    DWSPLAY=$PLAY/.dws-operator
    mkdir $PLAY
    mkdir $DWSPLAY
    cp -r .dws-operator/config $DWSPLAY
    cp .dws-operator/Makefile .dws-operator/.version $DWSPLAY
    cp -r config $PLAY
    cp .version Makefile playground.sh $PLAY
fi

