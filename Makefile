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

# Default container tool to use.
#   To use podman:
#   $ DOCKER=podman make docker-build
DOCKER ?= docker

# VERSION defines the project version for the bundle.
# Update this value when you upgrade the version of your project.
# To re-generate a bundle for another specific version without changing the standard setup, you can:
# - use the VERSION as arg of the bundle target (e.g make bundle VERSION=0.0.2)
# - use environment variables to overwrite this value (e.g export VERSION=0.0.2)
VERSION ?= $(shell sed 1q .version)

# CHANNELS define the bundle channels used in the bundle.
# Add a new line here if you would like to change its default config. (E.g CHANNELS = "preview,fast,stable")
# To re-generate a bundle for other specific channels without changing the standard setup, you can:
# - use the CHANNELS as arg of the bundle target (e.g make bundle CHANNELS=preview,fast,stable)
# - use environment variables to overwrite this value (e.g export CHANNELS="preview,fast,stable")
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif

# DEFAULT_CHANNEL defines the default channel used in the bundle.
# Add a new line here if you would like to change its default config. (E.g DEFAULT_CHANNEL = "stable")
# To re-generate a bundle for any other default channel without changing the default setup, you can:
# - use the DEFAULT_CHANNEL as arg of the bundle target (e.g make bundle DEFAULT_CHANNEL=stable)
# - use environment variables to overwrite this value (e.g export DEFAULT_CHANNEL="stable")
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

# GIT_TAG is the SHA of the current commit
GIT_TAG=$(shell git rev-parse --short HEAD)

# IMAGE_TAG_BASE defines the docker.io namespace and part of the image name for remote images.
# This variable is used to construct full image tags for bundle and catalog images.
#
# For example, running 'make bundle-build bundle-push catalog-build catalog-push' will build and push both
# cray.com/nnf-sos-bundle:$VERSION and cray.com/nnf-sos-catalog:$VERSION.
IMAGE_TAG_BASE ?= ghcr.io/nearnodeflash/nnf-sos

# BUNDLE_IMG defines the image:tag used for the bundle.
# You can use it as an arg. (E.g make bundle-build BUNDLE_IMG=<some-registry>/<project-name-bundle>:<tag>)
BUNDLE_IMG ?= $(IMAGE_TAG_BASE)-bundle:v$(VERSION)

# Image URL to use all building/pushing image targets
IMG ?= $(IMAGE_TAG_BASE):$(VERSION)

# Jenkins behaviors
# pipeline_service builds its target docker image and stores it into 1 of 3 destination folders.
# The behavior of where pipeline_service puts a build is dictated by name of the branch Jenkins
# is building. (See https://github.hpe.com/hpe/hpc-dst-jenkins-shared-library/vars/getArtiRepository.groovy)
#
#         arti.dev.cray.com folder                                  Contents
# ---------------------------------------------  -----------------------------------------------------
# arti.dev.cray.com/rabsw-docker-master-local    master branch builds
# arti.dev.cray.com/rabsw-docker-stable-local    release branch builds
# arti.dev.cray.com/rabsw-docker-unstable-local  non-master && non-release branches, i.e. everything else
#
# pipeline_service tags the build with the following tag:
#
#   VERSION = sh(returnStdout: true, script: "cat .version").trim()   // .version file here
#   def buildDate = new Date().format( 'yyyyMMddHHmmss' )
#   BUILD_DATE = "${buildDate}"
#   GIT_TAG = sh(returnStdout: true, script: "git rev-parse --short HEAD").trim()
#
#   IMAGE_TAG = getDockerImageTag(version: "${VERSION}", buildDate: "${BUILD_DATE}", gitTag: "${GIT_TAG}", gitBranch: "${GIT_BRANCH}")
#
# Because of the build date stamp, this tag is a bit difficult to use.
# NOTE: master-local and stable-local have a 'latest' tag that can be used to fetch the latest of either
#       the master or release branch.

# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true,preserveUnknownFields=false"

# Tell Kustomize to deploy the default config, or an overlay.
# To use the 'craystack' overlay:
#   export KUBECONFIG=/my/craystack/kubeconfig.file
#   make deploy OVERLAY=craystack
#
# To use the 'dp0' overlay:
#   export KUBECONFIG=/my/dp0/kubeconfig.file
#   make deploy OVERLAY=dp0
OVERLAY ?= kind

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

fmt: ## Run go fmt against code.
	go fmt ./...

vet: ## Run go vet against code.
	go vet ./...

##@ Test

ENVTEST_ASSETS_DIR=$(shell pwd)/testbin
# Explicitly specifying directories to test here to avoid running tests in .dws-operator for the time being.
# ./controllers/...
# ./api/...
# Below is a list of ginkgo test flags that may be used to generate different test patterns.
# Specifying 'count=1' is the idiomatic way to disable test caching
#   -ginkgo.debug
#         If set, ginkgo will emit node output to files when running in parallel.
#   -ginkgo.dryRun
#         If set, ginkgo will walk the test hierarchy without actually running anything.  Best paired with -v.
#   -ginkgo.failFast
#         If set, ginkgo will stop running a test suite after a failure occurs.
#   -ginkgo.failOnPending
#         If set, ginkgo will mark the test suite as failed if any specs are pending.
#   -ginkgo.flakeAttempts int
#         Make up to this many attempts to run each spec. Please note that if any of the attempts succeed, the suite will not be failed. But any failures will still be recorded. (default 1)
#   -ginkgo.focus value
#         If set, ginkgo will only run specs that match this regular expression. Can be specified multiple times, values are ORed.
#   -ginkgo.noColor
#         If set, suppress color output in default reporter.
#   -ginkgo.noisyPendings
#         If set, default reporter will shout about pending tests. (default true)
#   -ginkgo.noisySkippings
#         If set, default reporter will shout about skipping tests. (default true)
#   -ginkgo.parallel.node int
#         This worker node's (one-indexed) node number.  For running specs in parallel. (default 1)
#   -ginkgo.parallel.streamhost string
#         The address for the server that the running nodes should stream data to.
#   -ginkgo.parallel.synchost string
#         The address for the server that will synchronize the running nodes.
#   -ginkgo.parallel.total int
#         The total number of worker nodes.  For running specs in parallel. (default 1)
#   -ginkgo.progress
#         If set, ginkgo will emit progress information as each spec runs to the GinkgoWriter.
#   -ginkgo.randomizeAllSpecs
#         If set, ginkgo will randomize all specs together.  By default, ginkgo only randomizes the top level Describe, Context and When groups.
#   -ginkgo.regexScansFilePath
#         If set, ginkgo regex matching also will look at the file path (code location).
#   -ginkgo.reportFile string
#         Override the default reporter output file path.
#   -ginkgo.reportPassed
#         If set, default reporter prints out captured output of passed tests.
#   -ginkgo.seed int
#         The seed used to randomize the spec suite. (default 1635181882)
#   -ginkgo.skip value
#         If set, ginkgo will only run specs that do not match this regular expression. Can be specified multiple times, values are ORed.
#   -ginkgo.skipMeasurements
#         If set, ginkgo will skip any measurement specs.
#   -ginkgo.slowSpecThreshold float
#         (in seconds) Specs that take longer to run than this threshold are flagged as slow by the default reporter. (default 5)
#   -ginkgo.succinct
#         If set, default reporter prints out a very succinct report
#   -ginkgo.trace
#         If set, default reporter prints out the full stack trace when a failure occurs
#   -ginkgo.v
#         If set, default reporter print out all specs as they begin.
#
#  To see the test results as they execute:
# export KUBEBUILDER_ASSETS=$(pwd)/testbin/bin; export GOMEGA_DEFAULT_EVENTUALLY_TIMEOUT="20s"; export GOMEGA_DEFAULT_EVENTUALLY_INTERVAL="100ms"; go test -v -count=1 -cover ./controllers/... -args -ginkgo.v -ginkgo.failFast -ginkgo.randomizeAllSpecs

container-unit-test: ## Build docker image with the manager and execute unit tests.
	${DOCKER} build -f Dockerfile --label $(IMAGE_TAG_BASE)-$@:$(VERSION)-$@ -t $(IMAGE_TAG_BASE)-$@:$(VERSION) --target testing .
	${DOCKER} run --rm -t --name $@-nnf-sos  $(IMAGE_TAG_BASE)-$@:$(VERSION)

# Lengthen the default Eventually timeout to try to account for CI/CD issues
# https://onsi.github.io/gomega/ says: "By default, Eventually will poll every 10 milliseconds for up to 1 second"
EVENTUALLY_TIMEOUT ?= "20s"
EVENTUALLY_INTERVAL ?= "100ms"
TESTDIRS ?= controllers api
FAILFAST ?= no
test: manifests generate fmt vet ## Run tests.
	find controllers -name "*.db" -type d -exec rm -rf {} +
	mkdir -p ${ENVTEST_ASSETS_DIR}
	source test-tools.sh; prefix_webhook_names config/webhook ${ENVTEST_ASSETS_DIR}/webhook

	test -f ${ENVTEST_ASSETS_DIR}/setup-envtest.sh || curl -sSLo ${ENVTEST_ASSETS_DIR}/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.7.2/hack/setup-envtest.sh
	if [[ "${FAILFAST}" == yes ]]; then \
		failfast="-ginkgo.failFast"; \
	fi; \
	set -o errexit; \
	for subdir in ${TESTDIRS}; do \
		export GOMEGA_DEFAULT_EVENTUALLY_TIMEOUT=${EVENTUALLY_TIMEOUT}; \
		export GOMEGA_DEFAULT_EVENTUALLY_INTERVAL=${EVENTUALLY_INTERVAL}; \
		export WEBHOOK_DIR=${ENVTEST_ASSETS_DIR}/webhook; \
		source ${ENVTEST_ASSETS_DIR}/setup-envtest.sh; fetch_envtest_tools $(ENVTEST_ASSETS_DIR); setup_envtest_env $(ENVTEST_ASSETS_DIR); go test -v ./$$subdir/... -coverprofile cover.out -args -ginkgo.v -ginkgo.progress $$failfast; \
    	done

##@ Build

build: generate fmt vet ## Build manager binary.
	go build -o bin/manager main.go

run: manifests generate fmt vet ## Run a controller from your host.
	go run ./main.go

docker-build: test ## Build docker image with the manager.
	time ${DOCKER} build --build-arg FAILFAST -t ${IMG} .

docker-push: ## Push docker image with the manager.
	${DOCKER} push ${IMG}

kind-push: ## Push docker image to kind
	kind load docker-image --nodes `kubectl get node --no-headers -o custom-columns=":metadata.name" | paste -d, -s -` ${IMG}
	${DOCKER} pull gcr.io/kubebuilder/kube-rbac-proxy:v0.8.0
	kind load docker-image --nodes `kubectl get node -l cray.nnf.manager=true --no-headers -o custom-columns=":metadata.name" | paste -d, -s -` gcr.io/kubebuilder/kube-rbac-proxy:v0.8.0

##@ Deployment

install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

deploy: kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	./deploy.sh deploy $(KUSTOMIZE) $(IMG) $(OVERLAY)

undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
	./deploy.sh undeploy $(KUSTOMIZE) $(IMG) $(OVERLAY)


CONTROLLER_GEN = $(shell pwd)/bin/controller-gen
controller-gen: ## Download controller-gen locally if necessary.
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.4.1)

KUSTOMIZE = $(shell pwd)/bin/kustomize
kustomize: ## Download kustomize locally if necessary.
	$(call go-get-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v3@v3.8.7)

# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go get $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef

.PHONY: bundle
bundle: manifests kustomize ## Generate bundle manifests and metadata, then validate generated files.
	operator-sdk generate kustomize manifests -q
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG)
	$(KUSTOMIZE) build config/manifests | operator-sdk generate bundle -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)
	operator-sdk bundle validate ./bundle

.PHONY: bundle-build
bundle-build: ## Build the bundle image.
	${DOCKER} build -f bundle.Dockerfile -t $(BUNDLE_IMG) .

.PHONY: bundle-push
bundle-push: ## Push the bundle image.
	$(MAKE) docker-push IMG=$(BUNDLE_IMG)

.PHONY: opm
OPM = ./bin/opm
opm: ## Download opm locally if necessary.
ifeq (,$(wildcard $(OPM)))
ifeq (,$(shell which opm 2>/dev/null))
	@{ \
	set -e ;\
	mkdir -p $(dir $(OPM)) ;\
	OS=$(shell go env GOOS) && ARCH=$(shell go env GOARCH) && \
	curl -sSLo $(OPM) https://github.com/operator-framework/operator-registry/releases/download/v1.15.1/$${OS}-$${ARCH}-opm ;\
	chmod +x $(OPM) ;\
	}
else
OPM = $(shell which opm)
endif
endif

# A comma-separated list of bundle images (e.g. make catalog-build BUNDLE_IMGS=example.com/operator-bundle:v0.1.0,example.com/operator-bundle:v0.2.0).
# These images MUST exist in a registry and be pull-able.
BUNDLE_IMGS ?= $(BUNDLE_IMG)

# The image tag given to the resulting catalog image (e.g. make catalog-build CATALOG_IMG=example.com/operator-catalog:v0.2.0).
CATALOG_IMG ?= $(IMAGE_TAG_BASE)-catalog:v$(VERSION)

# Set CATALOG_BASE_IMG to an existing catalog image tag to add $BUNDLE_IMGS to that image.
ifneq ($(origin CATALOG_BASE_IMG), undefined)
FROM_INDEX_OPT := --from-index $(CATALOG_BASE_IMG)
endif

# Build a catalog image by adding bundle images to an empty catalog using the operator package manager tool, 'opm'.
# This recipe invokes 'opm' in 'semver' bundle add mode. For more information on add modes, see:
# https://github.com/operator-framework/community-operators/blob/7f1438c/docs/packaging-operator.md#updating-your-existing-operator
.PHONY: catalog-build
catalog-build: opm ## Build a catalog image.
	$(OPM) index add --container-tool ${DOCKER} --mode semver --tag $(CATALOG_IMG) --bundles $(BUNDLE_IMGS) $(FROM_INDEX_OPT)

# Push the catalog image.
.PHONY: catalog-push
catalog-push: ## Push a catalog image.
	$(MAKE) docker-push IMG=$(CATALOG_IMG)
