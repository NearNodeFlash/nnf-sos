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


# ---------------------------------------------------------------------------
# exec_pod
# ---------------------------------------------------------------------------


def test_exec_pod_strips_channel_bytes() -> None:
    """exec_pod strips WebSocket channel-marker bytes when opted in."""
    raw = "\x01{\"key\": \"value\"}"
    mock_api = MagicMock()
    with patch("nnf.k8s.get_core_v1_api", return_value=mock_api), \
            patch("kubernetes.stream.stream", return_value=raw):
        result = k8s.exec_pod("ns", "pod", ["echo"], strip_channel_bytes=True)

    assert result == '{"key": "value"}'


def test_exec_pod_strips_embedded_channel_bytes() -> None:
    """exec_pod removes channel-marker bytes at frame boundaries mid-string."""
    raw = "\x01{\"key\":\x01 \"value\"}"
    mock_api = MagicMock()
    with patch("nnf.k8s.get_core_v1_api", return_value=mock_api), \
            patch("kubernetes.stream.stream", return_value=raw):
        result = k8s.exec_pod("ns", "pod", ["echo"], strip_channel_bytes=True)

    assert result == '{"key": "value"}'


def test_exec_pod_preserves_channel_bytes_by_default() -> None:
    """exec_pod does not strip channel bytes unless asked."""
    raw = "\x01{\"key\": \"value\"}"
    mock_api = MagicMock()
    with patch("nnf.k8s.get_core_v1_api", return_value=mock_api), \
            patch("kubernetes.stream.stream", return_value=raw):
        result = k8s.exec_pod("ns", "pod", ["echo"])

    assert result == raw


def test_exec_pod_returns_clean_output_unchanged() -> None:
    """exec_pod returns already-clean output as-is."""
    clean = '{"key": "value"}'
    mock_api = MagicMock()
    with patch("nnf.k8s.get_core_v1_api", return_value=mock_api), \
            patch("kubernetes.stream.stream", return_value=clean):
        result = k8s.exec_pod("ns", "pod", ["echo"])

    assert result == clean
