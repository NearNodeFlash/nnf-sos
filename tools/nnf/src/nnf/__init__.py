"""nnf CLI entrypoint."""

import argparse
import importlib
import logging
import pkgutil
import sys

import kubernetes.config  # type: ignore[import-untyped]

from nnf import commands as _commands_pkg
from nnf import k8s
from nnf.commands import add_common_arguments


def build_parser() -> argparse.ArgumentParser:
    """Create the CLI argument parser."""
    parser = argparse.ArgumentParser(
        prog="nnf",
        description="CLI tool for managing NNF and DWS resources.",
    )
    parser.add_argument(
        "--kubeconfig",
        default=None,
        help=(
            "Optional path to a kubeconfig file. If omitted, uses the default "
            "kubeconfig resolution (for example KUBECONFIG or ~/.kube/config) "
            "and falls back to in-cluster config when available."
        ),
    )
    add_common_arguments(parser)

    subparsers = parser.add_subparsers(dest="command", metavar="<command>")
    subparsers.required = True

    for module_info in pkgutil.iter_modules(_commands_pkg.__path__):
        mod = importlib.import_module(f"nnf.commands.{module_info.name}")
        if hasattr(mod, "register"):
            mod.register(subparsers)

    return parser


def main() -> None:
    """Parse arguments and dispatch to the appropriate sub-command."""
    parser = build_parser()

    args = parser.parse_args()

    logging.basicConfig(
        level=logging.INFO if getattr(args, "verbose", False) else logging.WARNING,
        format="%(message)s",
    )

    try:
        k8s.load_config(kubeconfig=args.kubeconfig)
    except kubernetes.config.ConfigException as exc:
        parser.exit(2, f"error: failed to load Kubernetes config: {exc}\n")

    exit_code: int = args.func(args)
    sys.exit(exit_code)


if __name__ == "__main__":
    main()
