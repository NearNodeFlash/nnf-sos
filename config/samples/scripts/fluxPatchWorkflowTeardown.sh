#!/bin/bash

PROG=$(basename "$0")

printhelp()
{
  echo "Usage: $PROG -w WORKFLOW "
  echo
  echo "  -w WORKFLOW  Resource name of the workflow."
  echo
  exit 2
}

OPTIND=1
while getopts "hw:" opt
do
  case $opt in
  w) WORKFLOW=$OPTARG ;;
  *) printhelp ;;
  esac
done
shift $((OPTIND-1))

if [[ -z $WORKFLOW ]]
then
  echo "$PROG: Must specify -w"
  exit 2
fi

patch_workflow()
{
  desiredState="teardown"
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

# Starting timing here
patch_workflow

exit $?
