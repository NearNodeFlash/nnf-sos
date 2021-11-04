#!/bin/bash

MGT_SIZE_IN_BYTES=1073741824
MDT_SIZE_IN_BYTES=1099511627776

PROG=$(basename "$0")

printhelp()
{
  echo "Usage: $PROG -w WORKFLOW [-c COUNT] [-x XRABBITS | -i RABBITS] [-a ALLOCATION_COUNT] [-Y] [-X] [-C]"
  echo
  echo "  -w WORKFLOW  Resource name of the workflow."
  echo "  -C           Colocate MDT and MGT on same rabbit (but separate devices)."
  echo "  -c COUNT     Number of rabbit nodes to use."
  echo "  -i RABBITS   Comma-separated list of rabbit nodes to use."
  echo "  -x XRABBITS  Comma-separated list of rabbit nodes to exclude."
  echo "  -a ALLOCATION_COUNT Number of allocations per OST rabbit node (default 1)."
  echo "  -Y           Treat the OSTs as asymmetric."
  echo "  -X           Exclude rabbit nodes that have an existing nnfnodestorage."
  echo
  exit 2
}

ALLOCATION_COUNT=1
OPTIND=1
while getopts ":hw:c:x:i:a:YXC" opt
do
  case $opt in
  w) WORKFLOW=$OPTARG ;;
  c) COUNT=$OPTARG ;;
  C) COLOCATE_MDT_MGT=yes ;;
  i) IRABBITS=$OPTARG ;;
  x) XRABBITS=$OPTARG ;;
  a) ALLOCATION_COUNT=$OPTARG ;;
  Y) ASYMMETRIC_OSTS=yes ;;
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
  # Rabbit 0 and 1 in the list are used for 'mgt' and 'mdt'.
  # 'ost' rabbits start at 2.
  accumulator=""
  if [[ -n $ASYMMETRIC_OSTS ]]
  then
    # List each OST rabbit node with its own "ost" label in its own
    # "hosts" array.
    # This would allow asymmetric OSTs.
    for ((i = 2; i < RABBIT_COUNT; i++)); do
      accumulator+="- allocationSize: 1073741824\n"
      accumulator+="  label: ost\n"
      accumulator+="  hosts:\n"
      accumulator+="  - allocationCount: $ALLOCATION_COUNT\n"
      accumulator+="    name: ${RABBIT_ARRAY[i]}\n"
    done
  else
    # List all OST rabbit nodes in the same "hosts" array.  This implies that
    # the OSTs are symmetric.
    accumulator+="- allocationSize: 1073741824\n"
    accumulator+="  label: ost\n"
    accumulator+="  hosts:\n"
    for ((i = 2; i < RABBIT_COUNT; i++)); do
      accumulator+="  - allocationCount: $ALLOCATION_COUNT\n"
      accumulator+="    name: ${RABBIT_ARRAY[i]}\n"
    done
  fi
  echo -e "$accumulator"
}

patch_servers()
{
  SERVERS_PATCH=servers-patch.yaml

  # The MGT will be RABBIT_ARRAY[0].  The MDT will normally be RABBIT_ARRAY[1],
  # unless we need to put it on the same node as the MGT.
  MDT_IDX=1
  if (( RABBIT_COUNT == 1 )) || [[ -n $COLOCATE_MDT_MGT ]]
  then
    MDT_IDX=0
  fi

  cat > $SERVERS_PATCH << EOF
kind: Servers
name: $SERVERS
apiVersion: dws.cray.hpe.com/v1alpha1
spec:
  allocationSets:
  - allocationSize: $MGT_SIZE_IN_BYTES
    label: mgt
    storage:
    - allocationCount: 1
      name: ${RABBIT_ARRAY[0]}
  - allocationSize: $MDT_SIZE_IN_BYTES
    label: mdt
    storage:
    - allocationCount: 1
      name: ${RABBIT_ARRAY[$MDT_IDX]}
EOF

  case $RABBIT_COUNT in
  1)
cat >> $SERVERS_PATCH << EOF
  - allocationSize: 1073741824
    label: ost
    storage:
    - allocationCount: $ALLOCATION_COUNT
      name: ${RABBIT_ARRAY[0]}
EOF
  ;;

  2)
cat >> $SERVERS_PATCH << EOF
  - allocationSize: 1073741824
    label: ost
    storage:
    - allocationCount: $ALLOCATION_COUNT
      name: ${RABBIT_ARRAY[1]}
EOF
  ;;

  *)
    echo Building ost hosts...
    ost_section=$(build_ost_hosts)

cat >> $SERVERS_PATCH << EOF
$ost_section
EOF
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
