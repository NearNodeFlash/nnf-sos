"""Tests for servers helpers (now in workflow.py)."""

import math
from typing import Any, Dict, Optional
from unittest.mock import MagicMock, call, patch

import pytest

from nnf.workflow import (
    build_alloc_sets,
    fill_computes,
    fill_servers,
    fill_servers_default,
    get_rabbits_from_system_config,
    get_storage_profile,
    has_standalone_mgt,
)


# ---------------------------------------------------------------------------
# Resource shape helpers
# ---------------------------------------------------------------------------


def test_get_storage_profile_fetches_profile_from_nnf_system() -> None:
    """get_storage_profile fetches the named NNF storage profile."""
    mock_get = MagicMock(return_value={"metadata": {"name": "profile-a"}})

    with patch("nnf.workflow.k8s.get_object", mock_get):
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
    profile = {"data": {"lustreStorage": {"mgtOptions": {"standaloneMgtPoolName": "main-pool"}}}}
    assert has_standalone_mgt(profile) is True


def test_has_standalone_mgt_returns_false_when_missing() -> None:
    """has_standalone_mgt returns False when the nested field is absent."""
    assert has_standalone_mgt({"data": {"lustreStorage": {}}}) is False
    assert has_standalone_mgt({}) is False


def test_get_rabbits_from_system_config_filters_rabbit_nodes() -> None:
    """get_rabbits_from_system_config returns only nodes marked as Rabbit."""
    mock_get = MagicMock(return_value={
        "spec": {
            "storageNodes": [
                {"name": "rabbit-0", "type": "Rabbit"},
                {"name": "hare-0", "type": "Other"},
                {"name": "rabbit-1", "type": "Rabbit"},
            ]
        }
    })

    with patch("nnf.workflow.k8s.get_object", mock_get):
        rabbits = get_rabbits_from_system_config(namespace="ns-a", name="cfg-a")

    assert rabbits == ["rabbit-0", "rabbit-1"]
    mock_get.assert_called_once_with(
        group="dataworkflowservices.github.io",
        version="v1alpha7",
        namespace="ns-a",
        plural="systemconfigurations",
        name="cfg-a",
    )


def test_get_rabbits_from_system_config_raises_when_no_rabbits() -> None:
    """get_rabbits_from_system_config raises ValueError when no Rabbit nodes exist."""
    mock_get = MagicMock(return_value={"spec": {"storageNodes": [{"name": "n-0", "type": "Other"}]}})

    with patch("nnf.workflow.k8s.get_object", mock_get):
        with pytest.raises(ValueError, match="No Rabbit nodes found"):
            get_rabbits_from_system_config()


# ---------------------------------------------------------------------------
# build_alloc_sets
# ---------------------------------------------------------------------------


def _alloc_set(
    strategy: str,
    min_capacity: int,
    label: str = "ost",
    constraints: Optional[Dict[str, Any]] = None,
) -> Dict[str, Any]:
    result: Dict[str, Any] = {
        "allocationStrategy": strategy,
        "minimumCapacity": min_capacity,
        "label": label,
    }
    if constraints:
        result["constraints"] = constraints
    return result


def test_build_alloc_sets_single_server_uses_first_rabbit() -> None:
    """AllocateSingleServer always uses the first Rabbit, full capacity."""
    sets = build_alloc_sets([_alloc_set("AllocateSingleServer", 1024, "ost")], ["rabbit-0", "rabbit-1"])
    assert len(sets) == 1
    assert sets[0]["allocationSize"] == 1024
    assert sets[0]["storage"] == [{"name": "rabbit-0", "allocationCount": 1}]


def test_build_alloc_sets_per_compute_uses_all_rabbits() -> None:
    """AllocatePerCompute creates one entry per Rabbit."""
    sets = build_alloc_sets(
        [_alloc_set("AllocatePerCompute", 512, "xfs")], ["rabbit-0", "rabbit-1"]
    )
    assert len(sets[0]["storage"]) == 2
    assert {s["name"] for s in sets[0]["storage"]} == {"rabbit-0", "rabbit-1"}
    assert sets[0]["allocationSize"] == 512


def test_build_alloc_sets_across_servers_splits_capacity() -> None:
    """AllocateAcrossServers divides capacity across provided Rabbits."""
    sets = build_alloc_sets(
        [_alloc_set("AllocateAcrossServers", 1000, "ost")], ["rabbit-0", "rabbit-1"]
    )
    expected_size = math.ceil(1000 / 2)
    assert sets[0]["allocationSize"] == expected_size
    assert len(sets[0]["storage"]) == 2
    assert sets[0]["storage"][0]["allocationCount"] == 1


def test_build_alloc_sets_across_servers_alloc_count() -> None:
    """AllocateAcrossServers divides aggregate capacity by rabbits x alloc_count."""
    sets = build_alloc_sets(
        [_alloc_set("AllocateAcrossServers", 1000, "ost")],
        ["rabbit-0", "rabbit-1"],
        alloc_count=2,
    )
    # 2 rabbits x 2 allocations each = 4 total; ceil(1000 / 4) = 250
    assert sets[0]["allocationSize"] == math.ceil(1000 / 4)
    assert len(sets[0]["storage"]) == 2
    assert sets[0]["storage"][0]["allocationCount"] == 2


def test_build_alloc_sets_across_servers_with_count_constraint() -> None:
    """AllocateAcrossServers with count constraint uses only that many Rabbits."""
    sets = build_alloc_sets(
        [_alloc_set("AllocateAcrossServers", 900, "mdt", constraints={"count": 1})],
        ["rabbit-0", "rabbit-1", "rabbit-2"],
    )
    assert len(sets[0]["storage"]) == 1
    assert sets[0]["storage"][0]["name"] == "rabbit-0"
    assert sets[0]["allocationSize"] == 900


def test_build_alloc_sets_count_exceeds_rabbits_raises() -> None:
    """build_alloc_sets raises ValueError when count constraint exceeds available Rabbits (non-ost)."""
    with pytest.raises(ValueError, match="requires 3 Rabbit"):
        build_alloc_sets(
            [_alloc_set("AllocateAcrossServers", 1000, "mdt", constraints={"count": 3})],
            ["rabbit-0"],
        )


def test_build_alloc_sets_ost_ignores_count() -> None:
    """ost allocation sets ignore the count constraint and use all provided Rabbits."""
    sets = build_alloc_sets(
        [_alloc_set("AllocateAcrossServers", 1000, "ost", constraints={"count": 1})],
        ["rabbit-0", "rabbit-1", "rabbit-2"],
    )
    assert {s["name"] for s in sets[0]["storage"]} == {"rabbit-0", "rabbit-1", "rabbit-2"}


def test_build_alloc_sets_scale_min_selects_one_rabbit() -> None:
    """scale=1 selects exactly 1 Rabbit for non-ost labels."""
    sets = build_alloc_sets(
        [_alloc_set("AllocateAcrossServers", 1000, "mdt", constraints={"scale": 1})],
        ["rabbit-0", "rabbit-1", "rabbit-2", "rabbit-3", "rabbit-4"],
    )
    assert len(sets[0]["storage"]) == 1
    assert sets[0]["storage"][0]["name"] == "rabbit-0"


def test_build_alloc_sets_scale_max_selects_all_rabbits() -> None:
    """scale=10 selects all provided Rabbits for non-ost labels."""
    rabbits = [f"rabbit-{i}" for i in range(5)]
    sets = build_alloc_sets(
        [_alloc_set("AllocateAcrossServers", 1000, "mdt", constraints={"scale": 10})],
        rabbits,
    )
    assert len(sets[0]["storage"]) == 5


def test_build_alloc_sets_scale_mid_selects_proportional_rabbits() -> None:
    """scale=5 with 10 rabbits selects ~5 Rabbits for non-ost labels."""
    rabbits = [f"rabbit-{i}" for i in range(10)]
    sets = build_alloc_sets(
        [_alloc_set("AllocateAcrossServers", 1000, "mdt", constraints={"scale": 5})],
        rabbits,
    )
    assert len(sets[0]["storage"]) == 5


def test_build_alloc_sets_scale_ignored_for_ost() -> None:
    """scale constraint is ignored for ost; all provided Rabbits are used."""
    rabbits = ["rabbit-0", "rabbit-1", "rabbit-2"]
    sets = build_alloc_sets(
        [_alloc_set("AllocateAcrossServers", 1000, "ost", constraints={"scale": 1})],
        rabbits,
    )
    assert len(sets[0]["storage"]) == 3


def test_build_alloc_sets_count_overrides_scale() -> None:
    """count constraint takes precedence over scale."""
    rabbits = [f"rabbit-{i}" for i in range(5)]
    sets = build_alloc_sets(
        [_alloc_set("AllocateAcrossServers", 1000, "mdt", constraints={"count": 2, "scale": 10})],
        rabbits,
    )
    assert len(sets[0]["storage"]) == 2


def test_build_alloc_sets_label_rabbits_override() -> None:
    """label_rabbits overrides the default rabbit list for the matching label."""
    raw_sets = [
        _alloc_set("AllocateSingleServer", 512, "mgt"),
        _alloc_set("AllocateSingleServer", 1024, "mdt"),
        _alloc_set("AllocateAcrossServers", 4096, "ost"),
    ]
    result = build_alloc_sets(
        raw_sets,
        rabbits=["ost-0", "ost-1"],
        label_rabbits={"mgt": ["mgt-0"], "mdt": ["mdt-0"], "mgtmdt": ["mdt-0"]},
    )
    assert result[0]["storage"] == [{"name": "mgt-0", "allocationCount": 1}]
    assert result[1]["storage"] == [{"name": "mdt-0", "allocationCount": 1}]
    # ost falls back to the default rabbits list
    assert {s["name"] for s in result[2]["storage"]} == {"ost-0", "ost-1"}


def test_build_alloc_sets_label_rabbits_ignores_count_scale() -> None:
    """When label_rabbits overrides a label, count/scale constraints are ignored."""
    raw_sets = [
        _alloc_set("AllocateAcrossServers", 4096, "mdt", constraints={"count": 1}),
        _alloc_set("AllocateAcrossServers", 4096, "mgt", constraints={"scale": 2}),
        _alloc_set("AllocateAcrossServers", 8192, "ost"),
    ]
    result = build_alloc_sets(
        raw_sets,
        rabbits=["ost-0", "ost-1", "ost-2"],
        label_rabbits={"mdt": ["mdt-0", "mdt-1"], "mgt": ["mgt-0", "mgt-1", "mgt-2"]},
    )
    # mdt: count=1 in breakdown, but override has 2 rabbits — all 2 should be used
    assert {s["name"] for s in result[0]["storage"]} == {"mdt-0", "mdt-1"}
    # mgt: scale=2 in breakdown, but override has 3 rabbits — all 3 should be used
    assert {s["name"] for s in result[1]["storage"]} == {"mgt-0", "mgt-1", "mgt-2"}
    # ost: no override, uses default rabbits
    assert {s["name"] for s in result[2]["storage"]} == {"ost-0", "ost-1", "ost-2"}


def test_build_alloc_sets_empty_label_rabbits_override_raises_for_single() -> None:
    """An empty label override raises ValueError instead of indexing into an empty list."""
    with pytest.raises(ValueError, match="Allocation set 'mgt' must have at least one Rabbit"):
        build_alloc_sets(
            [_alloc_set("AllocateSingleServer", 512, "mgt")],
            rabbits=["ost-0"],
            label_rabbits={"mgt": []},
        )


def test_build_alloc_sets_empty_label_rabbits_override_raises_for_across_servers() -> None:
    """An empty label override raises ValueError instead of dividing by zero."""
    with pytest.raises(ValueError, match="Allocation set 'mdt' must have at least one Rabbit"):
        build_alloc_sets(
            [_alloc_set("AllocateAcrossServers", 4096, "mdt")],
            rabbits=["ost-0", "ost-1"],
            label_rabbits={"mdt": []},
        )


def test_build_alloc_sets_empty_rabbits_raises() -> None:
    """build_alloc_sets raises ValueError when no Rabbits are provided."""
    with pytest.raises(ValueError, match="At least one Rabbit"):
        build_alloc_sets([_alloc_set("AllocateSingleServer", 1024, "ost")], [])


def test_build_alloc_sets_multiple_sets() -> None:
    """build_alloc_sets handles multiple allocationSets (e.g. Lustre mgt+mdt+ost)."""
    raw_sets = [
        _alloc_set("AllocateSingleServer", 512, "mgt"),
        _alloc_set("AllocateSingleServer", 1024, "mdt"),
        _alloc_set("AllocateAcrossServers", 4096, "ost"),
    ]
    result = build_alloc_sets(raw_sets, ["rabbit-0", "rabbit-1"])
    assert len(result) == 3
    assert result[0]["label"] == "mgt"
    assert result[1]["label"] == "mdt"
    assert result[2]["label"] == "ost"


# ---------------------------------------------------------------------------
# fill_servers
# ---------------------------------------------------------------------------


def test_fill_servers_patches_named_servers_resource() -> None:
    """fill_servers patches the requested Servers resource directly."""
    alloc_sets = [{"label": "ost", "allocationSize": 1024, "storage": [{"name": "rabbit-0", "allocationCount": 1}]}]
    mock_patch = MagicMock(return_value={})

    with patch("nnf.workflow.k8s.patch_object", mock_patch):
        ok, msg = fill_servers("wf", "default", "servers-0", alloc_sets)

    assert ok is True
    assert msg == ""
    mock_patch.assert_called_once_with(
        group="dataworkflowservices.github.io",
        version="v1alpha7",
        namespace="default",
        plural="servers",
        name="servers-0",
        body={"spec": {"allocationSets": alloc_sets}},
    )


def test_fill_servers_returns_false_on_patch_failure() -> None:
    """fill_servers returns False when patching the Servers resource fails."""
    import kubernetes.client.exceptions  # type: ignore[import-untyped]

    exc = kubernetes.client.exceptions.ApiException(status=422, reason="Unprocessable")

    with patch("nnf.workflow.k8s.patch_object", side_effect=exc):
        ok, msg = fill_servers("wf", "default", "servers-0", [])

    assert ok is False
    assert "Failed to patch Servers 'servers-0'" in msg


# ---------------------------------------------------------------------------
# fill_servers_default
# ---------------------------------------------------------------------------


def _workflow_with_breakdowns() -> Dict[str, Any]:
    return {
        "status": {
            "directiveBreakdowns": [{"name": "wf-0", "namespace": "default"}],
            "computes": {"name": "computes-0", "namespace": "default"},
        }
    }


def _ready_breakdown() -> Dict[str, Any]:
    return {
        "status": {
            "ready": True,
            "storage": {
                "allocationSets": [
                    {"allocationStrategy": "AllocateSingleServer", "minimumCapacity": 1024, "label": "ost"}
                ],
                "reference": {"name": "servers-0", "namespace": "default"},
            },
        }
    }


def test_fill_servers_patches_servers_resource() -> None:
    """fill_servers_default patches the Servers resource for the matching DirectiveBreakdown."""
    # Calls: (1) fetch workflow, (2) _wait_for_breakdown poll, (3) fetch breakdown details
    mock_get = MagicMock(side_effect=[_workflow_with_breakdowns(), _ready_breakdown(), _ready_breakdown()])
    mock_patch = MagicMock(return_value={})

    with patch("nnf.workflow.k8s.get_object", mock_get), \
            patch("nnf.workflow.k8s.patch_object", mock_patch), \
            patch("nnf.workflow.time.sleep"):
        ok, msg = fill_servers_default("wf", "default", ["rabbit-0"], timeout=10)

    assert ok is True
    assert msg == ""
    assert mock_patch.call_count == 1  # Only Servers patched, not Computes.


def test_fill_servers_skips_when_no_storage_breakdown() -> None:
    """fill_servers_default skips Servers patching when status.storage is absent from a breakdown."""
    breakdown_no_storage: Dict[str, Any] = {"status": {"ready": True}}
    # Calls: (1) fetch workflow, (2) _wait_for_breakdown poll, (3) fetch breakdown details
    mock_get = MagicMock(side_effect=[_workflow_with_breakdowns(), breakdown_no_storage, breakdown_no_storage])
    mock_patch = MagicMock(return_value={})

    with patch("nnf.workflow.k8s.get_object", mock_get), \
            patch("nnf.workflow.k8s.patch_object", mock_patch), \
            patch("nnf.workflow.time.sleep"):
        ok, _ = fill_servers_default("wf", "default", ["rabbit-0"], timeout=10)

    assert ok is True
    assert mock_patch.call_count == 0


def test_fill_servers_returns_false_when_workflow_fetch_fails() -> None:
    """fill_servers_default returns False when the Workflow cannot be fetched."""
    import kubernetes.client.exceptions  # type: ignore[import-untyped]

    exc = kubernetes.client.exceptions.ApiException(status=404, reason="Not Found")
    with patch("nnf.workflow.k8s.get_object", side_effect=exc):
        ok, msg = fill_servers_default("wf", "default", ["rabbit-0"], timeout=10)
    assert ok is False
    assert "Failed to fetch Workflow" in msg


def test_fill_servers_returns_false_on_servers_patch_failure() -> None:
    """fill_servers_default returns False when patching Servers raises an ApiException."""
    import kubernetes.client.exceptions  # type: ignore[import-untyped]

    # Calls: (1) fetch workflow, (2) _wait_for_breakdown poll, (3) fetch breakdown details
    mock_get = MagicMock(side_effect=[_workflow_with_breakdowns(), _ready_breakdown(), _ready_breakdown()])
    exc = kubernetes.client.exceptions.ApiException(status=422, reason="Unprocessable")

    with patch("nnf.workflow.k8s.get_object", mock_get), \
            patch("nnf.workflow.k8s.patch_object", side_effect=exc), \
            patch("nnf.workflow.time.sleep"):
        ok, msg = fill_servers_default("wf", "default", ["rabbit-0"], timeout=10)

    assert ok is False
    assert "Failed to patch Servers" in msg


# ---------------------------------------------------------------------------
# fill_computes
# ---------------------------------------------------------------------------


def _workflow_with_computes_ref() -> Dict[str, Any]:
    return {
        "status": {
            "computes": {"name": "computes-0", "namespace": "default"},
        }
    }


def test_fill_computes_patches_computes_resource() -> None:
    """fill_computes patches the Computes resource."""
    mock_get = MagicMock(return_value=_workflow_with_computes_ref())
    mock_patch = MagicMock(return_value={})

    with patch("nnf.workflow.k8s.get_object", mock_get), \
            patch("nnf.workflow.k8s.patch_object", mock_patch):
        ok, msg = fill_computes("wf", "default", ["compute-0", "compute-1"])

    assert ok is True
    assert msg == ""
    assert mock_patch.call_count == 1
    _, kwargs = mock_patch.call_args
    patched_body = kwargs["body"]
    assert patched_body == {"data": [{"name": "compute-0"}, {"name": "compute-1"}]}


def test_fill_computes_skips_when_no_computes_ref() -> None:
    """fill_computes succeeds without patching when status.computes is absent."""
    mock_get = MagicMock(return_value={"status": {}})
    mock_patch = MagicMock(return_value={})

    with patch("nnf.workflow.k8s.get_object", mock_get), \
            patch("nnf.workflow.k8s.patch_object", mock_patch):
        ok, _ = fill_computes("wf", "default", [])

    assert ok is True
    assert mock_patch.call_count == 0


def test_fill_computes_returns_false_when_workflow_fetch_fails() -> None:
    """fill_computes returns False when the Workflow cannot be fetched."""
    import kubernetes.client.exceptions  # type: ignore[import-untyped]

    exc = kubernetes.client.exceptions.ApiException(status=500, reason="Internal Server Error")
    with patch("nnf.workflow.k8s.get_object", side_effect=exc):
        ok, msg = fill_computes("wf", "default", [])
    assert ok is False
    assert "Failed to fetch Workflow" in msg


def test_fill_computes_returns_false_on_patch_failure() -> None:
    """fill_computes returns False when patching Computes raises an ApiException."""
    import kubernetes.client.exceptions  # type: ignore[import-untyped]

    mock_get = MagicMock(return_value=_workflow_with_computes_ref())
    exc = kubernetes.client.exceptions.ApiException(status=422, reason="Unprocessable")

    with patch("nnf.workflow.k8s.get_object", mock_get), \
            patch("nnf.workflow.k8s.patch_object", side_effect=exc):
        ok, msg = fill_computes("wf", "default", ["compute-0"])

    assert ok is False
    assert "Failed to patch Computes" in msg
