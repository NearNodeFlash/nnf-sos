"""Tests for k8s.py helpers."""

from unittest.mock import MagicMock, patch

import pytest

from nnf import k8s


def test_load_config_kubeconfig() -> None:
    """load_config calls load_kube_config when a path is supplied."""
    with patch("kubernetes.config.load_kube_config") as mock_load:
        k8s.load_config(kubeconfig="/some/path")
        mock_load.assert_called_once_with(config_file="/some/path")


def test_load_config_fallback_to_incluster() -> None:
    """load_config falls back to load_incluster_config on ConfigException."""
    import kubernetes.config  # type: ignore[import-untyped]

    with patch(
            "kubernetes.config.load_kube_config",
            side_effect=kubernetes.config.ConfigException,
        ), patch("kubernetes.config.load_incluster_config") as mock_incluster:
        k8s.load_config()
        mock_incluster.assert_called_once()


def test_load_config_explicit_kubeconfig_does_not_fallback() -> None:
    """An explicit kubeconfig path should fail fast instead of using in-cluster config."""
    import kubernetes.config  # type: ignore[import-untyped]

    with patch(
            "kubernetes.config.load_kube_config",
            side_effect=kubernetes.config.ConfigException("bad kubeconfig"),
        ), patch("kubernetes.config.load_incluster_config") as mock_incluster:
        with pytest.raises(kubernetes.config.ConfigException, match="bad kubeconfig"):
            k8s.load_config(kubeconfig="/bad/path")

    mock_incluster.assert_not_called()


def test_get_object_calls_api() -> None:
    """get_object delegates to CustomObjectsApi.get_namespaced_custom_object."""
    mock_api = MagicMock()
    mock_api.get_namespaced_custom_object.return_value = {"metadata": {"name": "test"}}
    with patch("nnf.k8s.get_custom_objects_api", return_value=mock_api):
        result = k8s.get_object("g", "v", "ns", "plural", "test")
    mock_api.get_namespaced_custom_object.assert_called_once_with(
        group="g", version="v", namespace="ns", plural="plural", name="test"
    )
    assert result == {"metadata": {"name": "test"}}


def test_create_object_calls_api() -> None:
    """create_object delegates to CustomObjectsApi.create_namespaced_custom_object."""
    mock_api = MagicMock()
    body: dict = {"metadata": {"name": "test"}}
    mock_api.create_namespaced_custom_object.return_value = body
    with patch("nnf.k8s.get_custom_objects_api", return_value=mock_api):
        result = k8s.create_object("g", "v", "ns", "plural", body)
    mock_api.create_namespaced_custom_object.assert_called_once_with(
        group="g", version="v", namespace="ns", plural="plural", body=body
    )
    assert result == body


def test_delete_object_calls_api() -> None:
    """delete_object delegates to CustomObjectsApi.delete_namespaced_custom_object."""
    mock_api = MagicMock()
    with patch("nnf.k8s.get_custom_objects_api", return_value=mock_api):
        k8s.delete_object("g", "v", "ns", "plural", "test")
    mock_api.delete_namespaced_custom_object.assert_called_once_with(
        group="g", version="v", namespace="ns", plural="plural", name="test"
    )
