"""Shared utility helpers for the nnf CLI."""

from typing import Dict


def parse_capacity(value: str) -> int:
    """Convert a human-readable capacity string to bytes.

    Accepts values like '1GiB', '500MiB', '2TiB', or a plain integer (bytes).
    """
    suffixes: Dict[str, int] = {
        "TIB": 1024**4,
        "GIB": 1024**3,
        "MIB": 1024**2,
        "KIB": 1024,
        "TB": 10**12,
        "GB": 10**9,
        "MB": 10**6,
        "KB": 10**3,
    }
    upper = value.strip().upper()
    for suffix, multiplier in suffixes.items():
        if upper.endswith(suffix):
            number_part = upper[: -len(suffix)].strip()
            try:
                return int(float(number_part) * multiplier)
            except ValueError as exc:
                raise ValueError(f"Invalid capacity value: {value!r}") from exc
    try:
        return int(value)
    except ValueError as exc:
        raise ValueError(f"Invalid capacity value: {value!r}") from exc
