# LVM2 --nosync Flag Ignored with Deferred Activation

## Problem Summary

The `--nosync` flag is ignored when creating RAID LVs with `--activate n` (deferred activation), forcing unwanted synchronization when the LV is later activated. This defeats the purpose of the `--nosync` optimization for scenarios with known-zero storage.

## Current Behavior

```bash
# This works - nosync is honored
lvcreate --type raid5 --nosync --activate y -L 1G -n test vg

# This fails - nosync is ignored, sync occurs on activation
lvcreate --type raid5 --nosync --activate n -L 1G -n test vg
lvchange -ay vg/test  # Triggers unwanted synchronization
```

## Root Cause Analysis

**Source Code Investigation:**

1. **LV_NOTSYNCED flag is stored persistently** in LV metadata (`lib/metadata/lv_manip.c:9405`)
2. **Global `_mirror_in_sync` variable controls activation behavior** but is **reset to 0 on every command** (`lib/misc/lvm-globals.c:38`, `tools/lvmcmdline.c:2749`)
3. **RAID activation code only checks the global variable**, not the persistent flag (`lib/raid/raid.c:386`)

**The Problem:**

```c
// In lib/raid/raid.c - Missing check for LV_NOTSYNCED flag
if (mirror_in_sync())           // Only checks transient global
    flags = DM_NOSYNC;          // Should also check persistent flag
```

## Impact

- **Performance**: Forces unnecessary sync on large volumes with known-zero data
- **Workflow**: Requires `--activate y` workarounds, complicating automated deployment
- **Inconsistency**: User expectation that `--nosync` intent persists is violated

## Proposed Solution

Add persistent flag check to RAID activation:

```c
// In lib/raid/raid.c around line 385
if (mirror_in_sync() || (seg->lv->status & LV_NOTSYNCED))
    flags = DM_NOSYNC;
```

This one-line change would:

- ✅ Honor original `--nosync` intent during deferred activation
- ✅ Maintain backward compatibility
- ✅ Require no metadata format changes
- ✅ Fix the logical inconsistency

## Verification

The LV_NOTSYNCED flag can be verified using:

```bash
lvs -o name,attr vg/lv_name
# Look for 'R' (RAID not-synced) vs 'r' (RAID synced) in position 7
```

## Use Case

This fix is particularly valuable for:

- **HPC storage**: Creating large RAID volumes on fresh NVMe namespaces
- **Automated deployment**: Scripts that create inactive LVs for later activation
- **Container orchestration**: Deferred volume activation workflows

## Alternative Workarounds

Until fixed, use two-stage creation:

```bash
# Stage 1: Create with immediate activation (honors --nosync)
lvcreate --type raid5 --nosync --activate y -L 1G -n test vg

# Stage 2: Deactivate for later use
lvchange -an vg/test
```
