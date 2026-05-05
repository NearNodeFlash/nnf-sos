"""system state sub-command: report the state of Rabbit storage resources.

Displays a summary of NnfNode server health/status and DWS Storage
state/status, including disabled and drained nodes with their reasons.
"""

import argparse
import json
import logging
import os
import shutil
import subprocess
import sys
from typing import Any, Dict, List, Optional, Tuple

import kubernetes.client.exceptions  # type: ignore[import-untyped]

from nnf import crd
from nnf import k8s
from nnf.commands import add_command_parser
from nnf.table import print_table as _print_table


LOGGER = logging.getLogger(__name__)


# Health values to scan (in display order).
_HEALTH_VALUES = ["OK", "Critical", "Warning"]

# Status values for NnfNode servers (in display order).
_NODE_STATUS_VALUES = [
    "Ready", "Enabled", "Disabled", "NotPresent", "Offline",
    "Starting", "Deleting", "Deleted", "Failed", "Invalid", "Fenced",
]

# State values for Storage resources.
_STORAGE_STATES = ["Enabled", "Disabled"]

# Status values for Storage resources (in display order).
_STORAGE_STATUS_VALUES = [
    "Starting", "Ready", "Disabled", "NotPresent", "Offline",
    "Failed", "Degraded", "Drained", "Fenced", "Unknown",
]

_MISSING_BUCKET = "<missing>"


def register(subparsers: argparse._SubParsersAction) -> None:  # type: ignore[type-arg]
    """Register the system state sub-command."""
    parser: argparse.ArgumentParser = add_command_parser(
        subparsers,
        "state",
        help="Report the state of Rabbit storage resources.",
    )
    parser.set_defaults(func=run)


def _compress_hostlist(names: List[str]) -> str:
    """Compress a list of hostnames into bracketed range notation.

    For example: ["rabbit-node-1", "rabbit-node-2", "rabbit-compute-2",
    "rabbit-compute-3", "rabbit-compute-4", "rabbit-compute-5"]
    becomes "rabbit-compute-[2-5],rabbit-node-[1-2]".

    Names that don't end in digits are left as-is.
    """
    if not names:
        return ""
    if len(names) == 1:
        return names[0]

    # Split each name into (prefix, number).  Names without a numeric
    # suffix are kept verbatim in the output.
    groups: Dict[str, List[int]] = {}
    plain: List[str] = []
    for name in sorted(names):
        i = len(name)
        while i > 0 and name[i - 1].isdigit():
            i -= 1
        if i == len(name) or i == 0:
            plain.append(name)
        else:
            prefix = name[:i]
            groups.setdefault(prefix, []).append(int(name[i:]))

    parts: List[str] = []
    for prefix in sorted(groups):
        nums = sorted(groups[prefix])
        if len(nums) == 1:
            parts.append(f"{prefix}{nums[0]}")
            continue
        ranges: List[str] = []
        start = end = nums[0]
        for n in nums[1:]:
            if n == end + 1:
                end = n
            else:
                ranges.append(str(start) if start == end else f"{start}-{end}")
                start = end = n
        ranges.append(str(start) if start == end else f"{start}-{end}")
        parts.append(f"{prefix}[{','.join(ranges)}]")

    parts.extend(sorted(plain))
    return ",".join(parts)


def _get_all_nnfnodes() -> List[Dict[str, Any]]:
    """Fetch all NnfNode resources across all namespaces."""
    result = k8s.list_cluster_objects(
        group=crd.NNF_GROUP,
        version=crd.NNF_VERSION,
        plural=crd.NNF_NODE_PLURAL,
    )
    return result.get("items", [])  # type: ignore[no-any-return]


def _get_all_storages() -> List[Dict[str, Any]]:
    """Fetch all DWS Storage resources from the default namespace."""
    result = k8s.list_objects(
        group=crd.DWS_GROUP,
        version=crd.DWS_VERSION,
        namespace="default",
        plural=crd.DWS_STORAGE_PLURAL,
    )
    return result.get("items", [])  # type: ignore[no-any-return]


def _normalize_bucket(value: Any) -> str:
    """Normalize state/health values so missing data is still reported."""
    if value is None:
        return _MISSING_BUCKET
    text = str(value).strip()
    return text if text else _MISSING_BUCKET


def _ordered_values(observed: List[str], preferred: List[str]) -> List[str]:
    """Return observed values ordered by preferred display order, then extras."""
    extras = sorted({value for value in observed if value not in preferred})
    return [value for value in preferred if value in observed] + extras


def _build_node_status_rows(
    nnfnodes: List[Dict[str, Any]],
) -> List[Tuple[str, ...]]:
    """Build rows for the Node Status Summary table."""
    grouped_hostnames: Dict[Tuple[str, str], List[str]] = {}
    observed_healths: List[str] = []
    observed_statuses: List[str] = []

    for node in nnfnodes:
        for server in node.get("status", {}).get("servers", []):
            hostname = server.get("hostname")
            if not hostname:
                continue
            health = _normalize_bucket(server.get("health"))
            status = _normalize_bucket(server.get("status"))
            grouped_hostnames.setdefault((health, status), []).append(hostname)
            observed_healths.append(health)
            observed_statuses.append(status)

    rows: List[Tuple[str, ...]] = []
    for health in _ordered_values(observed_healths, _HEALTH_VALUES):
        for status in _ordered_values(observed_statuses, _NODE_STATUS_VALUES):
            hostnames = grouped_hostnames.get((health, status), [])
            if hostnames:
                hostnames.sort()
                compressed = _compress_hostlist(hostnames)
                rows.append((health, status, str(len(hostnames)), compressed))
    return rows


def _build_storage_status_rows(
    storages: List[Dict[str, Any]],
) -> List[Tuple[str, ...]]:
    """Build rows for the Storage Status Summary table."""
    grouped_names: Dict[Tuple[str, str], List[str]] = {}
    observed_states: List[str] = []
    observed_statuses: List[str] = []

    for storage in storages:
        state = _normalize_bucket(storage.get("spec", {}).get("state"))
        status = _normalize_bucket(storage.get("status", {}).get("status"))
        grouped_names.setdefault((state, status), []).append(storage["metadata"]["name"])
        observed_states.append(state)
        observed_statuses.append(status)

    rows: List[Tuple[str, ...]] = []
    for state in _ordered_values(observed_states, _STORAGE_STATES):
        for status in _ordered_values(observed_statuses, _STORAGE_STATUS_VALUES):
            names = grouped_names.get((state, status), [])
            if names:
                names.sort()
                compressed = _compress_hostlist(names)
                rows.append((state, status, str(len(names)), compressed))
    return rows


def _build_annotation_rows(
    storages: List[Dict[str, Any]],
) -> List[Tuple[str, ...]]:
    """Build rows for disabled/drained annotation details."""
    rows: List[Tuple[str, ...]] = []
    for s in storages:
        annotations = s.get("metadata", {}).get("annotations") or {}
        name = s["metadata"]["name"]
        if "disable_date" in annotations:
            rows.append((
                "Disabled",
                annotations.get("disable_date", ""),
                name,
                annotations.get("disable_reason", ""),
            ))
        if "drain_date" in annotations:
            rows.append((
                "Drained",
                annotations.get("drain_date", ""),
                name,
                annotations.get("drain_reason", ""),
            ))
    return rows


def run(args: argparse.Namespace) -> int:
    """Execute the system state sub-command."""
    errors = 0

    # --- Node Status Summary ---
    print("------------------------------")
    print("Node Status Summary")
    print("------------------------------")

    try:
        nnfnodes = _get_all_nnfnodes()
    except kubernetes.client.exceptions.ApiException as exc:
        print(f"error: failed to list NnfNode resources: {exc.reason}", file=sys.stderr)
        nnfnodes = []
        errors += 1

    node_rows = _build_node_status_rows(nnfnodes)
    if node_rows:
        _print_table(
            ("HEALTH", "STATUS", "NNODES", "NODELIST"),
            node_rows,
            right_align=(0, 1, 2),
        )

    # --- Storage Status Summary ---
    print()
    print("-------------------------------")
    print("Disabled/Drained Rabbit Summary")
    print("-------------------------------")

    try:
        storages = _get_all_storages()
    except kubernetes.client.exceptions.ApiException as exc:
        print(f"error: failed to list Storage resources: {exc.reason}", file=sys.stderr)
        storages = []
        errors += 1

    storage_rows = _build_storage_status_rows(storages)
    if storage_rows:
        _print_table(
            ("STATE", "STATUS", "NNODES", "NODELIST"),
            storage_rows,
            right_align=(0, 1, 2),
        )

    # --- Annotation details ---
    annotation_rows = _build_annotation_rows(storages)
    if annotation_rows:
        print()
        _print_table(
            ("STATE", "DATE", "NODELIST", "REASON"),
            annotation_rows,
            right_align=(0,),
        )

    # --- Disabled Computes (badrabbit+) ---
    _show_disabled_computes()

    return 1 if errors else 0


def _run_cmd(args: List[str]) -> Optional[str]:
    """Run a command and return its stripped stdout, or None on failure."""
    try:
        result = subprocess.run(
            args,
            capture_output=True,
            text=True,
            timeout=30,
        )
        if result.returncode != 0:
            LOGGER.info("%s failed (rc=%d): %s", args[0], result.returncode, result.stderr.strip())
            return None
        return result.stdout.strip()
    except (OSError, subprocess.TimeoutExpired) as exc:
        LOGGER.info("%s: %s", args[0], exc)
        return None


def _show_disabled_computes() -> None:
    """Show compute nodes paired with unavailable Rabbits, if flux is available."""
    if shutil.which("flux") is None:
        LOGGER.info("flux not found, skipping disabled computes section")
        return

    jgf_output = _run_cmd([
        "flux", "ion-resource", "find", "-q",
        "--format", "jgf", "property=badrabbit",
    ])
    if not jgf_output:
        return

    try:
        jgf = json.loads(jgf_output)
    except (json.JSONDecodeError, ValueError):
        LOGGER.info("Failed to parse JGF output")
        return

    nodes = jgf.get("graph", {}).get("nodes", [])
    hostnames = []
    for node in nodes:
        meta = node.get("metadata", {})
        if meta.get("type") == "node":
            hostnames.append(meta.get("basename", "") + str(meta.get("id", "")))

    if not hostnames:
        return

    hostlist = _run_cmd(["flux", "hostlist"] + hostnames)
    if not hostlist:
        hostlist = ", ".join(hostnames)

    print()
    print("------------------------------")
    print("Disabled Computes (badrabbit+)")
    print("------------------------------")

    # Determine whether to use a cluster prefix with nodeattr.
    scheduling_cfg = _run_cmd(["flux", "config", "get", "resource.scheduling"])
    if scheduling_cfg and os.path.isfile(scheduling_cfg):
        cluster_prefix = _run_cmd(["nodeattr", "-v", "cluster"])
        if cluster_prefix:
            _run_cmd_print(["flux", "resource", "list", "-i", cluster_prefix + hostlist])
            return

    _run_cmd_print(["flux", "resource", "list", "-i", hostlist])


def _run_cmd_print(args: List[str]) -> None:
    """Run a command and print its stdout directly."""
    try:
        subprocess.run(
            args,
            check=False,
            timeout=30,
        )
    except (OSError, subprocess.TimeoutExpired) as exc:
        LOGGER.info("%s: %s", args[0], exc)
