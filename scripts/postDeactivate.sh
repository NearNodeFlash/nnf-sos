#!/bin/bash

# Redirect stdout and stderr to syslog
exec 1> >(logger -s -t $(basename $0)) 2>&1

# 1. Check if cluster is running. If not, we are already "stopped", so success.
if ! pcs status > /dev/null 2>&1; then
    echo "Cluster is already stopped."
    exit 0
fi

# 2. Handle any UNCLEAN nodes before stopping the cluster
# UNCLEAN nodes will block `pcs cluster stop` indefinitely. If nodes have been
# fenced (e.g., via STONITH), we need to confirm the fence to mark them as cleanly
# removed from the cluster.
echo "Checking for UNCLEAN nodes..."
unclean_nodes=$(pcs status nodes 2>/dev/null | grep -oP 'UNCLEAN \(offline\):\s*\[\s*\K[^\]]+' | tr ' ' '\n' | grep -v '^$')

if [ -n "$unclean_nodes" ]; then
    echo "Found UNCLEAN nodes: $unclean_nodes"
    for node in $unclean_nodes; do
        echo "Confirming fence for UNCLEAN node: $node"
        # Use 'yes' to auto-confirm the fence operation
        echo 'y' | sudo pcs stonith confirm "$node" 2>/dev/null || true
    done
    
    # Wait a moment for the cluster to process the fence confirmations
    sleep 2
    
    # Also acknowledge fences in DLM to unblock any pending lock operations
    # DLM node IDs are typically derived from corosync node IDs
    echo "Acknowledging fences in DLM..."
    for nodeid in $(corosync-cmapctl -g nodelist 2>/dev/null | grep -oP 'nodeid\s*=\s*\K\d+' | sort -u); do
        # Only ack nodes that are not this node
        local_nodeid=$(corosync-cmapctl -g runtime.votequorum.this_node_id 2>/dev/null | grep -oP 'value\s*=\s*\K\d+' || echo "")
        if [ -n "$local_nodeid" ] && [ "$nodeid" != "$local_nodeid" ]; then
            dlm_tool fence_ack "$nodeid" 2>/dev/null || true
        fi
    done
fi

# 3. Stop the cluster
echo "Stopping cluster..."
sudo pcs cluster stop
exit 0
