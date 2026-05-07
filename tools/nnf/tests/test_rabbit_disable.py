"""Tests for the rabbit disable sub-command."""

import argparse
from typing import Dict
from unittest.mock import patch

import kubernetes.client.exceptions  # type: ignore[import-untyped]

from nnf.commands.rabbit.disable import _disable_storage, run


def _make_args(**kwargs: object) -> argparse.Namespace:
    defaults: Dict[str, object] = {
        "nodes": ["rabbit-0"],
        "reason": "none",
    }
    defaults.update(kwargs)
    return argparse.Namespace(**defaults)


# ---------------------------------------------------------------------------
# _disable_storage
# ---------------------------------------------------------------------------


def test_disable_storage_patches_state_and_annotations() -> None:
    """_disable_storage sets spec.state and both annotations."""
    with patch("nnf.commands.rabbit.disable.k8s.patch_object") as mock_patch:
        _disable_storage("rabbit-0", "2026-04-27T10:30:00", "maintenance")

    mock_patch.assert_called_once()
    _, kwargs = mock_patch.call_args
    assert kwargs["name"] == "rabbit-0"
    body = kwargs["body"]
    assert body["spec"]["state"] == "Disabled"
    annotations = body["metadata"]["annotations"]
    assert annotations["disable_date"] == "2026-04-27T10:30:00"
    assert annotations["disable_reason"] == "maintenance"


# ---------------------------------------------------------------------------
# run()
# ---------------------------------------------------------------------------


def test_run_success_single_node() -> None:
    """run() returns 0 for a single successful disable."""
    with patch("nnf.commands.rabbit.disable._disable_storage"):
        rc = run(_make_args())

    assert rc == 0


def test_run_success_multiple_nodes() -> None:
    """run() disables every node in the list."""
    with patch("nnf.commands.rabbit.disable._disable_storage") as mock_dis:
        rc = run(_make_args(nodes=["rabbit-0", "rabbit-1", "rabbit-2"]))

    assert rc == 0
    assert mock_dis.call_count == 3


def test_run_custom_reason() -> None:
    """run() passes the --reason value through."""
    with patch("nnf.commands.rabbit.disable._disable_storage") as mock_dis:
        run(_make_args(reason="bad disk"))

    assert mock_dis.call_args[0][2] == "bad disk"


def test_run_failure_continues_to_next_node() -> None:
    """run() continues to the next node when one fails."""
    exc = kubernetes.client.exceptions.ApiException(status=404, reason="NotFound")
    with patch("nnf.commands.rabbit.disable._disable_storage",
               side_effect=[exc, None]) as mock_dis:
        rc = run(_make_args(nodes=["bad-node", "good-node"]))

    assert rc == 1
    assert mock_dis.call_count == 2


def test_run_all_fail_returns_1() -> None:
    """run() returns 1 when all nodes fail."""
    exc = kubernetes.client.exceptions.ApiException(status=500, reason="InternalError")
    with patch("nnf.commands.rabbit.disable._disable_storage", side_effect=exc):
        rc = run(_make_args(nodes=["rabbit-0", "rabbit-1"]))

    assert rc == 1
