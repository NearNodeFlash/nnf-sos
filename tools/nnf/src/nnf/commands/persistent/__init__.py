"""persistent sub-command group: manage DWS PersistentStorageInstances."""

import argparse

from nnf.commands import add_command_parser
from nnf.commands.persistent import create, destroy


def register(subparsers: argparse._SubParsersAction) -> None:  # type: ignore[type-arg]
    """Register the persistent sub-command group."""
    persistent_parser: argparse.ArgumentParser = add_command_parser(
        subparsers,
        "persistent",
        help="Manage persistent storage instances.",
    )
    sub = persistent_parser.add_subparsers(dest="persistent_command", metavar="<action>")
    sub.required = True

    create.register(sub)
    destroy.register(sub)
