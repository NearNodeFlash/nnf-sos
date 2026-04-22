"""Shared helpers for creating, driving, and populating DWS Workflow resources.

These utilities are used by any sub-command that needs to create a Workflow,
advance it through the full state machine, populate its Servers/Computes
resources at the appropriate states, and clean up when done."""

import math
import logging
import time
from typing import Any, Callable, Dict, List, Optional, Tuple

import kubernetes.client.exceptions  # type: ignore[import-untyped]

from nnf import crd
from nnf import k8s

# Hook type: receives (workflow_name, namespace) injected by run_to_completion,
# returns (ok, message).
StateHook = Callable[[str, str], Tuple[bool, str]]

# Notification callback: receives the completed state name.
StateCompleteHook = Callable[[str], None]


LOGGER = logging.getLogger(__name__)

# Ordered list of states every Workflow passes through.
WORKFLOW_STATES: List[str] = [
    "Proposal",
    "Setup",
    "DataIn",
    "PreRun",
    "PostRun",
    "DataOut",
    "Teardown",
]

_STATUS_ERROR = "Error"

DEFAULT_TIMEOUT = 180  # seconds per state
POLL_INTERVAL = 3.0   # seconds between polls


class WorkflowRun:
    """Represents a Workflow to be created and driven to completion.

    Bundles the k8s manifest, routing information, and optional transition
    hooks into a single object passed to ``run_to_completion``.

    Hooks
    -----
    state_hooks:
        Mapping of state name → list of ``StateHook`` callables, each called
        in order after that state is ready and before advancing to the next.
        Any state may have one or more hooks.  Common entries:

        * ``"Proposal"`` — populate Servers resources from DirectiveBreakdowns.
        * ``"DataIn"`` — populate the Computes resource.
    on_state_complete:
        List of callbacks invoked after every state completes (after any
        transition hooks).  Each receives the state name.  Useful for
        progress reporting.
    """

    def __init__(
        self,
        name: str,
        namespace: str,
        user_id: int,
        group_id: int = 0,
        dw_directives: Optional[List[str]] = None,
        state_hooks: Optional[Dict[str, List[StateHook]]] = None,
        on_state_complete: Optional[List[StateCompleteHook]] = None,
    ) -> None:
        self.name = name
        self.namespace = namespace
        self.user_id = user_id
        self.group_id = group_id
        self.dw_directives = dw_directives if dw_directives is not None else []
        self.state_hooks = state_hooks if state_hooks is not None else {}
        self.on_state_complete = on_state_complete if on_state_complete is not None else []

    @property
    def manifest(self) -> Dict[str, Any]:
        """Return the Workflow manifest body ready for submission to k8s."""
        return {
            "apiVersion": f"{crd.DWS_GROUP}/{crd.DWS_VERSION}",
            "kind": "Workflow",
            "metadata": {
                "name": self.name,
                "namespace": self.namespace,
            },
            "spec": {
                "desiredState": "Proposal",
                "dwDirectives": self.dw_directives,
                "jobID": 0,
                "userID": self.user_id,
                "groupID": self.group_id,
                "wlmID": "nnf-cli",
            },
        }


def advance(workflow_name: str, namespace: str, state: str) -> None:
    """Patch the Workflow's ``spec.desiredState`` to *state*."""
    k8s.patch_object(
        group=crd.DWS_GROUP,
        version=crd.DWS_VERSION,
        namespace=namespace,
        plural=crd.DWS_WORKFLOW_PLURAL,
        name=workflow_name,
        body={"spec": {"desiredState": state}},
    )


def wait_for_state(
    workflow_name: str,
    namespace: str,
    state: str,
    timeout: int = DEFAULT_TIMEOUT,
) -> Tuple[bool, str]:
    """Poll the Workflow until ``status.state == state`` and ``ready == True``.

    Returns
    -------
    ``(True, "")`` on success.
    ``(False, message)`` if the Workflow enters an unrecoverable ``Error``
    status or the timeout is exceeded.
    """
    deadline = time.monotonic() + timeout
    while time.monotonic() < deadline:
        try:
            obj = k8s.get_object(
                group=crd.DWS_GROUP,
                version=crd.DWS_VERSION,
                namespace=namespace,
                plural=crd.DWS_WORKFLOW_PLURAL,
                name=workflow_name,
            )
        except kubernetes.client.exceptions.ApiException as exc:
            return False, f"API error while polling: {exc.reason} (HTTP {exc.status})"

        status: Dict[str, Any] = obj.get("status", {})
        current_state = status.get("state", "")
        ready = status.get("ready", False)
        wf_status = status.get("status", "")
        message = status.get("message", "")

        if wf_status == _STATUS_ERROR:
            return False, f"Workflow entered Error state: {message}"

        if current_state == state and ready:
            return True, ""

        time.sleep(POLL_INTERVAL)

    return False, f"Timed out waiting for workflow state '{state}' after {timeout}s"


def delete(workflow_name: str, namespace: str) -> None:
    """Delete the Workflow, ignoring 404 (already gone)."""
    try:
        k8s.delete_object(
            group=crd.DWS_GROUP,
            version=crd.DWS_VERSION,
            namespace=namespace,
            plural=crd.DWS_WORKFLOW_PLURAL,
            name=workflow_name,
        )
    except kubernetes.client.exceptions.ApiException as exc:
        if exc.status != 404:
            LOGGER.warning("Failed to delete Workflow '%s': %s", workflow_name, exc.reason)


def teardown_and_delete(
    workflow_name: str,
    namespace: str,
    timeout: int = DEFAULT_TIMEOUT,
) -> None:
    """Best-effort: advance to Teardown, wait for completion, then delete.

    Always attempts the delete even if Teardown fails or times out.
    """
    LOGGER.info("Advancing workflow '%s' to Teardown...", workflow_name)
    try:
        advance(workflow_name, namespace, "Teardown")
    except kubernetes.client.exceptions.ApiException as exc:
        LOGGER.warning("Could not advance to Teardown: %s", exc.reason)
    else:
        ok, msg = wait_for_state(workflow_name, namespace, "Teardown", timeout)
        if not ok:
            LOGGER.warning("Teardown did not complete cleanly: %s", msg)

    LOGGER.info("Deleting workflow '%s'...", workflow_name)
    delete(workflow_name, namespace)


def run_to_completion(
    wf: WorkflowRun,
    timeout: int = DEFAULT_TIMEOUT,
) -> Tuple[bool, str]:
    """Advance a Workflow through every state from Proposal to Teardown.

    The caller is responsible for creating the Workflow (via ``wf.manifest``)
    before calling this function.  On success the Workflow is deleted.
    On failure, Teardown is attempted and the Workflow is deleted before
    returning.

    Transition hooks on ``wf`` are invoked at the following points:

    * ``wf.state_hooks[state]`` — list of hooks called in order after each
      state is ready, before advancing to the next.  Any hook returning
      ``(False, message)`` aborts the remaining hooks and the run.
    * ``wf.on_state_complete`` — list of callbacks invoked after every state
      (and its hooks).

    A hook returning ``(False, message)`` aborts the run; Teardown is triggered
    automatically.

    Returns
    -------
    ``(True, "")`` on success.
    ``(False, message)`` on the first failure encountered.
    """
    for state in WORKFLOW_STATES:
        if state != "Proposal":
            LOGGER.info("Advancing to %s...", state)
            try:
                advance(wf.name, wf.namespace, state)
            except kubernetes.client.exceptions.ApiException as exc:
                msg = f"Failed to advance to {state}: {exc.reason}"
                if state != "Teardown":
                    teardown_and_delete(wf.name, wf.namespace, timeout)
                else:
                    delete(wf.name, wf.namespace)
                return False, msg

        LOGGER.info("Waiting for %s...", state)
        ok, msg = wait_for_state(wf.name, wf.namespace, state, timeout)
        if not ok:
            if state != "Teardown":
                teardown_and_delete(wf.name, wf.namespace, timeout)
            else:
                delete(wf.name, wf.namespace)
            return False, msg

        LOGGER.info("%s complete.", state)

        hooks = wf.state_hooks.get(state, [])
        for i, hook in enumerate(hooks):
            LOGGER.info("Running post-%s hook (%s/%s)...", state, i + 1, len(hooks))
            ok, msg = hook(wf.name, wf.namespace)
            if not ok:
                teardown_and_delete(wf.name, wf.namespace, timeout)
                return False, msg

        for cb in wf.on_state_complete:
            cb(state)

    LOGGER.info("Deleting workflow '%s'...", wf.name)
    delete(wf.name, wf.namespace)
    return True, ""


# ---------------------------------------------------------------------------
# Servers / Computes population helpers
# ---------------------------------------------------------------------------

_ALLOC_PER_COMPUTE = "AllocatePerCompute"
_ALLOC_ACROSS = "AllocateAcrossServers"
_ALLOC_SINGLE = "AllocateSingleServer"

_BREAKDOWN_POLL_INTERVAL = 2.0  # seconds

_OST_LABEL = "ost"


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
