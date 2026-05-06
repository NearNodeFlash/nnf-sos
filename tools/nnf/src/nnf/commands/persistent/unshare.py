"""persistent unshare sub-command: remove shared access from a PSI.

Removes the ``dataworkflowservices.github.io/ignore-uid`` annotation from the
PersistentStorageInstance so that only the owning user may access it.
"""

import argparse
import sys

import kubernetes.client.exceptions  # type: ignore[import-untyped]

from nnf import crd
from nnf import k8s
from nnf.commands import add_command_parser


def register(subparsers: argparse._SubParsersAction) -> None:  # type: ignore[type-arg]
    """Register the persistent unshare sub-command."""
    parser: argparse.ArgumentParser = add_command_parser(
        subparsers,
        "unshare",
        help="Unshare a persistent storage instance (remove ignore-uid annotation).",
    )
    parser.add_argument(
        "--name",
        required=True,
        help="Name of the persistent storage instance to unshare.",
    )
    parser.add_argument(
        "--namespace",
        default="default",
        help="Kubernetes namespace (default: default).",
    )
    parser.set_defaults(func=run)


def run(args: argparse.Namespace) -> int:
    """Execute the persistent unshare sub-command."""
    body = {
        "metadata": {
            "annotations": {
                crd.DWS_IGNORE_UID_ANNOTATION: None,
            }
        }
    }
    try:
        k8s.patch_object(
            group=crd.DWS_GROUP,
            version=crd.DWS_VERSION,
            namespace=args.namespace,
            plural=crd.DWS_PERSISTENT_STORAGE_PLURAL,
            name=args.name,
            body=body,
        )
    except kubernetes.client.exceptions.ApiException as exc:
        print(
            f"error: failed to unshare PersistentStorageInstance '{args.name}': {exc.reason}",
            file=sys.stderr,
        )
        return 1
    print(f"Unshared persistent storage '{args.name}'.")
    return 0
