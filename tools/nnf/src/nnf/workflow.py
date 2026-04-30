"""Shared helpers for creating, driving, and populating DWS Workflow resources.

These utilities are used by any sub-command that needs to create a Workflow,
advance it through the full state machine, populate its Servers/Computes
resources at the appropriate states, and clean up when done."""

import logging
import sys
import time
from typing import Any, Callable, Dict, List, Optional, Tuple

import kubernetes.client.exceptions  # type: ignore[import-untyped]

from nnf import crd
from nnf import k8s

# Hook type: receives (workflow_name, namespace) injected by run_to_completion,
# returns (ok, message).
StateHook = Callable[[str, str], Tuple[bool, str]]


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
    """

    def __init__(
        self,
        name: str,
        namespace: str,
        user_id: int,
        group_id: int = 0,
        dw_directives: Optional[List[str]] = None,
        state_hooks: Optional[Dict[str, List[StateHook]]] = None,
    ) -> None:
        self.name = name
        self.namespace = namespace
        self.user_id = user_id
        self.group_id = group_id
        self.dw_directives = dw_directives if dw_directives is not None else []
        self.state_hooks = state_hooks if state_hooks is not None else {}

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
            if exc.status is not None and (exc.status >= 500 or exc.status == 429):
                LOGGER.info("Transient API error (%s), retrying...", exc.status)
                time.sleep(POLL_INTERVAL)
                continue
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

    LOGGER.info("Deleting workflow '%s'...", wf.name)
    delete(wf.name, wf.namespace)
    return True, ""


def create_and_run(wf: WorkflowRun, timeout: int) -> int:
    """Create a Workflow resource and drive it to completion.

    Handles the standard create→run→delete sequence shared by all subcommands.
    Returns 0 on success, 2 on any handled failure.  Re-raises unexpected
    exceptions after triggering teardown.
    """
    try:
        k8s.create_object(
            group=crd.DWS_GROUP,
            version=crd.DWS_VERSION,
            namespace=wf.namespace,
            plural=crd.DWS_WORKFLOW_PLURAL,
            body=wf.manifest,
        )
    except kubernetes.client.exceptions.ApiException as exc:
        print(f"error: failed to create Workflow: {exc.reason} (HTTP {exc.status})", file=sys.stderr)
        LOGGER.info("Full error body: %s", exc.body)
        k8s.debug_api_group(crd.DWS_GROUP)
        return 2

    print(f"Workflow '{wf.name}' created.")

    try:
        ok, msg = run_to_completion(wf, timeout)
        if not ok:
            print(f"error: {msg}", file=sys.stderr)
            return 2
    except Exception:
        teardown_and_delete(wf.name, wf.namespace, timeout)
        raise

    return 0

