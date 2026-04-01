#!/bin/bash
#
# Purge and rediscover NNF storage on compute nodes.
#
# This is a standalone version of rabbit_wipe.sh Steps 2 and 14.
# Use it to refresh compute-side LVM/mount state without a full
# rabbit wipe — e.g. after manual recovery or debugging.
#
# Usage: compute_refresh.sh [-n] [-m mount_path ...] <computes>
#
#   -n          Do not add "e" ethernet prefix to compute hostnames.
#               By default, hostnames are prefixed with "e" for network
#               access (e.g. erabbit-compute-2). Use -n when the Kubernetes
#               node name is directly reachable.
#   -m path     Mount path for NNF filesystems (e.g. /mnt/nnf/system-storage,
#               /l/ssd). Can be specified multiple times — one per VG.
#               Used as fallback when no prior mounts were saved.
#   computes    Compute node name or hostlist
#               (e.g. "rabbit-compute-2" or "rabbit-compute-[2-3]")
#
# What it does:
#   1. Rescans NVMe namespaces on computes (new namespaces may not appear without this)
#   2. Unmounts NNF filesystems on the computes
#   3. Deactivates, removes NNF VGs, cleans DM entries and LVM metadata
#   4. Rediscovers NVMe-backed PVs/VGs
#   5. Activates VGs and mounts filesystems to saved or -m mount paths
#   6. Verifies mounts on each compute
#
# Requires: clush, flux hostlist, SSH access to the computes.
#

set -o pipefail

log() { echo "$(date '+%m-%d %H:%M:%S.%3N')  $*"; }

POLL_INTERVAL=5
POLL_TIMEOUT=300

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

MOUNT_PATHS=()
ETH_PREFIX=true

usage() {
    echo "Usage: $0 [-n] [-m mount_path ...] <computes>"
    echo ""
    echo "Purge and rediscover NNF storage on compute nodes."
    echo ""
    echo "Options:"
    echo "  -n          Do not add \"e\" ethernet prefix to compute hostnames."
    echo "              By default, hostnames are prefixed with \"e\" for network"
    echo "              access (e.g. erabbit-compute-2). Use -n when the Kubernetes"
    echo "              node name is directly reachable."
    echo "  -m path     Mount path for NNF filesystems (repeatable)."
    echo "              Used as fallback when no prior mounts were saved."
    echo ""
    echo "  computes    Compute node name or hostlist"
    echo "              (e.g. \"rabbit-compute-2\" or \"rabbit-compute-[2-3]\")"
}

# Parse options and positional args in any order
COMPUTES_ARG=""
while [[ $# -gt 0 ]]; do
    case "$1" in
        -n) ETH_PREFIX=false; shift ;;
        -m) MOUNT_PATHS+=("$2"); shift 2 ;;
        -h) usage; exit 0 ;;
        -*) echo "Unknown option: $1"; usage; exit 1 ;;
        *)  COMPUTES_ARG="$1"; shift ;;
    esac
done

if [[ -z "$COMPUTES_ARG" ]]; then
    usage
    exit 1
fi

COMPUTES="$COMPUTES_ARG"
FALLBACK_MOUNTS=$(printf '%s\n' "${MOUNT_PATHS[@]}")

# Derive network-reachable compute names. LLNL prefixes hostnames with "e"
# for the ethernet interface. Use -n to disable this.
if [ "$ETH_PREFIX" = true ]; then
    COMPUTES_NET=$(echo "$COMPUTES" | sed 's/\([^,]*\)/e\1/g')
else
    COMPUTES_NET="$COMPUTES"
fi

log "Purging NNF storage on computes: $COMPUTES_NET"
if [ ${#MOUNT_PATHS[@]} -gt 0 ]; then
    log "  Fallback mount paths: ${MOUNT_PATHS[*]}"
fi

# --- Step 1: Rescan NVMe namespaces ---
# New namespaces may not appear without an explicit rescan.
log "Rescanning NVMe namespaces on computes..."
clush -b -f 16 -w "$COMPUTES_NET" "
    for dev in /dev/nvme[0-9]*; do
        [ -c \"\$dev\" ] || continue
        nvme ns-rescan \$dev 2>/dev/null || true
    done
" || log "WARNING: NVMe ns-rescan had errors (may be harmless)"

# --- Step 2: Unmount and clean compute-side storage ---
log "Unmounting and tearing down NNF storage on computes..."
clush -b -f 16 -w "$COMPUTES_NET" "
    mkdir -p /run/nnf || { echo \"ERROR: Failed to create /run/nnf\"; exit 1; }
    > /run/nnf/mount_points || { echo \"ERROR: Failed to create /run/nnf/mount_points\"; exit 1; }
    NNF_VGS=\$(pvs --noheadings -o vg_name,pv_name 2>/dev/null | grep nvme | awk '{print \$1}' | sort -u)
    for vg in \$NNF_VGS; do
        vg_dm=\$(echo \$vg | sed 's/-/--/g')
        mount | grep \"/dev/mapper/\${vg_dm}-\" | awk '{print \$3}' >> /run/nnf/mount_points
    done
    FAIL=0
    while read mp; do
        if [ -n \"\$mp\" ]; then
            umount \$mp || { echo \"ERROR: umount \$mp failed\"; FAIL=1; }
        fi
    done < /run/nnf/mount_points
    for vg in \$NNF_VGS; do
        vgchange --activate n \$vg || echo \"WARNING: vgchange --activate n \$vg failed\"
        vg_dm=\$(echo \$vg | sed 's/-/--/g')
        for dm in \$(dmsetup ls 2>/dev/null | awk '{print \$1}' | grep \"^\${vg_dm}-\"); do
            dmsetup remove -f \$dm 2>/dev/null
        done
        vgreduce --removemissing --force \$vg 2>/dev/null || true
        vgremove --force --yes \$vg 2>/dev/null || true
        rm -f \"/etc/lvm/backup/\$vg\"
    done
    # Second pass: catch any remaining partial VGs (missing PVs but VG
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
    pvscan --cache 2>/dev/null || true
    exit \$FAIL
" || { log "ERROR: Failed to unmount NNF filesystems (see clush output above)"; exit 1; }
log "  Compute storage unmounted and cleaned."

# --- Step 3: Rediscover and mount ---
log "Waiting for NVMe-backed VGs to appear on computes..."
for c in $(flux hostlist --expand "$COMPUTES_NET"); do
    wait_for "NVMe VG visible on $c" "$POLL_TIMEOUT" \
        bash -c "clush -w '$c' 'pvscan --cache >/dev/null 2>&1; pvs --noheadings -o vg_name,pv_name 2>/dev/null | grep nvme | grep -v \"not used\"' 2>/dev/null | grep -q nvme" || exit 1
done

clush -b -f 16 -w "$COMPUTES_NET" "
    pvscan --cache || echo \"WARNING: pvscan --cache failed\"
    vgscan --cache || echo \"WARNING: vgscan --cache failed\"
    NNF_VGS=\$(pvs --noheadings -o vg_name,pv_name 2>/dev/null | grep nvme | awk '{print \$1}' | sort -u)
    for vg in \$NNF_VGS; do
        vgchange --activate y \$vg || echo \"WARNING: vgchange --activate y \$vg failed\"
    done
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

# --- Step 4: Verify ---
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
    log "WARNING: NNF mounts missing on:$FAILED_COMPUTES"
    log "These computes may need to be rebooted."
else
    log "All computes have NNF mounts."
fi

log ""
log "Compute refresh complete for $COMPUTES_NET"
