"""rabbit sub-command group: manage Rabbit node state."""

import argparse

from nnf.commands import add_command_parser
from nnf.commands.rabbit import df, disable, drain, enable, undrain


def register(subparsers: argparse._SubParsersAction) -> None:  # type: ignore[type-arg]
    """Register the rabbit sub-command group."""
    rabbit_parser: argparse.ArgumentParser = add_command_parser(
        subparsers,
        "rabbit",
        help="Manage Rabbit node state.",
    )
    sub = rabbit_parser.add_subparsers(dest="rabbit_command", metavar="<action>")
    sub.required = True

    df.register(sub)
    disable.register(sub)
    drain.register(sub)
    enable.register(sub)
    undrain.register(sub)
