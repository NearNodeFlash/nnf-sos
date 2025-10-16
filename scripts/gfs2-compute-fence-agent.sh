#!/bin/bash

# GFS2 Compute Node Fencing Agent
# This script fences a compute node by removing its access to specific GFS2 filesystems
# through the NNF-SOS Kubernetes infrastructure.

set -euo pipefail

# Configuration
KUBECTL_CMD="${KUBECTL_CMD:-kubectl}"
LOG_LEVEL="${LOG_LEVEL:-INFO}"
DRY_RUN="${DRY_RUN:-false}"

# Logging function
log() {
    local level="$1"
    shift
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] [$level] $*" >&2
}

log_info() {
    [[ "$LOG_LEVEL" =~ ^(DEBUG|INFO)$ ]] && log "INFO" "$@"
}

log_warn() {
    [[ "$LOG_LEVEL" =~ ^(DEBUG|INFO|WARN)$ ]] && log "WARN" "$@"
}

log_error() {
    log "ERROR" "$@"
}

log_debug() {
    [[ "$LOG_LEVEL" == "DEBUG" ]] && log "DEBUG" "$@"
}

# Usage function
usage() {
    cat << EOF
Usage: $0 [OPTIONS] COMMAND COMPUTE_NODE [GFS2_FILESYSTEM]

COMMANDS:
    fence       - Remove compute node access to GFS2 filesystem(s)
    unfence     - Restore compute node access to GFS2 filesystem(s)  
    status      - Show current fence status for compute node
    list-gfs2   - List all GFS2 filesystems accessible to compute node

OPTIONS:
    -n, --namespace NAMESPACE    Limit operations to specific namespace
    -f, --filesystem FILESYSTEM  Target specific GFS2 filesystem (alternative to positional arg)
    -d, --dry-run               Show what would be done without making changes
    -v, --verbose               Enable verbose logging
    -h, --help                  Show this help message

EXAMPLES:
    # Fence compute node from all GFS2 filesystems
    $0 fence x1000c0s0b0n0

    # Fence compute node from specific GFS2 filesystem
    $0 fence x1000c0s0b0n0 my-gfs2-storage

    # Show status for compute node
    $0 status x1000c0s0b0n0

    # List all GFS2 filesystems for compute node
    $0 list-gfs2 x1000c0s0b0n0

ENVIRONMENT VARIABLES:
    KUBECTL_CMD     - kubectl command to use (default: kubectl)
    LOG_LEVEL       - Logging level: DEBUG, INFO, WARN, ERROR (default: INFO)
    DRY_RUN         - Set to 'true' to enable dry-run mode (default: false)
EOF
}

# Parse command line arguments
parse_args() {
    local command=""
    local compute_node=""
    local gfs2_filesystem=""
    local namespace=""
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            -n|--namespace)
                namespace="$2"
                shift 2
                ;;
            -f|--filesystem)
                gfs2_filesystem="$2"
                shift 2
                ;;
            -d|--dry-run)
                DRY_RUN="true"
                shift
                ;;
            -v|--verbose)
                LOG_LEVEL="DEBUG"
                shift
                ;;
            -h|--help)
                usage
                exit 0
                ;;
            fence|unfence|status|list-gfs2)
                command="$1"
                shift
                ;;
            *)
                if [[ -z "$compute_node" ]]; then
                    compute_node="$1"
                elif [[ -z "$gfs2_filesystem" ]]; then
                    gfs2_filesystem="$1"
                else
                    log_error "Unknown argument: $1"
                    usage
                    exit 1
                fi
                shift
                ;;
        esac
    done
    
    if [[ -z "$command" ]]; then
        log_error "Command is required"
        usage
        exit 1
    fi
    
    if [[ -z "$compute_node" ]] && [[ "$command" != "list-gfs2" ]]; then
        log_error "Compute node is required"
        usage
        exit 1
    fi
    
    # Export for use in other functions
    export COMMAND="$command"
    export COMPUTE_NODE="$compute_node"
    export GFS2_FILESYSTEM="$gfs2_filesystem"
    export TARGET_NAMESPACE="$namespace"
}

# Get all GFS2 NnfStorage resources
get_gfs2_storages() {
    local namespace_filter=""
    if [[ -n "${TARGET_NAMESPACE:-}" ]]; then
        namespace_filter="--namespace $TARGET_NAMESPACE"
    else
        namespace_filter="-A"
    fi
    
    $KUBECTL_CMD get nnfstorage $namespace_filter -o json | jq -r '
        .items[] | 
        select(.spec.fileSystemType == "gfs2") |
        "\(.metadata.namespace)/\(.metadata.name)"
    '
}

# Get NnfNodeBlockStorage resources for a specific GFS2 storage and compute node
get_nnfnodeblockstorage_for_gfs2() {
    local gfs2_storage="$1"
    local compute_node="$2"
    local gfs2_namespace="${gfs2_storage%/*}"
    local gfs2_name="${gfs2_storage#*/}"
    
    log_debug "Looking for NnfNodeBlockStorage for GFS2: $gfs2_storage, Compute: $compute_node"
    
    # Step 1: Find NnfNodeStorage resources that reference this GFS2 storage
    local node_storages
    node_storages=$($KUBECTL_CMD get nnfnodestorage -A -o json | jq -r --arg fs "$gfs2_name" --arg ns "$gfs2_namespace" '
        .items[] | 
        select(.metadata.ownerReferences[]? | select(.name == $fs and .kind == "NnfStorage")) |
        select(.spec.fileSystemType == "gfs2") |
        "\(.metadata.namespace)/\(.metadata.name):\(.spec.blockReference.namespace)/\(.spec.blockReference.name)"
    ')
    
    if [[ -z "$node_storages" ]]; then
        log_debug "No NnfNodeStorage found for GFS2 storage: $gfs2_storage"
        return 0
    fi
    
    # Step 2: For each NnfNodeStorage, check its NnfNodeBlockStorage for compute node access
    while IFS=':' read -r node_storage block_ref; do
        if [[ -n "$block_ref" ]]; then
            log_debug "Checking NnfNodeBlockStorage: $block_ref for compute node: $compute_node"
            
            local block_namespace="${block_ref%/*}"
            local block_name="${block_ref#*/}"
            
            $KUBECTL_CMD get nnfnodeblockstorage "$block_name" -n "$block_namespace" -o json 2>/dev/null | jq -r --arg compute "$compute_node" '
                select(.status.allocations[]?.accesses[$compute]?) |
                {
                    nodeBlockStorage: "\(.metadata.namespace)/\(.metadata.name)",
                    computeNode: $compute,
                    allocations: [
                        .status.allocations[] | 
                        select(.accesses[$compute]?) | 
                        {
                            index: (. as $item | [.] | keys[0]),
                            storageGroupId: .accesses[$compute].storageGroupId,
                            devicePaths: .accesses[$compute].devicePaths
                        }
                    ]
                } | 
                "\(.nodeBlockStorage):\(.allocations | length):\(.allocations | map(.storageGroupId) | join(","))"
            ' || true
        fi
    done <<< "$node_storages"
}

# List all GFS2 filesystems accessible to a compute node
list_gfs2_for_compute() {
    local compute_node="$1"
    
    log_info "Listing GFS2 filesystems accessible to compute node: $compute_node"
    
    local gfs2_storages
    gfs2_storages=$(get_gfs2_storages)
    
    if [[ -z "$gfs2_storages" ]]; then
        log_info "No GFS2 storage found"
        return 0
    fi
    
    echo "GFS2_FILESYSTEM NAMESPACE/NNFNODEBLOCKSTORAGE STORAGE_GROUPS_COUNT STORAGE_GROUP_IDS"
    echo "=================================================================="
    
    while read -r gfs2_storage; do
        if [[ -n "$gfs2_storage" ]]; then
            local result
            result=$(get_nnfnodeblockstorage_for_gfs2 "$gfs2_storage" "$compute_node")
            
            if [[ -n "$result" ]]; then
                while IFS=':' read -r nodeblock count groups; do
                    if [[ -n "$nodeblock" && -n "$count" ]]; then
                        echo "$gfs2_storage $nodeblock $count $groups"
                    fi
                done <<< "$result"
            fi
        fi
    done <<< "$gfs2_storages"
}

# Get fence status for a compute node
get_fence_status() {
    local compute_node="$1"
    
    log_info "Getting fence status for compute node: $compute_node"
    
    if [[ -n "${GFS2_FILESYSTEM:-}" ]]; then
        # Check specific filesystem
        local result
        result=$(get_nnfnodeblockstorage_for_gfs2 "$GFS2_FILESYSTEM" "$compute_node")
        
        if [[ -n "$result" ]]; then
            echo "UNFENCED: $compute_node has access to GFS2 filesystem: $GFS2_FILESYSTEM"
            while IFS=':' read -r nodeblock count groups; do
                echo "  NnfNodeBlockStorage: $nodeblock"
                echo "  StorageGroups: $groups"
            done <<< "$result"
        else
            echo "FENCED: $compute_node has no access to GFS2 filesystem: $GFS2_FILESYSTEM"
        fi
    else
        # Check all GFS2 filesystems
        local found_access=false
        local gfs2_storages
        gfs2_storages=$(get_gfs2_storages)
        
        while read -r gfs2_storage; do
            if [[ -n "$gfs2_storage" ]]; then
                local result
                result=$(get_nnfnodeblockstorage_for_gfs2 "$gfs2_storage" "$compute_node")
                
                if [[ -n "$result" ]]; then
                    found_access=true
                    echo "UNFENCED: $compute_node has access to GFS2 filesystem: $gfs2_storage"
                fi
            fi
        done <<< "$gfs2_storages"
        
        if [[ "$found_access" == "false" ]]; then
            echo "FENCED: $compute_node has no access to any GFS2 filesystems"
        fi
    fi
}

# Remove compute node access from NnfNodeBlockStorage (fencing)
fence_compute_node() {
    local compute_node="$1"
    local gfs2_filesystem="$2"
    
    log_info "Fencing compute node: $compute_node from GFS2 filesystem: $gfs2_filesystem"
    
    local result
    result=$(get_nnfnodeblockstorage_for_gfs2 "$gfs2_filesystem" "$compute_node")
    
    if [[ -z "$result" ]]; then
        log_info "Compute node $compute_node already fenced from $gfs2_filesystem (no access found)"
        return 0
    fi
    
    while IFS=':' read -r nodeblock count groups; do
        if [[ -n "$nodeblock" && -n "$count" && "$count" != "0" ]]; then
            local block_namespace="${nodeblock%/*}"
            local block_name="${nodeblock#*/}"
            
            log_info "Removing access for $compute_node from NnfNodeBlockStorage: $nodeblock"
            log_info "StorageGroups to be removed: $groups"
            
            if [[ "$DRY_RUN" == "true" ]]; then
                log_info "[DRY-RUN] Would remove $compute_node from $nodeblock access list"
            else
                # Get current access list and remove the compute node
                local access_list
                access_list=$($KUBECTL_CMD get nnfnodeblockstorage "$block_name" -n "$block_namespace" -o json | jq -r --arg compute "$compute_node" '
                    .spec.allocations[] | 
                    select(.access[]? == $compute) |
                    .access | 
                    map(select(. != $compute))
                ')
                
                if [[ -n "$access_list" ]]; then
                    # Find allocation index and update
                    local allocation_index
                    allocation_index=$($KUBECTL_CMD get nnfnodeblockstorage "$block_name" -n "$block_namespace" -o json | jq -r --arg compute "$compute_node" '
                        .spec.allocations | 
                        to_entries[] | 
                        select(.value.access[]? == $compute) | 
                        .key
                    ')
                    
                    if [[ -n "$allocation_index" ]]; then
                        log_info "Updating allocation index $allocation_index for $nodeblock"
                        
                        # Remove compute node from access list
                        $KUBECTL_CMD patch nnfnodeblockstorage "$block_name" -n "$block_namespace" --type='json' -p="[
                            {
                                \"op\": \"replace\",
                                \"path\": \"/spec/allocations/$allocation_index/access\",
                                \"value\": $access_list
                            }
                        ]"
                        
                        log_info "Successfully removed $compute_node from $nodeblock"
                    else
                        log_error "Could not find allocation index for $compute_node in $nodeblock"
                    fi
                else
                    log_warn "No access list found for $compute_node in $nodeblock"
                fi
            fi
        fi
    done <<< "$result"
}

# Add compute node access to NnfNodeBlockStorage (unfencing)
unfence_compute_node() {
    local compute_node="$1"
    local gfs2_filesystem="$2"
    
    log_info "Unfencing compute node: $compute_node for GFS2 filesystem: $gfs2_filesystem"
    
    # This is more complex as it requires understanding the original access patterns
    # For now, log that this functionality needs to be implemented based on specific requirements
    log_warn "Unfencing functionality not yet implemented"
    log_warn "Manual intervention required to restore access for $compute_node to $gfs2_filesystem"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "[DRY-RUN] Would restore $compute_node access to $gfs2_filesystem"
    fi
}

# Main execution
main() {
    parse_args "$@"
    
    log_info "Starting GFS2 Compute Fence Agent"
    log_info "Command: $COMMAND"
    log_info "Compute Node: ${COMPUTE_NODE:-N/A}"
    log_info "GFS2 Filesystem: ${GFS2_FILESYSTEM:-ALL}"
    log_info "Target Namespace: ${TARGET_NAMESPACE:-ALL}"
    log_info "Dry Run: $DRY_RUN"
    
    case "$COMMAND" in
        "list-gfs2")
            if [[ -n "${COMPUTE_NODE:-}" ]]; then
                list_gfs2_for_compute "$COMPUTE_NODE"
            else
                get_gfs2_storages
            fi
            ;;
        "status")
            get_fence_status "$COMPUTE_NODE"
            ;;
        "fence")
            if [[ -n "${GFS2_FILESYSTEM:-}" ]]; then
                fence_compute_node "$COMPUTE_NODE" "$GFS2_FILESYSTEM"
            else
                # Fence from all GFS2 filesystems
                local gfs2_storages
                gfs2_storages=$(get_gfs2_storages)
                while read -r gfs2_storage; do
                    if [[ -n "$gfs2_storage" ]]; then
                        fence_compute_node "$COMPUTE_NODE" "$gfs2_storage"
                    fi
                done <<< "$gfs2_storages"
            fi
            ;;
        "unfence")
            if [[ -n "${GFS2_FILESYSTEM:-}" ]]; then
                unfence_compute_node "$COMPUTE_NODE" "$GFS2_FILESYSTEM"
            else
                log_error "Unfencing requires specific GFS2 filesystem (-f option)"
                exit 1
            fi
            ;;
        *)
            log_error "Unknown command: $COMMAND"
            usage
            exit 1
            ;;
    esac
    
    log_info "GFS2 Compute Fence Agent completed"
}

# Execute main function with all arguments
main "$@"