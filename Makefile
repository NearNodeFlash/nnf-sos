# Copyright 2021-2025 Hewlett Packard Enterprise Development LP
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

# VERSION defines the project version for the bundle.
# Update this value when you upgrade the version of your project.
# To re-generate a bundle for another specific version without changing the standard setup, you can:
# - use the VERSION as arg of the bundle target (e.g make bundle VERSION=0.0.2)
# - use environment variables to overwrite this value (e.g export VERSION=0.0.2)
# NOTE: git-version-gen will generate a value for VERSION, unless you override it.

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

# The NNF-MFU container image to use in NNFContainerProfile resources.
NNFMFU_TAG_BASE ?= ghcr.io/nearnodeflash/nnf-mfu
NNFMFU_VERSION ?= master

# BUNDLE_IMG defines the image:tag used for the bundle.
# You can use it as an arg. (E.g make bundle-build BUNDLE_IMG=<some-registry>/<project-name-bundle>:<tag>)

# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.29.0

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

# Tell Kustomize to deploy the default config, or an overlay.
# To use the 'craystack' overlay:
#   export KUBECONFIG=/my/craystack/kubeconfig.file
#   make deploy OVERLAY=craystack
#
# To use the 'dp0' overlay:
#   export KUBECONFIG=/my/dp0/kubeconfig.file
#   make deploy OVERLAY=dp0
OVERLAY ?= kind

# Tell Kustomize to deploy the default examples config, or an overlay
OVERLAY_EXAMPLES ?= examples

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# CONTAINER_TOOL defines the container tool to be used for building images.
# Be aware that the target commands are only tested with Docker which is
# scaffolded by default. However, you might want to replace it to use other
# tools. (i.e. podman)
CONTAINER_TOOL ?= docker

# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

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
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

fmt: ## Run go fmt against code.
	go fmt ./...

vet: ## Run go vet against code.
	go vet ./...

##@ Test

# Explicitly specifying directories to test here to avoid running tests in .dws-operator for the time being.
# ./internal/...
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

container-unit-test: VERSION ?= $(shell cat .version)
container-unit-test: .version ## Build docker image with the manager and execute unit tests.
	${CONTAINER_TOOL} build -f Dockerfile --label $(IMAGE_TAG_BASE)-$@:$(VERSION)-$@ -t $(IMAGE_TAG_BASE)-$@:$(VERSION) --target testing .
	${CONTAINER_TOOL} run --rm -t --name $@-nnf-sos  $(IMAGE_TAG_BASE)-$@:$(VERSION)

ENVTEST_ASSETS_DIR=$(shell pwd)/testbin
# Lengthen the default Eventually timeout to try to account for CI/CD issues
# https://onsi.github.io/gomega/ says: "By default, Eventually will poll every 10 milliseconds for up to 1 second"
EVENTUALLY_TIMEOUT ?= "20s"
EVENTUALLY_INTERVAL ?= "100ms"
TESTDIRS ?= internal api pkg github/cluster-api
FAILFAST ?= no
test: manifests generate fmt vet envtest ## Run tests.
	find internal -name "*.db" -type d -exec rm -rf {} +
	./hack/prefix-webhook-names.sh config/webhook ${ENVTEST_ASSETS_DIR}/webhook-nnf nnf
	./hack/prefix-webhook-names.sh vendor/github.com/NearNodeFlash/lustre-fs-operator/config/webhook ${ENVTEST_ASSETS_DIR}/webhook-lus lus
	./hack/prefix-webhook-names.sh vendor/github.com/DataWorkflowServices/dws/config/webhook ${ENVTEST_ASSETS_DIR}/webhook-dws dws
	if [[ "${FAILFAST}" == yes ]]; then \
		failfast="-ginkgo.fail-fast"; \
	fi; \
	set -o errexit; \
	export GOMEGA_DEFAULT_EVENTUALLY_TIMEOUT=${EVENTUALLY_TIMEOUT}; \
	export GOMEGA_DEFAULT_EVENTUALLY_INTERVAL=${EVENTUALLY_INTERVAL}; \
	export WEBHOOK_DIRS=${ENVTEST_ASSETS_DIR}/webhook-nnf:${ENVTEST_ASSETS_DIR}/webhook-lus:${ENVTEST_ASSETS_DIR}/webhook-dws; \
	for subdir in ${TESTDIRS}; do \
		KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path --bin-dir $(LOCALBIN))" go test -v ./$$subdir/... -coverprofile cover-$$(basename $$subdir.out) -ginkgo.v $$failfast; \
	done; \
	rm -rf internal/controller/nnf.db

##@ Build
RPM_PLATFORM ?= linux/amd64
RPM_TARGET ?= x86_64
.PHONY: build-daemon-rpm
build-daemon-rpm: RPM_VERSION ?= $(shell ./git-version-gen | sed -e 's/\-.*//')
build-daemon-rpm: $(RPMBIN)
build-daemon-rpm: fmt vet ## Build standalone clientmount binary and its rpm
	${CONTAINER_TOOL} build --platform=$(RPM_PLATFORM) --build-arg="RPMTARGET=$(RPM_TARGET)" --build-arg="RPMVERSION=$(RPM_VERSION)" --output=type=local,dest=$(RPMBIN) -f mount-daemon/Dockerfile.rpmbuild .

.PHONY: build-daemon-local
build-daemon-local: GOOS = $(shell go env GOOS)
build-daemon-local: GOARCH = $(shell go env GOARCH)
build-daemon-local: build-daemon-with

.PHONY: build-daemon
build-daemon: GOOS ?= linux
build-daemon: GOARCH ?= amd64
build-daemon: build-daemon-with

.PHONY: build-daemon-with
build-daemon-with: RPM_VERSION ?= $(shell ./git-version-gen)
build-daemon-with: PACKAGE = github.com/NearNodeFlash/nnf-sos/mount-daemon/version
build-daemon-with: $(LOCALBIN)
build-daemon-with: fmt vet ## Build standalone clientmount binary
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -ldflags="-X '$(PACKAGE).version=$(RPM_VERSION)'" -o bin/clientmountd mount-daemon/main.go

build: generate fmt vet ## Build manager binary.
	CGO_ENABLED=0 go build -o bin/manager cmd/main.go

run: manifests generate fmt vet ## Run a controller from your host.
	CGO_ENABLED=0 go run cmd/main.go

# If you wish to build the manager image targeting other platforms you can use the --platform flag.
# (i.e. docker build --platform linux/arm64). However, you must enable docker buildKit for it.
# More info: https://docs.docker.com/develop/develop-images/build_enhancements/
.PHONY: docker-build
docker-build: VERSION ?= $(shell cat .version)
docker-build: .version ## Build docker image with the manager.
	time ${CONTAINER_TOOL} build --build-arg FAILFAST -t $(IMAGE_TAG_BASE):$(VERSION) .

.PHONY: docker-push
docker-push: VERSION ?= $(shell cat .version)
docker-push: .version ## Push docker image with the manager.
	${CONTAINER_TOOL} push $(IMAGE_TAG_BASE):$(VERSION)

# PLATFORMS defines the target platforms for the manager image be built to provide support to multiple
# architectures. (i.e. make docker-buildx IMG=myregistry/mypoperator:0.0.1). To use this option you need to:
# - be able to use docker buildx. More info: https://docs.docker.com/build/buildx/
# - have enabled BuildKit. More info: https://docs.docker.com/develop/develop-images/build_enhancements/
# - be able to push the image to your registry (i.e. if you do not set a valid value via IMG=<myregistry/image:<tag>> then the export will fail)
# To adequately provide solutions that are compatible with multiple platforms, you should consider using this option.
PLATFORMS ?= linux/arm64,linux/amd64,linux/s390x,linux/ppc64le
.PHONY: docker-buildx
docker-buildx: VERSION ?= $(shell cat .version)
docker-buildx: ## Build and push docker image for the manager for cross-platform support
	# copy existing Dockerfile and insert --platform=${BUILDPLATFORM} into Dockerfile.cross, and preserve the original Dockerfile
	sed -e '1 s/\(^FROM\)/FROM --platform=\$$\{BUILDPLATFORM\}/; t' -e ' 1,// s//FROM --platform=\$$\{BUILDPLATFORM\}/' Dockerfile > Dockerfile.cross
	- $(CONTAINER_TOOL) buildx create --name project-v3-builder
	$(CONTAINER_TOOL) buildx use project-v3-builder
	- $(CONTAINER_TOOL) buildx build --push --platform=$(PLATFORMS) --tag $(IMAGE_TAG_BASE):$(VERSION) -f Dockerfile.cross .
	- $(CONTAINER_TOOL) buildx rm project-v3-builder
	rm Dockerfile.cross

kind-push: VERSION ?= $(shell cat .version)
kind-push: .version ## Push docker image to kind
	kind load docker-image $(IMAGE_TAG_BASE):$(VERSION)

##@ Deployment

install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl delete --ignore-not-found -f -

edit-image: VERSION ?= $(shell cat .version)
edit-image: .version
	$(KUSTOMIZE_IMAGE_TAG) config/begin $(OVERLAY) $(IMAGE_TAG_BASE) $(VERSION) $(VERSION)
	$(KUSTOMIZE_IMAGE_TAG) config/begin-examples $(OVERLAY_EXAMPLES) $(NNFMFU_TAG_BASE) $(NNFMFU_VERSION) $(VERSION)

deploy: kustomize edit-image ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	./deploy.sh deploy $(KUSTOMIZE) config/begin config/begin-examples

undeploy: VERSION ?= $(shell cat .version)
undeploy: .version kustomize ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
	./deploy.sh undeploy $(KUSTOMIZE) config/$(OVERLAY) config/$(OVERLAY_EXAMPLES)

# Let .version be phony so that a git update to the workarea can be reflected
# in it each time it's needed.
.PHONY: .version
.version: ## Uses the git-version-gen script to generate a tag version
	./git-version-gen --fallback `git rev-parse HEAD` > .version

clean:
	rm -f .version

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

.PHONY: clean-bin
clean-bin:
	if [[ -d $(LOCALBIN) ]]; then \
	  chmod -R u+w $(LOCALBIN) && rm -rf $(LOCALBIN); \
	fi

## Location to place rpms
RPMBIN ?= $(shell pwd)/rpms
$(RPMBIN):
	mkdir $(RPMBIN)

.PHONY: clean-rpmbin
clean-rpmbin:
	if [[ -d $(RPMBIN) ]]; then \
	  rm -rf $(RPMBIN); \
	fi

## Tool Binaries
KUSTOMIZE_IMAGE_TAG ?= ./hack/make-kustomization.sh
GO_INSTALL := ./github/cluster-api/scripts/go_install.sh
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest

CONVERSION_GEN_BIN := conversion-gen
CONVERSION_GEN := $(LOCALBIN)/$(CONVERSION_GEN_BIN)
CONVERSION_GEN_PKG := k8s.io/code-generator/cmd/conversion-gen

CONVERSION_VERIFIER_BIN := conversion-verifier
CONVERSION_VERIFIER := $(LOCALBIN)/$(CONVERSION_VERIFIER_BIN)
CONVERSION_VERIFIER_PKG := sigs.k8s.io/cluster-api/hack/tools/conversion-verifier

## Tool Versions
KUSTOMIZE_VERSION ?= v5.5.0
CONTROLLER_TOOLS_VERSION ?= v0.16.5
CONVERSION_GEN_VER := v0.31.5

# Can be "latest", but cannot be a tag, such as "v1.3.3".  However, it will
# work with the short-form git commit rev that has been tagged.
#CONVERSION_VERIFIER_VER := 09030092b # v1.3.3
CONVERSION_VERIFIER_VER := b0284d8b # v1.7.3

KUSTOMIZE_INSTALL_SCRIPT ?= "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"
.PHONY: kustomize
kustomize: $(LOCALBIN) ## Download kustomize locally if necessary.
	if [[ ! -s $(LOCALBIN)/kustomize || ! $$($(LOCALBIN)/kustomize version) =~ $(KUSTOMIZE_VERSION) ]]; then \
	  rm -f $(LOCALBIN)/kustomize && \
	  { curl -s $(KUSTOMIZE_INSTALL_SCRIPT) | bash -s -- $(subst v,,$(KUSTOMIZE_VERSION)) $(LOCALBIN); }; \
	fi

.PHONY: controller-gen
controller-gen: $(LOCALBIN) ## Download controller-gen locally if necessary.
	if [[ ! -s $(LOCALBIN)/controller-gen || $$($(LOCALBIN)/controller-gen --version | awk '{print $$2}') != $(CONTROLLER_TOOLS_VERSION) ]]; then \
	  rm -f $(LOCALBIN)/controller-gen && GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION); \
	fi

.PHONY: $(CONVERSION_GEN_BIN)
$(CONVERSION_GEN_BIN): $(CONVERSION_GEN) ## Build a local copy of conversion-gen.

## We are forcing a rebuild of conversion-gen via PHONY so that we're always using an up-to-date version.
## We can't use a versioned name for the binary, because that would be reflected in generated files.
.PHONY: $(CONVERSION_GEN)
$(CONVERSION_GEN): $(LOCALBIN) # Build conversion-gen from tools folder.
	GOBIN=$(LOCALBIN) $(GO_INSTALL) $(CONVERSION_GEN_PKG) $(CONVERSION_GEN_BIN) $(CONVERSION_GEN_VER)

.PHONY: generate-go-conversions
# The SRC_DIRS value is a space-separated list of paths to old versions.
# The --input-dirs value is a single path item; specify multiple --input-dirs
# parameters if you have multiple old versions.
SRC_DIRS=./api/v1alpha4 ./api/v1alpha5 ./api/v1alpha6
generate-go-conversions: $(CONVERSION_GEN) ## Generate conversions go code
	$(MAKE) clean-generated-conversions SRC_DIRS="$(SRC_DIRS)"
	$(CONVERSION_GEN) \
		--output-file=zz_generated.conversion.go \
		--go-header-file=./hack/boilerplate.go.txt \
		$(SRC_DIRS)

.PHONY: clean-generated-conversions
clean-generated-conversions: ## Remove files generated by conversion-gen from the mentioned dirs
	(IFS=','; for i in $(SRC_DIRS); do find $$i -type f -name 'zz_generated.conversion*' -exec rm -f {} \;; done)

## We are forcing a rebuild of conversion-verifier via PHONY so that we're always using an up-to-date version.
.PHONY: $(CONVERSION_VERIFIER)
$(CONVERSION_VERIFIER): $(LOCALBIN) # Build conversion-verifier from tools folder.
	GOBIN=$(LOCALBIN) $(GO_INSTALL) $(CONVERSION_VERIFIER_PKG) $(CONVERSION_VERIFIER_BIN) $(CONVERSION_VERIFIER_VER)

.PHONY: $(CONVERSION_VERIFIER_BIN)
$(CONVERSION_VERIFIER_BIN): $(CONVERSION_VERIFIER) ## Build a local copy of conversion-verifier.

## -------------
## verify
## -------------

ALL_VERIFY_CHECKS = gen conversions

.PHONY: verify
verify: $(addprefix verify-,$(ALL_VERIFY_CHECKS)) ## Run all verify-* targets

.PHONY: verify-gen
verify-gen: generate manifests generate-go-conversions ## Verify go generated files are up to date
	@if !(git diff --quiet HEAD); then \
		git diff; \
		echo "generated files are out of date, run make generate"; exit 1; \
	fi

.PHONY: verify-conversions
verify-conversions: $(CONVERSION_VERIFIER)  ## Verifies expected API conversion are in place
	$(CONVERSION_VERIFIER)

.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST): $(LOCALBIN)
	test -s $(LOCALBIN)/setup-envtest || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@release-0.17

.PHONY: bundle
bundle: manifests kustomize ## Generate bundle manifests and metadata, then validate generated files.
	operator-sdk generate kustomize manifests -q
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG)
	$(KUSTOMIZE) build config/manifests | operator-sdk generate bundle -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)
	operator-sdk bundle validate ./bundle

.PHONY: bundle-build
bundle-build: VERSION ?= $(shell cat .version)
bundle-build: BUNDLE_IMG ?= $(IMAGE_TAG_BASE)-bundle:v$(VERSION)
bundle-build: .version ## Build the bundle image.
	${CONTAINER_TOOL} build -f bundle.Dockerfile -t $(BUNDLE_IMG) .

.PHONY: bundle-push
bundle-push: VERSION ?= $(shell cat .version)
bundle-push: BUNDLE_IMG ?= $(IMAGE_TAG_BASE)-bundle:v$(VERSION)
bundle-push: .version ## Push the bundle image.
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
	$(OPM) index add --container-tool ${CONTAINER_TOOL} --mode semver --tag $(CATALOG_IMG) --bundles $(BUNDLE_IMGS) $(FROM_INDEX_OPT)

# Push the catalog image.
.PHONY: catalog-push
catalog-push: ## Push a catalog image.
	$(MAKE) docker-push IMG=$(CATALOG_IMG)
