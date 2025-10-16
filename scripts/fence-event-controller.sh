#!/bin/bash

# Simple Fence Event Controller
# Watches ComputeNodeFenceEvent resources and executes fencing operations

set -euo pipefail

# Configuration
KUBECTL_CMD="${KUBECTL_CMD:-kubectl}"
FENCE_NAMESPACE="${FENCE_NAMESPACE:-nnf-sos}"
POLL_INTERVAL="${POLL_INTERVAL:-10}"
LOG_LEVEL="${LOG_LEVEL:-INFO}"
FENCE_SCRIPT="${FENCE_SCRIPT:-/usr/local/bin/gfs2-compute-fence-agent.sh}"

# Logging function
log() {
    local level="$1"
    shift
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] [$level] [Controller] $*" >&2
}

log_info() {
    [[ "$LOG_LEVEL" =~ ^(DEBUG|INFO)$ ]] && log "INFO" "$@"
}

log_error() {
    log "ERROR" "$@"
}

log_debug() {
    [[ "$LOG_LEVEL" == "DEBUG" ]] && log "DEBUG" "$@"
}

# Update fence event status
update_fence_event_status() {
    local event_name="$1"
    local state="$2"
    local message="$3"
    local affected_resources="${4:-[]}"
    
    local current_time=$(date -u '+%Y-%m-%dT%H:%M:%SZ')
    
    log_debug "Updating fence event $event_name status to $state"
    
    local patch_data
    if [[ "$state" == "in-progress" ]]; then
        patch_data="{
            \"status\": {
                \"state\": \"$state\",
                \"message\": \"$message\",
                \"startTime\": \"$current_time\",
                \"lastTransitionTime\": \"$current_time\",
                \"affectedResources\": $affected_resources
            }
        }"
    elif [[ "$state" == "completed" || "$state" == "failed" || "$state" == "timeout" ]]; then
        patch_data="{
            \"status\": {
                \"state\": \"$state\",
                \"message\": \"$message\",
                \"completionTime\": \"$current_time\",
                \"lastTransitionTime\": \"$current_time\",
                \"affectedResources\": $affected_resources
            }
        }"
    else
        patch_data="{
            \"status\": {
                \"state\": \"$state\",
                \"message\": \"$message\",
                \"lastTransitionTime\": \"$current_time\"
            }
        }"
    fi
    
    $KUBECTL_CMD patch computenodefenceevent "$event_name" -n "$FENCE_NAMESPACE" \
        --type='merge' -p "$patch_data" || {
        log_error "Failed to update status for fence event: $event_name"
        return 1
    }
}

# Execute fence operation using the fence script
execute_fence_operation() {
    local event_name="$1"
    local compute_node="$2"
    local action="$3"
    local gfs2_filesystem="$4"
    local timeout="$5"
    
    log_info "Executing $action operation for compute node: $compute_node, filesystem: ${gfs2_filesystem:-ALL}"
    
    # Update status to in-progress
    update_fence_event_status "$event_name" "in-progress" "Executing $action operation for $compute_node"
    
    # Build fence script command
    local fence_cmd="$FENCE_SCRIPT $action $compute_node"
    if [[ -n "$gfs2_filesystem" ]]; then
        fence_cmd="$fence_cmd $gfs2_filesystem"
    fi
    
    log_debug "Executing: $fence_cmd"
    
    # Execute with timeout
    local output
    local exit_code=0
    
    if timeout "$timeout" bash -c "$fence_cmd" > /tmp/fence_output_$$ 2>&1; then
        output=$(cat /tmp/fence_output_$$ || echo "No output")
        rm -f /tmp/fence_output_$$
        
        log_info "Fence operation completed successfully for $compute_node"
        
        # Get list of affected NnfNodeBlockStorage resources
        local affected_resources
        affected_resources=$(get_affected_resources "$compute_node" "$gfs2_filesystem")
        
        update_fence_event_status "$event_name" "completed" "Successfully ${action}d $compute_node. Output: $output" "$affected_resources"
    else
        exit_code=$?
        output=$(cat /tmp/fence_output_$$ 2>/dev/null || echo "No output")
        rm -f /tmp/fence_output_$$
        
        if [[ $exit_code == 124 ]]; then
            log_error "Fence operation timed out for $compute_node"
            update_fence_event_status "$event_name" "timeout" "Fence operation timed out after ${timeout}s. Output: $output"
        else
            log_error "Fence operation failed for $compute_node (exit code: $exit_code)"
            update_fence_event_status "$event_name" "failed" "Fence operation failed with exit code $exit_code. Output: $output"
        fi
        return 1
    fi
}

# Get affected NnfNodeBlockStorage resources for reporting
get_affected_resources() {
    local compute_node="$1"
    local gfs2_filesystem="$2"
    
    # This would use similar logic to the fence script to identify affected resources
    # For now, return empty array - this would be populated by actual resource discovery
    echo "[]"
}

# Process a single fence event
process_fence_event() {
    local event_name="$1"
    
    log_debug "Processing fence event: $event_name"
    
    # Get fence event details
    local event_json
    event_json=$($KUBECTL_CMD get computenodefenceevent "$event_name" -n "$FENCE_NAMESPACE" -o json 2>/dev/null || {
        log_error "Failed to get fence event: $event_name"
        return 1
    })
    
    # Extract event details
    local compute_node
    local action
    local gfs2_filesystem
    local timeout
    local current_state
    
    compute_node=$(echo "$event_json" | jq -r '.spec.computeNodeName')
    action=$(echo "$event_json" | jq -r '.spec.action')
    gfs2_filesystem=$(echo "$event_json" | jq -r '.spec.gfs2Filesystem // ""')
    timeout=$(echo "$event_json" | jq -r '.spec.timeoutSeconds // 300')
    current_state=$(echo "$event_json" | jq -r '.status.state // "pending"')
    
    # Skip if not pending
    if [[ "$current_state" != "pending" ]]; then
        log_debug "Skipping fence event $event_name (state: $current_state)"
        return 0
    fi
    
    log_info "Processing $action fence event for compute node: $compute_node"
    
    # Check if fence script exists
    if [[ ! -x "$FENCE_SCRIPT" ]]; then
        log_error "Fence script not found or not executable: $FENCE_SCRIPT"
        update_fence_event_status "$event_name" "failed" "Fence script not found: $FENCE_SCRIPT"
        return 1
    fi
    
    # Execute the fence operation
    execute_fence_operation "$event_name" "$compute_node" "$action" "$gfs2_filesystem" "$timeout"
}

# Main controller loop
controller_loop() {
    log_info "Starting fence event controller loop"
    log_info "Namespace: $FENCE_NAMESPACE"
    log_info "Poll Interval: ${POLL_INTERVAL}s"
    log_info "Fence Script: $FENCE_SCRIPT"
    
    while true; do
        log_debug "Checking for pending fence events..."
        
        # Get all pending fence events
        local pending_events
        pending_events=$($KUBECTL_CMD get computenodefenceevents -n "$FENCE_NAMESPACE" \
            -o jsonpath='{.items[?(@.status.state=="pending")].metadata.name}' 2>/dev/null || echo "")
        
        if [[ -n "$pending_events" ]]; then
            log_info "Found pending fence events: $pending_events"
            
            # Process each pending event
            for event_name in $pending_events; do
                if [[ -n "$event_name" ]]; then
                    process_fence_event "$event_name" &
                fi
            done
            
            # Wait for background processes to complete
            wait
        else
            log_debug "No pending fence events found"
        fi
        
        # Check for stuck events (in-progress for too long)
        local current_time=$(date +%s)
        local stuck_events
        stuck_events=$($KUBECTL_CMD get computenodefenceevents -n "$FENCE_NAMESPACE" \
            -o json 2>/dev/null | jq -r --arg current_time "$current_time" '
            .items[] | 
            select(.status.state == "in-progress") |
            select(.status.startTime != null) |
            select(($current_time | tonumber) - (.status.startTime | fromdateiso8601) > (.spec.timeoutSeconds // 300)) |
            .metadata.name
        ' || echo "")
        
        if [[ -n "$stuck_events" ]]; then
            log_info "Found stuck fence events: $stuck_events"
            for event_name in $stuck_events; do
                if [[ -n "$event_name" ]]; then
                    log_error "Marking stuck fence event as timed out: $event_name"
                    update_fence_event_status "$event_name" "timeout" "Event stuck in in-progress state beyond timeout"
                fi
            done
        fi
        
        sleep "$POLL_INTERVAL"
    done
}

# Signal handlers for graceful shutdown
cleanup() {
    log_info "Received shutdown signal, cleaning up..."
    # Kill any background fence operations
    jobs -p | xargs -r kill 2>/dev/null || true
    log_info "Fence event controller shutdown complete"
    exit 0
}

trap cleanup SIGTERM SIGINT

# Usage
usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Simple Fence Event Controller - Watches ComputeNodeFenceEvent resources and executes fencing operations

OPTIONS:
    -n, --namespace NAMESPACE     Namespace to watch for fence events (default: nnf-sos)
    -i, --interval SECONDS        Poll interval in seconds (default: 10)
    -s, --script PATH             Path to fence script (default: /usr/local/bin/gfs2-compute-fence-agent.sh)
    -v, --verbose                 Enable verbose logging
    -h, --help                    Show this help message

ENVIRONMENT VARIABLES:
    KUBECTL_CMD       - kubectl command to use (default: kubectl)
    FENCE_NAMESPACE   - Namespace for fence events (default: nnf-sos)
    POLL_INTERVAL     - Poll interval in seconds (default: 10)
    LOG_LEVEL         - Logging level: DEBUG, INFO, WARN, ERROR (default: INFO)
    FENCE_SCRIPT      - Path to fence script (default: /usr/local/bin/gfs2-compute-fence-agent.sh)
EOF
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -n|--namespace)
            FENCE_NAMESPACE="$2"
            shift 2
            ;;
        -i|--interval)
            POLL_INTERVAL="$2"
            shift 2
            ;;
        -s|--script)
            FENCE_SCRIPT="$2"
            shift 2
            ;;
        -v|--verbose)
            LOG_LEVEL="DEBUG"
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            log_error "Unknown argument: $1"
            usage
            exit 1
            ;;
    esac
done

# Start the controller
log_info "Starting Simple Fence Event Controller"
controller_loop