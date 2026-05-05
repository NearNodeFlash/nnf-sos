"""Tests for the system flowschema sub-command."""

import argparse
import pytest
import json
from typing import Dict
from unittest.mock import MagicMock, patch

import kubernetes.client.exceptions  # type: ignore[import-untyped]

from nnf.commands.system.flowschema import (
    _discover_api_version,
    _list_flowschemas,
    _list_priority_levels,
    _print_table,
    _view_activity,
    _view_summary,
    run,
)


def _make_args(**kwargs: object) -> argparse.Namespace:
    defaults: Dict[str, object] = {
        "flow_schema_name": None,
        "list_flowschemas": False,
        "priority_levels": False,
        "summary": False,
    }
    defaults.update(kwargs)
    return argparse.Namespace(**defaults)


# ---------------------------------------------------------------------------
# Fixtures / canned responses
# ---------------------------------------------------------------------------

_DISCOVERY_RESPONSE = json.dumps({
    "kind": "APIGroup",
    "apiVersion": "v1",
    "name": "flowcontrol.apiserver.k8s.io",
    "versions": [
        {"groupVersion": "flowcontrol.apiserver.k8s.io/v1", "version": "v1"},
        {"groupVersion": "flowcontrol.apiserver.k8s.io/v1beta3", "version": "v1beta3"},
    ],
    "preferredVersion": {
        "groupVersion": "flowcontrol.apiserver.k8s.io/v1",
        "version": "v1",
    },
})

_FLOWSCHEMA_LIST = json.dumps({
    "items": [
        {
            "metadata": {"name": "catch-all"},
            "spec": {
                "priorityLevelConfiguration": {"name": "catch-all"},
                "matchingPrecedence": 10000,
                "distinguisherMethod": {"type": "ByUser"},
            },
        },
        {
            "metadata": {"name": "nnf-webhook"},
            "spec": {
                "priorityLevelConfiguration": {"name": "workload-high"},
                "matchingPrecedence": 1000,
                "distinguisherMethod": {"type": "ByNamespace"},
            },
        },
    ],
})

_PRIORITY_LEVEL_LIST = json.dumps({
    "items": [
        {
            "metadata": {"name": "catch-all"},
            "spec": {
                "type": "Limited",
                "limited": {
                    "nominalConcurrencyShares": 5,
                    "limitResponse": {
                        "type": "Queue",
                        "queuing": {
                            "queues": 64,
                            "handSize": 6,
                            "queueLengthLimit": 50,
                        },
                    },
                },
            },
        },
        {
            "metadata": {"name": "exempt"},
            "spec": {
                "type": "Exempt",
            },
        },
    ],
})

_METRICS_RESPONSE = (
    "# HELP apiserver_flowcontrol_request_concurrency_limit\n"
    'apiserver_flowcontrol_request_concurrency_limit{flow_schema="catch-all",'
    'priority_level="catch-all"} 5\n'
    'apiserver_flowcontrol_request_concurrency_limit{flow_schema="nnf-webhook",'
    'priority_level="workload-high"} 30\n'
    "# HELP apiserver_flowcontrol_dispatched_requests_total\n"
    'apiserver_flowcontrol_dispatched_requests_total{flow_schema="catch-all",'
    'priority_level="catch-all"} 100\n'
    'apiserver_flowcontrol_dispatched_requests_total{flow_schema="nnf-webhook",'
    'priority_level="workload-high"} 500\n'
)


# ---------------------------------------------------------------------------
# _print_table
# ---------------------------------------------------------------------------


def test_print_table_basic(capsys: pytest.CaptureFixture[str]) -> None:
    _print_table(("A", "BB"), [("x", "yy"), ("zzz", "w")])
    out = capsys.readouterr().out
    lines = out.strip().splitlines()
    assert len(lines) == 3
    assert "A" in lines[0]
    assert "BB" in lines[0]
    assert "zzz" in lines[2]


def test_print_table_empty(capsys: pytest.CaptureFixture[str]) -> None:
    _print_table(("COL1", "COL2"), [])
    out = capsys.readouterr().out
    lines = out.strip().splitlines()
    assert len(lines) == 1
    assert "COL1" in lines[0]


# ---------------------------------------------------------------------------
# _discover_api_version
# ---------------------------------------------------------------------------


@patch("nnf.commands.system.flowschema.k8s")
def test_discover_api_version_v1(mock_k8s: MagicMock) -> None:
    mock_k8s.get_raw.return_value = _DISCOVERY_RESPONSE
    assert _discover_api_version() == "v1"


@patch("nnf.commands.system.flowschema.k8s")
def test_discover_api_version_fallback_v1beta3(mock_k8s: MagicMock) -> None:
    response = json.dumps({"versions": [{"version": "v1beta3"}]})
    mock_k8s.get_raw.return_value = response
    assert _discover_api_version() == "v1beta3"


@patch("nnf.commands.system.flowschema.k8s")
def test_discover_api_version_not_available(mock_k8s: MagicMock) -> None:
    mock_k8s.get_raw.side_effect = kubernetes.client.exceptions.ApiException(status=404)
    assert _discover_api_version() is None


@patch("nnf.commands.system.flowschema.k8s")
def test_discover_api_version_no_known_version(mock_k8s: MagicMock) -> None:
    response = json.dumps({"versions": [{"version": "v1alpha1"}]})
    mock_k8s.get_raw.return_value = response
    assert _discover_api_version() is None


# ---------------------------------------------------------------------------
# _list_flowschemas
# ---------------------------------------------------------------------------


@patch("nnf.commands.system.flowschema.k8s")
def test_list_flowschemas(mock_k8s: MagicMock, capsys: pytest.CaptureFixture[str]) -> None:
    mock_k8s.get_raw.return_value = _FLOWSCHEMA_LIST
    assert _list_flowschemas("v1") == 0
    out = capsys.readouterr().out
    assert "NAME" in out
    assert "PRIORITYLEVEL" in out
    assert "catch-all" in out
    assert "nnf-webhook" in out


@patch("nnf.commands.system.flowschema.k8s")
def test_list_flowschemas_sorted(mock_k8s: MagicMock, capsys: pytest.CaptureFixture[str]) -> None:
    mock_k8s.get_raw.return_value = _FLOWSCHEMA_LIST
    _list_flowschemas("v1")
    out = capsys.readouterr().out
    lines = out.strip().splitlines()
    # Header + 2 rows, "catch-all" before "nnf-webhook"
    assert lines[1].startswith("catch-all")
    assert lines[2].startswith("nnf-webhook")


@patch("nnf.commands.system.flowschema.k8s")
def test_list_flowschemas_no_distinguisher(mock_k8s: MagicMock, capsys: pytest.CaptureFixture[str]) -> None:
    data = json.dumps({
        "items": [{
            "metadata": {"name": "test"},
            "spec": {
                "priorityLevelConfiguration": {"name": "pl"},
                "matchingPrecedence": 100,
            },
        }],
    })
    mock_k8s.get_raw.return_value = data
    assert _list_flowschemas("v1") == 0


@patch("nnf.commands.system.flowschema.k8s")
def test_list_flowschemas_api_error(mock_k8s: MagicMock, capsys: pytest.CaptureFixture[str]) -> None:
    mock_k8s.get_raw.side_effect = kubernetes.client.exceptions.ApiException(
        status=500, reason="Internal Server Error",
    )
    assert _list_flowschemas("v1") == 1
    err = capsys.readouterr().err
    assert "error" in err


# ---------------------------------------------------------------------------
# _list_priority_levels
# ---------------------------------------------------------------------------


@patch("nnf.commands.system.flowschema.k8s")
def test_list_priority_levels(mock_k8s: MagicMock, capsys: pytest.CaptureFixture[str]) -> None:
    mock_k8s.get_raw.return_value = _PRIORITY_LEVEL_LIST
    assert _list_priority_levels("v1") == 0
    out = capsys.readouterr().out
    assert "NAME" in out
    assert "catch-all" in out
    assert "exempt" in out
    assert "NOMINALCONCURRENCYSHARES" in out


@patch("nnf.commands.system.flowschema.k8s")
def test_list_priority_levels_exempt_empty_fields(mock_k8s: MagicMock, capsys: pytest.CaptureFixture[str]) -> None:
    mock_k8s.get_raw.return_value = _PRIORITY_LEVEL_LIST
    _list_priority_levels("v1")
    out = capsys.readouterr().out
    lines = out.strip().splitlines()
    # "exempt" row should have empty fields for shares/queues/etc.
    exempt_line = [line for line in lines if "Exempt" in line][0]
    # After "Exempt" there should be only whitespace (empty columns)
    after_exempt = exempt_line.split("Exempt", 1)[1]
    assert after_exempt.strip() == ""


@patch("nnf.commands.system.flowschema.k8s")
def test_list_priority_levels_api_error(mock_k8s: MagicMock, capsys: pytest.CaptureFixture[str]) -> None:
    mock_k8s.get_raw.side_effect = kubernetes.client.exceptions.ApiException(
        status=500, reason="Internal Server Error",
    )
    assert _list_priority_levels("v1") == 1
    err = capsys.readouterr().err
    assert "error" in err


# ---------------------------------------------------------------------------
# _view_activity
# ---------------------------------------------------------------------------


@patch("nnf.commands.system.flowschema.k8s")
def test_view_activity_found(mock_k8s: MagicMock, capsys: pytest.CaptureFixture[str]) -> None:
    mock_k8s.get_raw.side_effect = [_FLOWSCHEMA_LIST, _METRICS_RESPONSE]
    assert _view_activity("v1", "catch-all") == 0
    out = capsys.readouterr().out
    assert 'flow_schema="catch-all"' in out
    # Should not include lines for other flow schemas
    assert 'flow_schema="nnf-webhook"' not in out


@patch("nnf.commands.system.flowschema.k8s")
def test_view_activity_not_found(mock_k8s: MagicMock, capsys: pytest.CaptureFixture[str]) -> None:
    mock_k8s.get_raw.return_value = _FLOWSCHEMA_LIST
    assert _view_activity("v1", "nonexistent") == 1
    out = capsys.readouterr().out
    assert "Valid flowschema NAMEs are:" in out
    assert "catch-all" in out
    assert "nnf-webhook" in out


@patch("nnf.commands.system.flowschema.k8s")
def test_view_activity_metrics_error(mock_k8s: MagicMock, capsys: pytest.CaptureFixture[str]) -> None:
    mock_k8s.get_raw.side_effect = [
        _FLOWSCHEMA_LIST,
        kubernetes.client.exceptions.ApiException(status=403, reason="Forbidden"),
    ]
    assert _view_activity("v1", "catch-all") == 1
    err = capsys.readouterr().err
    assert "error" in err


# ---------------------------------------------------------------------------
# _view_summary
# ---------------------------------------------------------------------------


@patch("nnf.commands.system.flowschema.k8s")
def test_view_summary(mock_k8s: MagicMock, capsys: pytest.CaptureFixture[str]) -> None:
    mock_k8s.get_raw.return_value = "priority-level dump data\n"
    assert _view_summary() == 0
    out = capsys.readouterr().out
    assert "priority-level dump data" in out


@patch("nnf.commands.system.flowschema.k8s")
def test_view_summary_api_error(mock_k8s: MagicMock, capsys: pytest.CaptureFixture[str]) -> None:
    mock_k8s.get_raw.side_effect = kubernetes.client.exceptions.ApiException(
        status=500, reason="Internal Server Error",
    )
    assert _view_summary() == 1
    err = capsys.readouterr().err
    assert "error" in err


# ---------------------------------------------------------------------------
# run (integration)
# ---------------------------------------------------------------------------


@patch("nnf.commands.system.flowschema.k8s")
def test_run_list_flowschemas(mock_k8s: MagicMock, capsys: pytest.CaptureFixture[str]) -> None:
    mock_k8s.get_raw.side_effect = [_DISCOVERY_RESPONSE, _FLOWSCHEMA_LIST]
    args = _make_args(list_flowschemas=True)
    assert run(args) == 0
    out = capsys.readouterr().out
    assert "catch-all" in out


@patch("nnf.commands.system.flowschema.k8s")
def test_run_priority_levels(mock_k8s: MagicMock, capsys: pytest.CaptureFixture[str]) -> None:
    mock_k8s.get_raw.side_effect = [_DISCOVERY_RESPONSE, _PRIORITY_LEVEL_LIST]
    args = _make_args(priority_levels=True)
    assert run(args) == 0
    out = capsys.readouterr().out
    assert "catch-all" in out


@patch("nnf.commands.system.flowschema.k8s")
def test_run_view_activity(mock_k8s: MagicMock, capsys: pytest.CaptureFixture[str]) -> None:
    mock_k8s.get_raw.side_effect = [
        _DISCOVERY_RESPONSE,
        _FLOWSCHEMA_LIST,
        _METRICS_RESPONSE,
    ]
    args = _make_args(flow_schema_name="catch-all")
    assert run(args) == 0


@patch("nnf.commands.system.flowschema.k8s")
def test_run_summary(mock_k8s: MagicMock, capsys: pytest.CaptureFixture[str]) -> None:
    mock_k8s.get_raw.return_value = "dump data\n"
    args = _make_args(summary=True)
    assert run(args) == 0


@patch("nnf.commands.system.flowschema.k8s")
def test_run_no_api_available(mock_k8s: MagicMock, capsys: pytest.CaptureFixture[str]) -> None:
    mock_k8s.get_raw.side_effect = kubernetes.client.exceptions.ApiException(status=404)
    args = _make_args(list_flowschemas=True)
    assert run(args) == 1
    err = capsys.readouterr().err
    assert "not available" in err
