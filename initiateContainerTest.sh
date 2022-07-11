#!/usr/bin/env bash

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

CWD=`pwd`
echo "Running in $CWD, setting up envtest."

export ENVTEST_ASSETS_DIR=/nnf/testbin
export GOMEGA_DEFAULT_EVENTUALLY_TIMEOUT=20s
export GOMEGA_DEFAULT_EVENTUALLY_INTERVAL=100ms
mkdir -p ${ENVTEST_ASSETS_DIR}
test -f ${ENVTEST_ASSETS_DIR}/setup-envtest.sh || curl -sSLo ${ENVTEST_ASSETS_DIR}/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.7.2/hack/setup-envtest.sh
source ${ENVTEST_ASSETS_DIR}/setup-envtest.sh
fetch_envtest_tools ${ENVTEST_ASSETS_DIR}
setup_envtest_env ${ENVTEST_ASSETS_DIR}

echo Running unit tests

go test ./... -coverprofile cover.out -args -ginkgo.v -ginkgo.progress -ginkgo.failFast |/usr/bin/tee results.txt
cat results.txt

grep FAIL results.txt && echo "Unit tests failure" && rm results.txt && exit 1

echo "Unit tests successful" && rm results.txt

