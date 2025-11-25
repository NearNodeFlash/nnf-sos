#!/bin/bash

# Patch the default NnfStorageProfile to add GFS2 fence commands
# This adds preActivate and postDeactivate commands to start/stop
# corosync and pacemaker on compute nodes during PreRun/PostRun

# PreActivate command (formatted for readability):
# sudo pcs cluster start $(hostname -s)
# sleep 3
# sudo pcs node unstandby $(hostname -s)
# timeout=30; elapsed=0
# while ! pcs status nodes 2>/dev/null | grep -qE "^[[:space:]]*Online:.*[[:space:]]$(hostname -s)([[:space:]]|$)"; do
#   [ $elapsed -ge $timeout ] && exit 1
#   sleep 1
#   elapsed=$((elapsed + 1))
# done

# PostDeactivate command (formatted for readability):
# if pcs status > /dev/null 2>&1; then
#   sudo pcs node standby $(hostname -s)
#   timeout=30; elapsed=0
#   while pcs status nodes 2>/dev/null | grep -qE "^[[:space:]]*Online:.*[[:space:]]$(hostname -s)([[:space:]]|$)"; do
#     [ $elapsed -ge $timeout ] && exit 1
#     sleep 1
#     elapsed=$((elapsed + 1))
#   done
#   sudo pcs cluster stop --force $(hostname -s)
# fi

kubectl patch nnfstorageprofile default -n nnf-system --type='json' -p='[
  {
    "op": "replace",
    "path": "/data/gfs2Storage/blockDeviceCommands/computeCommands/userCommands/preActivate",
    "value": ["/usr/bin/preActivate.sh"]
  },
  {
    "op": "replace",
    "path": "/data/gfs2Storage/blockDeviceCommands/computeCommands/userCommands/postDeactivate",
    "value": ["/usr/bin/postDeactivate.sh"]
  }
]'

echo "Verifying the patch..."
kubectl get nnfstorageprofile default -n nnf-system -o jsonpath='{.data.gfs2Storage.blockDeviceCommands.computeCommands.userCommands}' | jq .
