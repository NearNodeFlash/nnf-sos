"""Shared Kubernetes client helpers."""

import logging
from typing import Any, Dict, List, Optional

import urllib.parse

import kubernetes  # type: ignore[import-untyped]
import kubernetes.client  # type: ignore[import-untyped]
import kubernetes.config  # type: ignore[import-untyped]
import kubernetes.stream  # type: ignore[import-untyped]


LOGGER = logging.getLogger(__name__)


def load_config(kubeconfig: Optional[str] = None) -> None:
    """Load kubeconfig from the given path, or fall back to in-cluster config."""
    try:
        kubernetes.config.load_kube_config(config_file=kubeconfig)
    except kubernetes.config.ConfigException:
        if kubeconfig is not None:
            raise
        kubernetes.config.load_incluster_config()

    cfg = kubernetes.client.Configuration.get_default_copy()
    LOGGER.info("Connected to cluster: %s", cfg.host)


def get_custom_objects_api() -> kubernetes.client.CustomObjectsApi:
    """Return a configured CustomObjectsApi instance."""
    return kubernetes.client.CustomObjectsApi()


def get_object(
    group: str,
    version: str,
    namespace: str,
    plural: str,
    name: str,
) -> Dict[str, Any]:
    """Fetch a single namespaced custom object."""
    api = get_custom_objects_api()
    return api.get_namespaced_custom_object(  # type: ignore[no-any-return]
        group=group,
        version=version,
        namespace=namespace,
        plural=plural,
        name=name,
    )


def debug_api_group(group: str) -> None:
    """Print the versions and resources available for a given API group."""
    client = kubernetes.client.ApiClient()
    # Query /apis/<group> to list available versions.
    path = f"/apis/{urllib.parse.quote(group, safe='.')}"
    try:
        response = client.call_api(
            path, "GET",
            auth_settings=["BearerToken"],
            response_type="object",
            _return_http_data_only=True,
        )
        versions = [v["version"] for v in response.get("versions", [])]
        LOGGER.info("API group '%s' available versions: %s", group, versions)
    except Exception as exc:  # noqa: BLE001
        LOGGER.info("Could not query API group '%s': %s", group, exc)


def create_object(
    group: str,
    version: str,
    namespace: str,
    plural: str,
    body: Dict[str, Any],
) -> Dict[str, Any]:
    """Create a namespaced custom object."""
    LOGGER.info(
        "Creating %s/%s '%s' in namespace '%s'",
        group,
        version,
        plural,
        namespace,
    )
    api = get_custom_objects_api()
    return api.create_namespaced_custom_object(  # type: ignore[no-any-return]
        group=group,
        version=version,
        namespace=namespace,
        plural=plural,
        body=body,
    )


def patch_object(
    group: str,
    version: str,
    namespace: str,
    plural: str,
    name: str,
    body: Dict[str, Any],
) -> Dict[str, Any]:
    """Merge-patch a namespaced custom object."""
    api = get_custom_objects_api()
    return api.patch_namespaced_custom_object(  # type: ignore[no-any-return]
        group=group,
        version=version,
        namespace=namespace,
        plural=plural,
        name=name,
        body=body,
    )


def delete_object(
    group: str,
    version: str,
    namespace: str,
    plural: str,
    name: str,
) -> None:
    """Delete a namespaced custom object."""
    api = get_custom_objects_api()
    api.delete_namespaced_custom_object(
        group=group,
        version=version,
        namespace=namespace,
        plural=plural,
        name=name,
    )


# ---------------------------------------------------------------------------
# Core-API helpers (Nodes)
# ---------------------------------------------------------------------------


def get_core_v1_api() -> kubernetes.client.CoreV1Api:
    """Return a configured CoreV1Api instance."""
    return kubernetes.client.CoreV1Api()


def patch_node(name: str, body: Dict[str, Any]) -> Any:
    """Strategic-merge-patch a Node object."""
    api = get_core_v1_api()
    return api.patch_node(name, body)


def list_objects(
    group: str,
    version: str,
    namespace: str,
    plural: str,
) -> Dict[str, Any]:
    """List namespaced custom objects."""
    api = get_custom_objects_api()
    return api.list_namespaced_custom_object(  # type: ignore[no-any-return]
        group=group,
        version=version,
        namespace=namespace,
        plural=plural,
    )


def list_pods(
    namespace: str,
    label_selector: str = "",
    field_selector: str = "",
) -> List[Any]:
    """List pods in a namespace, optionally filtering by label or field selector."""
    api = get_core_v1_api()
    result = api.list_namespaced_pod(
        namespace=namespace,
        label_selector=label_selector,
        field_selector=field_selector,
    )
    return result.items  # type: ignore[no-any-return]


def exec_pod(
    namespace: str,
    pod_name: str,
    command: List[str],
    container: Optional[str] = None,
    strip_channel_bytes: bool = False,
) -> str:
    """Execute a command inside a pod and return stdout.

    When *strip_channel_bytes* is ``True``, leading WebSocket channel-marker
    bytes (``\\x00``–``\\x03``) are stripped from the output.  Enable this
    when the response is expected to be text or JSON and the Kubernetes
    stream may include binary framing.
    """
    api = get_core_v1_api()
    kwargs: Dict[str, Any] = {
        "name": pod_name,
        "namespace": namespace,
        "command": command,
        "stderr": False,
        "stdin": False,
        "stdout": True,
        "tty": False,
    }
    if container is not None:
        kwargs["container"] = container
    raw: str = kubernetes.stream.stream(
        api.connect_get_namespaced_pod_exec, **kwargs
    )
    if strip_channel_bytes:
        raw = raw.lstrip("\x00\x01\x02\x03")
    return raw
