"""rabbit df sub-command: show disk space on Rabbit nodes.

Queries each Rabbit's node-manager pod for NVMe capacity information via
the Redfish CapacitySource endpoint and displays allocated, consumed,
guaranteed, and provisioned bytes.
"""

import argparse
import ast
import json
import logging
import sys
from typing import Any, Dict, List, Optional

import kubernetes.client.exceptions  # type: ignore[import-untyped]

from nnf import crd
from nnf import k8s
from nnf.commands import add_command_parser


LOGGER = logging.getLogger(__name__)

_NNF_SYSTEM_NS = "nnf-system"
_CAPACITY_URL = "localhost:50057/redfish/v1/StorageServices/NNF/CapacitySource"


def register(subparsers: argparse._SubParsersAction) -> None:  # type: ignore[type-arg]
    """Register the rabbit df sub-command."""
    parser: argparse.ArgumentParser = add_command_parser(
        subparsers,
        "df",
        help="Show disk space information for Rabbit nodes.",
    )
    parser.add_argument(
        "nodes",
        nargs="*",
        metavar="NODE",
        help="Rabbit node names to query (default: all Enabled/Ready rabbits).",
    )
    parser.set_defaults(func=run)


def _format_tib(byte_count: int) -> str:
    """Format a byte count as TiB with 6 decimal places."""
    tib = byte_count / (1024 ** 4)
    return f"{tib:.6f} TiB"


def _get_all_storages() -> List[Dict[str, Any]]:
    """Return all DWS Storage resources in the default namespace."""
    result = k8s.list_objects(
        group=crd.DWS_GROUP,
        version=crd.DWS_VERSION,
        namespace="default",
        plural=crd.DWS_STORAGE_PLURAL,
    )
    return result.get("items", [])  # type: ignore[no-any-return]


def _is_enabled_ready(storage: Dict[str, Any]) -> bool:
    """Return True if the Storage resource is Enabled and Ready."""
    spec_state = storage.get("spec", {}).get("state", "")
    status = storage.get("status", {})
    status_state = status.get("status", "")
    return spec_state == "Enabled" and status_state == "Ready"


def _find_node_manager_pod(rabbit_name: str, pods: List[Any]) -> Optional[str]:
    """Find the node-manager pod running on *rabbit_name*."""
    for pod in pods:
        if pod.spec.node_name == rabbit_name:
            return pod.metadata.name  # type: ignore[no-any-return]
    return None


def _get_capacity(pod_name: str) -> Dict[str, int]:
    """Exec into *pod_name* and return capacity data from the Redfish endpoint."""
    output = k8s.exec_pod(
        namespace=_NNF_SYSTEM_NS,
        pod_name=pod_name,
        command=["curl", "-sS", _CAPACITY_URL],
        strip_channel_bytes=True,
    )
    LOGGER.info("raw exec output for %s: %r", pod_name, output[:200])
    try:
        data = json.loads(output)
    except json.JSONDecodeError:
        # The Redfish endpoint may return a Python-dict-formatted response
        # (single quotes) instead of strict JSON.
        data = ast.literal_eval(output)
    provided = data.get("ProvidedCapacity", {}).get("Data", {})
    return {
        "AllocatedBytes": provided.get("AllocatedBytes", 0),
        "ConsumedBytes": provided.get("ConsumedBytes", 0),
        "GuaranteedBytes": provided.get("GuaranteedBytes", 0),
        "ProvisionedBytes": provided.get("ProvisionedBytes", 0),
    }


def run(args: argparse.Namespace) -> int:
    """Execute the rabbit df sub-command."""
    # Fetch all Storage resources and node-manager pods up front.
    try:
        storages = _get_all_storages()
    except kubernetes.client.exceptions.ApiException as exc:
        print(f"error: failed to list Storage resources: {exc.reason}", file=sys.stderr)
        return 1

    storage_by_name: Dict[str, Dict[str, Any]] = {
        s["metadata"]["name"]: s for s in storages
    }

    try:
        nm_pods = k8s.list_pods(
            namespace=_NNF_SYSTEM_NS,
            label_selector="cray.nnf.node=true",
            field_selector="status.phase=Running",
        )
    except kubernetes.client.exceptions.ApiException as exc:
        print(f"error: failed to list node-manager pods: {exc.reason}", file=sys.stderr)
        return 1

    # Determine which rabbits to query.
    if args.nodes:
        rabbits = args.nodes
    else:
        rabbits = sorted(storage_by_name.keys())

    skipped: List[str] = []
    errors = 0

    for rabbit in rabbits:
        storage = storage_by_name.get(rabbit)
        if storage is None:
            print(f"error: no Storage resource found for '{rabbit}'", file=sys.stderr)
            errors += 1
            continue

        if not _is_enabled_ready(storage):
            if args.nodes:
                spec_state = storage.get("spec", {}).get("state", "")
                status_state = storage.get("status", {}).get("status", "")
                print(
                    f"error: '{rabbit}' is not Enabled/Ready "
                    f"(spec.state={spec_state}, status={status_state})",
                    file=sys.stderr,
                )
                errors += 1
            else:
                skipped.append(rabbit)
            continue

        pod_name = _find_node_manager_pod(rabbit, nm_pods)
        if pod_name is None:
            print(f"error: no node-manager pod found on '{rabbit}'", file=sys.stderr)
            errors += 1
            continue

        try:
            cap = _get_capacity(pod_name)
        except (kubernetes.client.exceptions.ApiException, json.JSONDecodeError,
                KeyError, ValueError, SyntaxError) as exc:
            print(f"error: failed to get capacity from '{rabbit}': {exc}", file=sys.stderr)
            errors += 1
            continue

        print(
            f"{rabbit} {pod_name}"
            f" Allocated: {_format_tib(cap['AllocatedBytes'])},"
            f" Consumed: {_format_tib(cap['ConsumedBytes'])},"
            f" Guaranteed: {_format_tib(cap['GuaranteedBytes'])},"
            f" Provisioned: {_format_tib(cap['ProvisionedBytes'])}"
        )

    if skipped:
        print()
        print(f"{len(skipped)} rabbit(s) skipped:")
        for rabbit in skipped:
            storage = storage_by_name[rabbit]
            spec_state = storage.get("spec", {}).get("state", "")
            status_state = storage.get("status", {}).get("status", "")
            print(f"    {rabbit}  {spec_state}  {status_state}")

    return 1 if errors else 0
