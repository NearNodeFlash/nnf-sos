#!/usr/bin/env bash

# Copyright 2022-2024 Hewlett Packard Enterprise Development LP
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

# To allow suite_test.go to run webhooks from multiple repos, we have to
# adjust the webhook names. They were meant to be processed by kustomize,
# which would have prepended "namePrefix" to the names, but envtest doesn't
# use that. This tool copies the webhook config and adds the name prefix.

SOURCE_DIR=$1
DEST_DIR=$2
GROUP=$3

mkdir -p $DEST_DIR
cp $SOURCE_DIR/* $DEST_DIR
sed -i.bak -e "s/validating-webhook-configuration/$GROUP-validating-webhook-configuration/" -e "s/mutating-webhook-configuration/$GROUP-mutating-webhook-configuration/" $DEST_DIR/manifests.yaml
rm $DEST_DIR/manifests.yaml.bak

sed -i.bak -e "s/webhook-service/$GROUP-webhook-service/" $DEST_DIR/service.yaml
rm $DEST_DIR/service.yaml.bak

if [[ -f $DEST_DIR/service_account.yaml ]]; then
    sed -i.bak -e "s/webhook/$GROUP-webhook/" $DEST_DIR/service_account.yaml
    rm $DEST_DIR/service_account.yaml.bak
fi

exit 0
