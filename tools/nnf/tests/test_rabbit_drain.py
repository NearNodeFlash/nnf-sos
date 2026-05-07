"""Tests for the rabbit drain sub-command."""

import argparse
from typing import Any, Dict, List, Optional
from unittest.mock import MagicMock, patch

import kubernetes.client.exceptions  # type: ignore[import-untyped]
import pytest

from nnf.commands.rabbit.drain import (
    TAINT_KEY,
    TAINT_VALUE,
    _annotate_storage,
    _apply_drain_taints,
    run,
)


def _make_args(**kwargs: object) -> argparse.Namespace:
    defaults: Dict[str, object] = {
        "nodes": ["rabbit-0"],
        "reason": "none",
    }
    defaults.update(kwargs)
    return argparse.Namespace(**defaults)


# ---------------------------------------------------------------------------
# _apply_drain_taints
# ---------------------------------------------------------------------------


def _make_node(existing_taints: Optional[List[Any]] = None) -> MagicMock:
    """Return a mock Node object with optional existing taints."""
    node = MagicMock()
    node.spec.taints = existing_taints
    return node


def _make_taint(key: str, value: str, effect: str) -> MagicMock:
    t = MagicMock()
    t.key = key
    t.value = value
    t.effect = effect
    return t


def test_apply_drain_taints_no_existing_taints() -> None:
    """Applies both drain taints when the node has no existing taints."""
    node = _make_node(existing_taints=None)
    with patch("nnf.commands.rabbit.drain.k8s.get_core_v1_api") as mock_api_fn, \
            patch("nnf.commands.rabbit.drain.k8s.patch_node") as mock_patch:
        mock_api_fn.return_value.read_node.return_value = node
        _apply_drain_taints("rabbit-0")

    mock_patch.assert_called_once()
    body = mock_patch.call_args[0][1]
    taints = body["spec"]["taints"]
    assert len(taints) == 2
    assert taints[0] == {"key": TAINT_KEY, "value": TAINT_VALUE, "effect": "NoSchedule"}
    assert taints[1] == {"key": TAINT_KEY, "value": TAINT_VALUE, "effect": "NoExecute"}


def test_apply_drain_taints_preserves_existing_taints() -> None:
    """Existing non-drain taints are preserved."""
    other_taint = _make_taint("other-key", "val", "NoSchedule")
    node = _make_node(existing_taints=[other_taint])
    with patch("nnf.commands.rabbit.drain.k8s.get_core_v1_api") as mock_api_fn, \
            patch("nnf.commands.rabbit.drain.k8s.patch_node") as mock_patch:
        mock_api_fn.return_value.read_node.return_value = node
        _apply_drain_taints("rabbit-0")

    body = mock_patch.call_args[0][1]
    taints = body["spec"]["taints"]
    assert len(taints) == 3
    assert taints[0] == {"key": "other-key", "value": "val", "effect": "NoSchedule"}


def test_apply_drain_taints_replaces_existing_drain_taints() -> None:
    """Existing drain taints are replaced rather than duplicated."""
    old_drain = _make_taint(TAINT_KEY, "old", "NoSchedule")
    other_taint = _make_taint("keep-me", "yes", "NoExecute")
    node = _make_node(existing_taints=[old_drain, other_taint])
    with patch("nnf.commands.rabbit.drain.k8s.get_core_v1_api") as mock_api_fn, \
            patch("nnf.commands.rabbit.drain.k8s.patch_node") as mock_patch:
        mock_api_fn.return_value.read_node.return_value = node
        _apply_drain_taints("rabbit-0")

    body = mock_patch.call_args[0][1]
    taints = body["spec"]["taints"]
    # 1 preserved + 2 new drain taints
    assert len(taints) == 3
    keys = [t["key"] for t in taints]
    assert keys.count(TAINT_KEY) == 2
    assert keys.count("keep-me") == 1


# ---------------------------------------------------------------------------
# _annotate_storage
# ---------------------------------------------------------------------------


def test_annotate_storage_patches_with_drain_metadata() -> None:
    """_annotate_storage sends the correct merge-patch body."""
    with patch("nnf.commands.rabbit.drain.k8s.patch_object") as mock_patch:
        _annotate_storage("rabbit-0", "2026-04-27T10:30:00", "maintenance")

    mock_patch.assert_called_once()
    _, kwargs = mock_patch.call_args
    assert kwargs["name"] == "rabbit-0"
    annotations = kwargs["body"]["metadata"]["annotations"]
    assert annotations["drain_date"] == "2026-04-27T10:30:00"
    assert annotations["drain_reason"] == "maintenance"


# ---------------------------------------------------------------------------
# run()
# ---------------------------------------------------------------------------


def test_run_success_single_node() -> None:
    """run() returns 0 and prints success for a single node."""
    with patch("nnf.commands.rabbit.drain._apply_drain_taints") as mock_taint, \
            patch("nnf.commands.rabbit.drain._annotate_storage") as mock_annotate:
        rc = run(_make_args())

    assert rc == 0
    mock_taint.assert_called_once_with("rabbit-0")
    mock_annotate.assert_called_once()
    call_args = mock_annotate.call_args[0]
    assert call_args[0] == "rabbit-0"
    assert call_args[2] == "none"


def test_run_success_multiple_nodes() -> None:
    """run() drains every node in the list."""
    args = _make_args(nodes=["rabbit-0", "rabbit-1", "rabbit-2"])
    with patch("nnf.commands.rabbit.drain._apply_drain_taints"), \
            patch("nnf.commands.rabbit.drain._annotate_storage") as mock_annotate:
        rc = run(args)

    assert rc == 0
    assert mock_annotate.call_count == 3


def test_run_custom_reason() -> None:
    """run() passes the --reason value to _annotate_storage."""
    with patch("nnf.commands.rabbit.drain._apply_drain_taints"), \
            patch("nnf.commands.rabbit.drain._annotate_storage") as mock_annotate:
        run(_make_args(reason="disk failure"))

    assert mock_annotate.call_args[0][2] == "disk failure"


def test_run_taint_failure_continues_to_next_node() -> None:
    """run() continues to the next node when tainting fails."""
    exc = kubernetes.client.exceptions.ApiException(status=404, reason="NotFound")
    with patch("nnf.commands.rabbit.drain._annotate_storage"), \
            patch("nnf.commands.rabbit.drain._apply_drain_taints", side_effect=[exc, None]), \
            patch("nnf.commands.rabbit.drain.remove_drain_annotations"):
        rc = run(_make_args(nodes=["bad-node", "good-node"]))

    assert rc == 1


def test_run_annotate_failure_continues_to_next_node() -> None:
    """run() continues to the next node when annotation fails."""
    exc = kubernetes.client.exceptions.ApiException(status=500, reason="InternalError")
    with patch("nnf.commands.rabbit.drain._annotate_storage",
                  side_effect=[exc, None]), \
            patch("nnf.commands.rabbit.drain._apply_drain_taints"):
        rc = run(_make_args(nodes=["rabbit-0", "rabbit-1"]))

    assert rc == 1


def test_run_taint_failure_rollback_failure_continues() -> None:
    """run() continues to the next node when rollback of storage annotation also fails."""
    taint_exc = kubernetes.client.exceptions.ApiException(status=500, reason="InternalError")
    rollback_exc = kubernetes.client.exceptions.ApiException(status=500, reason="RollbackFailed")
    with patch("nnf.commands.rabbit.drain._annotate_storage"), \
            patch("nnf.commands.rabbit.drain._apply_drain_taints",
                  side_effect=[taint_exc, None]), \
            patch("nnf.commands.rabbit.drain.remove_drain_annotations",
                  side_effect=[rollback_exc, None]):
        rc = run(_make_args(nodes=["rabbit-0", "rabbit-1"]))

    # rabbit-0 failed (taint + rollback), rabbit-1 succeeded → errors > 0
    assert rc == 1


def test_run_all_fail_returns_1() -> None:
    """run() returns 1 when all nodes fail."""
    exc = kubernetes.client.exceptions.ApiException(status=404, reason="NotFound")
    with patch("nnf.commands.rabbit.drain._annotate_storage", side_effect=exc):
        rc = run(_make_args(nodes=["rabbit-0", "rabbit-1"]))

    assert rc == 1
