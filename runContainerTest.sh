#!/usr/bin/env bash 

CWD=`pwd`
echo "Running in $CWD, setting up envtest"

export ENVTEST_ASSETS_DIR=/nnf/testbin
mkdir -p ${ENVTEST_ASSETS_DIR}
test -f ${ENVTEST_ASSETS_DIR}/setup-envtest.sh || curl -sSLo ${ENVTEST_ASSETS_DIR}/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.7.2/hack/setup-envtest.sh
source ${ENVTEST_ASSETS_DIR}/setup-envtest.sh
fetch_envtest_tools ${ENVTEST_ASSETS_DIR}
setup_envtest_env ${ENVTEST_ASSETS_DIR}

echo Running unit tests

go test ./... -coverprofile cover.out > results.txt
cat results.txt

grep FAIL results.txt && echo "Unit tests failure" && rm results.txt && exit 1 

echo "Unit tests successful" && rm results.txt

