# GFS2 Compute Node Fencing Agent

This fencing agent system provides targeted fencing capabilities for compute nodes accessing GFS2 filesystems in the NNF-SOS environment. When a GFS2 filesystem detects issues that require fencing, this system can selectively remove compute node access to specific storage resources.

## Overview

The fencing system consists of several components:

1. **ComputeNodeFenceEvent CRD** - Kubernetes custom resource for tracking fence events
2. **Direct Fence Agent** (`gfs2-compute-fence-agent.sh`) - Direct execution script for fencing operations
3. **Kubernetes Fence Agent** (`k8s-gfs2-fence-agent.sh`) - Creates fence events for controller processing
4. **Fence Event Controller** (`fence-event-controller.sh`) - Watches fence events and executes operations
5. **Deployment manifests** - Kubernetes deployment for the controller

## Key Features

- **Targeted Fencing**: Only affects specific GFS2 filesystems, not all storage
- **Resource Mapping**: Maps GFS2 filesystems to specific `NnfNodeBlockStorage` resources
- **StorageGroup Removal**: Removes specific StorageGroups that connect namespaces to compute endpoints
- **Event Tracking**: Full audit trail of fencing events and their outcomes
- **Kubernetes-native**: Integrates with existing NNF-SOS Kubernetes infrastructure

## How It Works

### Resource Relationship Chain

```text
GFS2 Filesystem Issue → Fencing Trigger
                     ↓
              Identify Compute Node
                     ↓
        Find NnfStorage (fileSystemType: gfs2)
                     ↓
        Find NnfNodeStorage (references NnfStorage)
                     ↓
    Find NnfNodeBlockStorage (via BlockReference)
                     ↓
    Remove Compute Node from Access List
                     ↓
    Controller Deletes StorageGroups
                     ↓
         Compute Node Fenced from GFS2
```

### StorageGroup Fencing Process

1. **Identification**: Map the problematic GFS2 filesystem to specific `NnfNodeBlockStorage` resources
2. **Access Removal**: Remove the compute node from `NnfNodeBlockStorage.Spec.Allocations[].Access`
3. **Automatic Cleanup**: The `NnfNodeBlockStorageReconciler` automatically deletes the corresponding StorageGroups
4. **Effect**: Compute node loses access to the specific NVMe namespaces for that GFS2 filesystem

## Installation

### 1. Deploy the CRD and RBAC

```bash
kubectl apply -f config/crd/computenodefenceevent-crd.yaml
```

### 2. Deploy the Controller (Optional)

For Kubernetes-native operation:

```bash
# Update the ConfigMap with actual script content first
kubectl apply -f config/deploy/gfs2-fence-controller.yaml
```

### 3. Install Scripts

For direct execution:

```bash
# Copy scripts to appropriate location
sudo cp scripts/gfs2-compute-fence-agent.sh /usr/local/bin/
sudo cp scripts/k8s-gfs2-fence-agent.sh /usr/local/bin/
sudo chmod +x /usr/local/bin/gfs2-compute-fence-agent.sh
sudo chmod +x /usr/local/bin/k8s-gfs2-fence-agent.sh
```

## Usage Examples

### Direct Fence Agent

```bash
# List all GFS2 filesystems accessible to a compute node
./gfs2-compute-fence-agent.sh list-gfs2 x1000c0s0b0n0

# Check fence status for a compute node
./gfs2-compute-fence-agent.sh status x1000c0s0b0n0

# Fence compute node from specific GFS2 filesystem
./gfs2-compute-fence-agent.sh fence x1000c0s0b0n0 my-gfs2-storage

# Fence compute node from all GFS2 filesystems
./gfs2-compute-fence-agent.sh fence x1000c0s0b0n0

# Dry run mode (show what would be done)
DRY_RUN=true ./gfs2-compute-fence-agent.sh fence x1000c0s0b0n0
```

### Kubernetes Fence Agent

```bash
# Create fence event for specific filesystem
./k8s-gfs2-fence-agent.sh fence x1000c0s0b0n0 my-gfs2-storage "GFS2 lock contention detected"

# Create fence event with wait for completion
./k8s-gfs2-fence-agent.sh fence x1000c0s0b0n0 "" "Node became unresponsive" --wait

# Check status of fence events
./k8s-gfs2-fence-agent.sh status x1000c0s0b0n0

# List all fence events
./k8s-gfs2-fence-agent.sh list
```

### Kubernetes Resource Operations

```bash
# List fence events
kubectl get computenodefenceevents -n nnf-sos

# Get detailed fence event info
kubectl describe computenodefenceevent fence-x1000c0s0b0n0-1634567890 -n nnf-sos

# Monitor fence event status
kubectl get computenodefenceevents -n nnf-sos -w
```

## Integration with GFS2

### Automatic Fencing Trigger

To integrate with your GFS2 monitoring system:

```bash
#!/bin/bash
# Example GFS2 monitoring integration
# This would be called when GFS2 detects an issue requiring fencing

PROBLEMATIC_NODE="$1"
GFS2_FILESYSTEM="$2"
REASON="$3"

# Create fence event
/usr/local/bin/k8s-gfs2-fence-agent.sh fence "$PROBLEMATIC_NODE" "$GFS2_FILESYSTEM" "$REASON" --wait

# Check if fencing was successful
if /usr/local/bin/k8s-gfs2-fence-agent.sh status "$PROBLEMATIC_NODE" | grep -q "FENCED"; then
    echo "Successfully fenced $PROBLEMATIC_NODE from $GFS2_FILESYSTEM"
    exit 0
else
    echo "Failed to fence $PROBLEMATIC_NODE from $GFS2_FILESYSTEM"
    exit 1
fi
```

## Query Commands

### Find GFS2 Filesystems for Compute Node

```bash
# Get all GFS2 filesystems
kubectl get nnfstorage -A -o json | jq -r '.items[] | select(.spec.fileSystemType == "gfs2") | "\(.metadata.namespace)/\(.metadata.name)"'

# Find NnfNodeBlockStorage resources accessible to compute node
COMPUTE_NODE="x1000c0s0b0n0"
kubectl get nnfnodeblockstorage -A -o json | jq -r --arg compute "$COMPUTE_NODE" '
.items[] | 
select(.status.allocations[]?.accesses[$compute]?) |
{
  nodeBlockStorage: "\(.metadata.namespace)/\(.metadata.name)",
  storageGroups: [
    .status.allocations[] | 
    select(.accesses[$compute]?) | 
    .accesses[$compute].storageGroupId
  ]
}'
```

### Map GFS2 Filesystem to Resources

```bash
# Map specific GFS2 filesystem to NnfNodeBlockStorage
GFS2_NAME="my-gfs2-storage"
GFS2_NAMESPACE="default"

kubectl get nnfnodestorage -A -o json | jq -r --arg fs "$GFS2_NAME" --arg ns "$GFS2_NAMESPACE" '
.items[] | 
select(.metadata.ownerReferences[]? | select(.name == $fs and .kind == "NnfStorage")) |
select(.spec.fileSystemType == "gfs2") |
{
  nodeStorage: "\(.metadata.namespace)/\(.metadata.name)",
  blockReference: "\(.spec.blockReference.namespace)/\(.spec.blockReference.name)"
}'
```

## Troubleshooting

### Common Issues

1. **No NnfNodeBlockStorage found**
   - Verify the GFS2 filesystem name and namespace
   - Check that the compute node actually has access to the filesystem
   - Ensure the `NnfStorage` resource exists and has `fileSystemType: gfs2`

2. **Access removal fails**
   - Check RBAC permissions for the fence agent
   - Verify the `NnfNodeBlockStorage` resource exists and is accessible
   - Check controller logs for detailed error messages

3. **Fence events stuck in pending**
   - Verify the fence event controller is running
   - Check controller logs for errors
   - Ensure the fence scripts are executable and in the correct location

### Debugging Commands

```bash
# Check controller logs
kubectl logs -n nnf-sos deployment/gfs2-fence-controller

# Check fence event details
kubectl get computenodefenceevents -n nnf-sos -o yaml

# Verify NnfNodeBlockStorage access lists
kubectl get nnfnodeblockstorage -A -o json | jq '.items[].spec.allocations[].access'

# Check StorageGroup status
kubectl get nnfnodeblockstorage -A -o json | jq '.items[].status.allocations[].accesses'
```

## Security Considerations

- The fence agent requires significant permissions to modify `NnfNodeBlockStorage` resources
- Use dedicated service accounts with minimal required permissions
- Monitor fence events for unauthorized fencing attempts
- Implement proper authentication for fence agent triggers

## Future Enhancements

- **Automatic Unfencing**: Implement logic to restore access when conditions improve
- **Policy-based Fencing**: Define fencing policies based on filesystem health metrics
- **Integration with Cluster Managers**: Direct integration with Slurm, PBS, etc.
- **Metrics and Alerting**: Prometheus metrics for fencing events and success rates