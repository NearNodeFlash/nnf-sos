#!/bin/bash

# Redirect stdout and stderr to syslog
exec 1> >(logger -s -t $(basename $0)) 2>&1

# ============================================================================
# Configuration
# ============================================================================
# Maximum number of attempts to start corosync and achieve quorum.
# On each retry, the cluster is stopped and restarted with a random delay
# to avoid synchronizing with other computes that may also be retrying.
MAX_ATTEMPTS=3

# Maximum time (seconds) to wait for the node to become Online in pcs.
# This covers corosync ring formation and pacemaker startup.
ONLINE_TIMEOUT=40

# Maximum time (seconds) to wait for DLM and lvmlockd to become active.
# These are pacemaker-managed resources that start after the node is Online.
LOCKING_TIMEOUT=30

# Maximum random delay (seconds) before starting corosync.
# Staggers compute node joins to reduce TOTEM membership storms when all
# computes on a rabbit start simultaneously during PreRun.
MAX_JITTER=4

# Define the command to check node status
PCS_STATUS_CMD="pcs status nodes"

# ============================================================================
# Diagnostic functions
# ============================================================================

# dump_diagnostics collects detailed cluster and network state for debugging.
# Called on timeout/failure so we have the data to diagnose the issue.
dump_diagnostics() {
    local reason="$1"
    echo "========================================================================"
    echo "DIAGNOSTIC DUMP — reason: $reason"
    echo "Timestamp: $(date -Iseconds)"
    echo "Hostname: $(hostname -s)"
    echo "========================================================================"

    echo "--- pcs status ---"
    pcs status 2>&1 || true

    echo "--- pcs status nodes ---"
    pcs status nodes 2>&1 || true

    echo "--- corosync-quorumtool ---"
    corosync-quorumtool 2>&1 || true

    echo "--- corosync-cfgtool -s (ring status) ---"
    corosync-cfgtool -s 2>&1 || true

    echo "--- corosync-cmapctl | grep members ---"
    corosync-cmapctl 2>/dev/null | grep -i "members\|quorum\|token\|ring" || true

    echo "--- dlm_tool ls ---"
    dlm_tool ls 2>&1 || true

    echo "--- pidof lvmlockd ---"
    pidof lvmlockd 2>&1 || echo "(not running)"

    echo "--- ip addr show (corosync interfaces) ---"
    # Show all non-loopback interfaces — the corosync interface will be among them
    ip addr show 2>&1 | grep -A3 "state UP\|state DOWN" || true

    echo "--- corosync.conf member list ---"
    grep -A1 "ring0_addr\|name:" /etc/corosync/corosync.conf 2>/dev/null || true

    echo "--- systemctl status corosync pacemaker ---"
    systemctl status corosync --no-pager -l 2>&1 | tail -20 || true
    systemctl status pacemaker --no-pager -l 2>&1 | tail -20 || true

    echo "--- last 50 lines of corosync journal ---"
    journalctl -u corosync --no-pager -n 50 2>&1 || true

    echo "========================================================================"
    echo "END DIAGNOSTIC DUMP"
    echo "========================================================================"
}

# check_quorum returns 0 if this node has quorum, 1 otherwise.
check_quorum() {
    corosync-quorumtool -s 2>/dev/null | grep -q "Quorate:.*Yes"
}

# ============================================================================
# Main logic
# ============================================================================

attempt=0
while [ $attempt -lt $MAX_ATTEMPTS ]; do
    attempt=$((attempt + 1))
    echo "============================================================"
    echo "Attempt $attempt of $MAX_ATTEMPTS"
    echo "============================================================"

    # 1. Stagger corosync start with a random delay to reduce membership storms.
    #    When all 16 computes on a rabbit start simultaneously, rapid TOTEM ring
    #    reconfigurations can cause some nodes to be partitioned. A small jitter
    #    spreads the joins over a few seconds.
    #    Skip jitter on attempt 1 if cluster is already running (e.g. leftover
    #    from a previous state that just needs unstandby).
    if pcs status > /dev/null 2>&1; then
        echo "Cluster is already running."
    else
        if [ $attempt -gt 1 ] || [ $MAX_JITTER -gt 0 ]; then
            jitter=$((RANDOM % (MAX_JITTER + 1)))
            echo "Waiting ${jitter}s before starting cluster (jitter to stagger joins)..."
            sleep $jitter
        fi

        echo "Starting cluster..."
        sudo pcs cluster start
        if [ $? -ne 0 ]; then
            # Check status again in case it started concurrently
            if ! pcs status > /dev/null 2>&1; then
                echo "Failed to start cluster on attempt $attempt."
                if [ $attempt -lt $MAX_ATTEMPTS ]; then
                    echo "Will retry after stopping cluster."
                    sudo pcs cluster stop --force 2>/dev/null || true
                    sleep 2
                    continue
                fi
                dump_diagnostics "pcs cluster start failed after $MAX_ATTEMPTS attempts"
                exit 1
            fi
        fi
        # Give corosync a moment to form the TOTEM ring
        sleep 5
    fi

    # 2. Ensure node is not in standby (may be leftover state from previous run)
    #    and wait for node to be Online.
    echo "Ensuring node is not in standby..."
    sudo pcs node unstandby 2>/dev/null || true

    echo "Waiting for node to become Online (timeout: ${ONLINE_TIMEOUT}s)..."
    online=false
    elapsed=0
    while [ $elapsed -lt $ONLINE_TIMEOUT ]; do
        nodes_out=$($PCS_STATUS_CMD 2>&1)

        # Log status periodically (every 10 seconds) to avoid log spam
        if [ $((elapsed % 10)) -eq 0 ]; then
            echo "Current node status (elapsed: ${elapsed}s):"
            echo "$nodes_out"

            # Also check quorum periodically
            echo "Quorum status:"
            corosync-quorumtool -s 2>&1 | head -10 || true
        fi

        if echo "$nodes_out" | grep -qE "^[[:space:]]*Online:.*[[:space:]]e?$(hostname -s)([[:space:]]|$)"; then
            echo "Node is Online."
            online=true
            break
        fi

        # If node is in standby, try to unstandby it
        if echo "$nodes_out" | grep -qE "^[[:space:]]*Standby:.*[[:space:]]e?$(hostname -s)([[:space:]]|$)"; then
            echo "Node is in Standby, attempting to unstandby..."
            sudo pcs node unstandby 2>/dev/null || true
        fi

        # Early detection: if we've been waiting 30+ seconds and don't have
        # quorum, there's likely a connectivity issue. Log it so we can
        # diagnose, and break out to retry sooner.
        if [ $elapsed -ge 30 ] && [ $((elapsed % 30)) -eq 0 ]; then
            if ! check_quorum; then
                echo "WARNING: No quorum after ${elapsed}s — possible partition."
                echo "Quorum details:"
                corosync-quorumtool 2>&1 || true
                echo "Ring status:"
                corosync-cfgtool -s 2>&1 || true

                if [ $attempt -lt $MAX_ATTEMPTS ]; then
                    echo "Stopping cluster to retry with new jitter..."
                    sudo pcs cluster stop --force 2>/dev/null || true
                    sleep 2
                    break
                fi
            fi
        fi

        sleep 1
        elapsed=$((elapsed + 1))
    done

    # If we broke out of the loop to retry, continue to next attempt
    if [ "$online" != "true" ] && [ $attempt -lt $MAX_ATTEMPTS ]; then
        # Check if cluster was already stopped for retry
        if ! pcs status > /dev/null 2>&1; then
            echo "Cluster stopped for retry."
            continue
        fi
        echo "Timed out waiting for node to become Online on attempt $attempt."
        echo "Stopping cluster to retry..."
        dump_diagnostics "Online timeout on attempt $attempt"
        sudo pcs cluster stop --force 2>/dev/null || true
        sleep 2
        continue
    fi

    if [ "$online" != "true" ]; then
        echo "Timed out waiting for node to become Online after $MAX_ATTEMPTS attempts."
        dump_diagnostics "Online timeout after all attempts"
        exit 1
    fi

    # 3. Wait for Locking Services (DLM & lvmlockd)
    # Even if the node is "Online", the locking services might still be starting up.
    # These are pacemaker-managed resources that depend on quorum.
    echo "Waiting for locking services (dlm_controld, lvmlockd) to be active (timeout: ${LOCKING_TIMEOUT}s)..."
    locking_ok=false
    elapsed=0
    while [ $elapsed -lt $LOCKING_TIMEOUT ]; do
        # Check for dlm_controld (userspace DLM controller)
        # dlm_tool ls returns 0 if it can contact dlm_controld
        dlm_ok=false
        if dlm_tool ls > /dev/null 2>&1; then
            dlm_ok=true
        fi

        # Check for lvmlockd
        lvm_ok=false
        if pidof lvmlockd > /dev/null 2>&1; then
            lvm_ok=true
        fi

        if [ "$dlm_ok" = true ] && [ "$lvm_ok" = true ]; then
            echo "DLM and lvmlockd are active."
            locking_ok=true
            break
        fi

        # Log status periodically
        if [ $((elapsed % 10)) -eq 0 ]; then
            echo "Locking services status (elapsed: ${elapsed}s): DLM_OK=$dlm_ok LVM_OK=$lvm_ok"
            pcs status resources 2>&1 | head -20
        fi

        sleep 1
        elapsed=$((elapsed + 1))
    done

    if [ "$locking_ok" = true ]; then
        echo "preActivate script completed successfully."
        exit 0
    fi

    echo "Timed out waiting for locking services on attempt $attempt."
    echo "DLM OK: $dlm_ok"
    echo "LVM OK: $lvm_ok"

    if [ $attempt -lt $MAX_ATTEMPTS ]; then
        dump_diagnostics "Locking services timeout on attempt $attempt"
        echo "Stopping cluster to retry..."
        sudo pcs cluster stop --force 2>/dev/null || true
        sleep 2
        continue
    fi

    dump_diagnostics "Locking services timeout after all attempts"
    exit 1
done

# Should not reach here, but just in case
echo "preActivate script failed after $MAX_ATTEMPTS attempts."
dump_diagnostics "Exhausted all attempts"
exit 1
