"""system flowschema sub-command: display FlowSchema information.

Full documentation at:
https://nearnodeflash.github.io/latest/guides/monitoring-cluster/api-priority-and-fairness/
https://kubernetes.io/docs/concepts/cluster-administration/flow-control/
"""

import argparse
import json
import sys
from typing import Any, Dict, List, Optional, Tuple

import kubernetes.client.exceptions  # type: ignore[import-untyped]

from nnf import k8s
from nnf.commands import add_command_parser
from nnf.table import print_table as _print_table


_FLOWCONTROL_GROUP = "flowcontrol.apiserver.k8s.io"
_API_VERSIONS = ["v1", "v1beta3", "v1beta2"]


def register(subparsers: argparse._SubParsersAction) -> None:  # type: ignore[type-arg]
    """Register the system flowschema sub-command."""
    parser: argparse.ArgumentParser = add_command_parser(
        subparsers,
        "flowschema",
        help="Display FlowSchema information.",
    )
    group = parser.add_mutually_exclusive_group(required=True)
    group.add_argument(
        "-c",
        "--activity",
        dest="flow_schema_name",
        metavar="NAME",
        help="View activity for a specific flow schema.",
    )
    group.add_argument(
        "-l",
        "--list",
        dest="list_flowschemas",
        action="store_true",
        help="List flow schemas.",
    )
    group.add_argument(
        "-p",
        "--priority-levels",
        dest="priority_levels",
        action="store_true",
        help="List priority level configurations.",
    )
    group.add_argument(
        "-s",
        "--summary",
        dest="summary",
        action="store_true",
        help="Summary of requests by priority level.",
    )
    parser.set_defaults(func=run)


def _discover_api_version() -> Optional[str]:
    """Find an available API version for the flowcontrol group."""
    try:
        raw = k8s.get_raw(f"/apis/{_FLOWCONTROL_GROUP}")
        data = json.loads(raw)
        available = [v["version"] for v in data.get("versions", [])]
        for version in _API_VERSIONS:
            if version in available:
                return version
    except (kubernetes.client.exceptions.ApiException, json.JSONDecodeError, KeyError):
        pass
    return None


def _list_flowschemas(api_version: str) -> int:
    """List all FlowSchema resources."""
    path = f"/apis/{_FLOWCONTROL_GROUP}/{api_version}/flowschemas"
    try:
        raw = k8s.get_raw(path)
    except kubernetes.client.exceptions.ApiException as exc:
        print(f"error: failed to list flow schemas: {exc.reason}", file=sys.stderr)
        return 1

    data = json.loads(raw)
    items = sorted(data.get("items", []), key=lambda i: i["metadata"]["name"])

    rows: List[Tuple[str, ...]] = []
    for item in items:
        name = item["metadata"]["name"]
        spec = item.get("spec", {})
        pl = spec.get("priorityLevelConfiguration", {}).get("name", "")
        precedence = str(spec.get("matchingPrecedence", ""))
        dm = spec.get("distinguisherMethod")
        dm_type = dm.get("type", "") if dm else ""
        rows.append((name, pl, precedence, dm_type))

    _print_table(
        ("NAME", "PRIORITYLEVEL", "MATCHINGPRECEDENCE", "DISTINGUISHERMETHOD"),
        rows,
    )
    return 0


def _list_priority_levels(api_version: str) -> int:
    """List all PriorityLevelConfiguration resources."""
    path = f"/apis/{_FLOWCONTROL_GROUP}/{api_version}/prioritylevelconfigurations"
    try:
        raw = k8s.get_raw(path)
    except kubernetes.client.exceptions.ApiException as exc:
        print(
            f"error: failed to list priority level configurations: {exc.reason}",
            file=sys.stderr,
        )
        return 1

    data = json.loads(raw)
    items = sorted(data.get("items", []), key=lambda i: i["metadata"]["name"])

    rows: List[Tuple[str, ...]] = []
    for item in items:
        name = item["metadata"]["name"]
        spec = item.get("spec", {})
        type_val = spec.get("type", "")
        limited = spec.get("limited")
        if limited:
            shares = str(limited.get("nominalConcurrencyShares", ""))
            queuing = limited.get("limitResponse", {}).get("queuing", {})
            queues = str(queuing.get("queues", "")) if queuing else ""
            hand_size = str(queuing.get("handSize", "")) if queuing else ""
            queue_len = str(queuing.get("queueLengthLimit", "")) if queuing else ""
        else:
            shares = queues = hand_size = queue_len = ""
        rows.append((name, type_val, shares, queues, hand_size, queue_len))

    _print_table(
        ("NAME", "TYPE", "NOMINALCONCURRENCYSHARES", "QUEUES", "HANDSIZE", "QUEUELENGTHLIMIT"),
        rows,
    )
    return 0


def _view_activity(api_version: str, flow_schema_name: str) -> int:
    """View metrics for a specific FlowSchema."""
    path = f"/apis/{_FLOWCONTROL_GROUP}/{api_version}/flowschemas"
    try:
        raw = k8s.get_raw(path)
    except kubernetes.client.exceptions.ApiException as exc:
        print(f"error: failed to list flow schemas: {exc.reason}", file=sys.stderr)
        return 1

    data = json.loads(raw)
    names = [item["metadata"]["name"] for item in data.get("items", [])]

    if flow_schema_name not in names:
        print("Valid flowschema NAMEs are:")
        for name in sorted(names):
            print(name)
        return 1

    try:
        metrics = k8s.get_raw("/metrics")
    except kubernetes.client.exceptions.ApiException as exc:
        print(f"error: failed to get metrics: {exc.reason}", file=sys.stderr)
        return 1

    target = f'flow_schema="{flow_schema_name}"'
    for line in metrics.splitlines():
        if target in line:
            print(line)
    return 0


def _view_summary() -> int:
    """Dump priority level information."""
    try:
        raw = k8s.get_raw("/debug/api_priority_and_fairness/dump_priority_levels")
    except kubernetes.client.exceptions.ApiException as exc:
        print(
            f"error: failed to get priority level dump: {exc.reason}",
            file=sys.stderr,
        )
        return 1
    print(raw, end="")
    return 0


def run(args: argparse.Namespace) -> int:
    """Execute the system flowschema sub-command."""
    if args.summary:
        return _view_summary()

    api_version = _discover_api_version()
    if api_version is None:
        print("error: flowcontrol API is not available on this cluster", file=sys.stderr)
        return 1

    if args.flow_schema_name:
        return _view_activity(api_version, args.flow_schema_name)
    elif args.list_flowschemas:
        return _list_flowschemas(api_version)
    elif args.priority_levels:
        return _list_priority_levels(api_version)
    return 0
