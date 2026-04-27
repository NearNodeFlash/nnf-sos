"""destroy_persistent sub-command: destroy a DWS PersistentStorageInstance.

Builds a #DW destroy_persistent directive, submits it as a Workflow, and drives
that Workflow through every state.  The Workflow is always deleted on exit,
whether the command succeeds or fails.
"""

import argparse
import logging
import os

from nnf import utils
from nnf import workflow
from nnf.commands import add_command_parser, add_workflow_arguments


LOGGER = logging.getLogger(__name__)


def register(subparsers: argparse._SubParsersAction) -> None:  # type: ignore[type-arg]
    """Register the destroy_persistent sub-command."""
    parser: argparse.ArgumentParser = add_command_parser(
        subparsers,
        "destroy_persistent",
        help="Destroy a persistent storage instance.",
    )
    parser.add_argument(
        "--name",
        required=True,
        help="Name of the persistent storage instance to destroy.",
    )
    add_workflow_arguments(parser)
    parser.set_defaults(func=run)


def run(args: argparse.Namespace) -> int:
    """Execute the destroy_persistent sub-command."""
    try:
        utils.validate_k8s_name(args.name, "nnf-destroy-persistent-")
    except ValueError as exc:
        print(f"error: invalid --name: {exc}")
        return 1

    user_id: int = args.user_id if args.user_id is not None else os.getuid()
    group_id: int = args.group_id if args.group_id is not None else os.getgid()
    dw_directive = f"#DW destroy_persistent name={args.name}"

    workflow_name = f"nnf-destroy-persistent-{args.name}"
    wf = workflow.WorkflowRun(
        name=workflow_name,
        namespace=args.namespace,
        user_id=user_id,
        group_id=group_id,
        dw_directives=[dw_directive],
    )

    rc = workflow.create_and_run(wf, args.timeout)
    if rc == 0:
        print(f"Persistent storage '{args.name}' destroyed successfully.")
    return rc
