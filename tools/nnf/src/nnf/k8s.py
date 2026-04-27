"""Shared Kubernetes client helpers."""

import logging
from typing import Any, Dict, Optional

import urllib.parse

import kubernetes  # type: ignore[import-untyped]
import kubernetes.client  # type: ignore[import-untyped]
import kubernetes.config  # type: ignore[import-untyped]


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
