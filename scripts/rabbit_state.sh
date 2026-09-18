#!/bin/bash
#
# Output a status summary of the NNF software deployed to the Kubernetes cluster.
#

set -euo pipefail

for cmd in kubectl jq nodeattr awk paste grep; do
    command -v "$cmd" >/dev/null || { echo "error: $cmd not found" >&2; exit 1; }
done

export KUBECONFIG=/etc/kubernetes/admin.conf
export CLUSTER=$(nodeattr -q cluster | awk -F'[' '{print $1}')
[[ -n $CLUSTER ]] || { echo "error: could not determine cluster name" >&2; exit 1; }
echo "CLUSTER=$CLUSTER"

export VERSION=$(kubectl get deploy -n nnf-system nnf-controller-manager -o json | jq -rM .metadata.labels | awk -F'"' '/nnf-version/ { print $4 }')
echo "VERSION=$VERSION"

FMT="+%Y%m%d%H%M%S"
now=$(date $FMT)
echo "now=$now"
histdir="$TMPDIR/rabbit-hist-$CLUSTER"
if [[ ! -d $histdir ]]; then
  mkdir "$histdir" || exit 1
fi
if [[ -f $histdir/latest-stamp ]]; then
  prevstamp=$(<"$histdir/latest-stamp")
  echo "prevstamp=$prevstamp"
fi

kubectl get pods -n nnf-system -o wide > "$histdir/pods-$now"
kubectl get pods -n dws-system -o wide > "$histdir/pods-dws-$now"
kubectl get pods -n nnf-dm-system -o wide > "$histdir/pods-dm-$now"
kubectl get pods -n calico-system -o wide > "$histdir/pods-calico-$now"

kubectl get nodes -o wide > "$histdir/nodes-$now"


kubectl get nodes -o json | jq -cM '.items[] | {"name":.metadata.name,"taints":.spec.taints}' > "$histdir/taints-$now"

kubectl get systemconfiguration default -o yaml > $histdir/systemconfiguration-"$now"

kubectl get lustrefilesystems -A -o yaml > $histdir/lustrefilesystems-"$now"

(

  # DWS is expected to be on the worker nodes.
  # Find these with: nodeattr -q rabbitk8sworkerinfra
  worker_nodes=$(nodeattr -c rabbitk8sworkerinfra | tr ',' '|')
  ERR_NODES=$(awk '$7 ~ /'"$CLUSTER"'/ {print $7}' "$histdir/pods-dws-$now"  | grep -v -E "$worker_nodes" || true)
  if [[ -n $ERR_NODES ]]; then
    echo
    echo "DWS on unexpected nodes:"
    echo "$ERR_NODES" | paste -d, -s
    echo
  fi

  # The worker nodes should not be tainted.
  # Find these with: nodeattr -q rabbitk8sworkerinfra
  pat='"'"($worker_nodes)"'"'
  UNEXP_TAINTS=$(grep -E "$pat" "$histdir/taints-$now" | grep -v null || true)
  if [[ -n $UNEXP_TAINTS ]]; then
    echo
    echo "Unexpected taints on nodes:"
    echo "$UNEXP_TAINTS"
    echo
  fi

  # Any node other than the worker nodes should be tainted for the NLCs. These
  # are the rabbit nodes.
  # Find these with: nodeattr -q rabbit
  pat='"'"($worker_nodes)"'"'
  NULL_TAINTS=$(grep null "$histdir/taints-$now" | grep -v -E "$pat" || true)
  if [[ -n $NULL_TAINTS ]]; then
    echo
    echo "Null taints on unexpected nodes:"
    echo "$NULL_TAINTS"
    echo
  fi

  echo
  echo "Nodes not ready: $(grep -c NotReady "$histdir/nodes-$now")"
  echo "Nodes ready: $(grep -c -v -e NotReady -e NAME "$histdir/nodes-$now")"
  echo "Nodes tainted with cray.nnf.node.drain: $(grep -c -e '"cray.nnf.node.drain"' "$histdir/taints-$now")"
  echo "Nodes tainted with cray.nnf.node.drain.csi: $(grep -c -e '"cray.nnf.node.drain.csi"' "$histdir/taints-$now")"
  echo

  count_node_state() {
    local state="$1"
    local f="$2"
    local nodes_file="$3"

    local nodes_ready=0
    local nodes_notready=0
    local max_print=10
    local nodes_ready_pods=""
    local node
    local statefile

    unset did_ellipsis
    statefile=$(mktemp -t rabbit-state-XXXXXXXX)
    trap 'rm -f "$statefile"' RETURN

    grep "$state" "$f" > "$statefile"
    while read -r pod; do
      # shellcheck disable=SC2001
      #node=$(echo "$pod" | sed 's/.*\(elcap[^ ]*\).*/\1/')
      node=$(echo "$pod" | sed 's/.*\('"$CLUSTER"'[^ ]*\).*/\1/')
      if grep -q -E "$node"' *NotReady' "$nodes_file"; then
          (( nodes_notready = nodes_notready + 1 ))
      else
          (( nodes_ready = nodes_ready + 1 ))
          if (( nodes_ready <= max_print )); then
            nodes_ready_pods="$nodes_ready_pods
$pod"
          elif [[ -z $did_ellipsis ]]; then
            nodes_ready_pods="$nodes_ready_pods
[...]"
            did_ellipsis=true
          fi
      fi
    done < "$statefile"
    rm -f "$statefile"

    if (( nodes_notready > 0 )); then
      echo "      $nodes_notready are on NotReady nodes"
    fi
    if (( nodes_ready > 0 )); then
      echo "      $nodes_ready are on Ready nodes"
      echo "$nodes_ready_pods"
      echo
    fi
  }

  count_node_state_simple() {
    local f="$1"
    local nodes_file="$2"

    local nodes_notready=0
    local node
    local statefile

    statefile="$histdir/count_node_state"
    grep Running "$f" > "$statefile" || true
    while read -r pod; do
      # shellcheck disable=SC2001
      #node=$(echo "$pod" | sed 's/.*\(elcap[^ ]*\).*/\1/')
      node=$(echo "$pod" | sed 's/.*\('"$CLUSTER"'[^ ]*\).*/\1/')
      if grep -q -E "$node"' *NotReady' "$nodes_file"; then
          (( nodes_notready = nodes_notready + 1 ))
      fi
    done < "$statefile"
    rm -f "$statefile"

    if (( nodes_notready > 0 )); then
      echo "      $nodes_notready are on NotReady nodes"
    fi
  }

  summarize_pods() {
    local f="$2"
    local nodes_file="$3"
    echo "$1 pods"
    echo "  running: $(grep -c Running "$f")"
    count_node_state_simple "$f" "$nodes_file"

    local not_running
    local container_creating
    local crashloopbackoff
    local createcontainererror
    local errimagepull
    local imagepullbackoff
    local pending
    local terminating

    local others

    not_running=$(grep -c -v -e NAME -e Running "$f" || true)
    container_creating=$(grep -c ContainerCreating "$f" || true)
    crashloopbackoff=$(grep -c CrashLoopBackOff "$f" || true)
    createcontainererror=$(grep -c CreateContainerError "$f" || true)
    errimagepull=$(grep -c ErrImagePull "$f" || true)
    imagepullbackoff=$(grep -c ImagePullBackOff "$f" || true)
    pending=$(grep -c Pending "$f" || true)
    terminating=$(grep -c Terminating "$f" || true)

    others=$(( not_running - container_creating - crashloopbackoff - createcontainererror - errimagepull - imagepullbackoff - pending - terminating ))

    echo "  not running: $not_running"
    if (( container_creating > 0 )); then
        echo "    ContainerCreating: $container_creating"
        count_node_state "ContainerCreating" "$f" "$nodes_file"
    fi
    if (( crashloopbackoff > 0 )); then
        echo "    CrashLoopBackOff: $crashloopbackoff"
        count_node_state "CrashLoopBackOff" "$f" "$nodes_file"
    fi
    if (( createcontainererror > 0 )); then
        echo "    CreateContainerError: $createcontainererror"
        count_node_state "CreateContainerError" "$f" "$nodes_file"
    fi
    if (( errimagepull > 0 )); then
        echo "    ErrImagePull: $errimagepull"
        count_node_state "ErrImagePull" "$f" "$nodes_file"
    fi
    if (( imagepullbackoff > 0 )); then
        echo "    ImagePullBackOff: $imagepullbackoff"
        count_node_state "ImagePullBackOff" "$f" "$nodes_file"
    fi
    if (( terminating > 0 )); then
        echo "    Terminating: $terminating"
        count_node_state "Terminating" "$f" "$nodes_file"
    fi
    if (( pending > 0 )); then
        echo "    Pending: $pending"
        count_node_state "Pending" "$f" "$nodes_file"
    fi

    if (( others > 0 )); then
        echo "    others: $others"
    fi
    echo
  }

  summarize_pods "dws-system"    "$histdir/pods-dws-$now"     "$histdir/nodes-$now"
  summarize_pods "nnf-system"    "$histdir/pods-$now"         "$histdir/nodes-$now"
  summarize_pods "nnf-dm-system" "$histdir/pods-dm-$now"      "$histdir/nodes-$now"
  summarize_pods "calico-system" "$histdir/pods-calico-$now"  "$histdir/nodes-$now"

  disabled_storages=$(kubectl get storages --no-headers | grep -cv Enabled || true)
  if (( disabled_storages > 0 )); then
      echo
      echo "$disabled_storages Storages resources are not Enabled"
      echo
  fi

  echo
  echo "ArgoCD state:"
  kubectl get applications -n argocd

  echo
  no_self_heal=$(kubectl get application -n argocd -o json | jq -rM '.items[]|select(.spec.syncPolicy.automated.selfHeal==false)|.metadata.name')
  if [[ $no_self_heal != "" ]]; then
    echo "The following argocd applications have selfHeal=false:"
    echo "$no_self_heal"
  fi

) | tee "$histdir/report-$now"
echo "$now" > "$histdir/latest-stamp"

echo
echo "now=$now"
[[ -n $prevstamp ]] && echo "prevstamp=$prevstamp"
