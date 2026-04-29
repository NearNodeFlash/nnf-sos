"""Tests for the rabbit undrain sub-command."""

import argparse
from typing import Any, Dict, List, Optional
from unittest.mock import MagicMock, patch

import kubernetes.client.exceptions  # type: ignore[import-untyped]
import pytest

from nnf.commands.rabbit.drain import TAINT_KEY, _remove_drain_annotations
from nnf.commands.rabbit.undrain import (
    _remove_drain_taints,
    run,
)


def _make_args(**kwargs: object) -> argparse.Namespace:
    defaults: Dict[str, object] = {
        "nodes": ["rabbit-0"],
    }
    defaults.update(kwargs)
    return argparse.Namespace(**defaults)


def _make_taint(key: str, value: str, effect: str) -> MagicMock:
    t = MagicMock()
    t.key = key
    t.value = value
    t.effect = effect
    return t


def _make_node(existing_taints: Optional[List[Any]] = None) -> MagicMock:
    node = MagicMock()
    node.spec.taints = existing_taints
    return node


# ---------------------------------------------------------------------------
# _remove_drain_taints
# ---------------------------------------------------------------------------


def test_remove_drain_taints_removes_drain_keeps_others() -> None:
    """Drain taints are removed; other taints are preserved."""
    drain_taint = _make_taint(TAINT_KEY, "true", "NoSchedule")
    other_taint = _make_taint("other-key", "val", "NoSchedule")
    node = _make_node([drain_taint, other_taint])
    with patch("nnf.commands.rabbit.undrain.k8s.get_core_v1_api") as mock_api_fn, \
            patch("nnf.commands.rabbit.undrain.k8s.patch_node") as mock_patch:
        mock_api_fn.return_value.read_node.return_value = node
        _remove_drain_taints("rabbit-0")

    body = mock_patch.call_args[0][1]
    taints = body["spec"]["taints"]
    assert len(taints) == 1
    assert taints[0]["key"] == "other-key"


def test_remove_drain_taints_sets_none_when_no_taints_remain() -> None:
    """Taints field is set to None when no taints remain."""
    drain_taint = _make_taint(TAINT_KEY, "true", "NoExecute")
    node = _make_node([drain_taint])
    with patch("nnf.commands.rabbit.undrain.k8s.get_core_v1_api") as mock_api_fn, \
            patch("nnf.commands.rabbit.undrain.k8s.patch_node") as mock_patch:
        mock_api_fn.return_value.read_node.return_value = node
        _remove_drain_taints("rabbit-0")

    body = mock_patch.call_args[0][1]
    assert body["spec"]["taints"] is None


def test_remove_drain_taints_no_existing_taints() -> None:
    """No error when node has no taints at all."""
    node = _make_node(None)
    with patch("nnf.commands.rabbit.undrain.k8s.get_core_v1_api") as mock_api_fn, \
            patch("nnf.commands.rabbit.undrain.k8s.patch_node") as mock_patch:
        mock_api_fn.return_value.read_node.return_value = node
        _remove_drain_taints("rabbit-0")

    body = mock_patch.call_args[0][1]
    assert body["spec"]["taints"] is None


# ---------------------------------------------------------------------------
# _remove_drain_annotations
# ---------------------------------------------------------------------------


def test_remove_drain_annotations_nulls_annotations() -> None:
    """_remove_drain_annotations sends None for both drain annotations."""
    with patch("nnf.commands.rabbit.undrain.k8s.patch_object") as mock_patch:
        _remove_drain_annotations("rabbit-0")

    _, kwargs = mock_patch.call_args
    assert kwargs["name"] == "rabbit-0"
    annotations = kwargs["body"]["metadata"]["annotations"]
    assert annotations["drain_date"] is None
    assert annotations["drain_reason"] is None


# ---------------------------------------------------------------------------
# run()
# ---------------------------------------------------------------------------


def test_run_success_single_node() -> None:
    """run() returns 0 for a single successful undrain."""
    with patch("nnf.commands.rabbit.undrain._remove_drain_taints"), \
            patch("nnf.commands.rabbit.undrain._remove_drain_annotations"):
        rc = run(_make_args())

    assert rc == 0


def test_run_success_multiple_nodes() -> None:
    """run() undrains every node in the list."""
    with patch("nnf.commands.rabbit.undrain._remove_drain_taints"), \
            patch("nnf.commands.rabbit.undrain._remove_drain_annotations") as mock_ann:
        rc = run(_make_args(nodes=["rabbit-0", "rabbit-1", "rabbit-2"]))

    assert rc == 0
    assert mock_ann.call_count == 3


def test_run_taint_failure_continues_to_next_node() -> None:
    """run() continues to the next node when untainting fails."""
    exc = kubernetes.client.exceptions.ApiException(status=404, reason="NotFound")
    with patch("nnf.commands.rabbit.undrain._remove_drain_annotations"), \
            patch("nnf.commands.rabbit.undrain._remove_drain_taints", side_effect=[exc, None]):
        rc = run(_make_args(nodes=["bad-node", "good-node"]))

    assert rc == 1


def test_run_annotation_failure_continues_to_next_node() -> None:
    """run() continues to the next node when annotation removal fails."""
    exc = kubernetes.client.exceptions.ApiException(status=500, reason="InternalError")
    with patch("nnf.commands.rabbit.undrain._remove_drain_annotations",
                  side_effect=[exc, None]), \
            patch("nnf.commands.rabbit.undrain._remove_drain_taints"):
        rc = run(_make_args(nodes=["rabbit-0", "rabbit-1"]))

    assert rc == 1


def test_run_all_fail_returns_1() -> None:
    """run() returns 1 when all nodes fail."""
    exc = kubernetes.client.exceptions.ApiException(status=404, reason="NotFound")
    with patch("nnf.commands.rabbit.undrain._remove_drain_annotations", side_effect=exc):
        rc = run(_make_args(nodes=["rabbit-0", "rabbit-1"]))

    assert rc == 1
