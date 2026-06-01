"""HPC-style hostlist expansion and compression.

Wraps the python-hostlist library to provide a consistent interface used
throughout the nnf CLI.
"""

from typing import List, Optional

import hostlist as _hl


class BadHostlist(ValueError):
    """Raised when a hostlist pattern is malformed."""


def expand(pattern: str) -> List[str]:
    """Expand a single bracketed hostlist pattern into individual hostnames.

    Examples:
        expand("node[1-3]")       → ["node1", "node2", "node3"]
        expand("node[1,3,5-7]")   → ["node1", "node3", "node5", "node6", "node7"]
        expand("node[01-03]")     → ["node01", "node02", "node03"]
        expand("node5")           → ["node5"]

    Raises BadHostlist for malformed patterns.
    """
    try:
        return _hl.expand_hostlist(pattern)
    except _hl.BadHostlist as exc:
        raise BadHostlist(f"invalid hostlist '{pattern}': {exc}") from exc


def expand_args(values: Optional[List[str]]) -> List[str]:
    """Normalize a list of argparse node arguments into expanded hostnames.

    Handles:
      - None → []
      - Comma-splitting ("a,b,c" → ["a","b","c"])
      - Hostlist expansion ("node[1-3]" → ["node1","node2","node3"])

    This is the common normalization function for any argparse nargs="+"
    node argument in the CLI.

    Raises BadHostlist for malformed patterns.
    """
    if values is None:
        return []
    result: List[str] = []
    for v in values:
        result.extend(expand(v))
    return result


def compress(names: List[str]) -> str:
    """Compress a list of hostnames into bracketed range notation.

    Example: ["node1","node2","node3"] → "node[1-3]"
    Returns empty string for an empty list.
    """
    if not names:
        return ""
    return _hl.collect_hostlist(names)
