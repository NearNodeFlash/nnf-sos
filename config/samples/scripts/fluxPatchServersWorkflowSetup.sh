#!/bin/bash

MGT_SIZE_IN_BYTES=1073741824
MDT_SIZE_IN_BYTES=1099511627776

query_workflow()
{
  echo "Querying workflows..."
  WORKFLOW=$(kubectl get workflows.dws.cray.hpe.com --no-headers | awk '{print $1;}')
  if [ "$WORKFLOW" = "" ]; then
    echo "Can't find a workflow"
    exit 1
  fi
}

patch_workflow()
{
  desiredState="setup"
  WORKFLOW_PATCH=workflow-patch.yaml

  {
cat > $WORKFLOW_PATCH << EOF
kind: Workflow
name: $WORKFLOW
apiVersion: dws.cray.hpe.com/v1alpha1
spec:
  desiredState: $desiredState
EOF
  }

  # Patch the Workflow object
  kubectl patch workflows.dws.cray.hpe.com "$WORKFLOW" --type='merge' --patch-file $WORKFLOW_PATCH

  # show the resulting Servers object
  kubectl get workflows.dws.cray.hpe.com "$WORKFLOW" -o yaml
}

query_rabbits()
{
  echo "Querying rabbits..."
  RABBITS=$(kubectl get nodes --no-headers | awk '{print $1;}' | grep --invert-match "control-plane")

  RABBIT_ARRAY=($RABBITS)
  RABBIT_COUNT="${#RABBIT_ARRAY[@]}"
  if [ "$RABBIT_COUNT" = 0 ]; then
    echo "Can't find rabbits"
    exit 1
  fi

  echo RabbitCount: "$RABBIT_COUNT"
}

query_servers()
{
  echo "Querying Servers..."
  SERVERS=$(kubectl get servers.dws.cray.hpe.com --no-headers | awk '{print $1;}')

  if [ "$SERVERS" = "" ]; then
    echo "Can't find Servers"
    exit 1
  fi
}

build_ost_hosts()
{
  echo Building ost hosts...
  # Rabbit 0 and 1 in the list are used for 'mgt' and 'mdt'.
  # 'ost' rabbits start at 2.
  for ((i = 2; i < RABBIT_COUNT; i++)); do
accumulator=$(cat << EOF
  - allocationCount: 1
    name: ${RABBIT_ARRAY[i]}
EOF
)

    if [ "$hostString" == "" ]; then
hostString=$(cat << EOF
$accumulator
EOF
)
    else
hostString=$(cat << EOF
$hostString
$accumulator
EOF
)
    fi

  done
}

patch_servers()
{
  SERVERS_PATCH=servers-patch.yaml

  case "$RABBIT_COUNT" in
  "1")
    {
cat > $SERVERS_PATCH << EOF
kind: Servers
name: $SERVERS
apiVersion: dws.cray.hpe.com/v1alpha1
data:
- allocationSize: $MGT_SIZE_IN_BYTES
  label: mgt
  hosts:
  - allocationCount: 1
    name: ${RABBIT_ARRAY[0]}
- allocationSize: $MDT_SIZE_IN_BYTES
  label: mdt
  hosts:
  - allocationCount: 1
    name: ${RABBIT_ARRAY[0]}
- allocationSize: 10995116277760
  label: ost
  hosts:
  - allocationCount: 1
    name: ${RABBIT_ARRAY[0]}
EOF
    }
    ;;
  "2")
    {
cat > $SERVERS_PATCH << EOF
kind: Servers
name: $SERVERS
apiVersion: dws.cray.hpe.com/v1alpha1
data:
- allocationSize: $MGT_SIZE_IN_BYTES
  label: mgt
  hosts:
  - allocationCount: 1
    name: ${RABBIT_ARRAY[0]}
- allocationSize: $MDT_SIZE_IN_BYTES
  label: mdt
  hosts:
  - allocationCount: 1
    name: ${RABBIT_ARRAY[1]}
- allocationSize: 10995116277760
  label: ost
  hosts:
  - allocationCount: 1
    name: ${RABBIT_ARRAY[1]}
EOF
    }
    ;;
  *)
    {

    build_ost_hosts

cat > $SERVERS_PATCH << EOF
kind: Servers
name: $SERVERS
apiVersion: dws.cray.hpe.com/v1alpha1
data:
- allocationSize: $MGT_SIZE_IN_BYTES
  label: mgt
  hosts:
  - allocationCount: 1
    name: ${RABBIT_ARRAY[0]}
- allocationSize: $MDT_SIZE_IN_BYTES
  label: mdt
  hosts:
  - allocationCount: 1
    name: ${RABBIT_ARRAY[1]}
- allocationSize: 10995116277760
  label: ost
  hosts:
$hostString
EOF
    }
    ;;
  esac

  # Patch the Servers object with the nodes
  kubectl patch servers.dws.cray.hpe.com "$SERVERS" --type='merge' --patch-file $SERVERS_PATCH

  # show the resulting Servers object
  kubectl get servers.dws.cray.hpe.com "$SERVERS" -o yaml
}

query_workflow
echo "$WORKFLOW"

query_rabbits

query_servers
echo "$SERVERS"

patch_servers
rm "$SERVERS_PATCH"

# Starting timing here
patch_workflow
rm "$WORKFLOW_PATCH"

exit $?