#!/bin/bash
#
# Wipe a rabbit node to a clean state after a drive replacement.
#
# Prerequisites:
#   - The rabbit must be disabled before running this script.
#   - Requires: kubectl, flux, ssh access to the rabbit.
#   - Must be run from a node with /admin/scripts/nnf/ shims available.
#   - With -c: also requires clush and SSH access to the computes.
#
# This script:
#   1. Verifies the rabbit is disabled
#   2. (with -c) Unmounts, deactivates, and removes NNF VGs on computes,
#      purging all stale LVM/DM state so Step 14 starts clean
#   3. Drains the rabbit to stop nnf-node-manager
#   4-5. Strips finalizers and deletes NnfNodeStorage/NnfNodeBlockStorage
#      (NnfStorageReconciler may recreate them — that's harmless)
#   6. Cleans up stale LVM state (VGs, DM entries, /dev/ dirs) on the rabbit
#      that finalizer-stripping left behind
#   7. Deletes the nnf-ec database
#   8-10. Restarts nnf-node-manager, which wipes all NVMe namespaces on
#      startup and rebuilds from scratch
#   11. Re-enables the rabbit
#   12. Waits for storage resources to be recreated and ready
#   13. Annotates NnfAccess to trigger a reconcile that adds computes
#      to the wiped rabbit's NBS access lists (may wait for backoff)
#   14. (with -c) Discovers new NVMe-backed VGs, activates, and mounts
#
# After this script completes, either:
#   - Reboot the compute nodes (default), or
#   - Use -c to live-refresh computes without rebooting.
#

set -o pipefail

# Timestamped output — every log line gets a local timestamp for post-mortem.
log() { echo "$(date '+%m-%d %H:%M:%S.%3N')  $*"; }

POLL_INTERVAL=5
POLL_TIMEOUT=300
REFRESH_COMPUTES=false
ETH_PREFIX=true
MOUNT_PATHS=()

usage() {
    cat <<EOF
Wipe a rabbit node to a clean state after a drive replacement.

Usage: $0 [-c] [-n] [-m mount_path ...] <rabbit-node-name>

The rabbit must be disabled before running this script. This script
tears down all NNF storage on the rabbit, wipes the nnf-ec database,
and rebuilds everything from scratch.

Options:
    -c    Refresh compute nodes in-place instead of requiring a reboot.
          Unmounts stale NNF filesystems, clears LVM/DM state, and
          remounts after storage is rebuilt. Requires clush and SSH
          access to the computes.
          NOTE: If you powercycle Rabbit-s, YOU MUST reboot the computes!
    -n    Do not add "e" ethernet prefix to compute hostnames for SSH/clush.
          By default, compute hostnames are prefixed with "e" (e.g.
          erabbit-compute-2) for network access. Use -n in environments
          where the Kubernetes node name is directly reachable.
    -m path
          Mount path for NNF filesystems (repeatable, e.g. -m /l/ssd).
          Used as fallback when no prior mounts were saved on the computes.
          Only meaningful with -c.
    -h    Show this help

Without -c, you must reboot the compute nodes after this script
completes so they pick up the rebuilt storage.
EOF
}

# get_computes returns a comma-separated list of compute nodes for a rabbit.
# Tries flux getrabbit first, falls back to kubectl SystemConfiguration.
get_computes() {
    local rabbit="$1"
    # Try flux getrabbit (available on Flux-managed clusters)
    if flux getrabbit "$rabbit" 2>/dev/null; then
        return
    fi
    # Fall back: query the SystemConfiguration for compute names
    kubectl get systemconfiguration default -o json 2>/dev/null \
        | jq -r --arg r "$rabbit" '.spec.storageNodes[] | select(.name==$r) | .computesAccess[].name' \
        | paste -sd, -
}

# wait_for polls a command until it succeeds or times out.
# Usage: wait_for <description> <timeout_seconds> <command...>
wait_for() {
    local desc="$1" timeout="$2"
    shift 2
    local elapsed=0
    log "Waiting for $desc (timeout ${timeout}s)..."
    while ! "$@" >/dev/null 2>&1; do
        sleep "$POLL_INTERVAL"
        elapsed=$((elapsed + POLL_INTERVAL))
        if [ "$elapsed" -ge "$timeout" ]; then
            log "ERROR: Timed out waiting for $desc after ${timeout}s"
            return 1
        fi
    done
    log "  $desc: OK"
}

# Parse options and positional args in any order (e.g. -c rabbit or rabbit -c)
RABBIT_ARG=""
while [[ $# -gt 0 ]]; do
    case "$1" in
        -c) REFRESH_COMPUTES=true; shift ;;
        -n) ETH_PREFIX=false; shift ;;
        -m) MOUNT_PATHS+=("$2"); shift 2 ;;
        -h) usage; exit 0 ;;
        -*) echo "Unknown option: $1"; usage; exit 1 ;;
        *)  RABBIT_ARG="$1"; shift ;;
    esac
done

if [[ -z "$RABBIT_ARG" ]]; then
    usage
    exit 1
fi

export KUBECONFIG=/etc/kubernetes/admin.conf
export RABBIT=$RABBIT_ARG
FALLBACK_MOUNTS=$(printf '%s\n' "${MOUNT_PATHS[@]}")

# Always look up computes — Step 13 needs them to verify NBS access lists.
COMPUTES=$(get_computes "$RABBIT")
if [ -z "$COMPUTES" ]; then
    log "ERROR: No computes found for $RABBIT"
    exit 1
fi

# Derive network-reachable compute names. LLNL prefixes hostnames with "e"
# for the ethernet interface. Use -n to disable this for environments where
# the Kubernetes node name is directly reachable (e.g. htx-lustre).
if [ "$ETH_PREFIX" = true ]; then
    COMPUTES_NET=$(echo "$COMPUTES" | sed 's/\([^,]*\)/e\1/g')
else
    COMPUTES_NET="$COMPUTES"
fi

if [ "$REFRESH_COMPUTES" = true ]; then
    log "Wiping rabbit: $RABBIT (with compute refresh: $COMPUTES_NET)"
    if [ ${#MOUNT_PATHS[@]} -gt 0 ]; then
        log "  Fallback mount paths: ${MOUNT_PATHS[*]}"
    fi
else
    log "Wiping rabbit: $RABBIT (computes: $COMPUTES_NET)"
fi

# --- Step 1: Verify rabbit is disabled ---
log "Step 1: Verifying rabbit is disabled..."
RABBIT_STATUS=$(kubectl get storages "$RABBIT" -o json | jq -r ".status.status") || {
    log "ERROR: Failed to query rabbit status for $RABBIT"
    exit 1
}
if [ "$RABBIT_STATUS" != "Disabled" ]; then
    log "ERROR: The rabbit must be disabled before wiping."
    echo "This prevents new workflows from being scheduled to the rabbit."
    echo ""
    echo "  Rabbit status: $RABBIT_STATUS"
    echo "  To disable:    rabbit_disable $RABBIT"
    exit 1
fi

# --- Step 2: Unmount and clean compute-side storage (with -c) ---
if [ "$REFRESH_COMPUTES" = true ]; then
    log "Step 2: Unmounting and tearing down NNF storage on computes..."
    # Discover NNF mounts by finding VGs backed by NVMe PVs (rabbit NVMe
    # targets). This works regardless of mount path (/mnt/nnf/, /l/ssd, etc.).
    # Save only mount POINTS to /run/nnf/mount_points — device paths will change
    # after the wipe, so we discover new devices in Step 14.
    #
    # Full teardown: unmount, deactivate VGs, remove VGs, clean dm entries,
    # and purge LVM metadata. This ensures the compute has NO stale LVM state
    # when the rabbit creates new namespaces. Step 14 then does a clean
    # discovery: pvscan, activate, mount.
    clush -b -f 16 -w "$COMPUTES_NET" "
        mkdir -p /run/nnf || { echo \"ERROR: Failed to create /run/nnf\"; exit 1; }
        > /run/nnf/mount_points || { echo \"ERROR: Failed to create /run/nnf/mount_points\"; exit 1; }
        # Find VGs with NVMe PVs — these are the rabbit-connected NNF VGs
        NNF_VGS=\$(pvs --noheadings -o vg_name,pv_name 2>/dev/null | grep nvme | awk '{print \$1}' | sort -u)
        for vg in \$NNF_VGS; do
            vg_dm=\$(echo \$vg | sed 's/-/--/g')
            mount | grep \"/dev/mapper/\${vg_dm}-\" | awk '{print \$3}' >> /run/nnf/mount_points
        done
        # Unmount all discovered NNF filesystems
        FAIL=0
        while read mp; do
            if [ -n \"\$mp\" ]; then
                umount \$mp || { echo \"ERROR: umount \$mp failed\"; FAIL=1; }
            fi
        done < /run/nnf/mount_points
        # Deactivate NNF VGs, remove them, and clean all stale LVM state.
        # After the wipe, new namespaces get new PV UUIDs — any leftover VG
        # metadata referencing old UUIDs would make the VG appear partial.
        # Removing everything now means Step 14 starts with a clean slate.
        for vg in \$NNF_VGS; do
            vgchange --activate n \$vg || echo \"WARNING: vgchange --activate n \$vg failed\"
            vg_dm=\$(echo \$vg | sed 's/-/--/g')
            for dm in \$(dmsetup ls 2>/dev/null | awk '{print \$1}' | grep \"^\${vg_dm}-\"); do
                dmsetup remove -f \$dm 2>/dev/null
            done
            # Strip missing PVs first so vgremove doesn't choke on partial VGs
            vgreduce --removemissing --force \$vg 2>/dev/null || true
            vgremove --force --yes \$vg 2>/dev/null || true
            rm -f \"/etc/lvm/backup/\$vg\"
        done
        # Second pass: catch any remaining partial VGs with NVMe PVs that
        # weren't found by the first pass (e.g. all PVs missing but VG
        # metadata still cached). The 'p' in vg_attr indicates partial.
        PARTIAL_VGS=\$(vgs --noheadings -o vg_name,vg_attr 2>/dev/null | awk '\$2 ~ /p/ {print \$1}')
        for vg in \$PARTIAL_VGS; do
            echo \"Removing partial VG: \$vg\"
            vgchange --activate n \$vg 2>/dev/null || true
            vg_dm=\$(echo \$vg | sed 's/-/--/g')
            for dm in \$(dmsetup ls 2>/dev/null | awk '{print \$1}' | grep \"^\${vg_dm}-\"); do
                dmsetup remove -f \$dm 2>/dev/null
            done
            vgreduce --removemissing --force \$vg 2>/dev/null || true
            vgremove --force --yes \$vg 2>/dev/null || true
            rm -f \"/etc/lvm/backup/\$vg\"
        done
        # Flush PV cache so old NVMe PVs are forgotten
        pvscan --cache 2>/dev/null || true
        exit \$FAIL
    " || { log "ERROR: Failed to unmount NNF filesystems on computes (see clush output above)"; exit 1; }
    log "  Compute storage unmounted and cleaned."
fi

# --- Step 3: Drain the rabbit ---
# Drain stops the nnf-node-manager DaemonSet pod. We drain before
# deleting NnfNodeStorage resources so node-manager can't process
# finalizers (which would try to clean up NVMe state we're about to wipe
# anyway). Although the NnfSystemStorage controller skips disabled rabbits
# (ExcludeDisabledRabbits), the NnfStorageReconciler may still recreate
# NnfNodeStorage/NnfNodeBlockStorage because the parent NnfStorage resource
# persists. This is harmless — fresh node-manager startup without nnf.db
# wipes all NVMe namespaces regardless.
log "Step 3: Draining rabbit..."
/admin/scripts/nnf/rabbit_drain "$RABBIT" || { log "ERROR: Failed to drain $RABBIT"; exit 1; }

wait_for "non-DaemonSet pods stopped" 120 \
    bash -c "[ \$(kubectl get pods -n '$RABBIT' --field-selector=status.phase=Running --no-headers 2>/dev/null | grep -cv 'node-manager') -eq 0 ]" || exit 1

# --- Step 4: Strip finalizers and delete NnfNodeStorage resources ---
# Node-manager is stopped so finalizers won't be processed. Strip them
# so the API server can delete the resources immediately. The
# NnfStorageReconciler may recreate them (the parent NnfStorage still
# exists and its CreateOrUpdate has no disabled-state guard), but this is
# harmless — they'll be reconciled against clean hardware after restart.
#
# Save the NnfNodeBlockStorage count before deleting so Step 12 can wait
# for all of them to be recreated (not just the first one).
EXPECTED_NBS_COUNT=$(kubectl get -n "$RABBIT" nnfnodeblockstorages --no-headers 2>/dev/null | wc -l)
log "Step 4: Deleting NnfNodeStorage resources (expecting $EXPECTED_NBS_COUNT NnfNodeBlockStorage to be recreated)..."
for NS in $(kubectl get -n "$RABBIT" nnfnodestorages -o name 2>/dev/null); do
    kubectl patch -n "$RABBIT" "$NS" -p '{"metadata":{"finalizers":[]}}' --type=merge 2>/dev/null
    kubectl delete -n "$RABBIT" "$NS" --wait=false 2>/dev/null
done

# --- Step 5: Strip finalizers and delete NnfNodeBlockStorage resources ---
log "Step 5: Deleting NnfNodeBlockStorage resources..."
for NBS in $(kubectl get -n "$RABBIT" nnfnodeblockstorages -o name 2>/dev/null); do
    kubectl patch -n "$RABBIT" "$NBS" -p '{"metadata":{"finalizers":[]}}' --type=merge 2>/dev/null
    kubectl delete -n "$RABBIT" "$NBS" --wait=false 2>/dev/null
done

# --- Step 6: Clean up rabbit-side LVM state ---
# Stripping finalizers skipped the normal VG teardown (deactivate, vgremove).
# The VG metadata is gone (PVs were part of the now-deleted namespaces), but
# stale device-mapper entries and /dev/<vg_name> directories persist. If these
# aren't removed, the controller's vgcreate will fail with "already exists in
# filesystem" when it tries to recreate the VG with the same name.
log "Step 6: Cleaning up stale LVM state on rabbit..."
ssh "$RABBIT" '
    # Try to find and remove NNF VGs by SystemStorage tag
    NNF_VGS=$(vgs --noheadings -o vg_name --select "vg_tags=SystemStorage" 2>/dev/null | awk "{print \$1}")
    for vg in $NNF_VGS; do
        vgchange --activate n $vg 2>/dev/null
        vgremove --force --yes $vg 2>/dev/null
    done
    # Clean stale device-mapper entries and /dev/ directories matching UUID
    # pattern (these persist even when the VG is no longer in LVM metadata)
    for d in /dev/????????-????-????-????-????????????_*; do
        [ -d "$d" ] || continue
        vg=$(basename "$d")
        vg_dm=$(echo $vg | sed "s/-/--/g")
        for dm in $(dmsetup ls 2>/dev/null | awk "{print \$1}" | grep "^${vg_dm}-"); do
            dmsetup remove -f $dm 2>/dev/null
        done
        rm -rf "$d"
        rm -f "/etc/lvm/backup/$vg"
    done
    pvscan --cache 2>/dev/null || true
' || log "WARNING: Rabbit LVM cleanup had errors (may be harmless)"
log "  Rabbit LVM state cleaned."

# --- Step 7: Delete nnf-ec database ---
log "Step 7: Deleting nnf-ec database..."
ssh "$RABBIT" rm -rf /localdisk/nnf.db || { log "ERROR: Failed to delete nnf-ec database on $RABBIT"; exit 1; }

# --- Step 8: Restart nnf-node-manager ---
# Delete any existing pod so the DaemonSet controller recreates it after
# undrain. On startup without a database, it deletes all NVMe namespaces
# (cleaning hardware state) and recreates a fresh nnf.db from discovery.
# Capture the old pod UID so we can confirm a NEW pod starts (not the old one
# passing a Ready check before it terminates).
OLD_NM_UID=$(kubectl get pod -n nnf-system --field-selector spec.nodeName="$RABBIT" -l cray.nnf.node=true -o jsonpath='{.items[0].metadata.uid}' 2>/dev/null || true)
log "Step 8: Restarting nnf-node-manager (old UID: ${OLD_NM_UID:-none})..."
kubectl delete pod -n nnf-system --field-selector spec.nodeName="$RABBIT" -l cray.nnf.node=true --wait=false 2>/dev/null || true

# --- Step 9: Undrain the rabbit ---
log "Step 9: Undraining rabbit..."
/admin/scripts/nnf/rabbit_undrain "$RABBIT" || { log "ERROR: Failed to undrain $RABBIT"; exit 1; }

# --- Step 10: Wait for nnf-node-manager to be ready ---
# First wait for the old pod to be gone and a new pod (different UID) to
# appear. Then wait for the new pod to be Ready. Without this two-phase
# check, the Ready wait could pass on the old pod before it terminates.
log "Step 10: Waiting for new nnf-node-manager pod..."
wait_for "new nnf-node-manager pod (old UID gone)" "$POLL_TIMEOUT" \
    bash -c '
        uid=$(kubectl get pod -n nnf-system --field-selector spec.nodeName="'"$RABBIT"'" -l cray.nnf.node=true -o jsonpath="{.items[0].metadata.uid}" 2>/dev/null)
        [ -n "$uid" ] && [ "$uid" != "'"$OLD_NM_UID"'" ]
    ' || exit 1
log "Step 10: Waiting for nnf-node-manager to be ready..."
wait_for "nnf-node-manager pod ready" "$POLL_TIMEOUT" \
    kubectl wait pod -n nnf-system --field-selector spec.nodeName="$RABBIT" -l cray.nnf.node=true \
        --for=condition=Ready --timeout=0s || exit 1

# --- Step 11: Enable the rabbit ---
log "Step 11: Enabling rabbit..."
/admin/scripts/nnf/rabbit_enable "$RABBIT" || { log "ERROR: Failed to enable $RABBIT"; exit 1; }

# --- Step 12: Wait for storage resources to be recreated and ready ---
# Wait for the expected count of NnfNodeBlockStorage, not just "at least one".
# The controller-manager recreates them asynchronously and xfs-0 often appears
# 30-40s before the rest.
log "Step 12: Waiting for $EXPECTED_NBS_COUNT NnfNodeBlockStorage resources..."
if [ "$EXPECTED_NBS_COUNT" -gt 0 ] 2>/dev/null; then
    wait_for "all $EXPECTED_NBS_COUNT NnfNodeBlockStorage resources recreated" "$POLL_TIMEOUT" \
        bash -c "[ \$(kubectl get -n '$RABBIT' nnfnodeblockstorages --no-headers 2>/dev/null | wc -l) -ge $EXPECTED_NBS_COUNT ]" || exit 1
else
    wait_for "NnfNodeBlockStorage resources recreated" "$POLL_TIMEOUT" \
        bash -c "[ \$(kubectl get -n '$RABBIT' nnfnodeblockstorages --no-headers 2>/dev/null | wc -l) -gt 0 ]" || exit 1
fi

wait_for "all NnfNodeBlockStorage ready" "$POLL_TIMEOUT" \
    bash -c "[ \$(kubectl get -n '$RABBIT' nnfnodeblockstorages -o jsonpath='{range .items[*]}{.status.ready}{\"\\n\"}{end}' 2>/dev/null | grep -c false) -eq 0 ]" || exit 1

# Also wait for NnfNodeStorage ready. The NnfAccess controller's
# mapClientLocalStorage() skips allocations whose NnfNodeStorage isn't ready
# (nnf_access_controller.go:601), which would cause Step 13's reconciliation
# to succeed without adding computes to the NBS access lists.
wait_for "all NnfNodeStorage ready" "$POLL_TIMEOUT" \
    bash -c "[ \$(kubectl get -n '$RABBIT' nnfnodestorages -o jsonpath='{range .items[*]}{.status.ready}{\"\\n\"}{end}' 2>/dev/null | grep -c false) -eq 0 ]" || exit 1

# Wait for Storage to transition from Disabled to Ready. Step 11 enabled
# the rabbit, but the DWSStorage controller must reconcile to update the
# status. Until Status=Ready, the Storage.Status.Access.Computes list may
# be stale or the controller may still be processing the enable transition.
wait_for "Storage status Ready" "$POLL_TIMEOUT" \
    bash -c "[ \$(kubectl get storages '$RABBIT' -o jsonpath='{.status.status}' 2>/dev/null) = 'Ready' ]" || exit 1

# Wait for Storage.Status.Access.Computes to be populated. The NnfAccess
# controller's mapClientLocalStorage() reads this field (line 721-723) to
# build the compute-to-allocation mapping. If computes aren't listed here,
# addBlockStorageAccess() receives an empty nodeStorageMap and returns nil
# without updating NBS — causing NnfAccess to report ready=true while NBS
# still has rabbit-only access.
FIRST_COMPUTE=$(flux hostlist -n0 "$COMPUTES")
wait_for "computes in Storage.Status.Access" "$POLL_TIMEOUT" \
    bash -c "kubectl get storages '$RABBIT' -o jsonpath='{.status.access.computes[*].name}' 2>/dev/null | grep -q '$FIRST_COMPUTE'" || exit 1

# Wait for NnfNodeBlockStorage to have NVMe device info populated. The
# NnfAccess controller's mapClientLocalStorage() reads NBS status allocations
# (line 695) for NQN/namespace data. If devices aren't populated yet, the
# storageMapping entries will be incomplete.
wait_for "NBS devices populated" "$POLL_TIMEOUT" \
    bash -c "kubectl get nnfnodeblockstorages -n '$RABBIT' -o jsonpath='{.items[0].status.allocations[0].devices[0].NQN}' 2>/dev/null | grep -q 'nqn'" || exit 1

# --- Step 13: Annotate NnfAccess to trigger reconcile ---
# The resource deletion storm in Steps 4-5 triggers a flood of errors in the
# NnfStorage, NnfSystemStorage, and NnfAccess controllers (e.g. "incorrect
# number of NnfNodeStorages", "NnfNodeStorage has not been reconciled after
# 300 seconds"). These errors put the NnfAccess controller's workqueue key
# into exponential backoff that can last up to ~17 minutes.
#
# We need the NnfAccess controller to run addBlockStorageAccess() to add
# computes to the wiped rabbit's NBS access lists. Without this, NVMe
# namespaces won't be exposed to the computes — even a reboot won't help.
#
# Approach: repeatedly annotate NnfAccess with a timestamp. Each annotation
# enqueues a reconcile event. If the controller is in backoff, the event
# won't be processed until the backoff timer expires — but it WILL eventually
# run. We re-annotate every 10s so there's always a pending event ready to
# be processed the moment backoff expires.
#
# This is safe for multi-rabbit clusters: we never delete NnfAccess (which
# is a cluster-wide shared resource). We only add an annotation, which
# cannot affect other rabbits' access configuration.
#
# Max backoff in controller-runtime is 16m40s. We allow up to 20 minutes.
ACCESS_INTERVAL=10
ACCESS_TIMEOUT=1200
ACCESS_ELAPSED=0
PREV_NBS_WITH_COMPUTE=-1

log "Step 13: Annotating NnfAccess to trigger reconcile (polling every ${ACCESS_INTERVAL}s, timeout ${ACCESS_TIMEOUT}s)..."
while true; do
    # Annotate all SystemStorage NnfAccess resources
    for ACCESS in $(kubectl get nnfaccess -A -l nnf.cray.hpe.com/job_id=SystemStorage -o jsonpath='{range .items[*]}{.metadata.namespace}/{.metadata.name}{"\n"}{end}' 2>/dev/null); do
        ACCESS_NS=${ACCESS%%/*}
        ACCESS_NAME=${ACCESS##*/}
        kubectl annotate nnfaccess "$ACCESS_NAME" -n "$ACCESS_NS" \
            rabbit-wipe-reconcile-trigger="$(date -u +%Y%m%dT%H%M%SZ)" --overwrite 2>/dev/null || true
    done

    # Two-part completion check:
    # 1) Every compute must appear in at least one NBS access list.
    #    Per-compute NnfSystemStorages assign different computes to different
    #    NBS subsets, so we must check each compute individually.
    # 2) The NBS-with-compute count must stabilize (same for 2 consecutive
    #    checks). This catches multi-NnfSystemStorage configs where each
    #    compute needs access added to NBS from EVERY NnfSystemStorage.
    #    Without stabilization, we'd exit after the first NnfAccess reconciles
    #    and miss the second — leaving computes with half their namespaces.
    ALL_COMPUTES_FOUND=true
    for c in $(flux hostlist --expand "$COMPUTES"); do
        if ! kubectl get nnfnodeblockstorages -n "$RABBIT" \
            -o jsonpath='{.items[*].spec.allocations[*].access}' 2>/dev/null | grep -q "$c"; then
            ALL_COMPUTES_FOUND=false
            break
        fi
    done

    NBS_WITH_COMPUTE=$(kubectl get nnfnodeblockstorages -n "$RABBIT" -o json 2>/dev/null \
        | jq '[.items[] | select(.spec.allocations[]?.access | length > 1)] | length')

    if [ "$ALL_COMPUTES_FOUND" = true ] && [ "$NBS_WITH_COMPUTE" -eq "$PREV_NBS_WITH_COMPUTE" ] 2>/dev/null; then
        log "  All computes confirmed in NBS access lists ($NBS_WITH_COMPUTE/$EXPECTED_NBS_COUNT NBS updated, ${ACCESS_ELAPSED}s elapsed)."
        break
    fi
    PREV_NBS_WITH_COMPUTE=$NBS_WITH_COMPUTE

    ACCESS_ELAPSED=$((ACCESS_ELAPSED + ACCESS_INTERVAL))
    if [ "$ACCESS_ELAPSED" -ge "$ACCESS_TIMEOUT" ]; then
        log "ERROR: NBS access not fully updated after ${ACCESS_TIMEOUT}s (${NBS_WITH_COMPUTE:-0}/$EXPECTED_NBS_COUNT NBS have compute access)."
        log "  The NnfAccess controller may still be in backoff or broken."
        log "  Check nnf-controller-manager logs."
        exit 1
    fi

    sleep "$ACCESS_INTERVAL"
done

# --- Step 14: Refresh compute storage (with -c) or advise reboot ---
if [ "$REFRESH_COMPUTES" = true ]; then
    log "Step 14: Refreshing compute storage..."
    # Step 2 fully tore down LVM state on the computes (unmount, deactivate,
    # vgremove, dm cleanup). The rabbit has since created new namespaces with
    # fresh PVs/VGs/LVs and run mkfs. The compute's NVMe VFs now point to
    # these new namespaces. We just need to discover, activate, and mount.
    #
    # Wait for the NBS controller to finish exposing NVMe namespaces to the
    # computes. NnfAccess ready=true only means the NBS spec was updated —
    # the NBS controller processes the access list asynchronously, calling
    # nnf-ec to set up NVMe access for each compute. Poll until the
    # NVMe-backed VG is visible on EVERY compute (not just the first one),
    # since namespaces may appear at different times on different computes.
    # Rescan NVMe namespaces on each compute — sometimes new namespaces
    # from the rebuilt rabbit don't appear without an explicit rescan.
    log "  Rescanning NVMe namespaces on computes..."
    clush -b -f 16 -w "$COMPUTES_NET" "
        for dev in /dev/nvme*; do
            [ -c \"\$dev\" ] && nvme ns-rescan \$dev 2>/dev/null || true
        done
    " || log "WARNING: NVMe ns-rescan had errors (may be harmless)"

    log "  Waiting for NVMe-backed VGs to appear on computes..."
    for c in $(flux hostlist --expand "$COMPUTES_NET"); do
        wait_for "NVMe VG visible on $c" "$POLL_TIMEOUT" \
            bash -c "clush -w '$c' 'pvscan --cache >/dev/null 2>&1; pvs --noheadings -o vg_name,pv_name 2>/dev/null | grep nvme | grep -v \"not used\"' 2>/dev/null | grep -q nvme" || exit 1
    done

    clush -b -f 16 -w "$COMPUTES_NET" "
        # Discover new NVMe-backed PVs and VGs
        pvscan --cache || echo \"WARNING: pvscan --cache failed\"
        vgscan --cache || echo \"WARNING: vgscan --cache failed\"
        # Find and activate NNF VGs backed by NVMe PVs
        NNF_VGS=\$(pvs --noheadings -o vg_name,pv_name 2>/dev/null | grep nvme | awk '{print \$1}' | sort -u)
        for vg in \$NNF_VGS; do
            vgchange --activate y \$vg || echo \"WARNING: vgchange --activate y \$vg failed\"
        done
        # Mount each discovered LV to its saved or fallback mount point
        FAIL=0
        # Use saved mount points if available, otherwise fall back to -m paths
        if [ -s /run/nnf/mount_points ]; then
            MOUNT_POINTS=\$(cat /run/nnf/mount_points)
        elif [ -n '$FALLBACK_MOUNTS' ]; then
            MOUNT_POINTS='$FALLBACK_MOUNTS'
        else
            echo \"WARNING: No mount points saved and no -m paths provided, skipping remount\"
            exit 0
        fi
        NEW_LVS=\"\"
        for vg in \$NNF_VGS; do
            lv_path=\$(lvs --noheadings -o lv_path \$vg 2>/dev/null | awk '{print \$1}' | head -1)
            if [ -n \"\$lv_path\" ]; then
                NEW_LVS=\"\$NEW_LVS \$lv_path\"
            fi
        done
        set -- \$MOUNT_POINTS
        for lv in \$NEW_LVS; do
            mp=\$1; shift 2>/dev/null || true
            if [ -n \"\$mp\" ] && [ -n \"\$lv\" ]; then
                mkdir -p \$mp 2>/dev/null || true
                mount \$lv \$mp || { echo \"ERROR: mount \$lv \$mp failed\"; FAIL=1; }
            fi
        done
        rm -f /run/nnf/mount_points
        exit \$FAIL
    " || { log "ERROR: Failed to refresh compute storage (see clush output above)"; exit 1; }

    # Verify mounts on every compute — report which are missing
    log "Verifying compute mounts..."
    MOUNT_OUTPUT=$(clush -b -f 16 -w "$COMPUTES_NET" "mount | grep -E '/mnt/nnf/|/l/ssd'" 2>&1) || true
    echo "$MOUNT_OUTPUT"
    FAILED_COMPUTES=""
    for c in $(flux hostlist --expand "$COMPUTES_NET"); do
        if ! ssh -o ConnectTimeout=5 "$c" "mount | grep -qE '/mnt/nnf/|/l/ssd'" 2>/dev/null; then
            FAILED_COMPUTES="$FAILED_COMPUTES $c"
        fi
    done
    if [ -n "$FAILED_COMPUTES" ]; then
        log "ERROR: NNF mounts missing on:$FAILED_COMPUTES"
        log "These computes may need to be rebooted."
        exit 1
    fi
fi

log ""
log "========================================="
log "Rabbit wipe complete for $RABBIT"
log "========================================="
if [ "$REFRESH_COMPUTES" = true ]; then
    log ""
    log "Compute storage has been refreshed in-place (no reboot needed)."
else
    log ""
    log "NEXT STEP: Reboot the compute nodes to pick up the rebuilt storage."
    log "The ansible scripts will remount the filesystems during boot."
    log ""
    log "Computes for this rabbit: $(get_computes "$RABBIT")"
fi