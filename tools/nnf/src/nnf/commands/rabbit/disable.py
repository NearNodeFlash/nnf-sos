"""rabbit disable sub-command: disable a Rabbit node.

Sets the DWS Storage resource ``spec.state`` to ``Disabled`` and annotates it
with ``disable_date`` and ``disable_reason`` so that Flux does not use the
Rabbit for allocations.
"""

import argparse
import datetime
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
    """Register the rabbit disable sub-command."""
    parser: argparse.ArgumentParser = add_command_parser(
        subparsers,
        "disable",
        help="Disable a Rabbit node so Flux does not use it for allocations.",
    )
    parser.add_argument(
        "nodes",
        nargs="+",
        metavar="NODE",
        help="One or more Rabbit node names to disable.",
    )
    parser.add_argument(
        "-r",
        "--reason",
        default="none",
        help="Reason for disabling the node (default: none).",
    )
    parser.set_defaults(func=run)


def _disable_storage(node_name: str, disable_date: str, disable_reason: str) -> None:
    """Set the Storage resource state to Disabled and add annotations."""
    body: Dict[str, Any] = {
        "metadata": {
            "annotations": {
                "disable_date": disable_date,
                "disable_reason": disable_reason,
            }
        },
        "spec": {
            "state": "Disabled",
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
    LOGGER.info("Disabled storage '%s' (reason=%s)", node_name, disable_reason)


def run(args: argparse.Namespace) -> int:
    """Execute the rabbit disable sub-command."""
    disable_date = datetime.datetime.now().strftime("%Y-%m-%dT%H:%M:%S")

    def action(node_name: str) -> int:
        try:
            _disable_storage(node_name, disable_date, args.reason)
        except kubernetes.client.exceptions.ApiException as exc:
            print(f"error: failed to disable storage '{node_name}': {exc.reason}", file=sys.stderr)
            return ERROR
        return OK

    return for_each_node(args.nodes, action, "Disabled node '{}'.")
