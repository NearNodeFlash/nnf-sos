#!/bin/bash

# GFS2 Fencing Integration Example
# This script demonstrates how to integrate the fencing agent with your GFS2 monitoring system

set -euo pipefail

# Configuration
FENCE_AGENT="${FENCE_AGENT:-/usr/local/bin/k8s-gfs2-fence-agent.sh}"
LOG_FILE="${LOG_FILE:-/var/log/gfs2-fence-integration.log}"
FENCE_TIMEOUT="${FENCE_TIMEOUT:-300}"

# Logging function
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" | tee -a "$LOG_FILE"
}

# Function to handle GFS2 lock contention
handle_lock_contention() {
    local compute_node="$1"
    local gfs2_filesystem="$2"
    local lock_details="$3"
    
    log "INFO: GFS2 lock contention detected"
    log "INFO: Compute Node: $compute_node"
    log "INFO: GFS2 Filesystem: $gfs2_filesystem"
    log "INFO: Lock Details: $lock_details"
    
    # Fence the problematic compute node
    local reason="GFS2 lock contention detected: $lock_details"
    
    log "INFO: Initiating fence operation for $compute_node"
    
    if "$FENCE_AGENT" fence "$compute_node" "$gfs2_filesystem" "$reason" --wait --timeout "$FENCE_TIMEOUT"; then
        log "INFO: Successfully fenced $compute_node from $gfs2_filesystem"
        
        # Optionally send alerts or notifications
        send_fence_notification "$compute_node" "$gfs2_filesystem" "SUCCESS" "$reason"
        
        return 0
    else
        log "ERROR: Failed to fence $compute_node from $gfs2_filesystem"
        
        # Send alert about fence failure
        send_fence_notification "$compute_node" "$gfs2_filesystem" "FAILED" "$reason"
        
        return 1
    fi
}

# Function to handle unresponsive nodes
handle_unresponsive_node() {
    local compute_node="$1"
    local detection_method="$2"
    
    log "INFO: Unresponsive node detected: $compute_node (via $detection_method)"
    
    # Get all GFS2 filesystems this node has access to
    local gfs2_filesystems
    gfs2_filesystems=$("$FENCE_AGENT" list 2>/dev/null | grep "$compute_node" | awk '{print $1}' || echo "")
    
    if [[ -z "$gfs2_filesystems" ]]; then
        log "INFO: No GFS2 filesystems found for $compute_node, no fencing needed"
        return 0
    fi
    
    local reason="Node became unresponsive (detected via $detection_method)"
    
    # Fence from all GFS2 filesystems
    log "INFO: Fencing $compute_node from all GFS2 filesystems: $gfs2_filesystems"
    
    if "$FENCE_AGENT" fence "$compute_node" "" "$reason" --wait --timeout "$FENCE_TIMEOUT"; then
        log "INFO: Successfully fenced $compute_node from all GFS2 filesystems"
        
        # Send notification
        send_fence_notification "$compute_node" "ALL" "SUCCESS" "$reason"
        
        return 0
    else
        log "ERROR: Failed to fence $compute_node from GFS2 filesystems"
        
        # Send alert
        send_fence_notification "$compute_node" "ALL" "FAILED" "$reason"
        
        return 1
    fi
}

# Function to send fence notifications
send_fence_notification() {
    local compute_node="$1"
    local filesystem="$2"
    local status="$3"
    local reason="$4"
    
    # Example notification - adapt to your alerting system
    local message="GFS2 Fence Event: $status - Node: $compute_node, Filesystem: $filesystem, Reason: $reason"
    
    # Log the notification
    log "NOTIFICATION: $message"
    
    # Send to your alerting system (examples)
    # echo "$message" | mail -s "GFS2 Fence Alert" admin@example.com
    # curl -X POST -H 'Content-type: application/json' --data "{\"text\":\"$message\"}" "$SLACK_WEBHOOK_URL"
    # kubectl create event --for="node/$compute_node" --reason="GFS2Fence" --message="$message"
}

# Function to check if fence is still needed
check_fence_still_needed() {
    local compute_node="$1"
    local gfs2_filesystem="$2"
    
    # Add your logic here to determine if the node still needs to be fenced
    # For example, check if the node is responsive again
    
    # Example check - ping the node
    if ping -c 1 -W 5 "$compute_node" >/dev/null 2>&1; then
        log "INFO: $compute_node is now responsive, fence may not be needed"
        return 1  # Fence not needed
    else
        log "INFO: $compute_node is still unresponsive, fence is needed"
        return 0  # Fence needed
    fi
}

# Function to monitor GFS2 health and trigger fencing
monitor_gfs2_health() {
    log "INFO: Starting GFS2 health monitoring"
    
    while true; do
        # Check for GFS2 lock contentions
        # This is an example - adapt to your GFS2 monitoring method
        
        # Example: Parse GFS2 lock dumps
        if command -v gfs2_tool >/dev/null 2>&1; then
            # Check for stuck locks (example implementation)
            local stuck_locks
            stuck_locks=$(timeout 30 gfs2_tool lockdump /gfs2/filesystem 2>/dev/null | grep -E "STUCK|TIMEOUT" || echo "")
            
            if [[ -n "$stuck_locks" ]]; then
                log "WARNING: Detected stuck GFS2 locks: $stuck_locks"
                
                # Parse the output to identify problematic nodes
                # This is highly dependent on your GFS2 setup and lock dump format
                # Example parsing (adapt to your environment):
                local problematic_node
                problematic_node=$(echo "$stuck_locks" | grep -oE "node[0-9]+" | head -1 || echo "")
                
                if [[ -n "$problematic_node" ]]; then
                    handle_lock_contention "$problematic_node" "/gfs2/filesystem" "$stuck_locks"
                fi
            fi
        fi
        
        # Check for unresponsive nodes via other methods
        # Example: Check SLURM node state
        if command -v sinfo >/dev/null 2>&1; then
            local down_nodes
            down_nodes=$(sinfo -h -N -t down -o "%N" 2>/dev/null || echo "")
            
            for node in $down_nodes; do
                if [[ -n "$node" ]]; then
                    handle_unresponsive_node "$node" "SLURM"
                fi
            done
        fi
        
        # Sleep between checks
        sleep 60
    done
}

# Usage function
usage() {
    cat << EOF
Usage: $0 COMMAND [OPTIONS]

COMMANDS:
    monitor                           - Start continuous GFS2 monitoring
    fence-lock-contention NODE FS    - Handle GFS2 lock contention
    fence-unresponsive NODE METHOD   - Handle unresponsive node
    check-fence-needed NODE FS       - Check if fence is still needed
    test-notification NODE FS STATUS - Test notification system

EXAMPLES:
    # Start monitoring
    $0 monitor

    # Handle specific lock contention
    $0 fence-lock-contention x1000c0s0b0n0 /gfs2/scratch "Lock timeout on inode 12345"

    # Handle unresponsive node
    $0 fence-unresponsive x1000c0s0b0n0 "heartbeat-timeout"

    # Test notification
    $0 test-notification x1000c0s0b0n0 my-gfs2-fs SUCCESS "Test message"

ENVIRONMENT VARIABLES:
    FENCE_AGENT     - Path to fence agent script
    LOG_FILE        - Path to log file
    FENCE_TIMEOUT   - Timeout for fence operations in seconds
EOF
}

# Main execution
case "${1:-}" in
    "monitor")
        monitor_gfs2_health
        ;;
    "fence-lock-contention")
        if [[ $# -lt 4 ]]; then
            echo "Error: fence-lock-contention requires NODE FS DETAILS arguments"
            usage
            exit 1
        fi
        handle_lock_contention "$2" "$3" "$4"
        ;;
    "fence-unresponsive")
        if [[ $# -lt 3 ]]; then
            echo "Error: fence-unresponsive requires NODE METHOD arguments"
            usage
            exit 1
        fi
        handle_unresponsive_node "$2" "$3"
        ;;
    "check-fence-needed")
        if [[ $# -lt 3 ]]; then
            echo "Error: check-fence-needed requires NODE FS arguments"
            usage
            exit 1
        fi
        check_fence_still_needed "$2" "$3"
        ;;
    "test-notification")
        if [[ $# -lt 5 ]]; then
            echo "Error: test-notification requires NODE FS STATUS REASON arguments"
            usage
            exit 1
        fi
        send_fence_notification "$2" "$3" "$4" "$5"
        ;;
    *)
        echo "Error: Unknown command: ${1:-}"
        usage
        exit 1
        ;;
esac