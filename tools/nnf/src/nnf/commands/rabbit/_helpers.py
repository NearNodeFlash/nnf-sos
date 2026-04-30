"""Shared helpers for rabbit sub-commands."""

import logging
from typing import Any, Callable, Dict, List

from nnf import crd
from nnf import k8s


LOGGER = logging.getLogger(__name__)

# Action return codes.
OK = 0
ERROR = 1

TAINT_KEY = "cray.nnf.node.drain"
TAINT_VALUE = "true"


def remove_drain_taints(node_name: str) -> None:
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


def remove_drain_annotations(node_name: str) -> None:
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


def for_each_node(
    nodes: List[str],
    action: Callable[[str], int],
    success_msg: str,
) -> int:
    """Run *action* on each node, collecting errors.

    *action* receives a node name and returns one of:

    - ``OK`` (0): success — print *success_msg* and continue.
    - ``ERROR`` (1): skip this node, continue to the next.

    *success_msg* is a format string with one ``{}`` placeholder for the node
    name.
    """
    errors = 0
    for node_name in nodes:
        rc = action(node_name)
        if rc != OK:
            errors += 1
            continue
        print(success_msg.format(node_name))
    return 1 if errors else 0
