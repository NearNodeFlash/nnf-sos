"""persistent share sub-command: mark a PSI as shared (ignore UID checks).

Adds the ``dataworkflowservices.github.io/ignore-uid`` annotation to the
PersistentStorageInstance so that any user may access it.
"""

import argparse
import sys

import kubernetes.client.exceptions  # type: ignore[import-untyped]

from nnf import crd
from nnf import k8s
from nnf.commands import add_command_parser


def register(subparsers: argparse._SubParsersAction) -> None:  # type: ignore[type-arg]
    """Register the persistent share sub-command."""
    parser: argparse.ArgumentParser = add_command_parser(
        subparsers,
        "share",
        help="Share a persistent storage instance (add ignore-uid annotation).",
    )
    parser.add_argument(
        "--name",
        required=True,
        help="Name of the persistent storage instance to share.",
    )
    parser.add_argument(
        "--namespace",
        default="default",
        help="Kubernetes namespace (default: default).",
    )
    parser.set_defaults(func=run)


def run(args: argparse.Namespace) -> int:
    """Execute the persistent share sub-command."""
    body = {
        "metadata": {
            "annotations": {
                crd.DWS_IGNORE_UID_ANNOTATION: "true",
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
            f"error: failed to share PersistentStorageInstance '{args.name}': {exc.reason}",
            file=sys.stderr,
        )
        return 1
    print(f"Shared persistent storage '{args.name}'.")
    return 0
