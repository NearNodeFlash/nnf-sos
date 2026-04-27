"""Shared utility helpers for the nnf CLI."""

import re
from typing import Dict


_DNS_SUBDOMAIN_RE = re.compile(r'^[a-z0-9]([a-z0-9\-]*[a-z0-9])?$')
_MAX_K8S_NAME_LEN = 253


def validate_k8s_name(name: str, resource_prefix: str) -> None:
    """Raise ``ValueError`` if *name* would produce an invalid K8s resource name.

    The full resource name is ``{resource_prefix}{name}`` and must conform to
    DNS subdomain rules: lowercase alphanumeric and hyphens, must start and end
    with an alphanumeric character, max 253 characters.
    """
    full = f"{resource_prefix}{name}"
    if len(full) > _MAX_K8S_NAME_LEN:
        raise ValueError(
            f"resulting resource name is {len(full)} characters "
            f"(max {_MAX_K8S_NAME_LEN}): '{full}'"
        )
    if not _DNS_SUBDOMAIN_RE.match(full):
        raise ValueError(
            f"resulting resource name '{full}' is not a valid DNS subdomain: "
            f"must be lowercase alphanumeric or '-' and start/end with alphanumeric"
        )


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
