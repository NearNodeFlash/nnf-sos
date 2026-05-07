"""system sub-command group: cluster-level administration."""

import argparse

from nnf.commands import add_command_parser
from nnf.commands.system import df, flowschema, state, version


def register(subparsers: argparse._SubParsersAction) -> None:  # type: ignore[type-arg]
    """Register the system sub-command group."""
    system_parser: argparse.ArgumentParser = add_command_parser(
        subparsers,
        "system",
        help="Cluster-level administration.",
    )
    sub = system_parser.add_subparsers(dest="system_command", metavar="<action>")
    sub.required = True

    df.register(sub)
    flowschema.register(sub)
    state.register(sub)
    version.register(sub)
