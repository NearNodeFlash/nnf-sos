#!/bin/bash

# dlm-diagnostics.sh — Post-workflow DLM state diagnostic for rabbits
#
# Run this script on a rabbit node after a GFS2 workflow completes (PostRun
# teardown finished). It checks for stale DLM artifacts that could cause a
# compute node's dlm_controld to kill corosync on the rabbit during the
# next workflow.
#
# Expected clean state after workflow teardown:
#   - Only the rabbit should be in corosync membership
#   - lvm_global should be the only DLM lockspace
#   - No nodes should be marked "needs fencing" in dlm_controld
#   - No per-VG DLM lockspaces should remain
#
# Exit codes:
#   0 = clean — no stale DLM state detected (or --no-fail used)
#   1 = WARNINGS found — stale state that may cause next workflow to fail
#   2 = script error (missing tools, etc.)
#
# Use --no-fail when running as a storage profile callout so warnings
# are logged but do not block the workflow.

usage() {
    echo "Usage: $(basename "$0") [-h|--help] [--no-fail]"
    echo ""
    echo "Post-workflow DLM state diagnostic for rabbit nodes."
    echo "Run after GFS2 workflow teardown to detect stale DLM artifacts"
    echo "that could cause compute nodes to kill corosync on the rabbit."
    echo ""

    echo "Options:"
    echo "  --no-fail  Always exit 0, even if warnings are found."
    echo "             Use this in storage profile callouts so diagnostics"
    echo "             never block the workflow."
    echo ""
    echo "Exit codes:"
    echo "  0  Clean — no stale DLM state detected (or --no-fail)"
    echo "  1  Warnings found — stale state detected"
    echo "  2  Script error (missing tools, etc.)"
    exit 0
}

NO_FAIL=false
for arg in "$@"; do
    case "$arg" in
        -h|--help) usage ;;
        --no-fail) NO_FAIL=true ;;
    esac
done

set -o pipefail
set -u

# In --no-fail mode, suppress all verbose output and only show warnings.
# Save original stdout to fd 3 so warn() can still produce output.
if [ "$NO_FAIL" = true ]; then
    exec 3>&1 1>/dev/null
else
    exec 3>&1
fi

HOSTNAME_SHORT=$(hostname -s)
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

# Run a command with a timeout.  Returns 0 on timeout (so || true patterns
# in callers stay clean) and logs a warning so operators know the command
# was skipped.
timed() {
    local output
    output=$(timeout "$CMD_TIMEOUT" "$@" 2>&1) && { echo "$output"; return 0; }
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
    if ! command -v "$tool" > /dev/null 2>&1; then
        log "ERROR: Required tool '$tool' not found in PATH"
        MISSING_TOOLS=1
    fi
done

if [ "$MISSING_TOOLS" -eq 1 ]; then
    log "Cannot proceed without required tools."
    exit 2
fi

log "All required tools found."
log "Hostname: $HOSTNAME_SHORT"
log "Date: $(date)"

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

if command -v lvmlockctl > /dev/null 2>&1; then
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

if [ -f /etc/corosync/corosync.conf ]; then
    log "Current corosync.conf:"
    cat /etc/corosync/corosync.conf
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
    echo "" >&3
    if [ "$NO_FAIL" = true ]; then
        echo "[$(date '+%Y-%m-%d %H:%M:%S')] --no-fail: exiting 0 despite warnings (advisory mode)" >&3
        exit 0
    fi
    exit 1
fi
