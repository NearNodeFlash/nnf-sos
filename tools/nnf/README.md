# nnf Helper CLI

This directory contains a small Python helper CLI for working with NNF and DWS resources from the `nnf-sos` repository.

Today the tool focuses on helper workflows around persistent storage. It is expected to grow over time with additional subcommands, so this document is the local reference point for setup, usage, and examples.

## Scope

The `nnf` command is intended as a repository-local helper tool, not a standalone product. It is useful when you need to:

- create a persistent storage instance for testing
- destroy a persistent storage instance
- inspect and drive workflow-based storage operations from a simple CLI

The command operates against an existing Kubernetes environment where the relevant NNF and DWS CRDs are already installed.

## Setup

From this directory, create or activate a virtual environment and install the package in editable mode:

```bash
python3 -m venv .venv
source .venv/bin/activate
pip install -e .
```

After that, you can run either:

```bash
nnf --help
```

or:

```bash
python -m nnf --help
```

## Kubernetes configuration

By default, the CLI uses normal Kubernetes client resolution, such as `KUBECONFIG` or `~/.kube/config`. If no kubeconfig is available, it can fall back to in-cluster configuration.

You can also provide an explicit kubeconfig path:

```bash
nnf --kubeconfig /path/to/kubeconfig --help
```

## Common usage

Show top-level help:

```bash
nnf --help
```

Show command-specific help:

```bash
nnf create_persistent --help
nnf destroy_persistent --help
```

Enable verbose workflow logging:

```bash
nnf --verbose create_persistent --help
nnf create_persistent --verbose --help
```

Both subcommands support these common flags:

- `--namespace` to target a namespace other than `default`
- `--user-id` to override the owning user ID
- `--group-id` to override the owning group ID
- `--timeout` to control how long the CLI waits for each workflow state

The default timeout is `180` seconds per workflow state. On a slow or heavily loaded cluster, you may need to raise it:

```bash
nnf create_persistent --timeout 600 ...
nnf destroy_persistent --timeout 600 ...
```

## create_persistent

Show help for the `create_persistent` subcommand:

```bash
nnf create_persistent --help
```

Key `create_persistent` flags:

- `--name` required name of the persistent storage instance
- `--fs-type` required filesystem type: `raw`, `xfs`, `gfs2`, `lustre`
- `--capacity` allocation capacity per Rabbit; required unless the selected profile defines `standaloneMgtPoolName`
- `--namespace` target namespace, default `default`
- `--user-id` override owning user ID
- `--group-id` override owning group ID
- `--rabbits` explicit Rabbit node list
- `--rabbits-mdt` Rabbit node list for `mdt` and `mgtmdt`
- `--rabbits-mgt` Rabbit node list for `mgt`
- `--rabbit-count` choose Rabbits randomly from the default SystemConfiguration instead of specifying them directly
- `--alloc-count` number of allocations per Rabbit node, default `1`
- `--profile` storage profile name
- `--timeout` seconds to wait per workflow state, default `180`

Use `--profile` to select the `NnfStorageProfile` if the default profile is not desired. For Lustre, the profile drives allocation-set behavior for labels such as `ost`, `mdt`, `mgt`, and `mgtmdt`.

Create a simple persistent storage instance:

```bash
nnf create_persistent \
  --name demo-psi \
  --fs-type xfs \
  --capacity 1GiB \
  --rabbits rabbit-node-0
```

Create persistent storage using multiple Rabbits:

```bash
nnf create_persistent \
  --name demo-lustre \
  --fs-type lustre \
  --capacity 10GiB \
  --profile lustre-storage-profile \
  --rabbits rabbit-node-0 rabbit-node-1
```

For Lustre, you can optionally direct MDT and MGT allocation sets to specific Rabbits instead of using the default Rabbit list for every allocation set:

```bash
nnf create_persistent \
  --name demo-lustre-tiered \
  --fs-type lustre \
  --capacity 10GiB \
  --profile lustre-storage-profile \
  --rabbits rabbit-node-0 rabbit-node-1 rabbit-node-2 \
  --rabbits-mdt rabbit-node-0 \
  --rabbits-mgt rabbit-node-1
```

In that case:

- `--rabbits` remains the default Rabbit list for `ost` allocation sets
- `--rabbits-mdt` is used for `mdt` and `mgtmdt` allocation sets
- `--rabbits-mgt` is used for `mgt` allocation sets

If you do not specify `--rabbits-mdt` or `--rabbits-mgt`, the CLI falls back to the Rabbit list provided by `--rabbits` for those allocation sets as well. In that case, the number of Rabbits actually used for `mgt`, `mgtmdt`, and `mdt` is determined by the allocation set constraints from the `NnfStorageProfile`. In practice, the profile's `count` or `scale` fields determine how many Rabbits are chosen from the default `--rabbits` list for those allocation sets.

If you prefer to let the CLI choose Rabbits from the default SystemConfiguration, use `--rabbit-count` instead of `--rabbits`:

```bash
nnf create_persistent \
  --name demo-random \
  --fs-type xfs \
  --capacity 5GiB \
  --rabbit-count 2 \
  --alloc-count 16
```

When `--rabbit-count` is used, the CLI randomly selects that many Rabbit nodes from the `default/default` SystemConfiguration resource. This option is mutually exclusive with `--rabbits`, `--rabbits-mdt`, and `--rabbits-mgt`. This example also shows the `--alloc-count` option. This controls how many allocations are made on each Rabbit node.

Create a standalone MGT using a profile that defines `standaloneMgtPoolName`:

```bash
nnf create_persistent \
  --name demo-lustre-standalone-mgt \
  --fs-type lustre \
  --profile lustre-standalone-mgt-profile \
  --rabbits rabbit-node-0
```

For a standalone-MGT profile:

- do not pass `--capacity` (capacity comes from the `NnfStorageProfile`)
- exactly one Rabbit must be supplied
- `--alloc-count` must remain `1`

The CLI detects `standaloneMgtPoolName` from the selected profile and adjusts validation and the generated directive accordingly.

## destroy_persistent

Show help for the `destroy_persistent` subcommand:

```bash
nnf destroy_persistent --help
```

Key `destroy_persistent` flags:

- `--name` required name of the persistent storage instance to destroy
- `--namespace` target namespace, default `default`
- `--user-id` override owning user ID
- `--group-id` override owning group ID
- `--timeout` seconds to wait per workflow state, default `180`

Destroy a persistent storage instance:

```bash
nnf destroy_persistent --name demo-psi
```

## Development notes

- The package metadata lives in `pyproject.toml`.
- The console script entry point is installed as `nnf`.
- Unit tests live under `tests/` and can be run with `pytest tests/`.

Run the test suite from this directory with:

```bash
.venv/bin/pytest tests/
```