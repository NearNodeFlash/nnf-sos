"""Tests for the system version sub-command."""

import argparse
import pytest
import json
from typing import Dict
from unittest.mock import MagicMock, patch

import kubernetes.client.exceptions  # type: ignore[import-untyped]

from nnf.commands.system.version import _get_deployment_labels, run


def _make_args(**kwargs: object) -> argparse.Namespace:
    defaults: Dict[str, object] = {}
    defaults.update(kwargs)
    return argparse.Namespace(**defaults)


def _make_deployment(labels: Dict[str, str]) -> MagicMock:
    deploy = MagicMock()
    deploy.metadata.labels = labels
    return deploy


# ---------------------------------------------------------------------------
# _get_deployment_labels
# ---------------------------------------------------------------------------


@patch("nnf.commands.system.version.k8s.get_deployment")
def test_get_deployment_labels(mock_get_deploy: MagicMock) -> None:
    labels = {"app": "nnf", "version": "v0.1.0"}
    deploy = _make_deployment(labels)
    mock_get_deploy.return_value = deploy
    result = _get_deployment_labels()
    assert result == labels
    mock_get_deploy.assert_called_once_with(
        name="nnf-controller-manager",
        namespace="nnf-system",
    )


@patch("nnf.commands.system.version.k8s.get_deployment")
def test_get_deployment_labels_none(mock_get_deploy: MagicMock) -> None:
    deploy = MagicMock()
    deploy.metadata.labels = None
    mock_get_deploy.return_value = deploy
    result = _get_deployment_labels()
    assert result == {}


# ---------------------------------------------------------------------------
# run
# ---------------------------------------------------------------------------


@patch("nnf.commands.system.version._get_deployment_labels")
def test_run_success(mock_labels: MagicMock, capsys: pytest.CaptureFixture[str]) -> None:
    labels = {"app": "nnf", "version": "v0.1.0", "build": "abc123"}
    mock_labels.return_value = labels
    args = _make_args()
    assert run(args) == 0
    out = capsys.readouterr().out
    parsed = json.loads(out)
    assert parsed == labels


@patch("nnf.commands.system.version._get_deployment_labels")
def test_run_api_error(mock_labels: MagicMock, capsys: pytest.CaptureFixture[str]) -> None:
    mock_labels.side_effect = kubernetes.client.exceptions.ApiException(
        status=404, reason="Not Found",
    )
    args = _make_args()
    assert run(args) == 1
    err = capsys.readouterr().err
    assert "error" in err
    assert "nnf-controller-manager" in err
