"""Tests for the destroy_persistent sub-command."""

import argparse
import pytest
from typing import Dict
from unittest.mock import MagicMock, patch

import kubernetes.client.exceptions  # type: ignore[import-untyped]

from nnf.commands.persistent.destroy import _get_psi_user_id, run


def _make_args(**kwargs: object) -> argparse.Namespace:
    defaults: Dict[str, object] = {
        "name": "my-psi",
        "namespace": "default",
        "user_id": 1000,
        "group_id": 1000,
        "timeout": 60,
    }
    defaults.update(kwargs)
    return argparse.Namespace(**defaults)


def test_run_invalid_name_returns_1() -> None:
    """run() returns 1 when --name would produce an invalid K8s resource name."""
    assert run(_make_args(name="My_Storage!")) == 1


def test_run_success() -> None:
    """run() returns 0 when the Workflow is created and completes successfully."""
    mock_car = MagicMock(return_value=0)
    with patch("nnf.commands.persistent.destroy.workflow.create_and_run", mock_car):
        assert run(_make_args()) == 0

    mock_car.assert_called_once()
    wf_arg = mock_car.call_args[0][0]
    manifest = wf_arg.manifest
    assert manifest["kind"] == "Workflow"
    assert "#DW destroy_persistent name=my-psi" in manifest["spec"]["dwDirectives"]


def test_run_directive_contains_name() -> None:
    """run() builds the correct #DW destroy_persistent directive."""
    mock_car = MagicMock(return_value=0)
    with patch("nnf.commands.persistent.destroy.workflow.create_and_run", mock_car):
        assert run(_make_args(name="my-lustre")) == 0

    wf_arg = mock_car.call_args[0][0]
    assert "#DW destroy_persistent name=my-lustre" in wf_arg.dw_directives


def test_run_workflow_name() -> None:
    """run() uses nnf-destroy-persistent-{name} as the Workflow name."""
    mock_car = MagicMock(return_value=0)
    with patch("nnf.commands.persistent.destroy.workflow.create_and_run", mock_car):
        assert run(_make_args(name="my-psi")) == 0

    wf_arg = mock_car.call_args[0][0]
    assert wf_arg.name == "nnf-destroy-persistent-my-psi"


def test_run_create_api_error_returns_2() -> None:
    """run() returns exit code 2 when Workflow creation fails."""
    with patch("nnf.commands.persistent.destroy.workflow.create_and_run", return_value=2):
        assert run(_make_args()) == 2


def test_run_workflow_failure_returns_2() -> None:
    """run() returns exit code 2 when run_to_completion reports failure."""
    with patch("nnf.commands.persistent.destroy.workflow.create_and_run", return_value=2):
        assert run(_make_args()) == 2


def test_run_no_state_hooks() -> None:
    """run() creates a Workflow with no state hooks (destroy needs no Servers population)."""
    from nnf.workflow import WorkflowRun

    captured: Dict[str, object] = {}

    def fake_create_and_run(wf: WorkflowRun, timeout: int) -> int:
        captured["wf"] = wf
        return 0

    with patch("nnf.commands.persistent.destroy.workflow.create_and_run",
               side_effect=fake_create_and_run):
        assert run(_make_args()) == 0

    wf = captured["wf"]
    assert isinstance(wf, WorkflowRun)
    assert wf.state_hooks == {}


# ---------------------------------------------------------------------------
# _get_psi_user_id
# ---------------------------------------------------------------------------


@patch("nnf.commands.persistent.destroy.k8s.get_object")
def test_get_psi_user_id(mock_get: MagicMock) -> None:
    mock_get.return_value = {"spec": {"userID": 5000}}
    assert _get_psi_user_id("my-psi", "default") == 5000


@patch("nnf.commands.persistent.destroy.k8s.get_object")
def test_get_psi_user_id_string(mock_get: MagicMock) -> None:
    """userID is sometimes stored as a string in the spec."""
    mock_get.return_value = {"spec": {"userID": "1234"}}
    assert _get_psi_user_id("my-psi", "default") == 1234


# ---------------------------------------------------------------------------
# run – user_id from PSI lookup
# ---------------------------------------------------------------------------


def test_run_user_id_from_psi() -> None:
    """When --user-id is not supplied, run() fetches userID from the PSI."""
    mock_car = MagicMock(return_value=0)
    with patch("nnf.commands.persistent.destroy.workflow.create_and_run", mock_car), \
            patch("nnf.commands.persistent.destroy._get_psi_user_id", return_value=4242):
        assert run(_make_args(user_id=None)) == 0

    wf_arg = mock_car.call_args[0][0]
    assert wf_arg.user_id == 4242


def test_run_user_id_cli_overrides_psi() -> None:
    """When --user-id is supplied, run() uses it instead of the PSI."""
    mock_car = MagicMock(return_value=0)
    with patch("nnf.commands.persistent.destroy.workflow.create_and_run", mock_car):
        assert run(_make_args(user_id=9999)) == 0

    wf_arg = mock_car.call_args[0][0]
    assert wf_arg.user_id == 9999


def test_run_psi_not_found_returns_1() -> None:
    """run() returns 1 when the PSI does not exist."""
    with patch("nnf.commands.persistent.destroy._get_psi_user_id",
               side_effect=kubernetes.client.exceptions.ApiException(
                   status=404, reason="Not Found")):
        assert run(_make_args(user_id=None)) == 1


def test_run_psi_missing_user_id_returns_1(capsys: pytest.CaptureFixture[str]) -> None:
    """run() returns 1 when the PSI spec has no userID field."""
    with patch("nnf.commands.persistent.destroy._get_psi_user_id",
               side_effect=KeyError("userID")):
        assert run(_make_args(user_id=None)) == 1
    err = capsys.readouterr().err
    assert "userID" in err
