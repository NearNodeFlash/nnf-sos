"""Tests for profile helpers."""

from typing import Any, Dict
from unittest.mock import MagicMock, patch

from nnf.profile import get_storage_profile, has_standalone_mgt


def test_get_storage_profile_fetches_profile_from_nnf_system() -> None:
    """get_storage_profile fetches the named NNF storage profile."""
    mock_get = MagicMock(return_value={"metadata": {"name": "profile-a"}})

    with patch("nnf.profile.k8s.get_object", mock_get):
        result = get_storage_profile("profile-a")

    assert result["metadata"]["name"] == "profile-a"
    mock_get.assert_called_once_with(
        group="nnf.cray.hpe.com",
        version="v1alpha11",
        namespace="nnf-system",
        plural="nnfstorageprofiles",
        name="profile-a",
    )


def test_has_standalone_mgt_returns_true_when_pool_name_present() -> None:
    """has_standalone_mgt detects standaloneMgtPoolName in the profile."""
    profile: Dict[str, Any] = {
        "data": {"lustreStorage": {"mgtOptions": {"standaloneMgtPoolName": "main-pool"}}}
    }
    assert has_standalone_mgt(profile) is True


def test_has_standalone_mgt_returns_false_when_missing() -> None:
    """has_standalone_mgt returns False when the nested field is absent."""
    assert has_standalone_mgt({"data": {"lustreStorage": {}}}) is False
    assert has_standalone_mgt({}) is False
