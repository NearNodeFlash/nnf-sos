"""Sub-command modules for the nnf CLI."""

import argparse

from nnf import workflow as _workflow


def add_common_arguments(parser: argparse.ArgumentParser) -> None:
    """Add arguments shared by the top-level parser and subcommands."""
    parser.add_argument(
        "-v",
        "--verbose",
        action="store_true",
        default=argparse.SUPPRESS,
        help="Enable verbose logging for workflow progress and Kubernetes operations.",
    )


def add_command_parser(
    subparsers: argparse._SubParsersAction,  # type: ignore[type-arg]
    name: str,
    **kwargs: object,
) -> argparse.ArgumentParser:
    """Create a subcommand parser with the shared common arguments attached."""
    common_parser = argparse.ArgumentParser(add_help=False)
    add_common_arguments(common_parser)
    return subparsers.add_parser(name, parents=[common_parser], **kwargs)


def add_workflow_arguments(parser: argparse.ArgumentParser) -> None:
    """Add Kubernetes/workflow arguments shared by all workflow subcommands.

    Adds --namespace, --user-id, --group-id, and --timeout.
    """
    parser.add_argument(
        "--namespace",
        default="default",
        help="Kubernetes namespace (default: default).",
    )
    parser.add_argument(
        "--user-id",
        type=int,
        default=None,
        dest="user_id",
        help="User ID that owns this storage (default: current user).",
    )
    parser.add_argument(
        "--group-id",
        type=int,
        default=None,
        dest="group_id",
        help="Group ID that owns this storage (default: current group).",
    )
    parser.add_argument(
        "--timeout",
        type=int,
        default=_workflow.DEFAULT_TIMEOUT,
        help=f"Seconds to wait for each workflow state (default: {_workflow.DEFAULT_TIMEOUT}).",
    )
