"""destroy_persistent sub-command: destroy a DWS PersistentStorageInstance.

Builds a #DW destroy_persistent directive, submits it as a Workflow, and drives
that Workflow through every state.  The Workflow is always deleted on exit,
whether the command succeeds or fails.
"""

import argparse
import logging
import os

import kubernetes.client.exceptions  # type: ignore[import-untyped]

from nnf import crd
from nnf import k8s
from nnf import utils
from nnf import workflow
from nnf.commands import add_command_parser


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
        default=workflow.DEFAULT_TIMEOUT,
        help=f"Seconds to wait for each workflow state (default: {workflow.DEFAULT_TIMEOUT}).",
    )
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

    try:
        k8s.create_object(
            group=crd.DWS_GROUP,
            version=crd.DWS_VERSION,
            namespace=args.namespace,
            plural=crd.DWS_WORKFLOW_PLURAL,
            body=wf.manifest,
        )
    except kubernetes.client.exceptions.ApiException as exc:
        print(f"error: failed to create Workflow: {exc.reason} (HTTP {exc.status})")
        LOGGER.info("Full error body: %s", exc.body)
        k8s.debug_api_group(crd.DWS_GROUP)
        return 2

    print(f"Workflow '{workflow_name}' created.")

    try:
        ok, msg = workflow.run_to_completion(wf, args.timeout)
        if not ok:
            print(f"error: {msg}")
            return 2
    except Exception:
        workflow.teardown_and_delete(wf.name, wf.namespace, args.timeout)
        raise

    print(f"Persistent storage '{args.name}' destroyed successfully.")
    return 0
