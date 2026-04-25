"""Tests for the destroy_persistent sub-command."""

import argparse
from typing import Dict
from unittest.mock import MagicMock, patch

from nnf.commands.destroy_persistent import run


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
    mock_create = MagicMock(return_value={})
    with patch("nnf.commands.destroy_persistent.k8s.create_object", mock_create), \
            patch("nnf.commands.destroy_persistent.workflow.run_to_completion",
                  return_value=(True, "")):
        assert run(_make_args()) == 0

    mock_create.assert_called_once()
    _, kwargs = mock_create.call_args
    body = kwargs["body"]
    assert body["kind"] == "Workflow"
    assert "#DW destroy_persistent name=my-psi" in body["spec"]["dwDirectives"]


def test_run_directive_contains_name() -> None:
    """run() builds the correct #DW destroy_persistent directive."""
    mock_create = MagicMock(return_value={})
    with patch("nnf.commands.destroy_persistent.k8s.create_object", mock_create), \
            patch("nnf.commands.destroy_persistent.workflow.run_to_completion",
                  return_value=(True, "")):
        assert run(_make_args(name="my-lustre")) == 0

    _, kwargs = mock_create.call_args
    body = kwargs["body"]
    assert "#DW destroy_persistent name=my-lustre" in body["spec"]["dwDirectives"]


def test_run_workflow_name() -> None:
    """run() uses nnf-destroy-persistent-{name} as the Workflow name."""
    mock_create = MagicMock(return_value={})
    with patch("nnf.commands.destroy_persistent.k8s.create_object", mock_create), \
            patch("nnf.commands.destroy_persistent.workflow.run_to_completion",
                  return_value=(True, "")):
        assert run(_make_args(name="my-psi")) == 0

    _, kwargs = mock_create.call_args
    body = kwargs["body"]
    assert body["metadata"]["name"] == "nnf-destroy-persistent-my-psi"


def test_run_create_api_error_returns_2() -> None:
    """run() returns exit code 2 when Workflow creation fails."""
    import kubernetes.client.exceptions  # type: ignore[import-untyped]

    exc = kubernetes.client.exceptions.ApiException(status=409, reason="AlreadyExists")
    with patch("nnf.commands.destroy_persistent.k8s.create_object", side_effect=exc):
        assert run(_make_args()) == 2


def test_run_workflow_failure_returns_2() -> None:
    """run() returns exit code 2 when run_to_completion reports failure."""
    with patch("nnf.commands.destroy_persistent.k8s.create_object", return_value={}), \
            patch("nnf.commands.destroy_persistent.workflow.run_to_completion",
                  return_value=(False, "Teardown timed out")):
        assert run(_make_args()) == 2


def test_run_no_state_hooks() -> None:
    """run() creates a Workflow with no state hooks (destroy needs no Servers population)."""
    from nnf.workflow import WorkflowRun

    captured: Dict[str, object] = {}

    def fake_run_to_completion(wf: WorkflowRun, timeout: int) -> object:
        captured["wf"] = wf
        return True, ""

    with patch("nnf.commands.destroy_persistent.k8s.create_object", return_value={}), \
            patch("nnf.commands.destroy_persistent.workflow.run_to_completion",
                  side_effect=fake_run_to_completion):
        assert run(_make_args()) == 0

    wf = captured["wf"]
    assert isinstance(wf, WorkflowRun)
    assert wf.state_hooks == {}
