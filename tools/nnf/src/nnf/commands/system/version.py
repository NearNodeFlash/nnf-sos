"""system version sub-command: display NNF controller version labels."""

import argparse
import json
import sys
from typing import Any, Dict

import kubernetes.client.exceptions  # type: ignore[import-untyped]

from nnf import k8s
from nnf.commands import add_command_parser


_NNF_SYSTEM_NS = "nnf-system"
_DEPLOY_NAME = "nnf-controller-manager"


def register(subparsers: argparse._SubParsersAction) -> None:  # type: ignore[type-arg]
    """Register the system version sub-command."""
    parser: argparse.ArgumentParser = add_command_parser(
        subparsers,
        "version",
        help="Display version information for the NNF controller.",
    )
    parser.set_defaults(func=run)


def _get_deployment_labels() -> Dict[str, Any]:
    """Fetch labels from the nnf-controller-manager Deployment."""
    deploy = k8s.get_deployment(
        name=_DEPLOY_NAME,
        namespace=_NNF_SYSTEM_NS,
    )
    return dict(deploy.metadata.labels or {})


def run(args: argparse.Namespace) -> int:
    """Execute the system version sub-command."""
    try:
        labels = _get_deployment_labels()
    except kubernetes.client.exceptions.ApiException as exc:
        print(
            f"error: failed to get deployment {_DEPLOY_NAME}: {exc.reason}",
            file=sys.stderr,
        )
        return 1
    print(json.dumps(labels, indent=2))
    return 0
