"""rabbit drain sub-command: drain a Rabbit node.

Taints the Kubernetes node with ``cray.nnf.node.drain=true`` (NoSchedule +
NoExecute) and annotates the corresponding DWS Storage resource with
``drain_date`` and ``drain_reason``.
"""

import argparse
import datetime
import logging
import sys
from typing import Any, Dict, List

import kubernetes.client.exceptions  # type: ignore[import-untyped]

from nnf import crd
from nnf import k8s
from nnf.commands import add_command_parser
from nnf.commands.rabbit._helpers import ERROR, OK, for_each_node


LOGGER = logging.getLogger(__name__)

TAINT_KEY = "cray.nnf.node.drain"
TAINT_VALUE = "true"


def register(subparsers: argparse._SubParsersAction) -> None:  # type: ignore[type-arg]
    """Register the rabbit drain sub-command."""
    parser: argparse.ArgumentParser = add_command_parser(
        subparsers,
        "drain",
        help="Drain a Rabbit node so no new workflows are scheduled to it.",
    )
    parser.add_argument(
        "nodes",
        nargs="+",
        metavar="NODE",
        help="One or more Rabbit node names to drain.",
    )
    parser.add_argument(
        "-r",
        "--reason",
        default="none",
        help="Reason for draining the node (default: none).",
    )
    parser.set_defaults(func=run)


def _apply_drain_taints(node_name: str) -> None:
    """Taint a node with the NNF drain key (NoSchedule + NoExecute)."""
    taints: List[Dict[str, str]] = [
        {"key": TAINT_KEY, "value": TAINT_VALUE, "effect": "NoSchedule"},
        {"key": TAINT_KEY, "value": TAINT_VALUE, "effect": "NoExecute"},
    ]

    # Read the current node to get existing taints.
    api = k8s.get_core_v1_api()
    node: Any = api.read_node(node_name)
    existing: List[Dict[str, str]] = []
    if node.spec.taints:
        existing = [
            {"key": t.key, "value": t.value or "", "effect": t.effect}
            for t in node.spec.taints
            if t.key != TAINT_KEY
        ]

    body: Dict[str, Any] = {
        "spec": {
            "taints": existing + taints,
        }
    }
    k8s.patch_node(node_name, body)
    LOGGER.info("Tainted node '%s' with %s", node_name, TAINT_KEY)


def _remove_drain_taints(node_name: str) -> None:
    """Remove all drain taints from a node."""
    api = k8s.get_core_v1_api()
    node: Any = api.read_node(node_name)
    remaining: List[Dict[str, str]] = []
    if node.spec.taints:
        remaining = [
            {"key": t.key, "value": t.value or "", "effect": t.effect}
            for t in node.spec.taints
            if t.key != TAINT_KEY
        ]

    body: Dict[str, Any] = {
        "spec": {
            "taints": remaining or None,
        }
    }
    k8s.patch_node(node_name, body)
    LOGGER.info("Removed %s taint from node '%s'", TAINT_KEY, node_name)


def _annotate_storage(node_name: str, drain_date: str, drain_reason: str) -> None:
    """Annotate the DWS Storage resource for *node_name* with drain metadata."""
    body: Dict[str, Any] = {
        "metadata": {
            "annotations": {
                "drain_date": drain_date,
                "drain_reason": drain_reason,
            }
        }
    }
    k8s.patch_object(
        group=crd.DWS_GROUP,
        version=crd.DWS_VERSION,
        namespace="default",
        plural=crd.DWS_STORAGE_PLURAL,
        name=node_name,
        body=body,
    )
    LOGGER.info("Annotated storage '%s' with drain_date=%s drain_reason=%s",
                node_name, drain_date, drain_reason)


def _remove_drain_annotations(node_name: str) -> None:
    """Remove drain_date and drain_reason annotations from the DWS Storage resource."""
    body: Dict[str, Any] = {
        "metadata": {
            "annotations": {
                "drain_date": None,
                "drain_reason": None,
            }
        }
    }
    k8s.patch_object(
        group=crd.DWS_GROUP,
        version=crd.DWS_VERSION,
        namespace="default",
        plural=crd.DWS_STORAGE_PLURAL,
        name=node_name,
        body=body,
    )
    LOGGER.info("Removed drain annotations from storage '%s'", node_name)


def run(args: argparse.Namespace) -> int:
    """Execute the rabbit drain sub-command."""
    drain_date = datetime.datetime.now(datetime.timezone.utc).strftime("%Y-%m-%dT%H:%M:%S")

    def action(node_name: str) -> int:
        try:
            _annotate_storage(node_name, drain_date, args.reason)
        except kubernetes.client.exceptions.ApiException as exc:
            print(f"error: failed to annotate storage '{node_name}': {exc.reason}", file=sys.stderr)
            return ERROR

        try:
            _apply_drain_taints(node_name)
        except kubernetes.client.exceptions.ApiException as exc:
            print(f"error: failed to taint node '{node_name}': {exc.reason}", file=sys.stderr)
            try:
                _remove_drain_annotations(node_name)
            except kubernetes.client.exceptions.ApiException as rollback_exc:
                print(f"error: failed to roll back storage annotation on '{node_name}': {rollback_exc.reason}", file=sys.stderr)
            return ERROR
        return OK

    return for_each_node(args.nodes, action, "Drained node '{}'.")
