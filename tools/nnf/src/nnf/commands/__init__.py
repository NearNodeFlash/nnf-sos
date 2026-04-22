"""Sub-command modules for the nnf CLI."""

import argparse


def add_common_arguments(parser: argparse.ArgumentParser) -> None:
    """Add arguments shared by the top-level parser and subcommands."""
    parser.add_argument(
        "-v",
        "--verbose",
        action="store_true",
        default=argparse.SUPPRESS,
        help="Enable verbose logging for workflow progress and Kubernetes operations.",
    )


def add_command_parser(
    subparsers: argparse._SubParsersAction,  # type: ignore[type-arg]
    name: str,
    **kwargs: object,
) -> argparse.ArgumentParser:
    """Create a subcommand parser with the shared common arguments attached."""
    common_parser = argparse.ArgumentParser(add_help=False)
    add_common_arguments(common_parser)
    return subparsers.add_parser(name, parents=[common_parser], **kwargs)
