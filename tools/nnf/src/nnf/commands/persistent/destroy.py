"""persistent destroy sub-command: destroy a DWS PersistentStorageInstance.

Builds a #DW destroy_persistent directive, submits it as a Workflow, and drives
that Workflow through every state.  The Workflow is always deleted on exit,
whether the command succeeds or fails.
"""

import argparse
import logging
import os
import sys

import kubernetes.client.exceptions  # type: ignore[import-untyped]

from nnf import crd
from nnf import k8s
from nnf import utils
from nnf import workflow
from nnf.commands import add_command_parser, add_workflow_arguments


LOGGER = logging.getLogger(__name__)


def register(subparsers: argparse._SubParsersAction) -> None:  # type: ignore[type-arg]
    """Register the persistent destroy sub-command."""
    parser: argparse.ArgumentParser = add_command_parser(
        subparsers,
        "destroy",
        help="Destroy a persistent storage instance.",
    )
    parser.add_argument(
        "--name",
        required=True,
        help="Name of the persistent storage instance to destroy.",
    )
    add_workflow_arguments(parser)
    parser.set_defaults(func=run)


def _get_psi_user_id(name: str, namespace: str) -> int:
    """Fetch the userID from the PersistentStorageInstance spec."""
    psi = k8s.get_object(
        group=crd.DWS_GROUP,
        version=crd.DWS_VERSION,
        namespace=namespace,
        plural=crd.DWS_PERSISTENT_STORAGE_PLURAL,
        name=name,
    )
    return int(psi["spec"]["userID"])


def run(args: argparse.Namespace) -> int:
    """Execute the persistent destroy sub-command."""
    try:
        utils.validate_k8s_name(args.name, "nnf-destroy-persistent-")
    except ValueError as exc:
        print(f"error: invalid --name: {exc}", file=sys.stderr)
        return 1

    user_id: int
    if args.user_id is not None:
        user_id = args.user_id
    else:
        try:
            user_id = _get_psi_user_id(args.name, args.namespace)
        except kubernetes.client.exceptions.ApiException as exc:
            print(
                f"error: failed to get PersistentStorageInstance '{args.name}': {exc.reason}",
                file=sys.stderr,
            )
            return 1
        except (KeyError, ValueError) as exc:
            print(
                f"error: could not read userID from PersistentStorageInstance '{args.name}': {exc}",
                file=sys.stderr,
            )
            return 1
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
        print(f"Destroyed persistent storage '{args.name}'.")
    return rc
