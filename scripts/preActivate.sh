#!/bin/bash

# Redirect stdout and stderr to syslog
exec 1> >(logger -s -t $(basename $0)) 2>&1

# 1. Ensure Cluster is running
echo "Checking cluster status..."
if pcs status; then
    echo "Cluster is already running."
else
    sudo pcs cluster start
    if [ $? -ne 0 ]; then
        # Check status again to be sure, in case it started concurrently
        if ! pcs status; then
            echo "Failed to start cluster."
            exit 1
        fi
    fi
    # Give it a moment to stabilize
    sleep 5
fi

# Define the command to check node status
PCS_STATUS_CMD="pcs status nodes"

# 2. Ensure node is not in standby (may be leftover state from previous run)
# and wait for node to be Online
echo "Ensuring node is not in standby..."
sudo pcs node unstandby 2>/dev/null || true

echo "Waiting for node to become Online..."
timeout=120
elapsed=0
while true; do
    nodes_out=$($PCS_STATUS_CMD 2>&1)
    echo "Current node status (elapsed: $elapsed):"
    echo "$nodes_out"

    if echo "$nodes_out" | grep -qE "^[[:space:]]*Online:.*[[:space:]]$(hostname -s)([[:space:]]|$)"; then
        echo "Node is Online."
        break
    fi

    # If node is in standby, try to unstandby it
    if echo "$nodes_out" | grep -qE "^[[:space:]]*Standby:.*[[:space:]]$(hostname -s)([[:space:]]|$)"; then
        echo "Node is in Standby, attempting to unstandby..."
        sudo pcs node unstandby 2>/dev/null || true
    fi

    if [ $elapsed -ge $timeout ]; then
        echo "Timed out waiting for node to become Online."
        exit 1
    fi
    sleep 1
    elapsed=$((elapsed + 1))
done

# 3. Wait for Locking Services (DLM & lvmlockd)
# Even if the node is "Online", the locking services might still be starting up.
# We check for their presence at the system level.
echo "Waiting for locking services (dlm_controld, lvmlockd) to be active..."
timeout=60
elapsed=0
while true; do
    # Print resource status for debugging
    echo "Current resource status (elapsed: $elapsed):"
    pcs status resources 2>&1

    # Check for dlm_controld (userspace DLM controller)
    # dlm_tool ls returns 0 if it can contact dlm_controld
    dlm_ok=false
    if dlm_tool ls > /dev/null 2>&1; then
        dlm_ok=true
    fi

    # Check for lvmlockd
    # We can check if the process is running
    lvm_ok=false
    if pidof lvmlockd > /dev/null 2>&1; then
        lvm_ok=true
    fi

    if [ "$dlm_ok" = true ] && [ "$lvm_ok" = true ]; then
        echo "DLM and lvmlockd are active."
        break
    fi

    if [ $elapsed -ge $timeout ]; then
        echo "Timed out waiting for locking services."
        echo "DLM OK: $dlm_ok"
        echo "LVM OK: $lvm_ok"
        exit 1
    fi

    sleep 1
    elapsed=$((elapsed + 1))
done

echo "preActivate script completed successfully."
exit 0
