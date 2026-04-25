"""Servers, Computes, and resource population helpers for DWS Workflows.

This module provides helpers for populating Servers and Computes resources,
building allocation sets from DirectiveBreakdowns, and fetching related
NNF/DWS resources (storage profiles, system configurations)."""

import math
import logging
import time
from typing import Any, Dict, List, Optional, Tuple

import kubernetes.client.exceptions  # type: ignore[import-untyped]

from nnf import crd
from nnf import k8s


LOGGER = logging.getLogger(__name__)

# ---------------------------------------------------------------------------
# Resource helpers
# ---------------------------------------------------------------------------


def get_rabbits_from_system_config(
    namespace: str = "default",
    name: str = "default",
) -> List[str]:
    """Return all Rabbit node names from a SystemConfiguration resource.

    Parameters
    ----------
    namespace:
        Namespace of the SystemConfiguration resource.  Defaults to
        ``"default"``.
    name:
        Name of the SystemConfiguration resource.  Defaults to ``"default"``.

    Returns
    -------
    List of Rabbit node names from ``spec.storageNodes`` where ``type == "Rabbit"``.

    Raises
    ------
    kubernetes.client.exceptions.ApiException
        If the resource cannot be fetched.
    ValueError
        If no Rabbit nodes are found.
    """
    obj = k8s.get_object(
        group=crd.DWS_GROUP,
        version=crd.DWS_VERSION,
        namespace=namespace,
        plural=crd.DWS_SYSTEM_CONFIGURATION_PLURAL,
        name=name,
    )
    rabbits = [
        node["name"]
        for node in obj.get("spec", {}).get("storageNodes", [])
        if node.get("type") == "Rabbit"
    ]
    if not rabbits:
        raise ValueError(
            f"No Rabbit nodes found in SystemConfiguration '{namespace}/{name}'."
        )
    return rabbits


# ---------------------------------------------------------------------------
# Allocation-set math
# ---------------------------------------------------------------------------

_ALLOC_PER_COMPUTE = "AllocatePerCompute"
_ALLOC_ACROSS = "AllocateAcrossServers"
_ALLOC_SINGLE = "AllocateSingleServer"

_OST_LABEL = "ost"


def _scale_to_count(scale: int, num_rabbits: int) -> int:
    """Map a scale hint (1–10) to a number of Rabbits, linear between the extremes.

    scale=1  → 1 Rabbit
    scale=10 → num_rabbits
    """
    if num_rabbits <= 1:
        return num_rabbits
    return max(1, round(1 + (scale - 1) * (num_rabbits - 1) / 9.0))


def build_alloc_sets(
    breakdown_alloc_sets: List[Dict[str, Any]],
    rabbits: List[str],
    alloc_count: int = 1,
    label_rabbits: Optional[Dict[str, List[str]]] = None,
) -> List[Dict[str, Any]]:
    """Build ServersSpec.allocationSets from a DirectiveBreakdown's allocationSets.

    Parameters
    ----------
    breakdown_alloc_sets:
        The ``status.storage.allocationSets`` list from a DirectiveBreakdown.
    rabbits:
        Default ordered list of Rabbit node names to place storage on.
    alloc_count:
        Number of allocations per Rabbit for the ``AllocatePerCompute`` and
        ``AllocateAcrossServers``/``AllocatePerServer`` strategies.  For
        ``AllocateAcrossServers`` the breakdown's ``minimumCapacity`` is the
        *aggregate* across all allocations, so each individual allocation
        receives ``minimumCapacity / (num_rabbits * alloc_count)``.  Ignored
        for ``AllocateSingleServer``.  Defaults to ``1``.
    label_rabbits:
        Optional per-label Rabbit overrides.  When a label is present in this
        mapping its list is used instead of *rabbits* for that allocation set.
        E.g. ``{"mgt": ["rabbit-0"], "mdt": ["rabbit-1"], "mgtmdt": ["rabbit-1"]}``.

    Returns
    -------
    List of dicts suitable for patching into ``spec.allocationSets`` of a Servers resource.

    Raises
    ------
    ValueError
        If *rabbits* is empty or a ``count`` constraint exceeds the number of rabbits provided.

    Notes
    -----
    For the ``AllocateAcrossServers`` strategy the ``constraints.count`` field takes
    precedence.  If only ``constraints.scale`` is present it is mapped linearly:
    ``scale=1`` selects 1 Rabbit, ``scale=10`` selects all provided Rabbits.  Scale
    is **ignored** for ``ost`` allocation sets, which always use the full Rabbit list
    (or ``count`` if given).
    """
    if alloc_count < 1:
        raise ValueError("alloc_count must be at least 1")
    if not rabbits:
        raise ValueError("At least one Rabbit node must be specified (--rabbits).")
    if label_rabbits is None:
        label_rabbits = {}

    result: List[Dict[str, Any]] = []
    for alloc_set in breakdown_alloc_sets:
        strategy: str = alloc_set["allocationStrategy"]
        min_capacity: int = alloc_set["minimumCapacity"]
        label: str = alloc_set["label"]

        # Use the per-label override if provided, otherwise the default list.
        effective_rabbits = label_rabbits.get(label, rabbits)
        if not effective_rabbits:
            raise ValueError(
                f"Allocation set '{label}' must have at least one Rabbit node."
            )

        if strategy == _ALLOC_SINGLE:
            # One allocation on the first provided Rabbit.
            storage = [{"name": effective_rabbits[0], "allocationCount": 1}]
            alloc_size = min_capacity

        elif strategy == _ALLOC_PER_COMPUTE:
            storage = [{"name": r, "allocationCount": alloc_count} for r in effective_rabbits]
            alloc_size = min_capacity

        else:  # AllocateAcrossServers / AllocatePerServer
            constraints: Dict[str, Any] = alloc_set.get("constraints", {})
            count = int(constraints.get("count", 0))
            scale = int(constraints.get("scale", 0))

            if label in label_rabbits:
                # Explicit rabbit list provided for this label — use all of them,
                # ignoring any count/scale constraints from the breakdown.
                chosen = effective_rabbits
            elif count and label != _OST_LABEL:
                if count > len(effective_rabbits):
                    raise ValueError(
                        f"Allocation set '{label}' requires {count} Rabbit(s) "
                        f"but only {len(effective_rabbits)} were provided."
                    )
                chosen = effective_rabbits[:count]
            elif scale and label != _OST_LABEL:
                chosen = effective_rabbits[:_scale_to_count(scale, len(effective_rabbits))]
            else:
                # ost ignores both count and scale; always use the full rabbit list.
                chosen = effective_rabbits
            total_allocs = len(chosen) * alloc_count
            alloc_size = math.ceil(min_capacity / total_allocs)
            storage = [{"name": r, "allocationCount": alloc_count} for r in chosen]

        result.append({"label": label, "allocationSize": alloc_size, "storage": storage})
    return result


# ---------------------------------------------------------------------------
# Servers / Computes population helpers
# ---------------------------------------------------------------------------

_BREAKDOWN_POLL_INTERVAL = 2.0  # seconds


def _wait_for_breakdown(
    name: str,
    namespace: str,
    timeout: int,
) -> Tuple[bool, str]:
    """Poll a DirectiveBreakdown until ``status.ready`` is True."""
    deadline = time.monotonic() + timeout
    while time.monotonic() < deadline:
        try:
            obj = k8s.get_object(
                group=crd.DWS_GROUP,
                version=crd.DWS_VERSION,
                namespace=namespace,
                plural=crd.DWS_DIRECTIVE_BREAKDOWN_PLURAL,
                name=name,
            )
        except kubernetes.client.exceptions.ApiException as exc:
            if exc.status is not None and (exc.status >= 500 or exc.status == 429):
                LOGGER.info("Transient API error (%s), retrying...", exc.status)
                time.sleep(_BREAKDOWN_POLL_INTERVAL)
                continue
            return False, f"API error fetching DirectiveBreakdown '{name}': {exc.reason}"

        if obj.get("status", {}).get("ready", False):
            return True, ""

        time.sleep(_BREAKDOWN_POLL_INTERVAL)

    return False, f"Timed out waiting for DirectiveBreakdown '{name}' to be ready"


def fill_servers(
    workflow_name: str,
    namespace: str,
    servers_name: str,
    alloc_sets: List[Dict[str, Any]],
) -> Tuple[bool, str]:
    """Patch a named Servers resource with caller-supplied allocation sets.

    Unlike ``fill_servers_default``, this does not consult a DirectiveBreakdown.
    The caller supplies the target Servers resource name and the allocation sets
    to apply directly.  Use this when non-default allocation behaviour is needed.

    Parameters
    ----------
    workflow_name:
        Name of the Workflow (used in log and error messages).
    namespace:
        Kubernetes namespace for the Servers resource.
    servers_name:
        Name of the Servers resource to patch.
    alloc_sets:
        Allocation sets to write into ``spec.allocationSets``.

    Returns
    -------
    ``(True, "")`` on success, ``(False, message)`` on failure.
    """
    try:
        k8s.patch_object(
            group=crd.DWS_GROUP,
            version=crd.DWS_VERSION,
            namespace=namespace,
            plural=crd.DWS_SERVERS_PLURAL,
            name=servers_name,
            body={"spec": {"allocationSets": alloc_sets}},
        )
    except kubernetes.client.exceptions.ApiException as exc:
        return False, f"Failed to patch Servers '{servers_name}': {exc.reason}"

    LOGGER.info("Patched Servers '%s'.", servers_name)
    return True, ""


def fill_servers_default(
    workflow_name: str,
    namespace: str,
    rabbits: List[str],
    timeout: int,
    directive_index: int = 0,
    alloc_count: int = 1,
    rabbits_mdt: Optional[List[str]] = None,
    rabbits_mgt: Optional[List[str]] = None,
) -> Tuple[bool, str]:
    """Populate the Servers resource for a single directive of a Workflow.

    Locates the DirectiveBreakdown named ``<workflow-name>-<directive_index>``
    in ``status.directiveBreakdowns``, waits for it to be ready, then patches
    the Servers resource it references with allocation sets derived from the
    breakdown and the provided Rabbit nodes.

    For non-default allocation behaviour, use ``fill_servers`` and supply the
    Servers resource name and allocation sets directly.

    Call this after Proposal is complete, before advancing to Setup.

    Parameters
    ----------
    workflow_name:
        Name of the Workflow resource.
    namespace:
        Kubernetes namespace.
    rabbits:
        Rabbit node names to allocate storage on.
    timeout:
        Seconds to wait for the DirectiveBreakdown to become ready.
    directive_index:
        0-based index of the ``#DW`` directive whose Servers resource should
        be populated.  The corresponding DirectiveBreakdown is located by name
        (``<workflow-name>-<directive_index>``), not by list position, so
        directives that produce no breakdown do not affect the lookup.
        Defaults to ``0``.
    alloc_count:
        Number of allocations per Rabbit when the DirectiveBreakdown specifies
        ``AllocatePerCompute``.  Ignored for other strategies.  Defaults to ``1``.
    rabbits_mdt:
        Rabbit nodes to use for ``mdt`` and ``mgtmdt`` allocation sets.  Falls
        back to *rabbits* when not specified.
    rabbits_mgt:
        Rabbit nodes to use for ``mgt`` allocation sets.  Falls back to *rabbits*
        when not specified.

    Returns
    -------
    ``(True, "")`` on success, ``(False, message)`` on failure.
    """
    try:
        wf_obj = k8s.get_object(
            group=crd.DWS_GROUP,
            version=crd.DWS_VERSION,
            namespace=namespace,
            plural=crd.DWS_WORKFLOW_PLURAL,
            name=workflow_name,
        )
    except kubernetes.client.exceptions.ApiException as exc:
        return False, f"Failed to fetch Workflow '{workflow_name}': {exc.reason}"

    status: Dict[str, Any] = wf_obj.get("status", {})

    expected_bd_name = f"{workflow_name}-{directive_index}"
    bd_ref = None
    for ref in status.get("directiveBreakdowns", []):
        if ref.get("name") == expected_bd_name:
            bd_ref = ref
            break

    if bd_ref is None:
        return False, (
            f"No DirectiveBreakdown '{expected_bd_name}' found for "
            f"Workflow '{workflow_name}'."
        )

    bd_name: str = bd_ref["name"]
    bd_namespace: str = bd_ref.get("namespace", namespace)

    ok, msg = _wait_for_breakdown(bd_name, bd_namespace, timeout)
    if not ok:
        return False, msg

    try:
        breakdown = k8s.get_object(
            group=crd.DWS_GROUP,
            version=crd.DWS_VERSION,
            namespace=bd_namespace,
            plural=crd.DWS_DIRECTIVE_BREAKDOWN_PLURAL,
            name=bd_name,
        )
    except kubernetes.client.exceptions.ApiException as exc:
        return False, f"Failed to fetch DirectiveBreakdown '{bd_name}': {exc.reason}"

    bd_status: Dict[str, Any] = breakdown.get("status", {})
    storage_bd = bd_status.get("storage")
    if storage_bd is None:
        # Directive has no storage breakdown (e.g. #DW persist_dw).
        return True, ""

    server_ref: Dict[str, Any] = storage_bd.get("reference", {})
    if not server_ref:
        return True, ""

    server_name: str = server_ref["name"]
    server_namespace: str = server_ref.get("namespace", namespace)

    try:
        alloc_sets = build_alloc_sets(
            storage_bd.get("allocationSets", []),
            rabbits,
            alloc_count,
            label_rabbits={
                k: v
                for k, v in [
                    ("mdt", rabbits_mdt),
                    ("mgtmdt", rabbits_mdt),
                    ("mgt", rabbits_mgt),
                ]
                if v is not None
            },
        )
    except ValueError as exc:
        return False, str(exc)

    try:
        k8s.patch_object(
            group=crd.DWS_GROUP,
            version=crd.DWS_VERSION,
            namespace=server_namespace,
            plural=crd.DWS_SERVERS_PLURAL,
            name=server_name,
            body={"spec": {"allocationSets": alloc_sets}},
        )
    except kubernetes.client.exceptions.ApiException as exc:
        return False, f"Failed to patch Servers '{server_name}': {exc.reason}"

    for alloc_set in alloc_sets:
        entries = alloc_set["storage"]
        names = ", ".join(e["name"] for e in entries)
        count = entries[0]["allocationCount"] if entries else 0
        LOGGER.info(
            "[%s] %s Rabbit(s): %s (%s allocation(s) each)",
            alloc_set["label"],
            len(entries),
            names,
            count,
        )
    LOGGER.info("Patched Servers '%s'.", server_name)

    return True, ""


def fill_computes(
    workflow_name: str,
    namespace: str,
    computes: List[str],
) -> Tuple[bool, str]:
    """Populate the Computes resource for a Workflow that has reached DataIn.

    Reads ``status.computes`` from the Workflow and patches it with the
    provided compute node names.

    Call this after DataIn is complete, before advancing to PreRun.

    Parameters
    ----------
    workflow_name:
        Name of the Workflow resource.
    namespace:
        Kubernetes namespace.
    computes:
        Compute node names to associate with the Workflow.

    Returns
    -------
    ``(True, "")`` on success, ``(False, message)`` on failure.
    """
    try:
        wf_obj = k8s.get_object(
            group=crd.DWS_GROUP,
            version=crd.DWS_VERSION,
            namespace=namespace,
            plural=crd.DWS_WORKFLOW_PLURAL,
            name=workflow_name,
        )
    except kubernetes.client.exceptions.ApiException as exc:
        return False, f"Failed to fetch Workflow '{workflow_name}': {exc.reason}"

    computes_ref: Dict[str, Any] = wf_obj.get("status", {}).get("computes", {})
    if not computes_ref:
        return True, ""

    computes_name: str = computes_ref["name"]
    computes_namespace: str = computes_ref.get("namespace", namespace)

    try:
        k8s.patch_object(
            group=crd.DWS_GROUP,
            version=crd.DWS_VERSION,
            namespace=computes_namespace,
            plural=crd.DWS_COMPUTES_PLURAL,
            name=computes_name,
            body={"data": [{"name": c} for c in computes]},
        )
    except kubernetes.client.exceptions.ApiException as exc:
        return False, f"Failed to patch Computes '{computes_name}': {exc.reason}"

    LOGGER.info("Patched Computes '%s'.", computes_name)
    return True, ""
