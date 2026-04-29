"""Shared helpers for rabbit sub-commands."""

from typing import Callable, List


# Action return codes.
OK = 0
ERROR = 1


def for_each_node(
    nodes: List[str],
    action: Callable[[str], int],
    success_msg: str,
) -> int:
    """Run *action* on each node, collecting errors.

    *action* receives a node name and returns one of:

    - ``OK`` (0): success — print *success_msg* and continue.
    - ``ERROR`` (1): skip this node, continue to the next.

    *success_msg* is a format string with one ``{}`` placeholder for the node
    name.
    """
    errors = 0
    for node_name in nodes:
        rc = action(node_name)
        if rc != OK:
            errors += 1
            continue
        print(success_msg.format(node_name))
    return 1 if errors else 0
