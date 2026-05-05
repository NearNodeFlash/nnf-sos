"""Tests for the system state sub-command."""

import argparse
import pytest
import json
import subprocess
from typing import Any, Dict, List, Optional
from unittest.mock import MagicMock, patch

import kubernetes.client.exceptions  # type: ignore[import-untyped]

from nnf import crd
from nnf.commands.system.state import (
    _build_annotation_rows,
    _build_node_status_rows,
    _build_storage_status_rows,
    _compress_hostlist,
    _get_all_storages,
    _print_table,
    _run_cmd,
    _show_disabled_computes,
    run,
)


def _make_args(**kwargs: object) -> argparse.Namespace:
    defaults: Dict[str, object] = {}
    defaults.update(kwargs)
    return argparse.Namespace(**defaults)


# ---------------------------------------------------------------------------
# _compress_hostlist
# ---------------------------------------------------------------------------


def test_compress_hostlist_empty() -> None:
    assert _compress_hostlist([]) == ""


def test_compress_hostlist_single() -> None:
    assert _compress_hostlist(["rabbit-0"]) == "rabbit-0"


def test_compress_hostlist_consecutive() -> None:
    result = _compress_hostlist(["rabbit-0", "rabbit-1", "rabbit-2"])
    assert result == "rabbit-[0-2]"


def test_compress_hostlist_gaps() -> None:
    result = _compress_hostlist(["rabbit-0", "rabbit-1", "rabbit-3", "rabbit-5"])
    assert result == "rabbit-[0-1,3,5]"


def test_compress_hostlist_single_in_list() -> None:
    result = _compress_hostlist(["rabbit-5"])
    assert result == "rabbit-5"


def test_compress_hostlist_noncontiguous_ranges() -> None:
    result = _compress_hostlist([
        "rabbit-0", "rabbit-1", "rabbit-2",
        "rabbit-10", "rabbit-11",
    ])
    assert result == "rabbit-[0-2,10-11]"


def test_compress_hostlist_no_numeric_suffix() -> None:
    result = _compress_hostlist(["nodeA", "nodeB"])
    assert result == "nodeA,nodeB"


def test_compress_hostlist_different_prefixes() -> None:
    result = _compress_hostlist(["rabbit-0", "compute-1"])
    assert result == "compute-1,rabbit-0"


def test_compress_hostlist_multi_prefix_ranges() -> None:
    result = _compress_hostlist([
        "rabbit-compute-2", "rabbit-compute-3",
        "rabbit-compute-4", "rabbit-compute-5",
        "rabbit-node-1", "rabbit-node-2",
    ])
    assert result == "rabbit-compute-[2-5],rabbit-node-[1-2]"


def test_compress_hostlist_unsorted_input() -> None:
    result = _compress_hostlist(["rabbit-3", "rabbit-1", "rabbit-2"])
    assert result == "rabbit-[1-3]"


# ---------------------------------------------------------------------------
# _print_table
# ---------------------------------------------------------------------------


def test_print_table_basic(capsys: pytest.CaptureFixture[str]) -> None:
    _print_table(("A", "B"), [("x", "yy")], right_align=())
    out = capsys.readouterr().out
    assert "A" in out
    assert "B" in out


def test_print_table_right_align(capsys: pytest.CaptureFixture[str]) -> None:
    _print_table(("NUM",), [("42",)], right_align=(0,))
    out = capsys.readouterr().out
    # Right-aligned, so no trailing spaces before newline for single column
    assert "NUM" in out


def test_print_table_empty_rows(capsys: pytest.CaptureFixture[str]) -> None:
    _print_table(("A", "B"), [], right_align=())
    out = capsys.readouterr().out
    lines = out.strip().splitlines()
    assert len(lines) == 1
    assert "A" in lines[0]


def test_print_table_rejects_short_rows() -> None:
    with pytest.raises(ValueError, match="row 1 has 1 cells, expected 2"):
        _print_table(("A", "B"), [("x",)], right_align=())


def test_print_table_rejects_long_rows() -> None:
    with pytest.raises(ValueError, match="row 1 has 3 cells, expected 2"):
        _print_table(("A", "B"), [("x", "y", "z")], right_align=())


# ---------------------------------------------------------------------------
# _build_node_status_rows
# ---------------------------------------------------------------------------


def _make_nnfnode(servers: List[Dict[str, str]]) -> Dict[str, Any]:
    return {"status": {"servers": servers}}


def test_build_node_status_rows_basic() -> None:
    nodes = [
        _make_nnfnode([
            {"hostname": "rabbit-0", "status": "Ready", "health": "OK"},
            {"hostname": "rabbit-1", "status": "Ready", "health": "OK"},
        ]),
        _make_nnfnode([
            {"hostname": "rabbit-2", "status": "Failed", "health": "Critical"},
        ]),
    ]
    rows = _build_node_status_rows(nodes)
    assert len(rows) == 2
    # OK/Ready row
    assert rows[0] == ("OK", "Ready", "2", "rabbit-[0-1]")
    # Critical/Failed row
    assert rows[1] == ("Critical", "Failed", "1", "rabbit-2")


def test_build_node_status_rows_skips_null_hostname() -> None:
    nodes = [
        _make_nnfnode([
            {"hostname": None, "status": "Ready", "health": "OK"},
            {"hostname": "rabbit-0", "status": "Ready", "health": "OK"},
        ]),
    ]
    rows = _build_node_status_rows(nodes)
    assert len(rows) == 1
    assert rows[0][2] == "1"


def test_build_node_status_rows_empty() -> None:
    assert _build_node_status_rows([]) == []


def test_build_node_status_rows_includes_unexpected_values() -> None:
    nodes = [
        _make_nnfnode([
            {"hostname": "rabbit-9", "status": "Recovering", "health": "Degraded"},
            {"hostname": "rabbit-10", "status": None, "health": "OK"},
        ]),
    ]

    rows = _build_node_status_rows(nodes)

    assert ("OK", "<missing>", "1", "rabbit-10") in rows
    assert ("Degraded", "Recovering", "1", "rabbit-9") in rows


# ---------------------------------------------------------------------------
# _build_storage_status_rows
# ---------------------------------------------------------------------------


def _make_storage(
    name: str,
    state: str,
    status: str,
    annotations: Optional[Dict[str, str]] = None,
) -> Dict[str, Any]:
    s: Dict[str, Any] = {
        "metadata": {"name": name, "annotations": annotations or {}},
        "spec": {"state": state},
        "status": {"status": status},
    }
    return s


def test_build_storage_status_rows_basic() -> None:
    storages = [
        _make_storage("rabbit-0", "Enabled", "Ready"),
        _make_storage("rabbit-1", "Enabled", "Ready"),
        _make_storage("rabbit-2", "Disabled", "Drained"),
    ]
    rows = _build_storage_status_rows(storages)
    assert len(rows) == 2
    # Enabled/Ready first (by order in constants)
    assert rows[0] == ("Enabled", "Ready", "2", "rabbit-[0-1]")
    assert rows[1] == ("Disabled", "Drained", "1", "rabbit-2")


def test_build_storage_status_rows_empty() -> None:
    assert _build_storage_status_rows([]) == []


def test_build_storage_status_rows_includes_unexpected_values() -> None:
    storages = [
        _make_storage("rabbit-9", "Cordoned", "Recovering"),
        {
            "metadata": {"name": "rabbit-10", "annotations": {}},
            "spec": {"state": "Enabled"},
            "status": {"status": None},
        },
    ]

    rows = _build_storage_status_rows(storages)

    assert ("Enabled", "<missing>", "1", "rabbit-10") in rows
    assert ("Cordoned", "Recovering", "1", "rabbit-9") in rows


@patch("nnf.commands.system.state.k8s.list_objects")
def test_get_all_storages_uses_local_k8s_helper(mock_list_objects: MagicMock) -> None:
    mock_list_objects.return_value = {"items": [{"metadata": {"name": "rabbit-0"}}]}

    result = _get_all_storages()

    mock_list_objects.assert_called_once_with(
        group=crd.DWS_GROUP,
        version=crd.DWS_VERSION,
        namespace="default",
        plural=crd.DWS_STORAGE_PLURAL,
    )
    assert result == [{"metadata": {"name": "rabbit-0"}}]


# ---------------------------------------------------------------------------
# _build_annotation_rows
# ---------------------------------------------------------------------------


def test_build_annotation_rows_disabled() -> None:
    storages = [
        _make_storage("rabbit-0", "Disabled", "Disabled", {
            "disable_date": "2026-05-01",
            "disable_reason": "maintenance",
        }),
    ]
    rows = _build_annotation_rows(storages)
    assert len(rows) == 1
    assert rows[0] == ("Disabled", "2026-05-01", "rabbit-0", "maintenance")


def test_build_annotation_rows_drained() -> None:
    storages = [
        _make_storage("rabbit-1", "Enabled", "Ready", {
            "drain_date": "2026-05-02",
            "drain_reason": "testing",
        }),
    ]
    rows = _build_annotation_rows(storages)
    assert len(rows) == 1
    assert rows[0] == ("Drained", "2026-05-02", "rabbit-1", "testing")


def test_build_annotation_rows_both() -> None:
    storages = [
        _make_storage("rabbit-0", "Disabled", "Disabled", {
            "disable_date": "2026-05-01",
            "disable_reason": "hw issue",
            "drain_date": "2026-05-01",
            "drain_reason": "preparing",
        }),
    ]
    rows = _build_annotation_rows(storages)
    assert len(rows) == 2


def test_build_annotation_rows_none() -> None:
    storages = [_make_storage("rabbit-0", "Enabled", "Ready")]
    assert _build_annotation_rows(storages) == []


def test_build_annotation_rows_null_annotations() -> None:
    s: Dict[str, Any] = {
        "metadata": {"name": "rabbit-0", "annotations": None},
        "spec": {"state": "Enabled"},
        "status": {"status": "Ready"},
    }
    assert _build_annotation_rows([s]) == []


# ---------------------------------------------------------------------------
# run (integration)
# ---------------------------------------------------------------------------


@patch("nnf.commands.system.state._show_disabled_computes")
@patch("nnf.commands.system.state._get_all_storages")
@patch("nnf.commands.system.state._get_all_nnfnodes")
def test_run_success(
    mock_nodes: MagicMock,
    mock_storages: MagicMock,
    mock_computes: MagicMock,
    capsys: pytest.CaptureFixture[str],
) -> None:
    mock_nodes.return_value = [
        _make_nnfnode([
            {"hostname": "rabbit-0", "status": "Ready", "health": "OK"},
        ]),
    ]
    mock_storages.return_value = [
        _make_storage("rabbit-0", "Enabled", "Ready"),
    ]
    assert run(_make_args()) == 0
    out = capsys.readouterr().out
    assert "Node Status Summary" in out
    assert "Disabled/Drained Rabbit Summary" in out
    assert "rabbit-0" in out


@patch("nnf.commands.system.state._show_disabled_computes")
@patch("nnf.commands.system.state._get_all_storages")
@patch("nnf.commands.system.state._get_all_nnfnodes")
def test_run_with_annotations(
    mock_nodes: MagicMock,
    mock_storages: MagicMock,
    mock_computes: MagicMock,
    capsys: pytest.CaptureFixture[str],
) -> None:
    mock_nodes.return_value = []
    mock_storages.return_value = [
        _make_storage("rabbit-0", "Disabled", "Disabled", {
            "disable_date": "2026-05-01",
            "disable_reason": "test",
        }),
    ]
    assert run(_make_args()) == 0
    out = capsys.readouterr().out
    assert "2026-05-01" in out
    assert "test" in out


@patch("nnf.commands.system.state._show_disabled_computes")
@patch("nnf.commands.system.state._get_all_storages")
@patch("nnf.commands.system.state._get_all_nnfnodes")
def test_run_nnfnode_api_error(
    mock_nodes: MagicMock,
    mock_storages: MagicMock,
    mock_computes: MagicMock,
    capsys: pytest.CaptureFixture[str],
) -> None:
    mock_nodes.side_effect = kubernetes.client.exceptions.ApiException(
        status=500, reason="Internal Server Error",
    )
    mock_storages.return_value = []
    assert run(_make_args()) == 1
    err = capsys.readouterr().err
    assert "NnfNode" in err


@patch("nnf.commands.system.state._show_disabled_computes")
@patch("nnf.commands.system.state._get_all_storages")
@patch("nnf.commands.system.state._get_all_nnfnodes")
def test_run_storage_api_error(
    mock_nodes: MagicMock,
    mock_storages: MagicMock,
    mock_computes: MagicMock,
    capsys: pytest.CaptureFixture[str],
) -> None:
    mock_nodes.return_value = []
    mock_storages.side_effect = kubernetes.client.exceptions.ApiException(
        status=500, reason="Internal Server Error",
    )
    assert run(_make_args()) == 1
    err = capsys.readouterr().err
    assert "Storage" in err


# ---------------------------------------------------------------------------
# _run_cmd
# ---------------------------------------------------------------------------


@patch("nnf.commands.system.state.subprocess.run")
def test_run_cmd_success(mock_run: MagicMock) -> None:
    mock_run.return_value = MagicMock(returncode=0, stdout="hello\n", stderr="")
    assert _run_cmd(["echo", "hello"]) == "hello"


@patch("nnf.commands.system.state.subprocess.run")
def test_run_cmd_failure(mock_run: MagicMock) -> None:
    mock_run.return_value = MagicMock(returncode=1, stdout="", stderr="err")
    assert _run_cmd(["false"]) is None


@patch("nnf.commands.system.state.subprocess.run")
def test_run_cmd_oserror(mock_run: MagicMock) -> None:
    mock_run.side_effect = OSError("not found")
    assert _run_cmd(["missing"]) is None


@patch("nnf.commands.system.state.subprocess.run")
def test_run_cmd_timeout(mock_run: MagicMock) -> None:
    mock_run.side_effect = subprocess.TimeoutExpired("cmd", 30)
    assert _run_cmd(["slow"]) is None


# ---------------------------------------------------------------------------
# _show_disabled_computes
# ---------------------------------------------------------------------------


@patch("nnf.commands.system.state.shutil.which", return_value=None)
def test_show_disabled_computes_no_flux(mock_which: MagicMock, capsys: pytest.CaptureFixture[str]) -> None:
    _show_disabled_computes()
    out = capsys.readouterr().out
    assert "Disabled Computes" not in out


@patch("nnf.commands.system.state._run_cmd_print")
@patch("nnf.commands.system.state._run_cmd")
@patch("nnf.commands.system.state.shutil.which", return_value="/usr/bin/flux")
def test_show_disabled_computes_with_flux(
    mock_which: MagicMock,
    mock_run_cmd: MagicMock,
    mock_run_print: MagicMock,
    capsys: pytest.CaptureFixture[str],
) -> None:
    jgf = json.dumps({
        "graph": {
            "nodes": [
                {"metadata": {"type": "node", "basename": "compute", "id": 0}},
                {"metadata": {"type": "node", "basename": "compute", "id": 1}},
                {"metadata": {"type": "core", "basename": "core", "id": 0}},
            ],
        },
    })
    mock_run_cmd.side_effect = [
        jgf,           # flux ion-resource find
        "compute[0-1]",  # flux hostlist
        None,          # flux config get (not available)
    ]
    _show_disabled_computes()
    out = capsys.readouterr().out
    assert "Disabled Computes" in out
    mock_run_print.assert_called_once_with(
        ["flux", "resource", "list", "-i", "compute[0-1]"],
    )


@patch("nnf.commands.system.state.os.path.isfile", return_value=True)
@patch("nnf.commands.system.state._run_cmd_print")
@patch("nnf.commands.system.state._run_cmd")
@patch("nnf.commands.system.state.shutil.which", return_value="/usr/bin/flux")
def test_show_disabled_computes_with_nodeattr(
    mock_which: MagicMock,
    mock_run_cmd: MagicMock,
    mock_run_print: MagicMock,
    mock_isfile: MagicMock,
    capsys: pytest.CaptureFixture[str],
) -> None:
    jgf = json.dumps({
        "graph": {
            "nodes": [
                {"metadata": {"type": "node", "basename": "compute", "id": 5}},
            ],
        },
    })
    mock_run_cmd.side_effect = [
        jgf,             # flux ion-resource find
        "compute5",      # flux hostlist
        "/path/to/jgf",  # flux config get resource.scheduling
        "nid",           # nodeattr -v cluster
    ]
    _show_disabled_computes()
    mock_run_print.assert_called_once_with(
        ["flux", "resource", "list", "-i", "nidcompute5"],
    )


@patch("nnf.commands.system.state._run_cmd")
@patch("nnf.commands.system.state.shutil.which", return_value="/usr/bin/flux")
def test_show_disabled_computes_no_badrabbit(
    mock_which: MagicMock,
    mock_run_cmd: MagicMock,
    capsys: pytest.CaptureFixture[str],
) -> None:
    mock_run_cmd.return_value = None  # flux ion-resource find fails
    _show_disabled_computes()
    out = capsys.readouterr().out
    assert "Disabled Computes" not in out


@patch("nnf.commands.system.state._run_cmd")
@patch("nnf.commands.system.state.shutil.which", return_value="/usr/bin/flux")
def test_show_disabled_computes_empty_jgf(
    mock_which: MagicMock,
    mock_run_cmd: MagicMock,
    capsys: pytest.CaptureFixture[str],
) -> None:
    jgf = json.dumps({"graph": {"nodes": []}})
    mock_run_cmd.return_value = jgf
    _show_disabled_computes()
    out = capsys.readouterr().out
    assert "Disabled Computes" not in out
