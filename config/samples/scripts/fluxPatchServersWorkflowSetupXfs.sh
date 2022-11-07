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

XFS_CAPACITY_IN_BYTES=5000000000

PROG=$(basename "$0")

printhelp()
{
  echo "Usage: $PROG -w WORKFLOW [-c COUNT] [-x XRABBITS | -i RABBITS] [-a ALLOCATION_COUNT] [-X]"
  echo
  echo "  -w WORKFLOW  Resource name of the workflow."
  echo "  -c COUNT     Number of rabbit nodes to use."
  echo "  -i RABBITS   Comma-separated list of rabbit nodes to use."
  echo "  -x XRABBITS  Comma-separated list of rabbit nodes to exclude."
  echo "  -a ALLOCATION_COUNT Number of allocations per OST rabbit node (default 1)."
  echo "  -X           Exclude rabbit nodes that have an existing nnfnodestorage."
  echo
  exit 2
}

ALLOCATION_COUNT=1
OPTIND=1
while getopts ":hw:c:x:i:a:X" opt
do
  case $opt in
  w) WORKFLOW=$OPTARG ;;
  c) COUNT=$OPTARG ;;
  i) IRABBITS=$OPTARG ;;
  x) XRABBITS=$OPTARG ;;
  a) ALLOCATION_COUNT=$OPTARG ;;
  X) XNNFNODESTORAGES=yes ;;
  *) printhelp ;;
  esac
done
shift $((OPTIND-1))

if [[ -z $WORKFLOW ]]
then
  echo "$PROG: Must specify -w"
  exit 2
fi
if [[ -n $XRABBITS && -n $XNNFNODESTORAGES ]]
then
  echo "$PROG: Do not use -x and -X together"
  exit 2
fi
if [[ -n $XRABBITS ]]
then
  # Flip commas to pipes, to turn it into a regex.
  XRABBITS=$(echo "$XRABBITS" | tr ',' '|')
elif [[ -n $XNNFNODESTORAGES ]]
then
  # Get a list of rabbits that have an existing nnfnodestorage.  Make a
  # regex for them.
  XRABBITS=$(kubectl get nnfnodestorages -A -o json | jq -Mr '.items[].metadata|.namespace' | paste -s -d'|' -)
fi
if [[ -n $XRABBITS && -n $IRABBITS ]]
then
  echo "$PROG: Do not use -i and -x/-X together"
  exit 2
fi
if [[ -n $IRABBITS ]]
then
  # Flip commas to pipes, to turn it into a regex.
  IRABBITS=$(echo "$IRABBITS" | tr ',' '|')
fi

patch_workflow()
{
  desiredState="Setup"
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
  RABBITS=$(kubectl get nodes --no-headers -l cray.nnf.node=true | sort | awk '{print $1}')
  if [[ -n $XRABBITS ]]
  then
    RABBITS=$(echo "$RABBITS" | grep -Ev "$XRABBITS")
  elif [[ -n $IRABBITS ]]
  then
    RABBITS=$(echo "$RABBITS" | grep -E "$IRABBITS")
  fi
  if which shuf 1> /dev/null 2>&1
  then
    # Shuffle with the shuf(1) command when we can find it.
    RABBITS=$(echo "$RABBITS" | shuf)
  fi
  if [[ -n $COUNT ]]
  then
    RABBITS=$(echo "$RABBITS" | sed "$COUNT"q)
  fi

  # shellcheck disable=SC2206
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
  echo Rabbits: "$RABBITS"
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

get_capacity()
{
  # Retrieve the minimum capacity from the directiveBreakdown
  XFS_CAPACITY_IN_BYTES=$(kubectl get directivebreakdowns.dws.cray.hpe.com -n default -o json | jq -Mr '.items[0].status.allocationSet[0]' | jq -Mr '.minimumCapacity')
}

patch_servers()
{
  SERVERS_PATCH=servers-patch.yaml
  rm -f "$SERVERS_PATCH"

  get_capacity

  # For now, just pick the first rabbit for allocation
  cat > $SERVERS_PATCH << EOF
kind: Servers
name: $SERVERS
apiVersion: dws.cray.hpe.com/v1alpha1
spec:
  allocationSets:
  - allocationSize: $XFS_CAPACITY_IN_BYTES
    label: xfs
    storage:
    - allocationCount: 1
      name: ${RABBIT_ARRAY[0]}
EOF

  # Patch the Servers object with the nodes
  if ! kubectl patch servers.dws.cray.hpe.com "$SERVERS" --type='merge' --patch-file $SERVERS_PATCH
  then
    echo "Servers patch file: $SERVERS_PATCH"
    exit 2
  fi
  rm "$SERVERS_PATCH"
}

query_rabbits

query_servers

patch_servers

# Starting timing here
patch_workflow

exit $?
