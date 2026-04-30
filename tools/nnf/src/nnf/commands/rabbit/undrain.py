"""rabbit undrain sub-command: undrain a Rabbit node.

Removes the ``cray.nnf.node.drain`` taint from the Kubernetes node and removes
the ``drain_date`` and ``drain_reason`` annotations from the corresponding DWS
Storage resource.
"""

import argparse
import logging
import sys

import kubernetes.client.exceptions  # type: ignore[import-untyped]

from nnf.commands import add_command_parser
from nnf.commands.rabbit._helpers import (
    ERROR,
    OK,
    for_each_node,
    remove_drain_annotations,
    remove_drain_taints,
)


LOGGER = logging.getLogger(__name__)


def register(subparsers: argparse._SubParsersAction) -> None:  # type: ignore[type-arg]
    """Register the rabbit undrain sub-command."""
    parser: argparse.ArgumentParser = add_command_parser(
        subparsers,
        "undrain",
        help="Undrain a Rabbit node so new workflows can be scheduled to it.",
    )
    parser.add_argument(
        "nodes",
        nargs="+",
        metavar="NODE",
        help="One or more Rabbit node names to undrain.",
    )
    parser.set_defaults(func=run)


def run(args: argparse.Namespace) -> int:
    """Execute the rabbit undrain sub-command."""

    def action(node_name: str) -> int:
        try:
            remove_drain_taints(node_name)
        except kubernetes.client.exceptions.ApiException as exc:
            print(f"error: failed to untaint node '{node_name}': {exc.reason}", file=sys.stderr)
            return ERROR

        try:
            remove_drain_annotations(node_name)
        except kubernetes.client.exceptions.ApiException as exc:
            print(f"error: failed to remove annotations from storage '{node_name}': {exc.reason}", file=sys.stderr)
            return ERROR
        return OK

    return for_each_node(args.nodes, action, "Undrained node '{}'.")
