"""Tests for nnf.hostlist module."""

import pytest

from nnf.hostlist import BadHostlist, compress, expand, expand_args


# ---------------------------------------------------------------------------
# expand()
# ---------------------------------------------------------------------------


def test_expand_single_host() -> None:
    assert expand("rabbit-node-1") == ["rabbit-node-1"]


def test_expand_simple_range() -> None:
    assert expand("node[1-3]") == ["node1", "node2", "node3"]


def test_expand_enumeration() -> None:
    assert expand("node[1,3,5]") == ["node1", "node3", "node5"]


def test_expand_mixed_range_and_enumeration() -> None:
    result = expand("node[1-3,7,10-12]")
    assert result == ["node1", "node2", "node3", "node7", "node10", "node11", "node12"]


def test_expand_zero_padded() -> None:
    assert expand("node[001-003]") == ["node001", "node002", "node003"]


def test_expand_comma_separated_hosts() -> None:
    assert expand("node1,node2,node3") == ["node1", "node2", "node3"]


def test_expand_realistic_hostlist() -> None:
    result = expand("rzadams[201-208]")
    expected = [f"rzadams{i}" for i in range(201, 209)]
    assert result == expected


def test_expand_realistic_mixed() -> None:
    result = expand("rzadams[201,207-208]")
    assert result == ["rzadams201", "rzadams207", "rzadams208"]


# ---------------------------------------------------------------------------
# expand_args()
# ---------------------------------------------------------------------------


def test_expand_args_none() -> None:
    assert expand_args(None) == []


def test_expand_args_plain_names() -> None:
    assert expand_args(["rabbit-1", "rabbit-2"]) == ["rabbit-1", "rabbit-2"]


def test_expand_args_comma_separated() -> None:
    assert expand_args(["a,b,c"]) == ["a", "b", "c"]


def test_expand_args_hostlist_pattern() -> None:
    result = expand_args(["node[1-3]"])
    assert result == ["node1", "node2", "node3"]


def test_expand_args_mixed_arguments() -> None:
    result = expand_args(["node[1-3]", "gpu[1,2]"])
    assert result == ["node1", "node2", "node3", "gpu1", "gpu2"]


def test_expand_args_comma_with_brackets() -> None:
    result = expand_args(["node[1-2],gpu[1-2]"])
    assert result == ["node1", "node2", "gpu1", "gpu2"]


# ---------------------------------------------------------------------------
# compress()
# ---------------------------------------------------------------------------


def test_compress_empty() -> None:
    assert compress([]) == ""


def test_compress_single() -> None:
    assert compress(["rabbit-0"]) == "rabbit-0"


def test_compress_consecutive() -> None:
    result = compress(["rabbit-0", "rabbit-1", "rabbit-2"])
    # python-hostlist may format slightly differently — just check round-trip
    expanded = expand(result)
    assert sorted(expanded) == sorted(["rabbit-0", "rabbit-1", "rabbit-2"])


def test_compress_gaps() -> None:
    result = compress(["rabbit-0", "rabbit-1", "rabbit-3", "rabbit-5"])
    expanded = expand(result)
    assert sorted(expanded) == sorted(["rabbit-0", "rabbit-1", "rabbit-3", "rabbit-5"])


def test_compress_different_prefixes() -> None:
    result = compress(["rabbit-0", "compute-1"])
    expanded = expand(result)
    assert sorted(expanded) == sorted(["rabbit-0", "compute-1"])


def test_compress_round_trip() -> None:
    """Expanding a compressed hostlist yields the original names."""
    names = [f"rzadams{i}" for i in range(201, 209)]
    compressed = compress(names)
    assert expand(compressed) == sorted(names)


# ---------------------------------------------------------------------------
# Error handling
# ---------------------------------------------------------------------------


def test_expand_unmatched_bracket_raises() -> None:
    with pytest.raises(BadHostlist, match="invalid hostlist"):
        expand("rabbit-[")


def test_expand_reversed_range_raises() -> None:
    with pytest.raises(BadHostlist, match="invalid hostlist"):
        expand("rabbit-[5-2]")


def test_expand_args_bad_pattern_raises() -> None:
    with pytest.raises(BadHostlist, match="invalid hostlist"):
        expand_args(["good-node", "bad-["])
