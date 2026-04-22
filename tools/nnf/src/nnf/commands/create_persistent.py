"""create_persistent sub-command: create a DWS PersistentStorageInstance.

Builds a #DW create_persistent directive, submits it as a Workflow, and drives
that Workflow through every state.  The Workflow is always deleted on exit,
whether the command succeeds or fails.
"""

import argparse
import functools
import logging
import os
import random
from typing import List, Optional

import kubernetes.client.exceptions  # type: ignore[import-untyped]

from nnf import crd
from nnf import k8s
from nnf import utils
from nnf import workflow
from nnf.commands import add_command_parser


LOGGER = logging.getLogger(__name__)


def register(subparsers: argparse._SubParsersAction) -> None:  # type: ignore[type-arg]
    """Register the create_persistent sub-command."""
    parser: argparse.ArgumentParser = add_command_parser(
        subparsers,
        "create_persistent",
        help="Create a persistent storage instance.",
    )
    parser.add_argument(
        "--name",
        required=True,
        help="Name of the persistent storage instance.",
    )
    parser.add_argument(
        "--fs-type",
        required=True,
        choices=["raw", "xfs", "gfs2", "lustre"],
        dest="fs_type",
        help="Filesystem type for the persistent storage.",
    )
    parser.add_argument(
        "--capacity",
        default=None,
        help='Allocation capacity per Rabbit (e.g. "1GiB", "500MiB"). Required unless the profile specifies standaloneMgtPoolName.',
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
        "--rabbits",
        nargs="+",
        default=None,
        metavar="RABBIT",
        help="One or more Rabbit node names to allocate storage on (e.g. rabbit-node-0). Mutually exclusive with --rabbit-count.",
    )
    parser.add_argument(
        "--rabbits-mdt",
        nargs="+",
        default=None,
        dest="rabbits_mdt",
        metavar="RABBIT",
        help="Rabbit nodes to use for mdt and mgtmdt allocation sets (default: --rabbits). Cannot be used with --rabbit-count.",
    )
    parser.add_argument(
        "--rabbits-mgt",
        nargs="+",
        default=None,
        dest="rabbits_mgt",
        metavar="RABBIT",
        help="Rabbit nodes to use for mgt allocation sets (default: --rabbits). Cannot be used with --rabbit-count.",
    )
    parser.add_argument(
        "--rabbit-count",
        type=int,
        default=None,
        dest="rabbit_count",
        metavar="N",
        help="Number of Rabbits to pick at random from the default/default SystemConfiguration. Mutually exclusive with --rabbits, --rabbits-mdt, --rabbits-mgt.",
    )
    parser.add_argument(
        "--alloc-count",
        type=int,
        default=1,
        dest="alloc_count",
        help="Number of allocations per Rabbit for the AllocatePerCompute strategy (default: 1).",
    )
    parser.add_argument(
        "--profile",
        default=None,
        help="Storage profile name to pass as profile= in the #DW directive (optional).",
    )
    parser.add_argument(
        "--timeout",
        type=int,
        default=workflow.DEFAULT_TIMEOUT,
        help=f"Seconds to wait for each workflow state (default: {workflow.DEFAULT_TIMEOUT}).",
    )
    parser.set_defaults(func=run)


def _split_nodes(values: Optional[List[str]]) -> List[str]:
    """Flatten a list of node name arguments, splitting each on commas.

    Allows users to write either ``--rabbits a b c`` or ``--rabbits a,b,c``
    (or a mix of both).
    """
    if values is None:
        return []
    result = []
    for v in values:
        result.extend(n.strip() for n in v.split(",") if n.strip())
    return result


def run(args: argparse.Namespace) -> int:
    """Execute the create_persistent sub-command."""
    # Resolve profile and check standaloneMgtPoolName (Lustre only).
    standalone_mgt = False
    if args.profile is not None and args.fs_type == "lustre":
        try:
            profile = workflow.get_storage_profile(args.profile)
        except kubernetes.client.exceptions.ApiException as exc:
            print(f"error: failed to fetch storage profile '{args.profile}': {exc}")
            return 1
        standalone_mgt = workflow.has_standalone_mgt(profile)

    if standalone_mgt:
        if args.capacity is not None:
            print(
                f"error: --capacity cannot be used with profile '{args.profile}' "
                f"because it specifies standaloneMgtPoolName"
            )
            return 1
    else:
        if args.capacity is None:
            print("error: --capacity is required when the profile does not specify standaloneMgtPoolName")
            return 1
        try:
            utils.parse_capacity(args.capacity)
        except ValueError as exc:
            print(f"error: {exc}")
            return 1

    # Validate --rabbit-count mutual exclusion.
    if args.rabbit_count is not None:
        conflicts = [
            name
            for name, val in [
                ("--rabbits", args.rabbits),
                ("--rabbits-mdt", args.rabbits_mdt),
                ("--rabbits-mgt", args.rabbits_mgt),
            ]
            if val is not None
        ]
        if conflicts:
            print(f"error: --rabbit-count cannot be combined with: {', '.join(conflicts)}")
            return 1
        if args.rabbit_count < 1:
            print("error: --rabbit-count must be at least 1")
            return 1
        try:
            all_rabbits = workflow.get_rabbits_from_system_config()
        except (kubernetes.client.exceptions.ApiException, ValueError) as exc:
            print(f"error: failed to read SystemConfiguration: {exc}")
            return 1
        if args.rabbit_count > len(all_rabbits):
            print(
                f"error: --rabbit-count {args.rabbit_count} exceeds the "
                f"{len(all_rabbits)} Rabbit(s) in the SystemConfiguration"
            )
            return 1
        rabbits = random.sample(all_rabbits, args.rabbit_count)
        rabbits_mdt = None
        rabbits_mgt = None
    elif args.rabbits is None:
        print("error: one of --rabbits or --rabbit-count is required")
        return 1
    else:
        rabbits = _split_nodes(args.rabbits)
        rabbits_mdt = _split_nodes(args.rabbits_mdt) or None
        rabbits_mgt = _split_nodes(args.rabbits_mgt) or None

    if standalone_mgt:
        if len(rabbits) != 1:
            print(
                f"error: profile '{args.profile}' specifies standaloneMgtPoolName "
                f"and requires exactly 1 Rabbit, but {len(rabbits)} were provided"
            )
            return 1
        if args.alloc_count != 1:
            print(
                f"error: profile '{args.profile}' specifies standaloneMgtPoolName "
                f"and requires --alloc-count 1"
            )
            return 1

    user_id: int = args.user_id if args.user_id is not None else os.getuid()
    group_id: int = args.group_id if args.group_id is not None else os.getgid()
    dw_directive = f"#DW create_persistent name={args.name} type={args.fs_type}"
    if not standalone_mgt:
        dw_directive += f" capacity={args.capacity}"
    if args.profile is not None:
        dw_directive += f" profile={args.profile}"

    workflow_name = f"nnf-create-persistent-{args.name}"
    proposal_hook = functools.partial(
        workflow.fill_servers_default,
        rabbits=rabbits,
        timeout=args.timeout,
        directive_index=0,
        alloc_count=args.alloc_count,
        rabbits_mdt=rabbits_mdt,
        rabbits_mgt=rabbits_mgt,
    )
    wf = workflow.WorkflowRun(
        name=workflow_name,
        namespace=args.namespace,
        user_id=user_id,
        group_id=group_id,
        dw_directives=[dw_directive],
        state_hooks={"Proposal": [proposal_hook]},
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

    ok, msg = workflow.run_to_completion(wf, args.timeout)
    if not ok:
        print(f"error: {msg}")
        return 2

    print(f"Persistent storage '{args.name}' created successfully.")
    return 0
