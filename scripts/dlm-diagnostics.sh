#!/bin/bash

# dlm-diagnostics.sh — Post-workflow DLM state diagnostic for rabbits
#
# Run this script on a rabbit node after a GFS2 workflow completes (PostRun
# teardown finished). It checks for stale DLM artifacts that could cause a
# compute node's dlm_controld to kill corosync on the rabbit during the
# next workflow.
#
# Two modes of operation:
#
#   Per-VG mode (VG_NAME argument given):
#     Only checks that the named VG's lockspace was properly released from
#     lvmlockd and DLM.  Other VGs' lockspaces (from sibling allocations
#     still being torn down) are expected and ignored.  Use this when the
#     script is called as a PostTeardown callout — the controller substitutes
#     $VG_NAME automatically.
#
#   Global mode (no VG_NAME):
#     Full diagnostic — checks corosync membership, pacemaker, all DLM
#     lockspaces, fence status, lvmlockd, and kernel state.  Expects only
#     lvm_global to be active.  Use for manual post-workflow verification.
#
# Expected clean state after workflow teardown (global mode):
#   - Only the rabbit should be in corosync membership
#   - lvm_global should be the only DLM lockspace
#   - No nodes should be marked "needs fencing" in dlm_controld
#   - No per-VG DLM lockspaces should remain
#
# When run as a storage profile postTeardown callout, a non-zero exit
# causes the workflow to fail (.WithMajor).  All output is saved to
# /var/log/nnf-dlm-diagnostics.log on the rabbit for post-mortem.
#
# Container support:
#   The script auto-detects whether it is running inside a container
#   (e.g., nnf-node-manager pod with hostPID: true) by comparing mount
#   namespaces.  When in a container, host-side tools (dlm_tool, pcs,
#   corosync-cmapctl, lvmlockctl) are invoked via nsenter.  No flags or
#   separate scripts are needed — it works the same on the host or in
#   a pod.
#
# Exit codes:
#   0 = clean — no stale DLM state detected (or --no-fail used)
#   1 = WARNINGS found — stale state that may cause next workflow to fail
#   2 = script error (missing tools, etc.)
#
# Use --no-fail for advisory-only manual runs where you do not want a
# non-zero exit code.

usage() {
    echo "Usage: $(basename "$0") [-h|--help] [--no-fail] [VG_NAME]"
    echo ""
    echo "Post-workflow DLM state diagnostic for rabbit nodes."
    echo "Run after GFS2 workflow teardown to detect stale DLM artifacts"
    echo "that could cause compute nodes to kill corosync on the rabbit."
    echo ""
    echo "Arguments:"
    echo "  VG_NAME    Optional LVM volume group name.  When provided, the"
    echo "             script runs in per-VG mode: it only checks whether"
    echo "             that specific VG's lockspace is properly cleaned up"
    echo "             from lvmlockd and DLM, skipping global cluster checks."
    echo "             Use this when the script is called as a PostTeardown"
    echo "             callout (\$VG_NAME is substituted by the controller)."
    echo ""
    echo "Options:"
    echo "  --no-fail  Always exit 0, even if warnings are found."
    echo "             Use for advisory-only manual runs."
    echo ""
    echo "Modes:"
    echo "  Per-VG mode (VG_NAME given):"
    echo "    Only checks that the named VG's lockspace was removed from"
    echo "    lvmlockd and DLM after block device teardown.  Other VGs'"
    echo "    lockspaces (from sibling allocations) are ignored."
    echo ""
    echo "  Global mode (no VG_NAME):"
    echo "    Full diagnostic — checks corosync membership, pacemaker,"
    echo "    DLM lockspaces, fence status, lvmlockd, and kernel state."
    echo ""
    echo "Exit codes:"
    echo "  0  Clean — no stale DLM state detected (or --no-fail)"
    echo "  1  Warnings found — stale state detected"
    echo "  2  Script error (missing tools, etc.)"
    exit 0
}

NO_FAIL=false
VG_NAME=""
for arg in "$@"; do
    case "$arg" in
        -h|--help) usage ;;
        --no-fail) NO_FAIL=true ;;
        -*) echo "Unknown option: $arg" >&2; exit 2 ;;
        *) VG_NAME="$arg" ;;
    esac
done

# ---------------------------------------------------------------------------
# Container auto-detection
# ---------------------------------------------------------------------------
# When running inside a Kubernetes container (e.g., nnf-node-manager pod
# with hostPID: true), host-side tools like dlm_tool, pcs, corosync-cmapctl,
# and lvmlockctl live in the host mount namespace, not the container's.
# Detect this by comparing our mount namespace to PID 1 (host init).
# If they differ, we prefix host commands with nsenter to enter the
# host's mount, IPC, and network namespaces.  IPC is needed for
# corosync-cmapctl (shared memory) and pcs/crm_mon.  Network is
# needed for dlm_tool (abstract Unix sockets).
# ---------------------------------------------------------------------------
NSENTER=""
if [ -r /proc/1/ns/mnt ] && [ "$(readlink /proc/self/ns/mnt)" != "$(readlink /proc/1/ns/mnt)" ]; then
    NSENTER="nsenter -t 1 -m -i -n --"
    # Log file must be written to the host filesystem via nsenter.
    TEE_CMD="$NSENTER tee"
else
    TEE_CMD="tee"
fi

# Save full output to a persistent log file for post-mortem inspection.
# In normal mode, tee mirrors everything to both stdout and the log file.
# In --no-fail mode, the log is not written (advisory-only manual runs).
LOG_FILE="/var/log/nnf-dlm-diagnostics.log"
if [ -z "${_DLM_DIAG_WRAPPED:-}" ] && [ "$NO_FAIL" = false ]; then
    export _DLM_DIAG_WRAPPED=1
    export NSENTER
    "$0" "$@" 2>&1 | $TEE_CMD "$LOG_FILE"
    exit "${PIPESTATUS[0]}"
fi

set -o pipefail
set -u

# In --no-fail mode, suppress all verbose output and only show warnings.
# Save original stdout to fd 3 so warn() can still produce output.
if [ "$NO_FAIL" = true ]; then
    exec 3>&1 1>/dev/null
else
    exec 3>&1
fi

HOSTNAME_SHORT=$($NSENTER hostname -s)
WARNINGS=0
WARN_MESSAGES=()
DIVIDER="================================================================"
WARN_DIVIDER="################################################################"

# ---------------------------------------------------------------------------
# Timeout for daemon-communicating commands.
# Commands like lvmlockctl, dlm_tool, pcs, corosync-cmapctl talk to daemons
# via Unix sockets and can block indefinitely if the daemon is busy or hung.
# When run as a storage profile callout, a hang blocks the controller from
# reconciling any further resources, so we enforce a per-command timeout.
# ---------------------------------------------------------------------------
CMD_TIMEOUT=10

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*"
}

warn() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] >>> WARNING: $*" >&3
    WARN_MESSAGES+=("$*")
    WARNINGS=$((WARNINGS + 1))
}

header() {
    echo ""
    echo "$DIVIDER"
    echo "  $1"
    echo "$DIVIDER"
}

# Run a host command with a timeout.  Automatically prefixes with nsenter
# when running inside a container.  Returns 0 on timeout (so || true
# patterns in callers stay clean) and logs a warning so operators know the
# command was skipped.
timed() {
    local output
    output=$(timeout "$CMD_TIMEOUT" $NSENTER "$@" 2>&1) && { echo "$output"; return 0; }
    local rc=$?
    if [ "$rc" -eq 124 ]; then
        log "TIMEOUT: '$*' did not complete within ${CMD_TIMEOUT}s — skipping"
    else
        echo "$output"
    fi
    return "$rc"
}

# ---------------------------------------------------------------------------
# Preflight: verify required tools exist
# ---------------------------------------------------------------------------
header "Preflight Checks"

MISSING_TOOLS=0
for tool in dlm_tool corosync-cmapctl pcs lvmlockctl; do
    if ! $NSENTER command -v "$tool" > /dev/null 2>&1; then
        log "ERROR: Required tool '$tool' not found in PATH"
        MISSING_TOOLS=1
    fi
done

if [ "$MISSING_TOOLS" -eq 1 ]; then
    log "Cannot proceed without required tools."
    exit 2
fi

if [ -n "$NSENTER" ]; then
    log "Running in container mode (nsenter into host mount namespace)"
fi

log "All required tools found."
log "Hostname: $HOSTNAME_SHORT"
log "Date: $(date)"

# ---------------------------------------------------------------------------
# Per-VG mode: when VG_NAME is given, only check that VG's lockspace
# ---------------------------------------------------------------------------
# PostTeardown runs per-allocation inside deleteAllocation(), AFTER the
# block device is destroyed.  With multiple allocations on the same rabbit,
# sibling allocations' VGs may still be active.  In per-VG mode we only
# check that THIS allocation's VG lockspace was properly released from
# lvmlockd and DLM — other VGs are expected and ignored.
# ---------------------------------------------------------------------------
if [ -n "$VG_NAME" ]; then
    header "Per-VG Teardown Check: $VG_NAME"
    log "Running in per-VG mode — checking only lockspace for VG '$VG_NAME'."

    # Check lvmlockctl for our VG.  After Destroy(), the VG should be gone
    # from lvmlockd.  If it's still there, check whether the VG still has
    # LVs from sibling allocations.  For shared GFS2 VGs, Destroy() only
    # removes this allocation's LV; the VG (and its lockspace) stays until
    # the last LV is removed.  That's expected, not a warning.
    LVMLOCKCTL=$(timed lvmlockctl -i 2>&1 || true)

    # lvmlockctl -i output format:
    #   VG <vg_name> lock_type=dlm vg_lock_args=<version>:<vg_uuid>
    #     LV <lv_name> lock_args=...
    VG_LOCK_LINE=$(echo "$LVMLOCKCTL" | grep -E "^VG $VG_NAME " || true)

    if [ -n "$VG_LOCK_LINE" ]; then
        # VG lockspace still active — check if there are remaining LVs.
        # If sibling allocations still have LVs, the VG is expected to
        # stay until their Destroy() runs.
        LV_COUNT=$(timed lvs --noheadings --nosuffix -o lv_name "$VG_NAME" 2>/dev/null | wc -l || echo 0)
        LV_COUNT=$(echo "$LV_COUNT" | tr -d ' ')

        if [ "$LV_COUNT" -gt 0 ]; then
            log "VG '$VG_NAME' lockspace still active, but $LV_COUNT LV(s) remain from sibling allocations — expected."
            log "The lockspace will be released when the last allocation is torn down."
        else
            warn "VG '$VG_NAME' still has an active lockspace in lvmlockd after teardown (0 LVs remaining):"
            echo "  $VG_LOCK_LINE"

            # Extract the VG UUID from vg_lock_args to check DLM directly.
            # Format: vg_lock_args=<version>:<uuid>  e.g. vg_lock_args=1.0.0:AbCd1234...
            VG_UUID=$(echo "$VG_LOCK_LINE" | sed -n 's/.*vg_lock_args=[^:]*:\([^ ]*\).*/\1/p')
            if [ -n "$VG_UUID" ]; then
                DLM_LS_NAME="lvm_${VG_UUID}"
                DLM_LS=$(timed dlm_tool ls 2>&1 || true)
                if echo "$DLM_LS" | grep -q "^name.*${DLM_LS_NAME}"; then
                    warn "DLM lockspace '$DLM_LS_NAME' still active for VG '$VG_NAME'."
                else
                    log "DLM lockspace for VG UUID not found in dlm_tool ls (may have been partially cleaned up)."
                fi
            fi
        fi
    else
        log "OK — VG '$VG_NAME' lockspace properly cleaned up from lvmlockd."

        # Double-check: see if any DLM lockspace references our VG name.
        # This shouldn't happen if lvmlockctl is clean, but catches edge cases.
        DLM_LS=$(timed dlm_tool ls 2>&1 || true)
        LS_COUNT=$(echo "$DLM_LS" | grep -c "^name" || true)
        log "Active DLM lockspaces: $LS_COUNT"
        if [ "$LS_COUNT" -gt 0 ]; then
            echo "$DLM_LS" | grep "^name" || true
        fi
    fi

    # --- Per-VG Summary ---
    header "SUMMARY (per-VG: $VG_NAME)"

    if [ "$WARNINGS" -eq 0 ]; then
        echo "[$(date '+%Y-%m-%d %H:%M:%S')] CLEAN — VG '$VG_NAME' lockspace properly cleaned up on $HOSTNAME_SHORT." >&3
        exit 0
    else
        echo "" >&3
        echo "$WARN_DIVIDER" >&3
        echo "  FOUND $WARNINGS WARNING(S) FOR VG '$VG_NAME' ON $HOSTNAME_SHORT" >&3
        echo "$WARN_DIVIDER" >&3
        for i in "${!WARN_MESSAGES[@]}"; do
            echo "  $((i+1)). ${WARN_MESSAGES[$i]}" >&3
        done
        echo "$WARN_DIVIDER" >&3
        echo "  Full diagnostics: $LOG_FILE on $HOSTNAME_SHORT" >&3
        echo "" >&3
        if [ "$NO_FAIL" = true ]; then
            echo "[$(date '+%Y-%m-%d %H:%M:%S')] --no-fail: exiting 0 despite warnings (advisory mode)" >&3
            exit 0
        fi
        exit 1
    fi
fi

# ===========================================================================
# Global mode: full cluster diagnostic (no VG_NAME given)
# ===========================================================================

# ---------------------------------------------------------------------------
# 1. Corosync Membership — only the rabbit should be present
# ---------------------------------------------------------------------------
header "1. Corosync Membership"

log "Checking corosync ring membership..."
COROSYNC_MEMBERS=$(timed corosync-cmapctl 2>/dev/null | grep "runtime.members\." || true)

if [ -z "$COROSYNC_MEMBERS" ]; then
    log "No corosync membership info available (corosync may not be running)."
else
    JOINED_COUNT=$(echo "$COROSYNC_MEMBERS" | grep -c 'status (str) = joined' || true)
    log "Corosync joined members: $JOINED_COUNT"
    echo "$COROSYNC_MEMBERS"

    if [ "$JOINED_COUNT" -gt 1 ]; then
        warn "Expected only 1 member (rabbit) but found $JOINED_COUNT joined."
        warn "Compute nodes may still be in corosync — stale membership."
    else
        log "OK — only 1 member (rabbit) in corosync."
    fi
fi

echo ""
log "Corosync quorum status:"
timed corosync-quorumtool 2>&1 || true

# ---------------------------------------------------------------------------
# 2. Pacemaker Node Status — look for UNCLEAN or stuck Standby nodes
# ---------------------------------------------------------------------------
header "2. Pacemaker Node Status"

PCS_NODES=$(timed pcs status nodes 2>&1 || true)
log "Current pacemaker node status:"
echo "$PCS_NODES"

# Check for UNCLEAN nodes — these have an unexpected departure that pacemaker
# wants to fence before recovering resources.
UNCLEAN_NODES=$(echo "$PCS_NODES" | grep -i "UNCLEAN" || true)
if [ -n "$UNCLEAN_NODES" ]; then
    warn "UNCLEAN node(s) in pacemaker — pending fence operation: $UNCLEAN_NODES"
else
    log "OK — no UNCLEAN nodes in pacemaker."
fi

# Check for 'Standby with resource(s) running' — resources didn't drain before
# the cluster stopped, which can leave GFS2/DLM in a bad state.
# pcs always prints the category header line even when empty, so check for
# actual node names (non-whitespace) after the colon.
STANDBY_RES=$(echo "$PCS_NODES" | grep -i "Standby with resource" | grep -v ":$" | grep -v ":[[:space:]]*$" || true)
if [ -n "$STANDBY_RES" ]; then
    warn "Node(s) in Standby with resources still running — DLM/GFS2 may not have shut down cleanly: $STANDBY_RES"
else
    log "OK — no nodes stuck in Standby-with-resources."
fi

# ---------------------------------------------------------------------------
# 3. DLM Lockspaces — only lvm_global should remain
# ---------------------------------------------------------------------------
header "3. DLM Lockspaces"

log "Active DLM lockspaces (dlm_tool ls):"
DLM_LS=$(timed dlm_tool ls 2>&1 || true)
echo "$DLM_LS"

# Each lockspace entry starts with a "name" line; count those.
LS_COUNT=$(echo "$DLM_LS" | grep -c "^name" || true)
log "Lockspace count: $LS_COUNT"

# 0 lockspaces is normal in idle state between workflows (DLM is not joined).
# 1 lockspace named lvm_global is also normal when lvmlockd is running on a
# rabbit that is holding quorum but has no active workflow.
# Any other lockspace indicates a GFS2 filesystem was not unmounted cleanly.
if [ "$LS_COUNT" -eq 0 ]; then
    log "OK — no DLM lockspaces active (idle state)."
elif [ "$LS_COUNT" -eq 1 ]; then
    if echo "$DLM_LS" | grep -q "^name.*lvm_global"; then
        log "OK — only lvm_global lockspace is active."
    else
        EXTRA_NAME=$(echo "$DLM_LS" | awk '/^name/{print $2}')
        warn "Unexpected lockspace active (not lvm_global): $EXTRA_NAME"
    fi
else
    EXTRA_LS=$(echo "$DLM_LS" | awk '/^name/{print $2}' | grep -v "^lvm_global$" || true)
    if [ -n "$EXTRA_LS" ]; then
        warn "Per-VG GFS2 lockspaces still active after teardown: $EXTRA_LS"
        warn "These should have been removed when GFS2 was unmounted during postDeactivate."
    else
        log "OK — $LS_COUNT lockspaces active (lvm_global only or expected)."
    fi
fi

# Show lockspace membership from the already-captured output
MEMBER_LINES=$(echo "$DLM_LS" | grep -i "members" || true)
if [ -n "$MEMBER_LINES" ]; then
    log "Lockspace membership:"
    echo "$MEMBER_LINES"
fi

# ---------------------------------------------------------------------------
# 4. DLM Fence Status — no nodes should need fencing
# ---------------------------------------------------------------------------
header "4. DLM Fence Status"

log "dlm_controld status:"
DLM_STATUS=$(timed dlm_tool status 2>&1 || true)
echo "$DLM_STATUS"

# Check for nodes needing fencing
NEEDS_FENCING=$(echo "$DLM_STATUS" | grep -i "needs fencing\|wait fencing\|need_fencing" || true)
if [ -n "$NEEDS_FENCING" ]; then
    warn "Nodes marked as needing fencing in dlm_controld:"
    echo "$NEEDS_FENCING"
    warn "These stale fence records WILL cause dlm_controld to kill"
    warn "the rabbit when these nodes rejoin in the next workflow."
    warn "Fix: run 'dlm_tool fence_ack <nodeid>' for each."
else
    log "OK — no nodes need fencing."
fi

# ---------------------------------------------------------------------------
# 5. DLM Internal State Dump
# ---------------------------------------------------------------------------
header "5. DLM Internal State Dump"

log "dlm_controld internal state (dlm_tool dump):"
DLM_DUMP=$(timed dlm_tool dump 2>&1 || true)
echo "$DLM_DUMP"

# Get the rabbit's own nodeid from the already-captured dlm_tool status output
RABBIT_NODEID=$(echo "$DLM_STATUS" | awk '/^cluster nodeid/{print $3}' || true)
log "Rabbit nodeid: ${RABBIT_NODEID:-(unknown)}"

# Look for concerning patterns in the dump:
# - stateful merge detections (always bad)
# - nodes that left with need_fencing 1 (required fencing but may not have been acked)
# NOTE: "daemon remove N leave need_fencing 0" is a CLEAN departure — not a problem.
MERGE_DUMP=$(echo "$DLM_DUMP" | grep -i "stateful merge\|kill due" || true)
if [ -n "$MERGE_DUMP" ]; then
    warn "dlm_controld dump contains stateful merge or kill events:"
    echo "$MERGE_DUMP"
fi

# Check the dump for historical records of nodes that departed requiring fencing.
# On systems running the master branch DLM fence-ack code, these are always
# auto-acked within milliseconds and only appear as historical records.
# The authoritative live fence state is Section 4 (dlm_tool status).
HISTORICAL_FENCES=$(echo "$DLM_DUMP" | grep -E "daemon remove [0-9]+ .* need_fencing [1-9]" || true)
if [ -n "$HISTORICAL_FENCES" ]; then
    log "NOTE: dlm_controld dump has historical records of nodes that left requiring fencing:"
    echo "$HISTORICAL_FENCES"
    log "This is past history from crashes/unclean departures. It does not block"
    log "future joins when no active GFS2 lockspaces are present. See Section 4"
    log "(dlm_tool status) for the authoritative active fence state."
else
    log "OK — no unacknowledged fence departures in dump."
fi

# Look for 'daemon joined <N>' without a matching 'daemon remove <N>' — node
# joined DLM but never cleanly left. Only warn if the node is also NOT an
# active corosync member (active members are expected to have no remove record).
if [ -n "$RABBIT_NODEID" ]; then
    # Get only JOINED member indices — departed nodes stay in corosync-cmapctl
    # with status=left and must not suppress stale-member warnings.
    CURRENT_MEMBERS=$(timed corosync-cmapctl 2>/dev/null | awk -F. '/runtime\.members\.[0-9]+\.status.*= joined/{print $3}' | sort -u || true)
    FOREIGN_JOINED=$(echo "$DLM_DUMP" | grep -E "daemon joined [0-9]+" | grep -v "daemon joined ${RABBIT_NODEID}" | awk '{print $NF}' | sort -u || true)
    STALE_COUNT=0
    if [ -n "$FOREIGN_JOINED" ]; then
        for nodeid in $FOREIGN_JOINED; do
            REMOVE_LINE=$(echo "$DLM_DUMP" | grep -E "daemon remove ${nodeid} " || true)
            if [ -z "$REMOVE_LINE" ]; then
                # Use word matching to ensure we don't match partial IDs
                if echo "$CURRENT_MEMBERS" | grep -qw "$nodeid"; then
                    log "Node $nodeid joined DLM and is an active corosync member (workflow in progress — expected)."
                else
                    warn "Node $nodeid joined DLM but has no remove record and is NOT in corosync — may be a stale member."
                    STALE_COUNT=$((STALE_COUNT + 1))
                fi
            fi
        done
        
        if [ "$STALE_COUNT" -eq 0 ]; then
            log "OK — all foreign nodes in dump have clean remove records (normal workflow history)."
        fi
    else
        log "OK — no foreign node join records found in dump."
    fi
fi

# ---------------------------------------------------------------------------
# 6. lvmlockd Status — check for lingering lock state
# ---------------------------------------------------------------------------
header "6. lvmlockd Status"

if $NSENTER command -v lvmlockctl > /dev/null 2>&1; then
    log "lvmlockd lock info (lvmlockctl -i):"
    LVMLOCKCTL=$(timed lvmlockctl -i 2>&1 || true)
    echo "$LVMLOCKCTL"

    # Count VG lockspaces beyond lvm_global — only flag real DLM-locked VGs
    VG_LOCKS=$(echo "$LVMLOCKCTL" | grep -E "^VG .+ lock_type=dlm" | grep -v "lvm_global" || true)
    if [ -n "$VG_LOCKS" ]; then
        warn "lvmlockd has active VG lockspaces beyond lvm_global:"
        echo "$VG_LOCKS"
    else
        log "OK — no stale VG lockspaces in lvmlockd."
    fi
else
    log "lvmlockctl not found — skipping lvmlockd check."
fi

# ---------------------------------------------------------------------------
# 7. DLM Kernel State — check /sys/kernel/config/dlm
# ---------------------------------------------------------------------------
header "7. DLM Kernel Configuration"

DLM_CLUSTER_DIR="/sys/kernel/config/dlm/cluster"
if [ -d "$DLM_CLUSTER_DIR" ]; then
    log "DLM kernel cluster settings:"
    for param in recover_timer toss_secs scan_secs protocol; do
        if [ -f "$DLM_CLUSTER_DIR/$param" ]; then
            VAL=$(cat "$DLM_CLUSTER_DIR/$param" 2>/dev/null)
            log "  $param = $VAL"
        fi
    done
else
    log "DLM cluster config directory not found."
fi

# Check for active DLM lockspaces in /sys
DLM_LS_DIR="/sys/kernel/dlm"
if [ -d "$DLM_LS_DIR" ]; then
    log "Kernel DLM lockspaces in /sys/kernel/dlm/:"
    ls -la "$DLM_LS_DIR" 2>/dev/null || true
fi

# ---------------------------------------------------------------------------
# 8. Corosync Configuration — record for reference
# ---------------------------------------------------------------------------
header "8. Corosync Configuration (for reference)"

if $NSENTER test -f /etc/corosync/corosync.conf; then
    log "Current corosync.conf:"
    $NSENTER cat /etc/corosync/corosync.conf
else
    log "No corosync.conf found."
fi

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------
header "SUMMARY"

if [ "$WARNINGS" -eq 0 ]; then
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] CLEAN — No stale DLM state detected on $HOSTNAME_SHORT." >&3
    exit 0
else
    echo "" >&3
    echo "$WARN_DIVIDER" >&3
    echo "  FOUND $WARNINGS WARNING(S) ON $HOSTNAME_SHORT" >&3
    echo "$WARN_DIVIDER" >&3
    for i in "${!WARN_MESSAGES[@]}"; do
        echo "  $((i+1)). ${WARN_MESSAGES[$i]}" >&3
    done
    echo "$WARN_DIVIDER" >&3
    echo "  Full diagnostics: $LOG_FILE on $HOSTNAME_SHORT" >&3
    echo "" >&3
    if [ "$NO_FAIL" = true ]; then
        echo "[$(date '+%Y-%m-%d %H:%M:%S')] --no-fail: exiting 0 despite warnings (advisory mode)" >&3
        exit 0
    fi
    exit 1
fi
