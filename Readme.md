# NNF Storage Orchestration Services (SOS)

The NNF SOS project is a collection of Kubernetes Custom Resource Definitions (CRDs) and associated controllers that permit the creation, management, and destruction of storage on an NNF cluster.

## Setup

This project depends on additional CRDs present in the dws-operator project. It makes use of `git submodules` to make the dependecy in version control. To clone this project, use the additional `--recurse-submodules` option.

```bash
git clone --recurse-submodules ssh://git@stash.us.cray.com:7999/rabsw/nnf-sos.git

```

If you've already clone the repo, initialize the submodules with `git submodule update --init --recursive --remote`

## Building

Run `make`

## Testing

Run `make test`. This ensures the proper utilities (kube-apiserver, kubectl, etcd) are installed in the `./testbin` directory

If you use VS Code, the Test Package configuration exists to start a live debugger of the test suite.
