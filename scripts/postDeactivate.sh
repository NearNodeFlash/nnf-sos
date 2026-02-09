#!/bin/bash

# Redirect stdout and stderr to syslog
exec 1> >(logger -s -t $(basename $0)) 2>&1

# This script implements a "standby-first" approach to leaving the pacemaker
# cluster. The goal is to avoid a corosync membership storm when multiple
# compute nodes leave simultaneously during PostRun teardown.
#
# Problem: If all compute nodes run `pcs cluster stop` at the same time,
# they all drop out of corosync within seconds. DLM on the remaining nodes
# (especially the rabbit) sees rapid membership changes while lockspaces are
# still active, triggering a "stateful merge" detection that kills DLM and
# causes the rabbit to lose all GFS2 lockspaces.
#
# Solution: Put the node in standby first. Pacemaker will cleanly stop all
# resources (DLM lockspaces, lvmlockd, etc.) while the node remains a
# corosync member. Other nodes see a graceful resource departure — not a
# sudden membership loss. Only after resources are fully stopped do we
# leave corosync via `pcs cluster stop --force`.

STANDBY_TIMEOUT=120

# 1. Check if cluster is running. If not, we are already "stopped", so success.
if ! pcs status > /dev/null 2>&1; then
    echo "Cluster is already stopped."
    exit 0
fi

# 2. Handle any UNCLEAN nodes before entering standby
# UNCLEAN nodes can block resource operations. If nodes have been fenced
# (e.g., via STONITH), we need to confirm the fence to mark them as cleanly
# removed from the cluster.
echo "Checking for UNCLEAN nodes..."
unclean_nodes=$(pcs status nodes 2>/dev/null | grep -oP 'UNCLEAN \(offline\):\s*\[\s*\K[^\]]+' | tr ' ' '\n' | grep -v '^$')

if [ -n "$unclean_nodes" ]; then
    echo "Found UNCLEAN nodes: $unclean_nodes"
    for node in $unclean_nodes; do
        echo "Confirming fence for UNCLEAN node: $node"
        echo 'y' | sudo pcs stonith confirm "$node" 2>/dev/null || true
    done

    # Wait a moment for the cluster to process the fence confirmations
    sleep 2

    # Also acknowledge fences in DLM to unblock any pending lock operations
    echo "Acknowledging fences in DLM..."
    for nodeid in $(corosync-cmapctl -g nodelist 2>/dev/null | grep -oP 'nodeid\s*=\s*\K\d+' | sort -u); do
        local_nodeid=$(corosync-cmapctl -g runtime.votequorum.this_node_id 2>/dev/null | grep -oP 'value\s*=\s*\K\d+' || echo "")
        if [ -n "$local_nodeid" ] && [ "$nodeid" != "$local_nodeid" ]; then
            dlm_tool fence_ack "$nodeid" 2>/dev/null || true
        fi
    done
fi

# 3. Put node in standby to drain all resources gracefully
# This tells pacemaker to stop all resources on this node (including DLM
# lockspaces and lvmlockd) while keeping the node in corosync. Other nodes
# see the resources depart cleanly rather than seeing a sudden membership loss.
echo "Putting node in standby to drain resources..."
if ! sudo pcs node standby; then
    echo "WARNING: pcs node standby failed. Falling through to cluster stop."
    sudo pcs cluster stop --force
    exit 0
fi

# 4. Wait for this node to appear in the Standby list (resources drained)
echo "Waiting for node to reach Standby state (timeout: ${STANDBY_TIMEOUT}s)..."
hostname_short=$(hostname -s)
elapsed=0
while true; do
    nodes_out=$(pcs status nodes 2>&1)

    # Check if our node appears in the Standby list
    # Node names in pcs may be prefixed with 'e' (e.g., etuolumne1289)
    if echo "$nodes_out" | grep -qE "^[[:space:]]*Standby:.*[[:space:]]e?${hostname_short}([[:space:]]|$)"; then
        echo "Node is in Standby — resources have been drained."
        break
    fi

    if [ $elapsed -ge $STANDBY_TIMEOUT ]; then
        echo "Timed out waiting for Standby state after ${STANDBY_TIMEOUT}s."
        echo "Current node status:"
        echo "$nodes_out"
        echo "Proceeding with cluster stop anyway."
        break
    fi

    if [ $((elapsed % 10)) -eq 0 ]; then
        echo "Still waiting for Standby (elapsed: ${elapsed}s)..."
        echo "$nodes_out"
    fi

    sleep 1
    elapsed=$((elapsed + 1))
done

# 5. Now leave corosync. Use --force since resources are already stopped
# (or timed out). The node cleanly departs corosync membership.
echo "Stopping cluster (leaving corosync)..."
sudo pcs cluster stop --force
exit 0
