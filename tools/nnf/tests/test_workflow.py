"""Tests for workflow.py shared helpers."""

from typing import List, Tuple
from unittest.mock import MagicMock, patch

import pytest

from nnf import workflow
from nnf.workflow import (
    WorkflowRun,
    _wait_for_breakdown,
    advance,
    delete,
    run_to_completion,
    teardown_and_delete,
    wait_for_state,
)


# ---------------------------------------------------------------------------
# WorkflowRun
# ---------------------------------------------------------------------------


def test_workflow_run_manifest_structure() -> None:
    """WorkflowRun.manifest returns a correctly structured k8s body."""
    wf = WorkflowRun("wf", "default", 1000, dw_directives=["#DW jobdw type=lustre capacity=1GiB"])
    body = wf.manifest
    assert body["kind"] == "Workflow"
    assert body["spec"]["desiredState"] == "Proposal"
    assert body["spec"]["userID"] == 1000
    assert "#DW jobdw" in body["spec"]["dwDirectives"][0]


def test_workflow_run_manifest_multiple_directives() -> None:
    """WorkflowRun.manifest includes all provided directives."""
    directives = ["#DW jobdw type=xfs capacity=1GiB", "#DW swap type=xfs capacity=512MiB"]
    wf = WorkflowRun("wf", "ns", 42, dw_directives=directives)
    assert len(wf.manifest["spec"]["dwDirectives"]) == 2


def test_workflow_run_state_hooks_default_is_empty() -> None:
    """state_hooks defaults to an empty dict when not provided."""
    wf = WorkflowRun("wf", "default", 0, [])
    assert wf.state_hooks == {}
    assert wf.on_state_complete == []


# ---------------------------------------------------------------------------
# advance
# ---------------------------------------------------------------------------


def test_advance_patches_desired_state() -> None:
    """advance() sends a merge-patch with the new desiredState."""
    mock_patch = MagicMock()
    with patch("nnf.workflow.k8s.patch_object", mock_patch):
        advance("wf", "default", "Setup")
    _, kwargs = mock_patch.call_args
    assert kwargs["body"] == {"spec": {"desiredState": "Setup"}}
    assert kwargs["name"] == "wf"


# ---------------------------------------------------------------------------
# wait_for_state
# ---------------------------------------------------------------------------


def _obj(state: str, ready: bool, status: str = "Completed") -> dict:
    return {"status": {"state": state, "ready": ready, "status": status, "message": ""}}


def test_wait_for_state_succeeds_immediately() -> None:
    """Returns True when the workflow is already at the desired state."""
    with patch("nnf.workflow.k8s.get_object", return_value=_obj("Setup", True)):
        ok, msg = wait_for_state("wf", "default", "Setup", timeout=10)
    assert ok is True
    assert msg == ""


def test_wait_for_state_polls_until_ready() -> None:
    """Retries until the workflow becomes ready."""
    responses = [
        _obj("Setup", False, "DriverWait"),
        _obj("Setup", False, "DriverWait"),
        _obj("Setup", True),
    ]
    with patch("nnf.workflow.k8s.get_object", side_effect=responses), \
            patch("nnf.workflow.time.sleep"):
        ok, _ = wait_for_state("wf", "default", "Setup", timeout=60)
    assert ok is True


def test_wait_for_state_error_status() -> None:
    """Returns False immediately when status is Error."""
    with patch("nnf.workflow.k8s.get_object", return_value=_obj("Setup", False, "Error")), \
            patch("nnf.workflow.time.sleep"):
        ok, msg = wait_for_state("wf", "default", "Setup", timeout=60)
    assert ok is False
    assert "Error" in msg


def test_wait_for_state_timeout() -> None:
    """Returns False after the timeout is exceeded."""
    with patch("nnf.workflow.k8s.get_object", return_value=_obj("Setup", False, "DriverWait")), \
            patch("nnf.workflow.time.sleep"), \
            patch("nnf.workflow.time.monotonic", side_effect=[0.0, 999.0, 999.0]):
        ok, msg = wait_for_state("wf", "default", "Setup", timeout=10)
    assert ok is False
    assert "Timed out" in msg


def test_wait_for_state_api_exception() -> None:
    """Returns False when the k8s API raises an exception while polling."""
    import kubernetes.client.exceptions  # type: ignore[import-untyped]

    exc = kubernetes.client.exceptions.ApiException(status=500, reason="Server Error")
    with patch("nnf.workflow.k8s.get_object", side_effect=exc):
        ok, msg = wait_for_state("wf", "default", "Setup", timeout=10)
    assert ok is False
    assert "API error" in msg


# ---------------------------------------------------------------------------
# _wait_for_breakdown
# ---------------------------------------------------------------------------


def test_wait_for_breakdown_succeeds_immediately() -> None:
    """Returns True when the DirectiveBreakdown is already ready."""
    with patch("nnf.workflow.k8s.get_object", return_value={"status": {"ready": True}}):
        ok, msg = _wait_for_breakdown("bd-0", "default", timeout=10)

    assert ok is True
    assert msg == ""


def test_wait_for_breakdown_timeout() -> None:
    """Returns False after the DirectiveBreakdown timeout is exceeded."""
    with patch("nnf.workflow.k8s.get_object", return_value={"status": {"ready": False}}), \
            patch("nnf.workflow.time.sleep"), \
            patch("nnf.workflow.time.monotonic", side_effect=[0.0, 999.0, 999.0]):
        ok, msg = _wait_for_breakdown("bd-0", "default", timeout=10)

    assert ok is False
    assert "Timed out waiting for DirectiveBreakdown 'bd-0'" in msg


def test_wait_for_breakdown_api_exception() -> None:
    """Returns False when the k8s API raises while polling a DirectiveBreakdown."""
    import kubernetes.client.exceptions  # type: ignore[import-untyped]

    exc = kubernetes.client.exceptions.ApiException(status=500, reason="Server Error")
    with patch("nnf.workflow.k8s.get_object", side_effect=exc):
        ok, msg = _wait_for_breakdown("bd-0", "default", timeout=10)

    assert ok is False
    assert "API error fetching DirectiveBreakdown 'bd-0'" in msg


# ---------------------------------------------------------------------------
# delete
# ---------------------------------------------------------------------------


def test_delete_ignores_404() -> None:
    """delete() does not raise when the Workflow is already gone."""
    import kubernetes.client.exceptions  # type: ignore[import-untyped]

    exc = kubernetes.client.exceptions.ApiException(status=404, reason="Not Found")
    with patch("nnf.workflow.k8s.delete_object", side_effect=exc):
        delete("wf", "default")  # must not raise


def test_delete_logs_warning_on_other_errors(caplog: pytest.LogCaptureFixture) -> None:
    """delete() logs a warning for non-404 API errors."""
    import kubernetes.client.exceptions  # type: ignore[import-untyped]

    exc = kubernetes.client.exceptions.ApiException(status=500, reason="Internal Error")
    with caplog.at_level("WARNING"), \
            patch("nnf.workflow.k8s.delete_object", side_effect=exc):
        delete("wf", "default")
    assert "Failed to delete Workflow 'wf': Internal Error" in caplog.text


# ---------------------------------------------------------------------------
# teardown_and_delete
# ---------------------------------------------------------------------------


def test_teardown_and_delete_advances_then_deletes() -> None:
    """teardown_and_delete advances to Teardown and then deletes."""
    mock_advance = MagicMock()
    mock_delete = MagicMock()
    with patch("nnf.workflow.advance", mock_advance), \
            patch("nnf.workflow.wait_for_state", return_value=(True, "")), \
            patch("nnf.workflow.delete", mock_delete):
        teardown_and_delete("wf", "default", timeout=10)
    mock_advance.assert_called_once_with("wf", "default", "Teardown")
    mock_delete.assert_called_once_with("wf", "default")


# ---------------------------------------------------------------------------
# run_to_completion
# ---------------------------------------------------------------------------


def _make_wf(
    state_hooks: object = None,
    on_state_complete: object = None,
) -> WorkflowRun:
    return WorkflowRun(
        name="wf",
        namespace="default",
        user_id=1000,
        dw_directives=["#DW jobdw type=lustre capacity=1GiB"],
        state_hooks=state_hooks or {},  # type: ignore[arg-type]
        on_state_complete=on_state_complete,  # type: ignore[arg-type]
    )


def test_run_to_completion_on_state_complete_called_for_every_state() -> None:
    """on_state_complete is invoked once for each workflow state."""
    completed: List[str] = []

    with patch("nnf.workflow.advance", MagicMock()), \
            patch("nnf.workflow.wait_for_state", return_value=(True, "")), \
            patch("nnf.workflow.delete", MagicMock()):
        ok, _ = run_to_completion(
            _make_wf(on_state_complete=[completed.append]), timeout=60
        )

    assert ok is True
    assert completed == workflow.WORKFLOW_STATES


def test_run_to_completion_success() -> None:
    """run_to_completion advances through all states and deletes on success."""
    state_iter = iter(workflow.WORKFLOW_STATES)

    def fake_wait(name: str, ns: str, state: str, timeout: int) -> Tuple[bool, str]:
        assert state == next(state_iter)
        return True, ""

    mock_advance = MagicMock()
    mock_delete = MagicMock()
    with patch("nnf.workflow.advance", mock_advance), \
            patch("nnf.workflow.wait_for_state", side_effect=fake_wait), \
            patch("nnf.workflow.delete", mock_delete):
        ok, msg = run_to_completion(_make_wf(), timeout=60)

    assert ok is True
    assert msg == ""
    assert mock_advance.call_count == len(workflow.WORKFLOW_STATES) - 1
    mock_delete.assert_called_once()


def test_run_to_completion_failure_triggers_teardown() -> None:
    """run_to_completion calls teardown_and_delete when a state fails."""
    # Proposal succeeds; Setup fails.
    responses = [(True, ""), (False, "Setup timed out")]

    mock_advance = MagicMock()
    mock_teardown = MagicMock()
    with patch("nnf.workflow.advance", mock_advance), \
            patch("nnf.workflow.wait_for_state", side_effect=responses), \
            patch("nnf.workflow.teardown_and_delete", mock_teardown):
        ok, msg = run_to_completion(_make_wf(), timeout=60)

    assert ok is False
    assert "Setup timed out" in msg
    mock_teardown.assert_called_once()


def test_run_to_completion_post_proposal_called_after_proposal() -> None:
    """Proposal hook is invoked after Proposal succeeds, before Setup."""
    call_order: List[str] = []

    state_iter = iter(workflow.WORKFLOW_STATES)

    def fake_wait(name: str, ns: str, state: str, timeout: int) -> Tuple[bool, str]:
        assert state == next(state_iter)
        call_order.append(f"wait:{state}")
        return True, ""

    def fake_hook(name: str, ns: str) -> Tuple[bool, str]:
        call_order.append("hook")
        return True, ""

    with patch("nnf.workflow.advance", MagicMock()), \
            patch("nnf.workflow.wait_for_state", side_effect=fake_wait), \
            patch("nnf.workflow.delete", MagicMock()):
        ok, _ = run_to_completion(
            _make_wf(state_hooks={"Proposal": [fake_hook]}), timeout=60
        )

    assert ok is True
    proposal_idx = call_order.index("wait:Proposal")
    hook_idx = call_order.index("hook")
    setup_idx = call_order.index("wait:Setup")
    assert proposal_idx < hook_idx < setup_idx


def test_run_to_completion_post_data_in_called_after_data_in() -> None:
    """DataIn hook is invoked after DataIn succeeds, before PreRun."""
    call_order: List[str] = []

    state_iter = iter(workflow.WORKFLOW_STATES)

    def fake_wait(name: str, ns: str, state: str, timeout: int) -> Tuple[bool, str]:
        assert state == next(state_iter)
        call_order.append(f"wait:{state}")
        return True, ""

    def fake_hook(name: str, ns: str) -> Tuple[bool, str]:
        call_order.append("hook")
        return True, ""

    with patch("nnf.workflow.advance", MagicMock()), \
            patch("nnf.workflow.wait_for_state", side_effect=fake_wait), \
            patch("nnf.workflow.delete", MagicMock()):
        ok, _ = run_to_completion(
            _make_wf(state_hooks={"DataIn": [fake_hook]}), timeout=60
        )

    assert ok is True
    data_in_idx = call_order.index("wait:DataIn")
    hook_idx = call_order.index("hook")
    pre_run_idx = call_order.index("wait:PreRun")
    assert data_in_idx < hook_idx < pre_run_idx


def test_run_to_completion_post_data_in_failure_triggers_teardown() -> None:
    """Teardown is triggered when the DataIn hook returns failure."""
    responses = [(True, "")] * 3  # Proposal, Setup, DataIn
    mock_teardown = MagicMock()
    hook = MagicMock(return_value=(False, "no computes available"))

    with patch("nnf.workflow.advance", MagicMock()), \
            patch("nnf.workflow.wait_for_state", side_effect=responses), \
            patch("nnf.workflow.teardown_and_delete", mock_teardown):
        ok, msg = run_to_completion(
            _make_wf(state_hooks={"DataIn": [hook]}), timeout=60
        )

    assert ok is False
    assert "no computes available" in msg
    mock_teardown.assert_called_once()


def test_run_to_completion_post_proposal_failure_triggers_teardown() -> None:
    """Teardown is triggered when the Proposal hook returns failure."""
    mock_teardown = MagicMock()
    hook = MagicMock(return_value=(False, "not enough rabbits"))

    with patch("nnf.workflow.advance", MagicMock()), \
            patch("nnf.workflow.wait_for_state", return_value=(True, "")), \
            patch("nnf.workflow.teardown_and_delete", mock_teardown):
        ok, msg = run_to_completion(
            _make_wf(state_hooks={"Proposal": [hook]}), timeout=60
        )

    assert ok is False
    assert "not enough rabbits" in msg
    mock_teardown.assert_called_once()
