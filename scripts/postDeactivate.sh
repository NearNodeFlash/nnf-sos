#!/bin/bash

# Redirect stdout and stderr to syslog
exec 1> >(logger -s -t $(basename $0)) 2>&1

# 1. Check if cluster is running. If not, we are already "stopped", so success.
if ! pcs status > /dev/null 2>&1; then
    echo "Cluster is already stopped."
    exit 0
fi

# 2. Stop the cluster
echo "Stopping cluster..."
sudo pcs cluster stop
exit 0
