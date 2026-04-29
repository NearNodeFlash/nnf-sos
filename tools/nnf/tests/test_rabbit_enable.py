"""Tests for the rabbit enable sub-command."""

import argparse
from typing import Dict
from unittest.mock import patch

import kubernetes.client.exceptions  # type: ignore[import-untyped]

from nnf.commands.rabbit.enable import _enable_storage, run


def _make_args(**kwargs: object) -> argparse.Namespace:
    defaults: Dict[str, object] = {
        "nodes": ["rabbit-0"],
    }
    defaults.update(kwargs)
    return argparse.Namespace(**defaults)


# ---------------------------------------------------------------------------
# _enable_storage
# ---------------------------------------------------------------------------


def test_enable_storage_patches_state_and_removes_annotations() -> None:
    """_enable_storage sets spec.state to Enabled and nulls disable annotations."""
    with patch("nnf.commands.rabbit.enable.k8s.patch_object") as mock_patch:
        _enable_storage("rabbit-0")

    mock_patch.assert_called_once()
    _, kwargs = mock_patch.call_args
    assert kwargs["name"] == "rabbit-0"
    body = kwargs["body"]
    assert body["spec"]["state"] == "Enabled"
    annotations = body["metadata"]["annotations"]
    assert annotations["disable_date"] is None
    assert annotations["disable_reason"] is None


# ---------------------------------------------------------------------------
# run()
# ---------------------------------------------------------------------------


def test_run_success_single_node() -> None:
    """run() returns 0 for a single successful enable."""
    with patch("nnf.commands.rabbit.enable._enable_storage"):
        rc = run(_make_args())

    assert rc == 0


def test_run_success_multiple_nodes() -> None:
    """run() enables every node in the list."""
    with patch("nnf.commands.rabbit.enable._enable_storage") as mock_en:
        rc = run(_make_args(nodes=["rabbit-0", "rabbit-1", "rabbit-2"]))

    assert rc == 0
    assert mock_en.call_count == 3


def test_run_failure_continues_to_next_node() -> None:
    """run() continues to the next node when one fails."""
    exc = kubernetes.client.exceptions.ApiException(status=404, reason="NotFound")
    with patch("nnf.commands.rabbit.enable._enable_storage",
               side_effect=[exc, None]) as mock_en:
        rc = run(_make_args(nodes=["bad-node", "good-node"]))

    assert rc == 1
    assert mock_en.call_count == 2


def test_run_all_fail_returns_1() -> None:
    """run() returns 1 when all nodes fail."""
    exc = kubernetes.client.exceptions.ApiException(status=500, reason="InternalError")
    with patch("nnf.commands.rabbit.enable._enable_storage", side_effect=exc):
        rc = run(_make_args(nodes=["rabbit-0", "rabbit-1"]))

    assert rc == 1
