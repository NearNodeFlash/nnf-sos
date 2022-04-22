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

# Use `source`` to push the environment variables back into the calling shell
# "source ./setDevVersion.sh"

# Command to access arti and list all of the docker images there filtering out the tag for the
# version matching the current branch's git SHA
unset VERSION
unset IMAGE_TAG_BASE

# Setup some artifactory paths to the developer and master branch locations
ARTI_URL_DEV=https://arti.dev.cray.com/artifactory/rabsw-docker-unstable-local/cray-dp-nnf-sos/
ARTI_URL_MASTER=https://arti.dev.cray.com/artifactory/rabsw-docker-master-local/cray-dp-nnf-sos/

# Retrieve the name of the current branch. If we are in detached HEAD state, assume it is master.
CURRENT_BRANCH_NAME=$(git rev-parse --abbrev-ref HEAD)

# Depending on whether we are on the master branch or not, setup for deployment from artifactory
# NOTE: Detached HEAD state is assumed to match 'master'
if [[ "$CURRENT_BRANCH_NAME" == "master" ]] || [[ "$CURRENT_BRANCH_NAME" == "HEAD" ]]; then
    ARTI_URL="$ARTI_URL_MASTER"
else    # not on the master branch
    ARTI_URL="$ARTI_URL_DEV"

    # Deploying a developer build requires the IMAGE_TAG_BASE to change as well.
    # Master branch is the default, so we don't change it when we are on the master branch.
    IMAGE_TAG_BASE=arti.dev.cray.com/rabsw-docker-unstable-local/cray-dp-nnf-sos
    export IMAGE_TAG_BASE
    echo IMAGE_TAG_BASE: "$IMAGE_TAG_BASE"
fi

# Locate the container tags in arti to set the VERSION environment variable
# which allows us to run `make deploy` and pull the correct version from ARTI.
LATEST_LOCAL_COMMIT=$(git rev-parse --short HEAD)
ARTI_TAG=$(wget --spider --recursive --no-parent -l1 "$ARTI_URL" 2>&1 | grep -- ^-- | awk '{print $3}' | grep "$LATEST_LOCAL_COMMIT" | tail -1)
VERSION=$(basename "$ARTI_TAG")
export VERSION
echo VERSION: "$VERSION"
