"""Tests for the persistent share sub-command."""

import argparse
import pytest
from typing import Dict
from unittest.mock import MagicMock, patch

import kubernetes.client.exceptions  # type: ignore[import-untyped]

from nnf import crd
from nnf.commands.persistent.share import run


def _make_args(**kwargs: object) -> argparse.Namespace:
    defaults: Dict[str, object] = {
        "name": "my-psi",
        "namespace": "default",
    }
    defaults.update(kwargs)
    return argparse.Namespace(**defaults)


@patch("nnf.commands.persistent.share.k8s.patch_object")
def test_run_success(mock_patch: MagicMock, capsys: pytest.CaptureFixture[str]) -> None:
    assert run(_make_args()) == 0
    mock_patch.assert_called_once()
    body = mock_patch.call_args[1]["body"]
    assert body["metadata"]["annotations"][crd.DWS_IGNORE_UID_ANNOTATION] == "true"
    out = capsys.readouterr().out
    assert "Shared" in out


@patch("nnf.commands.persistent.share.k8s.patch_object")
def test_run_patches_correct_resource(mock_patch: MagicMock) -> None:
    run(_make_args(name="foo", namespace="bar"))
    assert mock_patch.call_args[1]["name"] == "foo"
    assert mock_patch.call_args[1]["namespace"] == "bar"


@patch("nnf.commands.persistent.share.k8s.patch_object")
def test_run_api_error(mock_patch: MagicMock, capsys: pytest.CaptureFixture[str]) -> None:
    mock_patch.side_effect = kubernetes.client.exceptions.ApiException(
        status=404, reason="Not Found",
    )
    assert run(_make_args()) == 1
    err = capsys.readouterr().err
    assert "error" in err
    assert "my-psi" in err
