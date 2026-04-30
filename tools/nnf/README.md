# nnf Helper CLI

This directory contains a small Python helper CLI for working with NNF and DWS resources from the `nnf-sos` repository.

Today the tool focuses on helper workflows around persistent storage and Rabbit node management. It is expected to grow over time with additional subcommands, so this document is the local reference point for setup, usage, and examples.

## Scope

The `nnf` command is intended as a repository-local helper tool, not a standalone product. It is useful when you need to:

- create a persistent storage instance for testing
- destroy a persistent storage instance
- inspect and drive workflow-based storage operations from a simple CLI
- drain or undrain Rabbit nodes to control workflow scheduling
- disable or enable Rabbit nodes to control workload manager allocations
- check disk space usage on Rabbit nodes

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
nnf persistent create --help
nnf persistent destroy --help
nnf rabbit --help
```

Enable verbose workflow logging:

```bash
nnf --verbose persistent create --help
nnf persistent create --verbose --help
```

Both subcommands support these common flags:

- `--namespace` to target a namespace other than `default`
- `--user-id` to override the owning user ID
- `--group-id` to override the owning group ID
- `--timeout` to control how long the CLI waits for each workflow state

The default timeout is `180` seconds per workflow state. On a slow or heavily loaded cluster, you may need to raise it:

```bash
nnf persistent create --timeout 600 ...
nnf persistent destroy --timeout 600 ...
```

## persistent create

Show help for the `persistent create` subcommand:

```bash
nnf persistent create --help
```

Key `persistent create` flags:

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
nnf persistent create \
  --name demo-psi \
  --fs-type xfs \
  --capacity 1GiB \
  --rabbits rabbit-node-0
```

Create persistent storage using multiple Rabbits:

```bash
nnf persistent create \
  --name demo-lustre \
  --fs-type lustre \
  --capacity 10GiB \
  --profile lustre-storage-profile \
  --rabbits rabbit-node-0 rabbit-node-1
```

For Lustre, you can optionally direct MDT and MGT allocation sets to specific Rabbits instead of using the default Rabbit list for every allocation set:

```bash
nnf persistent create \
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
nnf persistent create \
  --name demo-random \
  --fs-type xfs \
  --capacity 5GiB \
  --rabbit-count 2 \
  --alloc-count 16
```

When `--rabbit-count` is used, the CLI randomly selects that many Rabbit nodes from the `default/default` SystemConfiguration resource. This option is mutually exclusive with `--rabbits`, `--rabbits-mdt`, and `--rabbits-mgt`. This example also shows the `--alloc-count` option. This controls how many allocations are made on each Rabbit node.

Create a standalone MGT using a profile that defines `standaloneMgtPoolName`:

```bash
nnf persistent create \
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

## persistent destroy

Show help for the `persistent destroy` subcommand:

```bash
nnf persistent destroy --help
```

Key `persistent destroy` flags:

- `--name` required name of the persistent storage instance to destroy
- `--namespace` target namespace, default `default`
- `--user-id` override owning user ID
- `--group-id` override owning group ID
- `--timeout` seconds to wait per workflow state, default `180`

Destroy a persistent storage instance:

```bash
nnf persistent destroy --name demo-psi
```

## rabbit drain

Drain one or more Rabbit nodes so that no new workflows are scheduled to them. This taints the Kubernetes node with `cray.nnf.node.drain=true` (both `NoSchedule` and `NoExecute`) and annotates the corresponding DWS Storage resource with `drain_date` and `drain_reason`.

```bash
nnf rabbit drain --help
```

Key flags:

- `NODE` one or more Rabbit node names (positional)
- `-r`, `--reason` reason for draining the node (default: `none`)

Drain a single node:

```bash
nnf rabbit drain rabbit-node-0
```

Drain multiple nodes with a reason:

```bash
nnf rabbit drain rabbit-node-0 rabbit-node-1 --reason "firmware update"
```

If the node taint fails after the storage annotation succeeds, the CLI attempts to roll back the annotation. If the rollback also fails, the CLI logs the error and continues to the next node.

## rabbit undrain

Undrain one or more Rabbit nodes so that new workflows can be scheduled to them again. This removes the `cray.nnf.node.drain` taint from the Kubernetes node and removes the `drain_date` and `drain_reason` annotations from the DWS Storage resource.

```bash
nnf rabbit undrain --help
```

Undrain a node:

```bash
nnf rabbit undrain rabbit-node-0
```

## rabbit disable

Disable one or more Rabbit nodes so that the workload manager does not use them for allocations. This sets the DWS Storage resource `spec.state` to `Disabled` and annotates it with `disable_date` and `disable_reason`.

```bash
nnf rabbit disable --help
```

Key flags:

- `NODE` one or more Rabbit node names (positional)
- `-r`, `--reason` reason for disabling the node (default: `none`)

Disable a node:

```bash
nnf rabbit disable rabbit-node-0 --reason "bad disk"
```

## rabbit enable

Enable one or more Rabbit nodes so that the workload manager can use them for allocations again. This sets the DWS Storage resource `spec.state` to `Enabled` and removes the `disable_date` and `disable_reason` annotations.

```bash
nnf rabbit enable --help
```

Enable a node:

```bash
nnf rabbit enable rabbit-node-0
```

## rabbit df

Show disk space information for Rabbit nodes. Queries each Rabbit's node-manager pod for NVMe capacity information via the Redfish CapacitySource endpoint.

```bash
nnf rabbit df --help
```

Show disk space for all enabled and ready Rabbits:

```bash
nnf rabbit df
```

Show disk space for specific nodes:

```bash
nnf rabbit df rabbit-node-0 rabbit-node-1
```

## Development notes

- The package metadata lives in `pyproject.toml`.
- The console script entry point is installed as `nnf`.
- Unit tests live under `tests/` and can be run with `pytest tests/`.

Run the test suite from this directory with:

```bash
.venv/bin/pytest tests/
```