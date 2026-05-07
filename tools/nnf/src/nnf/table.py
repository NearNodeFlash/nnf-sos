"""Shared table-printing utilities."""

from typing import List, Tuple


def print_table(
    headers: Tuple[str, ...],
    rows: List[Tuple[str, ...]],
    right_align: Tuple[int, ...] = (),
) -> None:
    """Print a column-aligned table with optional right-alignment.

    Always prints the header row, even when *rows* is empty.
    """
    expected_width = len(headers)
    for index, row in enumerate(rows, start=1):
        if len(row) != expected_width:
            raise ValueError(
                f"row {index} has {len(row)} cells, expected {expected_width}"
            )

    widths = [len(h) for h in headers]
    for row in rows:
        for i, cell in enumerate(row):
            widths[i] = max(widths[i], len(cell))
    parts = []
    for i, (h, w) in enumerate(zip(headers, widths)):
        parts.append(h.rjust(w) if i in right_align else h.ljust(w))
    print("  ".join(parts))
    for row in rows:
        parts = []
        for i, (cell, w) in enumerate(zip(row, widths)):
            parts.append(cell.rjust(w) if i in right_align else cell.ljust(w))
        print("  ".join(parts))
