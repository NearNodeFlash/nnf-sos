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
