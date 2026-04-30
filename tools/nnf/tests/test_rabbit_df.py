"""Tests for the rabbit df sub-command."""

import argparse
import json
from typing import Any, Dict, List
from unittest.mock import MagicMock, patch

import kubernetes.client.exceptions  # type: ignore[import-untyped]
import pytest

from nnf.commands.rabbit.df import (
    _find_node_manager_pod,
    _format_tib,
    _get_all_storages,
    _get_capacity,
    _is_enabled_ready,
    run,
)


def _make_args(**kwargs: object) -> argparse.Namespace:
    defaults: Dict[str, object] = {
        "nodes": [],
    }
    defaults.update(kwargs)
    return argparse.Namespace(**defaults)


# ---------------------------------------------------------------------------
# _format_tib
# ---------------------------------------------------------------------------


def test_format_tib_one_tib() -> None:
    assert _format_tib(1024**4) == "1.000000 TiB"


def test_format_tib_zero() -> None:
    assert _format_tib(0) == "0.000000 TiB"


def test_format_tib_fractional() -> None:
    result = _format_tib(512 * 1024**3)
    assert result == "0.500000 TiB"


# ---------------------------------------------------------------------------
# _is_enabled_ready
# ---------------------------------------------------------------------------


def test_is_enabled_ready_true() -> None:
    storage: Dict[str, Any] = {
        "spec": {"state": "Enabled"},
        "status": {"status": "Ready"},
    }
    assert _is_enabled_ready(storage) is True


def test_is_enabled_ready_disabled() -> None:
    storage: Dict[str, Any] = {
        "spec": {"state": "Disabled"},
        "status": {"status": "Ready"},
    }
    assert _is_enabled_ready(storage) is False


def test_is_enabled_ready_not_ready() -> None:
    storage: Dict[str, Any] = {
        "spec": {"state": "Enabled"},
        "status": {"status": "NotReady"},
    }
    assert _is_enabled_ready(storage) is False


# ---------------------------------------------------------------------------
# _find_node_manager_pod
# ---------------------------------------------------------------------------


def _make_pod(name: str, node_name: str) -> MagicMock:
    pod = MagicMock()
    pod.metadata.name = name
    pod.spec.node_name = node_name
    return pod


def test_find_node_manager_pod_found() -> None:
    pods = [_make_pod("nm-abc", "rabbit-0"), _make_pod("nm-def", "rabbit-1")]
    assert _find_node_manager_pod("rabbit-1", pods) == "nm-def"


def test_find_node_manager_pod_not_found() -> None:
    pods = [_make_pod("nm-abc", "rabbit-0")]
    assert _find_node_manager_pod("rabbit-99", pods) is None


# ---------------------------------------------------------------------------
# _get_capacity
# ---------------------------------------------------------------------------


def test_get_capacity_parses_redfish_json() -> None:
    redfish_json = json.dumps({
        "ProvidedCapacity": {
            "Data": {
                "AllocatedBytes": 1000,
                "ConsumedBytes": 200,
                "GuaranteedBytes": 800,
                "ProvisionedBytes": 900,
            }
        }
    })
    with patch("nnf.commands.rabbit.df.k8s.exec_pod", return_value=redfish_json):
        cap = _get_capacity("nm-pod")

    assert cap == {
        "AllocatedBytes": 1000,
        "ConsumedBytes": 200,
        "GuaranteedBytes": 800,
        "ProvisionedBytes": 900,
    }


def test_get_capacity_rejects_non_json() -> None:
    """_get_capacity raises on non-JSON/non-dict responses."""
    non_json = "this is not json"
    with patch("nnf.commands.rabbit.df.k8s.exec_pod", return_value=non_json):
        with pytest.raises((json.JSONDecodeError, ValueError, SyntaxError)):
            _get_capacity("nm-pod")


def test_get_capacity_with_channel_byte_prefix() -> None:
    """_get_capacity parses JSON correctly when exec_pod strips channel bytes."""
    prefixed = "\x01" + json.dumps({
        "ProvidedCapacity": {
            "Data": {
                "AllocatedBytes": 100,
                "ConsumedBytes": 50,
                "GuaranteedBytes": 80,
                "ProvisionedBytes": 90,
            }
        }
    })
    # Patch the raw kubernetes stream to return channel-byte-prefixed output;
    # exec_pod (with strip_channel_bytes=True) will clean it before
    # _get_capacity calls json.loads.
    with patch("nnf.k8s.get_core_v1_api"), \
            patch("kubernetes.stream.stream", return_value=prefixed):
        cap = _get_capacity("nm-pod")

    assert cap == {
        "AllocatedBytes": 100,
        "ConsumedBytes": 50,
        "GuaranteedBytes": 80,
        "ProvisionedBytes": 90,
    }


def test_get_capacity_python_dict_response() -> None:
    """_get_capacity handles Python-dict-formatted responses (single quotes)."""
    python_dict = (
        "{'ProvidedCapacity': {'Data': {"
        "'AllocatedBytes': 200, 'ConsumedBytes': 100, "
        "'GuaranteedBytes': 160, 'ProvisionedBytes': 180}}}"
    )
    with patch("nnf.commands.rabbit.df.k8s.exec_pod", return_value=python_dict):
        cap = _get_capacity("nm-pod")

    assert cap == {
        "AllocatedBytes": 200,
        "ConsumedBytes": 100,
        "GuaranteedBytes": 160,
        "ProvisionedBytes": 180,
    }


# ---------------------------------------------------------------------------
# _get_all_storages
# ---------------------------------------------------------------------------


def test_get_all_storages_returns_items() -> None:
    items = [{"metadata": {"name": "r-0"}}]
    with patch("nnf.commands.rabbit.df.k8s.list_objects", return_value={"items": items}):
        assert _get_all_storages() == items


# ---------------------------------------------------------------------------
# run() — happy path
# ---------------------------------------------------------------------------


def _make_storage(name: str, state: str = "Enabled", status: str = "Ready") -> Dict[str, Any]:
    return {
        "metadata": {"name": name},
        "spec": {"state": state},
        "status": {"status": status},
    }


def test_run_success(capsys: pytest.CaptureFixture[str]) -> None:
    """run() prints capacity for Enabled/Ready rabbits."""
    storages = [_make_storage("rabbit-0")]
    pods = [_make_pod("nm-abc", "rabbit-0")]
    cap_json = json.dumps({
        "ProvidedCapacity": {"Data": {
            "AllocatedBytes": 1024**4,
            "ConsumedBytes": 0,
            "GuaranteedBytes": 1024**4,
            "ProvisionedBytes": 1024**4,
        }}
    })

    with patch("nnf.commands.rabbit.df._get_all_storages", return_value=storages), \
            patch("nnf.commands.rabbit.df.k8s.list_pods", return_value=pods), \
            patch("nnf.commands.rabbit.df.k8s.exec_pod", return_value=cap_json):
        rc = run(_make_args())

    assert rc == 0
    out = capsys.readouterr().out
    assert "rabbit-0" in out
    assert "nm-abc" in out
    assert "Allocated:" in out


def test_run_specific_nodes(capsys: pytest.CaptureFixture[str]) -> None:
    """run() only queries requested nodes."""
    storages = [_make_storage("rabbit-0"), _make_storage("rabbit-1")]
    pods = [_make_pod("nm-abc", "rabbit-0"), _make_pod("nm-def", "rabbit-1")]
    cap_json = json.dumps({
        "ProvidedCapacity": {"Data": {
            "AllocatedBytes": 0, "ConsumedBytes": 0,
            "GuaranteedBytes": 0, "ProvisionedBytes": 0,
        }}
    })

    with patch("nnf.commands.rabbit.df._get_all_storages", return_value=storages), \
            patch("nnf.commands.rabbit.df.k8s.list_pods", return_value=pods), \
            patch("nnf.commands.rabbit.df.k8s.exec_pod", return_value=cap_json):
        rc = run(_make_args(nodes=["rabbit-1"]))

    assert rc == 0
    out = capsys.readouterr().out
    assert "rabbit-1" in out
    assert "rabbit-0" not in out


def test_run_skips_disabled(capsys: pytest.CaptureFixture[str]) -> None:
    """run() skips rabbits that are not Enabled/Ready and reports them."""
    storages = [_make_storage("rabbit-0", state="Disabled", status="Ready")]

    with patch("nnf.commands.rabbit.df._get_all_storages", return_value=storages), \
            patch("nnf.commands.rabbit.df.k8s.list_pods", return_value=[]):
        rc = run(_make_args())

    assert rc == 0
    out = capsys.readouterr().out
    assert "skipped" in out
    assert "rabbit-0" in out


def test_run_explicit_disabled_node_returns_1(capsys: pytest.CaptureFixture[str]) -> None:
    """run() returns an error when a user-specified node is not Enabled/Ready."""
    storages = [_make_storage("rabbit-0", state="Disabled", status="Ready")]

    with patch("nnf.commands.rabbit.df._get_all_storages", return_value=storages), \
            patch("nnf.commands.rabbit.df.k8s.list_pods", return_value=[]):
        rc = run(_make_args(nodes=["rabbit-0"]))

    assert rc == 1
    err = capsys.readouterr().err
    assert "not Enabled/Ready" in err
    assert "rabbit-0" in err


def test_run_no_storage_resource_returns_1(capsys: pytest.CaptureFixture[str]) -> None:
    """run() reports an error when a requested rabbit has no Storage resource."""
    with patch("nnf.commands.rabbit.df._get_all_storages", return_value=[]), \
            patch("nnf.commands.rabbit.df.k8s.list_pods", return_value=[]):
        rc = run(_make_args(nodes=["rabbit-missing"]))

    assert rc == 1
    assert "no Storage resource" in capsys.readouterr().err


def test_run_no_pod_returns_1(capsys: pytest.CaptureFixture[str]) -> None:
    """run() reports an error when no node-manager pod is found."""
    storages = [_make_storage("rabbit-0")]

    with patch("nnf.commands.rabbit.df._get_all_storages", return_value=storages), \
            patch("nnf.commands.rabbit.df.k8s.list_pods", return_value=[]):
        rc = run(_make_args())

    assert rc == 1
    assert "no node-manager pod" in capsys.readouterr().err


def test_run_capacity_error_continues(capsys: pytest.CaptureFixture[str]) -> None:
    """run() continues to the next rabbit when capacity fetch fails."""
    storages = [_make_storage("rabbit-0"), _make_storage("rabbit-1")]
    pods = [_make_pod("nm-abc", "rabbit-0"), _make_pod("nm-def", "rabbit-1")]
    cap_json = json.dumps({
        "ProvidedCapacity": {"Data": {
            "AllocatedBytes": 0, "ConsumedBytes": 0,
            "GuaranteedBytes": 0, "ProvisionedBytes": 0,
        }}
    })
    exc = kubernetes.client.exceptions.ApiException(status=500, reason="InternalError")

    with patch("nnf.commands.rabbit.df._get_all_storages", return_value=storages), \
            patch("nnf.commands.rabbit.df.k8s.list_pods", return_value=pods), \
            patch("nnf.commands.rabbit.df.k8s.exec_pod", side_effect=[exc, cap_json]):
        rc = run(_make_args())

    assert rc == 1
    out = capsys.readouterr().out
    assert "rabbit-1" in out  # second rabbit succeeded


def test_run_storage_list_failure_returns_1() -> None:
    """run() returns 1 when listing Storage resources fails."""
    exc = kubernetes.client.exceptions.ApiException(status=500, reason="InternalError")
    with patch("nnf.commands.rabbit.df._get_all_storages", side_effect=exc):
        rc = run(_make_args())

    assert rc == 1


def test_run_pod_list_failure_returns_1() -> None:
    """run() returns 1 when listing pods fails."""
    exc = kubernetes.client.exceptions.ApiException(status=500, reason="InternalError")
    with patch("nnf.commands.rabbit.df._get_all_storages", return_value=[]), \
            patch("nnf.commands.rabbit.df.k8s.list_pods", side_effect=exc):
        rc = run(_make_args())

    assert rc == 1
