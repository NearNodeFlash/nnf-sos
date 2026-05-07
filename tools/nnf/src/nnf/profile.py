"""NnfStorageProfile helpers."""

from typing import Any, Dict

from nnf import crd
from nnf import k8s


def get_storage_profile(
    name: str,
    namespace: str = "nnf-system",
) -> Dict[str, Any]:
    """Fetch an NnfStorageProfile resource by name.

    Parameters
    ----------
    name:
        Name of the NnfStorageProfile.
    namespace:
        Namespace containing the profile.  Defaults to ``"nnf-system"``.

    Returns
    -------
    The full resource dict.

    Raises
    ------
    kubernetes.client.exceptions.ApiException
        If the resource cannot be fetched.
    """
    return k8s.get_object(
        group=crd.NNF_GROUP,
        version=crd.NNF_VERSION,
        namespace=namespace,
        plural=crd.NNF_STORAGE_PROFILE_PLURAL,
        name=name,
    )


def has_standalone_mgt(profile: Dict[str, Any]) -> bool:
    """Return True if the profile has ``standaloneMgtPoolName`` set.

    Checks ``data.lustreStorage.mgtOptions.standaloneMgtPoolName`` in the
    profile dict (v1alpha11 layout).
    """
    lustre = profile.get("data", {}).get("lustreStorage", {})
    return bool(lustre.get("mgtOptions", {}).get("standaloneMgtPoolName", ""))
