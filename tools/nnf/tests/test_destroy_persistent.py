"""Tests for the destroy_persistent sub-command."""

import argparse
from typing import Dict
from unittest.mock import MagicMock, patch

from nnf.commands.persistent.destroy import run


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
