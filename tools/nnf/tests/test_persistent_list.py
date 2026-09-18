"""Tests for the persistent list sub-command."""

import argparse
from typing import Any, Dict, List, Optional, Union
from unittest.mock import MagicMock, patch

import kubernetes.client.exceptions  # type: ignore[import-untyped]
import pytest

from nnf import crd
from nnf.commands.persistent.list import run


def _make_args(**kwargs: object) -> argparse.Namespace:
    defaults: Dict[str, object] = {
        "namespace": "default",
        "user_id": None,
        "wide": False,
    }
    defaults.update(kwargs)
    return argparse.Namespace(**defaults)


def _psi(
    name: str,
    user_id: int = 1000,
    fs_type: str = "lustre",
    state: str = "Active",
    servers_name: str = "",
    servers_namespace: Optional[str] = "default",
    shared: Union[bool, str] = False,
) -> Dict[str, Any]:
    """Build a PSI. *servers_namespace* of None omits the key from the ref."""
    metadata: Dict[str, Any] = {"name": name}
    if shared:
        value = shared if isinstance(shared, str) else "true"
        metadata["annotations"] = {crd.DWS_IGNORE_UID_ANNOTATION: value}
    servers_ref: Dict[str, Any] = {"name": servers_name or name}
    if servers_namespace is not None:
        servers_ref["namespace"] = servers_namespace
    return {
        "metadata": metadata,
        "spec": {"userID": user_id, "fsType": fs_type},
        "status": {
            "state": state,
            "servers": servers_ref,
        },
    }


def _servers(name: str, rabbits: List[str]) -> Dict[str, Any]:
    return _servers_sets(name, [("ost", rabbits)])


def _servers_sets(name: str, sets: List[Any]) -> Dict[str, Any]:
    return {
        "metadata": {"name": name},
        "spec": {
            "allocationSets": [
                {
                    "label": label,
                    "storage": [{"name": r, "allocationCount": 1} for r in rabbits],
                }
                for label, rabbits in sets
            ]
        },
    }


def _list_objects(psis: List[Dict[str, Any]], servers: List[Dict[str, Any]]) -> Any:
    def side_effect(**kwargs: Any) -> Dict[str, Any]:
        if kwargs["plural"] == crd.DWS_PERSISTENT_STORAGE_PLURAL:
            return {"items": psis}
        return {"items": servers}

    return side_effect


@patch("nnf.commands.persistent.list.k8s.list_objects")
def test_run_lists_instances(mock_list: MagicMock, capsys: pytest.CaptureFixture[str]) -> None:
    mock_list.side_effect = _list_objects(
        [_psi("demo")], [_servers("demo", ["rabbit-node-1", "rabbit-node-2"])]
    )
    assert run(_make_args()) == 0
    out = capsys.readouterr().out
    assert "NAME" in out
    assert "demo" in out
    assert "1000" in out
    assert "lustre" in out
    assert "Active" in out
    assert "rabbit-node-[1-2]" in out


@patch("nnf.commands.persistent.list.k8s.list_objects")
def test_run_sorts_by_name(mock_list: MagicMock, capsys: pytest.CaptureFixture[str]) -> None:
    mock_list.side_effect = _list_objects([_psi("zeta"), _psi("alpha")], [])
    assert run(_make_args()) == 0
    lines = capsys.readouterr().out.splitlines()
    assert lines[1].startswith("alpha")
    assert lines[2].startswith("zeta")


@patch("nnf.commands.persistent.list.k8s.list_objects")
def test_run_filters_by_user_id(mock_list: MagicMock, capsys: pytest.CaptureFixture[str]) -> None:
    mock_list.side_effect = _list_objects(
        [_psi("mine", user_id=1000), _psi("theirs", user_id=2000)], []
    )
    assert run(_make_args(user_id=1000)) == 0
    out = capsys.readouterr().out
    assert "mine" in out
    assert "theirs" not in out


@patch("nnf.commands.persistent.list.k8s.list_objects")
def test_run_filters_by_user_id_zero(mock_list: MagicMock, capsys: pytest.CaptureFixture[str]) -> None:
    mock_list.side_effect = _list_objects(
        [_psi("root-owned", user_id=0), _psi("user-owned", user_id=1000)], []
    )
    assert run(_make_args(user_id=0)) == 0
    out = capsys.readouterr().out
    assert "root-owned" in out
    assert "user-owned" not in out


@patch("nnf.commands.persistent.list.k8s.list_objects")
def test_run_reports_shared(mock_list: MagicMock, capsys: pytest.CaptureFixture[str]) -> None:
    mock_list.side_effect = _list_objects(
        [_psi("open", shared=True), _psi("closed")], []
    )
    assert run(_make_args()) == 0
    lines = capsys.readouterr().out.splitlines()
    assert lines[1].split()[4] == "no"  # closed
    assert lines[2].split()[4] == "yes"  # open


@pytest.mark.parametrize("annotation", ["true", "True", "TRUE"])
@patch("nnf.commands.persistent.list.k8s.list_objects")
def test_run_reports_shared_case_insensitively(
    mock_list: MagicMock, capsys: pytest.CaptureFixture[str], annotation: str
) -> None:
    # The workflow controller matches this annotation with strings.EqualFold.
    mock_list.side_effect = _list_objects([_psi("open", shared=annotation)], [])
    assert run(_make_args()) == 0
    lines = capsys.readouterr().out.splitlines()
    assert lines[1].split()[4] == "yes"


@patch("nnf.commands.persistent.list.k8s.list_objects")
def test_run_does_not_report_shared_for_other_values(
    mock_list: MagicMock, capsys: pytest.CaptureFixture[str]
) -> None:
    mock_list.side_effect = _list_objects([_psi("closed", shared="false")], [])
    assert run(_make_args()) == 0
    lines = capsys.readouterr().out.splitlines()
    assert lines[1].split()[4] == "no"


@patch("nnf.commands.persistent.list.k8s.list_objects")
def test_run_ignores_servers_in_other_namespace(
    mock_list: MagicMock, capsys: pytest.CaptureFixture[str]
) -> None:
    mock_list.side_effect = _list_objects(
        [_psi("demo", servers_namespace="other")],
        [_servers("demo", ["rabbit-node-1"])],
    )
    assert run(_make_args()) == 0
    assert "rabbit-node-1" not in capsys.readouterr().out


@patch("nnf.commands.persistent.list.k8s.list_objects")
def test_run_servers_ref_without_namespace_defaults_to_requested(
    mock_list: MagicMock, capsys: pytest.CaptureFixture[str]
) -> None:
    mock_list.side_effect = _list_objects(
        [_psi("demo", servers_namespace=None)],
        [_servers("demo", ["rabbit-node-1"])],
    )
    assert run(_make_args()) == 0
    assert "rabbit-node-1" in capsys.readouterr().out


@patch("nnf.commands.persistent.list.k8s.list_objects")
def test_run_missing_servers_resource(
    mock_list: MagicMock, capsys: pytest.CaptureFixture[str]
) -> None:
    mock_list.side_effect = _list_objects([_psi("demo")], [])
    assert run(_make_args()) == 0
    assert "demo" in capsys.readouterr().out


@patch("nnf.commands.persistent.list.k8s.list_objects")
def test_run_wide_breaks_out_allocation_sets(
    mock_list: MagicMock, capsys: pytest.CaptureFixture[str]
) -> None:
    mock_list.side_effect = _list_objects(
        [_psi("demo")],
        [
            _servers_sets(
                "demo",
                [
                    ("ost", ["rabbit-node-1", "rabbit-node-2"]),
                    ("mgtmdt", ["rabbit-node-1"]),
                ],
            )
        ],
    )
    assert run(_make_args(wide=True)) == 0
    out = capsys.readouterr().out
    assert "ost:rabbit-node-[1-2] mgtmdt:rabbit-node-1" in out


@patch("nnf.commands.persistent.list.k8s.list_objects")
def test_run_narrow_merges_allocation_sets(
    mock_list: MagicMock, capsys: pytest.CaptureFixture[str]
) -> None:
    mock_list.side_effect = _list_objects(
        [_psi("demo")],
        [
            _servers_sets(
                "demo",
                [
                    ("ost", ["rabbit-node-1", "rabbit-node-2"]),
                    ("mgtmdt", ["rabbit-node-1"]),
                ],
            )
        ],
    )
    assert run(_make_args()) == 0
    out = capsys.readouterr().out
    assert "rabbit-node-[1-2]" in out
    assert "ost:" not in out


@patch("nnf.commands.persistent.list.k8s.list_objects")
def test_run_wide_skips_empty_allocation_sets(
    mock_list: MagicMock, capsys: pytest.CaptureFixture[str]
) -> None:
    mock_list.side_effect = _list_objects(
        [_psi("demo")], [_servers_sets("demo", [("ost", [])])]
    )
    assert run(_make_args(wide=True)) == 0
    lines = capsys.readouterr().out.splitlines()
    assert lines[1].split()[5] == "-"


@patch("nnf.commands.persistent.list.k8s.list_objects")
def test_run_wide_marks_unlabeled_allocation_set(
    mock_list: MagicMock, capsys: pytest.CaptureFixture[str]
) -> None:
    mock_list.side_effect = _list_objects(
        [_psi("demo")],
        [
            {
                "metadata": {"name": "demo"},
                "spec": {"allocationSets": [{"storage": [{"name": "rabbit-node-3"}]}]},
            }
        ],
    )
    assert run(_make_args(wide=True)) == 0
    assert "unlabeled:rabbit-node-3" in capsys.readouterr().out


@patch("nnf.commands.persistent.list.k8s.list_objects")
def test_run_empty_prints_header_and_succeeds(
    mock_list: MagicMock, capsys: pytest.CaptureFixture[str]
) -> None:
    mock_list.side_effect = _list_objects([], [])
    assert run(_make_args()) == 0
    lines = capsys.readouterr().out.splitlines()
    assert len(lines) == 1
    assert lines[0].startswith("NAME")


@patch("nnf.commands.persistent.list.k8s.list_objects")
def test_run_uses_requested_namespace(mock_list: MagicMock) -> None:
    mock_list.side_effect = _list_objects([], [])
    run(_make_args(namespace="mine"))
    for call in mock_list.call_args_list:
        assert call[1]["namespace"] == "mine"


@patch("nnf.commands.persistent.list.k8s.list_objects")
def test_run_psi_api_error(mock_list: MagicMock, capsys: pytest.CaptureFixture[str]) -> None:
    mock_list.side_effect = kubernetes.client.exceptions.ApiException(
        status=403, reason="Forbidden",
    )
    assert run(_make_args()) == 1
    assert "error" in capsys.readouterr().err


@patch("nnf.commands.persistent.list.k8s.list_objects")
def test_run_servers_api_error(mock_list: MagicMock, capsys: pytest.CaptureFixture[str]) -> None:
    def side_effect(**kwargs: Any) -> Dict[str, Any]:
        if kwargs["plural"] == crd.DWS_PERSISTENT_STORAGE_PLURAL:
            return {"items": []}
        raise kubernetes.client.exceptions.ApiException(status=403, reason="Forbidden")

    mock_list.side_effect = side_effect
    assert run(_make_args()) == 1
    assert "Servers" in capsys.readouterr().err
