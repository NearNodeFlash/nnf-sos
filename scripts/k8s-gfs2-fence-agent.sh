#!/bin/bash

# Kubernetes-native GFS2 Compute Node Fencing Agent
# This script creates ComputeNodeFenceEvent resources to trigger fencing actions
# and monitors their status.

set -euo pipefail

# Configuration
KUBECTL_CMD="${KUBECTL_CMD:-kubectl}"
FENCE_NAMESPACE="${FENCE_NAMESPACE:-nnf-sos}"
LOG_LEVEL="${LOG_LEVEL:-INFO}"
TIMEOUT="${TIMEOUT:-300}"

# Logging function
log() {
    local level="$1"
    shift
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] [$level] $*" >&2
}

log_info() {
    [[ "$LOG_LEVEL" =~ ^(DEBUG|INFO)$ ]] && log "INFO" "$@"
}

log_error() {
    log "ERROR" "$@"
}

# Usage function
usage() {
    cat << EOF
Usage: $0 [OPTIONS] COMMAND COMPUTE_NODE [GFS2_FILESYSTEM] [REASON]

COMMANDS:
    fence       - Create fence event for compute node
    unfence     - Create unfence event for compute node
    status      - Check status of fence events for compute node
    wait        - Wait for fence event to complete
    list        - List all fence events

OPTIONS:
    -n, --namespace NAMESPACE    Namespace for fence events (default: nnf-sos)
    -t, --timeout SECONDS       Timeout in seconds (default: 300)
    -p, --priority PRIORITY     Priority 0-100 (default: 50)
    -w, --wait                  Wait for fence action to complete
    -h, --help                  Show this help message

EXAMPLES:
    # Fence compute node from all GFS2 filesystems
    $0 fence x1000c0s0b0n0 "" "GFS2 lock contention detected"

    # Fence compute node from specific GFS2 filesystem  
    $0 fence x1000c0s0b0n0 my-gfs2-storage "Node became unresponsive"

    # Check status of fence events
    $0 status x1000c0s0b0n0

    # Wait for fence to complete
    $0 wait x1000c0s0b0n0

ENVIRONMENT VARIABLES:
    KUBECTL_CMD      - kubectl command to use (default: kubectl)
    FENCE_NAMESPACE  - Namespace for fence events (default: nnf-sos)
    LOG_LEVEL        - Logging level: DEBUG, INFO, WARN, ERROR (default: INFO)
    TIMEOUT          - Default timeout in seconds (default: 300)
EOF
}

# Generate fence event name
generate_fence_event_name() {
    local compute_node="$1"
    local action="$2"
    local timestamp=$(date +%s)
    echo "${action}-${compute_node}-${timestamp}"
}

# Create ComputeNodeFenceEvent resource
create_fence_event() {
    local compute_node="$1"
    local action="$2"
    local gfs2_filesystem="${3:-}"
    local reason="${4:-Fence event triggered by external agent}"
    local priority="${5:-50}"
    local timeout="${6:-$TIMEOUT}"
    
    local event_name
    event_name=$(generate_fence_event_name "$compute_node" "$action")
    
    log_info "Creating $action event for compute node: $compute_node"
    
    local gfs2_field=""
    if [[ -n "$gfs2_filesystem" ]]; then
        gfs2_field="  gfs2Filesystem: \"$gfs2_filesystem\""
    fi
    
    $KUBECTL_CMD apply -f - << EOF
apiVersion: nnf.cray.hpe.com/v1alpha1
kind: ComputeNodeFenceEvent
metadata:
  name: $event_name
  namespace: $FENCE_NAMESPACE
  labels:
    compute-node: $compute_node
    action: $action
spec:
  computeNodeName: $compute_node
  action: $action
$gfs2_field
  reason: "$reason"
  priority: $priority
  timeoutSeconds: $timeout
EOF
    
    echo "Created fence event: $event_name"
    return 0
}

# Get fence event status
get_fence_status() {
    local compute_node="$1"
    
    log_info "Getting fence event status for compute node: $compute_node"
    
    $KUBECTL_CMD get computenodefenceevents -n "$FENCE_NAMESPACE" \
        -l "compute-node=$compute_node" \
        -o custom-columns="NAME:.metadata.name,ACTION:.spec.action,STATE:.status.state,GFS2:.spec.gfs2Filesystem,AGE:.metadata.creationTimestamp" \
        --sort-by='.metadata.creationTimestamp'
}

# Wait for fence event to complete
wait_for_fence_completion() {
    local compute_node="$1"
    local timeout="${2:-$TIMEOUT}"
    
    log_info "Waiting for fence events to complete for compute node: $compute_node (timeout: ${timeout}s)"
    
    local start_time=$(date +%s)
    local end_time=$((start_time + timeout))
    
    while [[ $(date +%s) -lt $end_time ]]; do
        local pending_events
        pending_events=$($KUBECTL_CMD get computenodefenceevents -n "$FENCE_NAMESPACE" \
            -l "compute-node=$compute_node" \
            -o jsonpath='{.items[?(@.status.state=="pending")].metadata.name}' 2>/dev/null || echo "")
        
        local in_progress_events
        in_progress_events=$($KUBECTL_CMD get computenodefenceevents -n "$FENCE_NAMESPACE" \
            -l "compute-node=$compute_node" \
            -o jsonpath='{.items[?(@.status.state=="in-progress")].metadata.name}' 2>/dev/null || echo "")
        
        if [[ -z "$pending_events" && -z "$in_progress_events" ]]; then
            log_info "All fence events completed for compute node: $compute_node"
            
            # Show final status
            get_fence_status "$compute_node"
            return 0
        fi
        
        log_info "Waiting... Pending: [$pending_events] In-Progress: [$in_progress_events]"
        sleep 5
    done
    
    log_error "Timeout waiting for fence events to complete"
    get_fence_status "$compute_node"
    return 1
}

# List all fence events
list_fence_events() {
    log_info "Listing all fence events"
    
    $KUBECTL_CMD get computenodefenceevents -n "$FENCE_NAMESPACE" \
        -o custom-columns="NAME:.metadata.name,COMPUTE_NODE:.spec.computeNodeName,ACTION:.spec.action,STATE:.status.state,GFS2:.spec.gfs2Filesystem,REASON:.spec.reason,AGE:.metadata.creationTimestamp" \
        --sort-by='.metadata.creationTimestamp'
}

# Parse command line arguments
parse_args() {
    local command=""
    local compute_node=""
    local gfs2_filesystem=""
    local reason=""
    local priority="50"
    local wait_flag="false"
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            -n|--namespace)
                FENCE_NAMESPACE="$2"
                shift 2
                ;;
            -t|--timeout)
                TIMEOUT="$2"
                shift 2
                ;;
            -p|--priority)
                priority="$2"
                shift 2
                ;;
            -w|--wait)
                wait_flag="true"
                shift
                ;;
            -h|--help)
                usage
                exit 0
                ;;
            fence|unfence|status|wait|list)
                command="$1"
                shift
                ;;
            *)
                if [[ -z "$compute_node" ]]; then
                    compute_node="$1"
                elif [[ -z "$gfs2_filesystem" ]]; then
                    gfs2_filesystem="$1"
                elif [[ -z "$reason" ]]; then
                    reason="$1"
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
    
    if [[ -z "$compute_node" ]] && [[ "$command" != "list" ]]; then
        log_error "Compute node is required for command: $command"
        usage
        exit 1
    fi
    
    # Export for use in other functions
    export COMMAND="$command"
    export COMPUTE_NODE="$compute_node"
    export GFS2_FILESYSTEM="$gfs2_filesystem"
    export FENCE_REASON="$reason"
    export FENCE_PRIORITY="$priority"
    export WAIT_FLAG="$wait_flag"
}

# Main execution
main() {
    parse_args "$@"
    
    log_info "Starting Kubernetes GFS2 Fence Agent"
    log_info "Command: $COMMAND"
    log_info "Compute Node: ${COMPUTE_NODE:-N/A}"
    log_info "GFS2 Filesystem: ${GFS2_FILESYSTEM:-ALL}"
    log_info "Fence Namespace: $FENCE_NAMESPACE"
    log_info "Timeout: ${TIMEOUT}s"
    
    case "$COMMAND" in
        "fence"|"unfence")
            create_fence_event "$COMPUTE_NODE" "$COMMAND" "$GFS2_FILESYSTEM" "$FENCE_REASON" "$FENCE_PRIORITY" "$TIMEOUT"
            
            if [[ "$WAIT_FLAG" == "true" ]]; then
                echo "Waiting for fence event to complete..."
                wait_for_fence_completion "$COMPUTE_NODE" "$TIMEOUT"
            fi
            ;;
        "status")
            get_fence_status "$COMPUTE_NODE"
            ;;
        "wait")
            wait_for_fence_completion "$COMPUTE_NODE" "$TIMEOUT"
            ;;
        "list")
            list_fence_events
            ;;
        *)
            log_error "Unknown command: $COMMAND"
            usage
            exit 1
            ;;
    esac
    
    log_info "Kubernetes GFS2 Fence Agent completed"
}

# Execute main function with all arguments
main "$@"