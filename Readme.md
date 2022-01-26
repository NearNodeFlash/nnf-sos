# NNF Storage Orchestration Services (SOS)

NNF SOS is a collection of Kubernetes Custom Resource Definitions (CRDs) and associated controllers that permit the creation, management, and destruction of storage on an NNF cluster.

## Setup

---
This project depends on additional CRDs present in other repositories. Those repositories are joined to ***nnf-sos*** as `git submodules` to give ***nnf-sos*** access to those CRDs. At the present time, the following directories are submodules included from other repos:

```bash
$ git submodule status
 31ee9c7ce7be6f4170231afea68d5629f3bdacb5 .dws-operator (remotes/origin/HEAD)
+40c966e92bf48bac8fa497f30bdffdb229c2cf15 .nnf-dm (remotes/origin/HEAD)
```

## k8s cluster setup

***nnf-ec*** (NNF Element Controller) is a vendor-ed module within within ***nnf-sos*** that provides hardware access functions for the NNF hardware. In order for this code to function correctly within a k8s pod when it performs storage operations on Rabbit hardware, a set of base OS configuration operations is required. Those operations are encapsulated in the `/rabbit-os-mods/nnf-ec.sh` script. This script sets up various OS configuration files on Rabbit that are required for nnf-ec.

### Clone

To clone this project, use the additional `--recurse-submodules` option to retrieve its submodules:

```bash
git clone --recurse-submodules git@github.hpe.com:hpe/hpc-rabsw-nnf-sos.git
```

### Update submodules

To update your submodules as external repos change run:

```bash
git submodule update --remote
```

## Build

---

The makefile controls most functions within ***nnf-sos*** including: Development, Test, Build, and Deployment. The make targets available may be listed with:

```bash
make help
```

In your local machine's environment, `make` builds the executable. This is useful as a sanity check to verify that code compiles and it is much quicker than the docker builds.

```bash
make
```

### Docker builds

Docker containers provide an isolated environment in which tooling and code is specified to provide consistent results. Builds within the CI/CD pipeline require containerization to allow the container to build and execute tests regardless of the configuration of the CI/CD execution environment.

The downside of building and executing containers is that the time to create the container is additive to the build time, and that time can be substantial (minutes at this point).

#### Unit test

Build a docker container with the unit test code and execute standalone unit tests within that docker container. This mimics testing actions of the CI/CD pipeline.

```bash
make container-ut
```

#### Deployable container

Build a docker container containing the code and tools necessary to build the executable, execute standalone unit tests within that docker container, then create the docker image to be deployed to k8s. This mimics the actions of the CI/CD pipeline

```bash
make docker-build
```

## Test

---

### Local Machine testing

Execute unit tests:

```bash
# Reduced output
make test
```

#### Optimization

```bash

# If you've already executed unit tests least once, the $(pwd)/testbin/bin
# directory contains the envtest assets required. No need to download them again.
# Specify the KUBEBUILDER_ASSETS environment variable to avoid the download.
$ ls testbin/bin
etcd*           kube-apiserver* kubectl*

$ export KUBEBUILDER_ASSETS=$(pwd)/testbin/bin
$ ls $KUBEBUILDER_ASSETS
etcd*           kube-apiserver* kubectl*

# Execute tests without downloading the k8s assets
go test -v -count=1 -cover ./controllers/... -args -ginkgo.v
```

## VS Code Debugging

You may execute unit tests inside the VS Code debugger using the `Run and Debug` widget. The following configuration is provided in `.vscode/launch.json`. Select `Debug Unit Test` from the drop-down to execute your test within the debugger. Set breakpoints in your code, etc.

```json
{
    "name": "Debug Unit Test",
    "type": "go",
    "request": "launch",
    "mode": "test",
    "program": "${workspaceFolder}/controllers",
    "args": [
        "-ginkgo.v",
        "-ginkgo.progress",
    ],
    "env": {
        "KUBEBUILDER_ASSETS": "${workspaceFolder}/testbin/bin"
    },
    "showLog": true
},
```

## Deploy

### Deploy to Kind

To deploy the nnf-sos code to a kind environment execute:

```bash
make kind-push
make deploy
```

### Deploy to DP0/DP1a Cluster

We don't have local container registries configured yet for our DPxx clusters. Therefore, we use artifactory to be the container registry. This means that in order to deploy to a DPxx cluster, you must push your branch to github.hpe.com to force `jenkins` to build it and save the resulting docker image in artifactory.

The artifactory path to the image is built into the makefile, but the version tag that `jenkins` applies to each docker image contains a timestamp which is impossible to reliably calculate. The `setDevVersion.sh` script instead queries artifactory to look for the tag that contains the `git SHA` of your local branch. If found, the `VERSION` and `IMAGE_TAG_BASE` environment variables are set such that each member of the cluster can `docker pull` the image form artifactory into its local docker cache.

#### Master branch

To deploy the ***master*** branch of nnf-sos code to DP0 or DP1b systems:

```bash
# set your kubectl context to the cluster you are interested in
kubectl config use-context dp1a-dev

# checkout the master branch and update to the latest
git checkout master
git pull --recurse-submodules

# set the environment variable to the pre-built docker image in
# https://arti.dev.cray.com/artifactory/rabsw-docker-master-local/cray-dp-nnf-sos/
source ./setDevVersion.sh

# Deploy the latest master build
make deploy
```

#### Other than master branch

To deploy a feature or bugfix branch of nnf-sos code to DP0 or DP1b systems:

```bash
# set your kubectl context to the cluster you are interested in
kubectl config use-context dp1a-dev

# commit all of your changes
git commit -a -m "some useful comment"

# push your changes to github
git push origin

# wait a few minutes for a jenkins pipeline to build your changes and save
# the resulting docker image in artifactory

# setup the arti path and version tag of the resulting jenkins build in environment
# variables that the makefile will use.
# Example below...
$ source ./setDevVersion.sh
IMAGE_TAG_BASE: arti.dev.cray.com/rabsw-docker-unstable-local/cray-dp-nnf-sos
VERSION: 0.0.1-20220131231031_11b401a

# Deploy the freshly built docker image to your cluster
make deploy
```
