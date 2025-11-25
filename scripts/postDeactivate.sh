#!/bin/bash

# Redirect stdout and stderr to syslog
exec 1> >(logger -s -t $(basename $0)) 2>&1

# 1. Check if cluster is running. If not, we are already "stopped", so success.
if ! pcs status > /dev/null 2>&1; then
    echo "Cluster is already stopped."
    exit 0
fi

# 2. Standby the node
echo "Setting node to standby..."
sudo pcs node standby

# 3. Wait for node to be NOT Online
timeout=30
elapsed=0
while pcs status nodes 2>/dev/null | grep -qE "^[[:space:]]*Online:.*[[:space:]]$(hostname -s)([[:space:]]|$)"; do
  if [ $elapsed -ge $timeout ]; then
    echo "Timed out waiting for node to standby."
    exit 1
  fi
  sleep 1
  elapsed=$((elapsed + 1))
done

# 4. Stop the cluster
echo "Stopping cluster..."
sudo pcs cluster stop
exit 0
