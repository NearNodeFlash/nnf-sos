"""Tests for the create_persistent sub-command."""

import argparse
from typing import Dict, Tuple
from unittest.mock import MagicMock, patch

import pytest

from nnf.commands.create_persistent import run
from nnf.utils import parse_capacity


# ---------------------------------------------------------------------------
# parse_capacity
# ---------------------------------------------------------------------------


@pytest.mark.parametrize(
    "value, expected",
    [
        ("1GiB", 1024**3),
        ("500MiB", 500 * 1024**2),
        ("2TiB", 2 * 1024**4),
        ("1024KiB", 1024 * 1024),
        ("1073741824", 1073741824),
    ],
)
def test_parse_capacity_valid(value: str, expected: int) -> None:
    """parse_capacity correctly converts known suffixes."""
    assert parse_capacity(value) == expected


def test_parse_capacity_invalid() -> None:
    """parse_capacity raises ValueError for unparseable input."""
    with pytest.raises(ValueError):
        parse_capacity("bogus")


# ---------------------------------------------------------------------------
# run()
# ---------------------------------------------------------------------------


def _make_args(**kwargs: object) -> argparse.Namespace:
    defaults: Dict[str, object] = {
        "name": "my-psi",
        "fs_type": "lustre",
        "capacity": "1GiB",
        "namespace": "default",
        "user_id": 1000,
        "group_id": 1000,
        "timeout": 60,
        "rabbits": ["rabbit-0"],
        "rabbits_mdt": None,
        "rabbits_mgt": None,
        "alloc_count": 1,
        "profile": None,
        "rabbit_count": None,
    }
    defaults.update(kwargs)
    return argparse.Namespace(**defaults)


def test_run_success() -> None:
    """run() returns 0 when the Workflow is created and completes successfully."""
    mock_create = MagicMock(return_value={})
    with patch("nnf.commands.create_persistent.k8s.create_object", mock_create), \
            patch("nnf.commands.create_persistent.workflow.run_to_completion", return_value=(True, "")):
        assert run(_make_args()) == 0

    mock_create.assert_called_once()
    _, kwargs = mock_create.call_args
    body = kwargs["body"]
    assert body["kind"] == "Workflow"
    assert "create_persistent" in body["spec"]["dwDirectives"][0]


def test_run_comma_separated_rabbits() -> None:
    """run() splits comma-separated rabbit names into individual nodes."""
    from nnf.workflow import WorkflowRun

    captured: Dict[str, object] = {}

    def fake_run_to_completion(wf: WorkflowRun, timeout: int) -> Tuple[bool, str]:
        captured["wf"] = wf
        return True, ""

    with patch("nnf.commands.create_persistent.k8s.create_object", return_value={}), \
            patch("nnf.commands.create_persistent.workflow.run_to_completion",
                  side_effect=fake_run_to_completion), \
            patch("nnf.commands.create_persistent.workflow.fill_servers_default",
                  return_value=(True, "")) as mock_fill:
        assert run(_make_args(rabbits=["rabbit-0,rabbit-1", "rabbit-2"])) == 0
        captured["wf"]  # ensure captured
        hooks = captured["wf"].state_hooks["Proposal"]  # type: ignore[index]
        hooks[0]("wf-name", "default")

    mock_fill.assert_called_once()
    _, call_kwargs = mock_fill.call_args
    assert call_kwargs["rabbits"] == ["rabbit-0", "rabbit-1", "rabbit-2"]


def test_run_post_proposal_hook_calls_fill_servers_default() -> None:
    """run() attaches a Proposal hook that delegates to fill_servers_default."""
    from nnf.workflow import WorkflowRun

    captured: Dict[str, object] = {}

    def fake_run_to_completion(wf: WorkflowRun, timeout: int) -> Tuple[bool, str]:
        captured["wf"] = wf
        return True, ""

    with patch("nnf.commands.create_persistent.k8s.create_object", return_value={}), \
            patch(
                "nnf.commands.create_persistent.workflow.run_to_completion",
                side_effect=fake_run_to_completion,
            ), \
            patch("nnf.commands.create_persistent.workflow.fill_servers_default", return_value=(True, "")) as mock_fill:
        assert run(_make_args(rabbits=["rabbit-0", "rabbit-1"])) == 0

        wf = captured["wf"]
        assert isinstance(wf, WorkflowRun)
        hooks = wf.state_hooks["Proposal"]
        assert len(hooks) == 1
        assert callable(hooks[0])
        hooks[0]("wf-name", "default")

    mock_fill.assert_called_once_with(
        "wf-name",
        "default",
        rabbits=["rabbit-0", "rabbit-1"],
        timeout=60,
        directive_index=0,
        alloc_count=1,
        rabbits_mdt=None,
        rabbits_mgt=None,
    )


def test_run_bad_capacity_returns_1() -> None:
    """run() returns exit code 1 when capacity cannot be parsed."""
    assert run(_make_args(capacity="bad")) == 1


def test_run_no_capacity_no_profile_returns_1() -> None:
    """run() returns 1 when --capacity is omitted and no profile is specified."""
    assert run(_make_args(capacity=None)) == 1


def test_run_no_capacity_profile_without_standalone_mgt_returns_1() -> None:
    """run() returns 1 when --capacity is omitted and the profile has no standaloneMgtPoolName."""
    plain_profile: Dict[str, object] = {"data": {"lustreStorage": {}}}
    with patch("nnf.commands.create_persistent.workflow.get_storage_profile",
               return_value=plain_profile):
        assert run(_make_args(capacity=None, profile="plain")) == 1


def test_run_capacity_with_standalone_mgt_profile_returns_1() -> None:
    """run() returns 1 when --capacity is given but the profile has standaloneMgtPoolName."""
    mgt_profile: Dict[str, object] = {
        "data": {"lustreStorage": {"mgtOptions": {"standaloneMgtPoolName": "main-pool"}}}
    }
    with patch("nnf.commands.create_persistent.workflow.get_storage_profile",
               return_value=mgt_profile):
        assert run(_make_args(capacity="1GiB", profile="mgt-profile")) == 1


def test_run_standalone_mgt_profile_omits_capacity_from_directive() -> None:
    """run() omits capacity= from the #DW directive when standaloneMgtPoolName is set."""
    mgt_profile: Dict[str, object] = {
        "data": {"lustreStorage": {"mgtOptions": {"standaloneMgtPoolName": "main-pool"}}}
    }
    mock_create = MagicMock(return_value={})
    with patch("nnf.commands.create_persistent.workflow.get_storage_profile",
               return_value=mgt_profile), \
            patch("nnf.commands.create_persistent.k8s.create_object", mock_create), \
            patch("nnf.commands.create_persistent.workflow.run_to_completion",
                  return_value=(True, "")):
        assert run(_make_args(capacity=None, profile="mgt-profile")) == 0

    _, kwargs = mock_create.call_args
    body = kwargs["body"]
    directive = body["spec"]["dwDirectives"][0]
    assert "capacity=" not in directive
    assert "profile=mgt-profile" in directive


def test_run_profile_fetch_error_returns_1() -> None:
    """run() returns 1 when fetching the storage profile fails."""
    import kubernetes.client.exceptions  # type: ignore[import-untyped]
    exc = kubernetes.client.exceptions.ApiException(status=404, reason="NotFound")
    with patch("nnf.commands.create_persistent.workflow.get_storage_profile",
               side_effect=exc):
        assert run(_make_args(profile="missing-profile")) == 1


def test_run_profile_unexpected_error_propagates() -> None:
    """run() should not swallow unexpected profile lookup failures."""
    with patch("nnf.commands.create_persistent.workflow.get_storage_profile",
               side_effect=RuntimeError("boom")):
        with pytest.raises(RuntimeError, match="boom"):
            run(_make_args(profile="broken-profile"))


def test_run_non_lustre_fs_type_skips_profile_check() -> None:
    """run() does not fetch the storage profile when fs_type is not lustre."""
    with patch("nnf.commands.create_persistent.workflow.get_storage_profile") as mock_get, \
            patch("nnf.commands.create_persistent.k8s.create_object", return_value={}), \
            patch("nnf.commands.create_persistent.workflow.run_to_completion",
                  return_value=(True, "")):
        assert run(_make_args(fs_type="xfs", profile="some-profile")) == 0
    mock_get.assert_not_called()


def test_run_standalone_mgt_multiple_rabbits_returns_1() -> None:
    """run() returns 1 when standaloneMgtPoolName is set but more than one rabbit is given."""
    mgt_profile: Dict[str, object] = {
        "data": {"lustreStorage": {"mgtOptions": {"standaloneMgtPoolName": "main-pool"}}}
    }
    with patch("nnf.commands.create_persistent.workflow.get_storage_profile",
               return_value=mgt_profile):
        assert run(_make_args(capacity=None, profile="mgt-profile",
                              rabbits=["rabbit-0", "rabbit-1"])) == 1


def test_run_standalone_mgt_alloc_count_gt1_returns_1() -> None:
    """run() returns 1 when standaloneMgtPoolName is set but alloc_count > 1."""
    mgt_profile: Dict[str, object] = {
        "data": {"lustreStorage": {"mgtOptions": {"standaloneMgtPoolName": "main-pool"}}}
    }
    with patch("nnf.commands.create_persistent.workflow.get_storage_profile",
               return_value=mgt_profile):
        assert run(_make_args(capacity=None, profile="mgt-profile",
                              rabbits=["rabbit-0"], alloc_count=2)) == 1


def test_run_no_rabbits_source_returns_1() -> None:
    """run() returns 1 when neither --rabbits nor --rabbit-count is given."""
    assert run(_make_args(rabbits=None, rabbit_count=None)) == 1


def test_run_rabbit_count_with_rabbits_returns_1() -> None:
    """run() returns 1 when --rabbit-count is combined with --rabbits."""
    assert run(_make_args(rabbit_count=2, rabbits=["rabbit-0"])) == 1


def test_run_rabbit_count_with_rabbits_mdt_returns_1() -> None:
    """run() returns 1 when --rabbit-count is combined with --rabbits-mdt."""
    assert run(_make_args(rabbit_count=2, rabbits=None, rabbits_mdt=["rabbit-0"])) == 1


def test_run_rabbit_count_with_rabbits_mgt_returns_1() -> None:
    """run() returns 1 when --rabbit-count is combined with --rabbits-mgt."""
    assert run(_make_args(rabbit_count=2, rabbits=None, rabbits_mgt=["rabbit-0"])) == 1


def test_run_rabbit_count_zero_returns_1() -> None:
    """run() returns 1 when --rabbit-count is 0."""
    assert run(_make_args(rabbit_count=0, rabbits=None)) == 1


def test_run_rabbit_count_exceeds_available_returns_1() -> None:
    """run() returns 1 when --rabbit-count exceeds the available rabbits."""
    _sys_config = {
        "spec": {"storageNodes": [{"name": "r-0", "type": "Rabbit"}]}
    }
    with patch("nnf.commands.create_persistent.workflow.get_rabbits_from_system_config",
               return_value=["r-0"]):
        assert run(_make_args(rabbit_count=5, rabbits=None)) == 1


def test_run_rabbit_count_picks_random_rabbits() -> None:
    """run() picks the requested number of rabbits at random from SystemConfiguration."""
    from nnf.workflow import WorkflowRun

    all_rabbits = [f"rabbit-{i}" for i in range(10)]
    captured: Dict[str, object] = {}

    def fake_run_to_completion(wf: WorkflowRun, timeout: int) -> Tuple[bool, str]:
        captured["wf"] = wf
        return True, ""

    with patch("nnf.commands.create_persistent.k8s.create_object", return_value={}), \
            patch("nnf.commands.create_persistent.workflow.run_to_completion",
                  side_effect=fake_run_to_completion), \
            patch("nnf.commands.create_persistent.workflow.get_rabbits_from_system_config",
                  return_value=all_rabbits), \
            patch("nnf.commands.create_persistent.workflow.fill_servers_default",
                  return_value=(True, "")) as mock_fill:
        assert run(_make_args(rabbit_count=3, rabbits=None)) == 0
        hooks = captured["wf"].state_hooks["Proposal"]  # type: ignore[index]
        hooks[0]("wf-name", "default")

    _, call_kwargs = mock_fill.call_args
    chosen = call_kwargs["rabbits"]
    assert len(chosen) == 3
    assert all(r in all_rabbits for r in chosen)
    # mdt/mgt not overridden — scale/count constraints will apply
    assert call_kwargs["rabbits_mdt"] is None
    assert call_kwargs["rabbits_mgt"] is None


def test_run_rabbit_count_unexpected_error_propagates() -> None:
    """run() should not swallow unexpected SystemConfiguration lookup failures."""
    with patch("nnf.commands.create_persistent.workflow.get_rabbits_from_system_config",
               side_effect=RuntimeError("boom")):
        with pytest.raises(RuntimeError, match="boom"):
            run(_make_args(rabbit_count=1, rabbits=None))


def test_run_create_api_error_returns_2() -> None:
    """run() returns exit code 2 when the initial Workflow creation fails."""
    import kubernetes.client.exceptions  # type: ignore[import-untyped]

    exc = kubernetes.client.exceptions.ApiException(status=409, reason="AlreadyExists")
    with patch("nnf.commands.create_persistent.k8s.create_object", side_effect=exc):
        assert run(_make_args()) == 2


def test_run_workflow_failure_returns_2() -> None:
    """run() returns exit code 2 when run_to_completion reports failure."""
    with patch("nnf.commands.create_persistent.k8s.create_object", return_value={}), \
            patch(
                "nnf.commands.create_persistent.workflow.run_to_completion",
                return_value=(False, "Setup timed out"),
            ):
        assert run(_make_args()) == 2
