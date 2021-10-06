#!/bin/bash

MGT_SIZE_IN_BYTES=1073741824
MDT_SIZE_IN_BYTES=1099511627776

PROG=$(basename "$0")

printhelp()
{
  echo "Usage: $PROG -w WORKFLOW [-c COUNT] [-x XRABBITS]"
  echo
  echo "  -w WORKFLOW  Resource name of the workflow."
  echo "  -c COUNT     Number of rabbit nodes to use."
  echo "  -x XRABBITS  Comma-separated list of rabbit nodes to exclude."
  echo
  exit 2
}

OPTIND=1
while getopts ":hw:c:x:" opt
do
  case $opt in
  w) WORKFLOW=$OPTARG ;;
  c) COUNT=$OPTARG ;;
  x) XRABBITS=$OPTARG ;;
  *) printhelp ;;
  esac
done
shift $((OPTIND-1))

if [[ -z $WORKFLOW ]]
then
  echo "$PROG: Must specify -w"
  exit 2
fi
if [[ -n $XRABBITS ]]
then
  # Flip commas to pipes, to turn it into a regex.
  XRABBITS=$(echo "$XRABBITS" | tr ',' '|')
fi

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
  echo "Setting desired state to $desiredState"
  time kubectl patch workflows.dws.cray.hpe.com "$WORKFLOW" --type='merge' --patch-file $WORKFLOW_PATCH
  rm "$WORKFLOW_PATCH"
}

query_rabbits()
{
  echo "Querying rabbits..."
  RABBITS=$(kubectl get nodes --no-headers | grep --invert-match "control-plane" | awk '{print $1}')
  if [[ -n $XRABBITS ]]
  then
    RABBITS=$(echo "$RABBITS" | grep -Ev "$XRABBITS")
  fi
  if [[ -n $COUNT ]]
  then
    RABBITS=$(echo "$RABBITS" | sed "$COUNT"q)
  fi

  RABBIT_ARRAY=($RABBITS)
  RABBIT_COUNT="${#RABBIT_ARRAY[@]}"
  if [ "$RABBIT_COUNT" = 0 ]; then
    echo "Can't find rabbits"
    exit 1
  fi
  if [[ -n $COUNT ]] && (( RABBIT_COUNT < COUNT ))
  then
    echo "Found $RABBIT_COUNT nodes, wanted $COUNT"
    exit 1
  fi

  echo RabbitCount: "$RABBIT_COUNT"
}

query_servers()
{
  echo "Querying Servers..."
  # Hard-code to take the first server from the workflow for now.
  WANTED_SERVERS=$(kubectl get -n default workflow "$WORKFLOW" -o json | jq -Mr '.status.directiveBreakdowns[].name' | sed 1q)
  if [[ -z $WANTED_SERVERS ]]
  then
    echo "Cannot find wanted server name"
    exit 1
  fi
  KNOWN_SERVERS=$(kubectl get servers.dws.cray.hpe.com -o json | jq -Mr '.items[].metadata.name')
  if [[ -z $KNOWN_SERVERS ]]
  then
    echo "Cannot find existing servers"
    exit 1
  fi
  SERVERS=$(echo "$KNOWN_SERVERS" | grep "$WANTED_SERVERS")
  if [[ -z $SERVERS ]]
  then
    echo "Can't find Servers"
    exit 1
  fi
  echo "Picked servers: $SERVERS"
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
  rm "$SERVERS_PATCH"
}

query_rabbits

query_servers

patch_servers

# Starting timing here
patch_workflow

exit $?
