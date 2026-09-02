"""persistent list sub-command: list DWS PersistentStorageInstances.

Shows the owning UID, filesystem type, state, whether the instance has been
shared, and the Rabbits backing it (from the Servers resource referenced by
``status.servers``).
"""

import argparse
import sys
from typing import Any, Dict, List, Optional, Tuple

import kubernetes.client.exceptions  # type: ignore[import-untyped]

from nnf import crd
from nnf import hostlist
from nnf import k8s
from nnf.commands import add_command_parser
from nnf.table import print_table


_HEADERS = ("NAME", "USERID", "FSTYPE", "STATE", "SHARED", "RABBITS")

# Placeholder for a field the resource does not carry yet.
_EMPTY = "-"

# Placeholder for an allocation set that carries no label.
_UNLABELED = "unlabeled"


def register(subparsers: argparse._SubParsersAction) -> None:  # type: ignore[type-arg]
    """Register the persistent list sub-command."""
    parser: argparse.ArgumentParser = add_command_parser(
        subparsers,
        "list",
        help="List persistent storage instances.",
    )
    parser.add_argument(
        "-u",
        "--user-id",
        type=int,
        default=None,
        dest="user_id",
        help="Only show instances owned by this user ID.",
    )
    parser.add_argument(
        "--namespace",
        default="default",
        help="Kubernetes namespace (default: default).",
    )
    parser.add_argument(
        "-w",
        "--wide",
        action="store_true",
        help="Break the RABBITS column down by allocation set label.",
    )
    parser.set_defaults(func=run)


def _servers_name(psi: Dict[str, Any], namespace: str) -> Optional[str]:
    """Return the name of the PSI's Servers resource if it lives in *namespace*."""
    ref = psi.get("status", {}).get("servers") or {}
    name = ref.get("name")
    if not name or ref.get("namespace", namespace) != namespace:
        return None
    return str(name)


def _allocation_sets(servers: Dict[str, Any]) -> List[Tuple[str, List[str]]]:
    """Return (label, Rabbit names) for each allocation set, in spec order."""
    sets: List[Tuple[str, List[str]]] = []
    for alloc_set in servers.get("spec", {}).get("allocationSets", []):
        names = {s["name"] for s in alloc_set.get("storage", []) if s.get("name")}
        if names:
            sets.append((str(alloc_set.get("label") or _UNLABELED), sorted(names)))
    return sets


def _rabbits(servers: Dict[str, Any], wide: bool) -> str:
    """Return the Rabbits used by a Servers resource in compressed hostlist form.

    When *wide*, each allocation set is listed separately as ``label:rabbits``;
    otherwise the sets are merged into a single hostlist.
    """
    sets = _allocation_sets(servers)
    if wide:
        return " ".join(f"{label}:{hostlist.compress(names)}" for label, names in sets)
    merged = sorted({name for _, names in sets for name in names})
    return hostlist.compress(merged)


def _is_shared(psi: Dict[str, Any]) -> bool:
    """Return True if the PSI carries the ignore-uid annotation.

    Matched case-insensitively to agree with the workflow controller's
    ``strings.EqualFold`` check in ``persistentStorageIgnoresUID``.
    """
    annotations = psi.get("metadata", {}).get("annotations") or {}
    value = annotations.get(crd.DWS_IGNORE_UID_ANNOTATION)
    return value is not None and str(value).lower() == "true"


def _build_row(
    psi: Dict[str, Any],
    servers_by_name: Dict[str, Dict[str, Any]],
    namespace: str,
    wide: bool,
) -> Tuple[str, ...]:
    """Build a single table row for a PersistentStorageInstance."""
    spec = psi.get("spec", {})
    user_id = spec.get("userID")
    name = _servers_name(psi, namespace)
    servers = servers_by_name.get(name) if name else None
    rabbits = _rabbits(servers, wide) if servers else ""
    return (
        str(psi["metadata"]["name"]),
        str(user_id) if user_id is not None else _EMPTY,
        spec.get("fsType") or _EMPTY,
        psi.get("status", {}).get("state") or _EMPTY,
        "yes" if _is_shared(psi) else "no",
        rabbits or _EMPTY,
    )


def run(args: argparse.Namespace) -> int:
    """Execute the persistent list sub-command."""
    try:
        psis = k8s.list_objects(
            group=crd.DWS_GROUP,
            version=crd.DWS_VERSION,
            namespace=args.namespace,
            plural=crd.DWS_PERSISTENT_STORAGE_PLURAL,
        ).get("items", [])
    except kubernetes.client.exceptions.ApiException as exc:
        print(
            f"error: failed to list PersistentStorageInstances: {exc.reason}",
            file=sys.stderr,
        )
        return 1

    try:
        servers_items = k8s.list_objects(
            group=crd.DWS_GROUP,
            version=crd.DWS_VERSION,
            namespace=args.namespace,
            plural=crd.DWS_SERVERS_PLURAL,
        ).get("items", [])
    except kubernetes.client.exceptions.ApiException as exc:
        print(f"error: failed to list Servers: {exc.reason}", file=sys.stderr)
        return 1

    servers_by_name: Dict[str, Dict[str, Any]] = {
        s["metadata"]["name"]: s for s in servers_items
    }

    if args.user_id is not None:
        psis = [p for p in psis if p.get("spec", {}).get("userID") == args.user_id]

    rows: List[Tuple[str, ...]] = [
        _build_row(psi, servers_by_name, args.namespace, args.wide)
        for psi in sorted(psis, key=lambda p: str(p["metadata"]["name"]))
    ]

    print_table(_HEADERS, rows)
    return 0
