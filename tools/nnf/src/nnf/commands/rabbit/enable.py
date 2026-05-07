"""rabbit enable sub-command: enable a Rabbit node.

Sets the DWS Storage resource ``spec.state`` to ``Enabled`` and removes the
``disable_date`` and ``disable_reason`` annotations so that Flux can use the
Rabbit for allocations again.
"""

import argparse
import logging
import sys
from typing import Any, Dict

import kubernetes.client.exceptions  # type: ignore[import-untyped]

from nnf import crd
from nnf import k8s
from nnf.commands import add_command_parser
from nnf.commands.rabbit._helpers import ERROR, OK, for_each_node


LOGGER = logging.getLogger(__name__)


def register(subparsers: argparse._SubParsersAction) -> None:  # type: ignore[type-arg]
    """Register the rabbit enable sub-command."""
    parser: argparse.ArgumentParser = add_command_parser(
        subparsers,
        "enable",
        help="Enable a Rabbit node so Flux can use it for allocations.",
    )
    parser.add_argument(
        "nodes",
        nargs="+",
        metavar="NODE",
        help="One or more Rabbit node names to enable.",
    )
    parser.set_defaults(func=run)


def _enable_storage(node_name: str) -> None:
    """Set the Storage resource state to Enabled and remove disable annotations."""
    body: Dict[str, Any] = {
        "metadata": {
            "annotations": {
                "disable_date": None,
                "disable_reason": None,
            }
        },
        "spec": {
            "state": "Enabled",
        },
    }
    k8s.patch_object(
        group=crd.DWS_GROUP,
        version=crd.DWS_VERSION,
        namespace="default",
        plural=crd.DWS_STORAGE_PLURAL,
        name=node_name,
        body=body,
    )
    LOGGER.info("Enabled storage '%s'", node_name)


def run(args: argparse.Namespace) -> int:
    """Execute the rabbit enable sub-command."""

    def action(node_name: str) -> int:
        try:
            _enable_storage(node_name)
        except kubernetes.client.exceptions.ApiException as exc:
            print(f"error: failed to enable storage '{node_name}': {exc.reason}", file=sys.stderr)
            return ERROR
        return OK

    return for_each_node(args.nodes, action, "Enabled node '{}'.")
