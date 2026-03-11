#!/bin/bash

# Patch the default NnfStorageProfile to add GFS2 fence commands
# This adds preActivate and postDeactivate commands to start/stop
# corosync and pacemaker on compute nodes during PreRun/PostRun.
# It also adds postTeardown on rabbit nodes to run DLM diagnostics
# after workflow teardown.

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

# PostTeardown command (rabbit-side):
# /usr/bin/dlm-diagnostics.sh $VG_NAME
#   Runs DLM state diagnostics after file system teardown on the rabbit.
#   $VG_NAME is substituted by the controller with the actual LVM volume
#   group name for the current allocation.  In per-VG mode, the script
#   only checks that THIS VG's lockspace was cleaned up — sibling
#   allocations' lockspaces are expected and ignored.
#   Errors (stale DLM state) cause the workflow to fail (.WithMajor).
#   Full output is saved to /var/log/nnf-dlm-diagnostics.log on the rabbit.

# --- Deploy scripts to rabbit nodes ---
RABBITS="${RABBITS:-rabbit-node-1 rabbit-node-2}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

echo "Deploying dlm-diagnostics.sh to rabbit nodes: $RABBITS"
for rabbit in $RABBITS; do
    scp "$SCRIPT_DIR/dlm-diagnostics.sh" "${rabbit}:/usr/bin/dlm-diagnostics.sh"
    ssh "$rabbit" chmod 755 /usr/bin/dlm-diagnostics.sh
done

# --- Patch the storage profile ---
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
  },
  {
    "op": "replace",
    "path": "/data/gfs2Storage/userCommands/postTeardown",
    "value": ["/usr/bin/dlm-diagnostics.sh $VG_NAME"]
  }
]'

echo ""
echo "Verifying compute commands..."
kubectl get nnfstorageprofile default -n nnf-system -o jsonpath='{.data.gfs2Storage.blockDeviceCommands.computeCommands.userCommands}' | jq .
echo ""
echo "Verifying userCommands..."
kubectl get nnfstorageprofile default -n nnf-system -o jsonpath='{.data.gfs2Storage.userCommands}' | jq .
