# ADR 0008: Pod Disk Fill Implementation

**Status**: Proposed
**Date**: 2025-12-18
**Author**: k8s-chaos team

## Context

Applications need to handle disk space exhaustion gracefully. Common scenarios include:
- Log file growth consuming disk space
- Temporary file accumulation
- Database storage limits
- Cache directory overflow
- User upload storage filling up

Currently, k8s-chaos supports CPU stress, memory stress, and network chaos, but lacks disk/storage chaos capabilities. This limits our ability to test:
- Disk space monitoring and alerting
- Application behavior when disk is full
- Log rotation effectiveness
- Cleanup job reliability
- PVC auto-expansion triggers

## Decision

Implement `pod-disk-fill` action using ephemeral containers to fill disk space in target pods.

### Implementation Approach

**Method**: Ephemeral Container Injection
- Use Kubernetes ephemeral containers feature (consistent with other stress actions)
- Inject a lightweight container (alpine or busybox) into target pods
- Create a large file using `fallocate` or `dd` to consume disk space
- Share the pod's filesystem namespace to fill actual pod storage
- Automatic cleanup when experiment completes (file deletion)

**Disk Fill Tool**: `fallocate` (preferred) with `dd` fallback
- `fallocate`: Fast, efficient, doesn't actually write data (just allocates space)
- `dd`: Fallback for filesystems that don't support fallocate (e.g., tmpfs)
- Both available in standard alpine/busybox images

### Configuration Parameters

Add to ChaosExperimentSpec:

```go
// FillPercentage specifies the percentage of disk space to fill (for pod-disk-fill)
// Range: 50-95. Conservative limit to prevent total disk exhaustion.
// +kubebuilder:validation:Minimum=50
// +kubebuilder:validation:Maximum=95
// +kubebuilder:default=80
// +optional
FillPercentage int `json:"fillPercentage,omitempty"`

// TargetPath specifies where to create the fill file (for pod-disk-fill)
// Default: /tmp (safe, typically has space, automatically cleaned up)
// Warning: Filling root filesystem (/) can crash the pod
// +kubebuilder:default="/tmp"
// +optional
TargetPath string `json:"targetPath,omitempty"`

// VolumeName optionally targets a specific mounted volume (for pod-disk-fill)
// If specified, fills that volume instead of TargetPath
// +optional
VolumeName string `json:"volumeName,omitempty"`
```

### Example Usage

```yaml
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: disk-fill-test
spec:
  action: pod-disk-fill
  namespace: default
  selector:
    app: logging-service
  count: 2
  duration: "5m"               # Keep disk full for 5 minutes
  fillPercentage: 85           # Fill to 85% capacity
  targetPath: "/var/log"       # Fill log directory

  # Safety features
  dryRun: false
  maxPercentage: 30
```

### Ephemeral Container Command

**Primary approach (fallocate)**:
```bash
# 1. Get filesystem size
SIZE=$(df -k /tmp | tail -1 | awk '{print $2}')

# 2. Calculate fill size (85% of total)
FILL_SIZE=$((SIZE * 85 / 100))

# 3. Create sparse file
fallocate -l ${FILL_SIZE}k /tmp/chaos-disk-fill.img

# 4. Wait for duration
sleep 300

# 5. Cleanup
rm -f /tmp/chaos-disk-fill.img
```

**Fallback approach (dd)**:
```bash
# For filesystems that don't support fallocate
dd if=/dev/zero of=/tmp/chaos-disk-fill.img bs=1M count=$((FILL_SIZE / 1024)) 2>/dev/null
sleep 300
rm -f /tmp/chaos-disk-fill.img
```

### Implementation Flow

1. **Validation**: Verify duration, fillPercentage in range
2. **Pod Selection**: Get eligible pods (respect exclusions, safety limits)
3. **Dry-run Check**: Preview affected pods if enabled
4. **Disk Fill**: For each pod:
   - Inject ephemeral container with alpine image
   - Mount pod's filesystem
   - Calculate available space on target path
   - Create fill file to reach fillPercentage
   - Sleep for duration
   - Delete fill file (cleanup)
5. **Status Update**: Record affected pods, success/failure
6. **Metrics**: Track disk fill experiments

## Alternatives Considered

### Alternative 1: Direct Exec Instead of Ephemeral Container

**Approach**: Use `kubectl exec` to run disk fill commands in existing containers

**Pros**:
- No ephemeral container overhead
- Simpler implementation

**Cons**:
- Requires shell/tools in target container (may not exist)
- Harder to isolate and cleanup
- No resource limits on disk fill operation
- Less portable (busybox vs bash vs sh)

**Decision**: Rejected - Ephemeral container provides better isolation and consistency

---

### Alternative 2: Fill Specific Volumes Instead of Paths

**Approach**: Allow users to specify PVC/volume names to fill

**Pros**:
- More precise targeting
- Better for testing PVC auto-expansion

**Cons**:
- More complex configuration
- Need to resolve volume mounts
- Path-based approach is simpler and covers most use cases

**Decision**: Deferred - Add path-based approach first, volume targeting later if needed

---

### Alternative 3: Gradual Fill Over Time

**Approach**: Slowly fill disk over duration instead of immediately

**Pros**:
- More realistic simulation of gradual disk growth
- Can test monitoring alert thresholds

**Cons**:
- More complex implementation
- Harder to predict exact behavior
- Immediate fill is simpler and tests worst case

**Decision**: Rejected for v1 - Can be added as `fillMode: gradual` option later

---

### Alternative 4: Fill Multiple Paths Simultaneously

**Approach**: Allow array of target paths to fill in parallel

**Pros**:
- Can test multiple directories at once

**Cons**:
- Adds complexity
- Single path covers 90% of use cases

**Decision**: Deferred - Single path sufficient for MVP

## Consequences

### Positive

- **Storage testing coverage**: Completes chaos testing suite (CPU, Memory, Network, Disk)
- **Realistic scenarios**: Tests actual out-of-disk-space situations
- **Safe defaults**: Conservative limits (50-95%, defaults to /tmp)
- **Automatic cleanup**: Files deleted when experiment ends
- **Consistent implementation**: Reuses ephemeral container pattern
- **All safety features**: dry-run, maxPercentage, exclusions work automatically

### Negative

- **Pod crash risk**: Filling root filesystem (/) can crash pods
- **Node disk pressure**: Can affect node if using hostPath volumes
- **Slower than other chaos**: Creating large files takes time
- **Filesystem dependency**: fallocate doesn't work on all filesystems (tmpfs, NFS)

### Neutral

- **New validation paths**: Need to validate paths, percentages
- **Documentation needs**: Must clearly warn about crash risks

### Risks and Mitigations

**Risk**: Filling root filesystem (/) causes pod to crash
- **Mitigation**: Default to /tmp (safer location)
- **Mitigation**: Conservative max limit (95%, not 100%)
- **Mitigation**: Document warning about path selection
- **Mitigation**: Consider blacklisting dangerous paths (/, /var, /etc)

**Risk**: Node disk fills up via hostPath volumes
- **Mitigation**: Safety features (maxPercentage) limit blast radius
- **Mitigation**: Document hostPath risks
- **Mitigation**: Recommend testing in non-production first

**Risk**: Filesystem doesn't support fallocate
- **Mitigation**: Automatic fallback to dd
- **Mitigation**: Detect and log which method was used

**Risk**: File isn't deleted on failure
- **Mitigation**: Track affected pods in status for manual cleanup
- **Mitigation**: Use unique filename with timestamp
- **Mitigation**: Cleanup on next reconciliation if found

## Validation Requirements

1. **OpenAPI Schema**:
   - `fillPercentage`: Range 50-95, default 80
   - `targetPath`: String, default "/tmp"
   - `volumeName`: Optional string

2. **Admission Webhook**:
   - Verify duration is specified
   - Validate fillPercentage in safe range
   - Warn if targetPath is dangerous (/, /var, /etc, /usr)
   - Check volume exists if volumeName specified

3. **Controller**:
   - Calculate available space correctly
   - Handle fallocate failures gracefully (fallback to dd)
   - Ensure file cleanup even on errors
   - Track affected pods for status

## Testing Strategy

1. **Unit Tests**:
   - Test percentage calculations
   - Test path validation
   - Test command generation

2. **Integration Tests**:
   - Test with different fillPercentage values
   - Verify cleanup occurs
   - Test fallocate vs dd fallback
   - Test safety features integration

3. **E2E Tests**:
   - Create pod with known disk size
   - Fill to 80%
   - Verify disk usage
   - Verify cleanup
   - Check pod remains healthy

4. **Manual Testing**:
   - Test on different filesystems (ext4, xfs, tmpfs)
   - Verify monitoring alerts trigger
   - Test log rotation behavior
   - Validate cleanup on experiment deletion

## Implementation Status

### Planned
- [ ] CRD/schema fields and webhook validation
- [ ] Controller logic for disk fill injection
- [ ] Ephemeral container with fallocate/dd logic
- [ ] Cleanup tracking and automatic file removal
- [ ] Safety wiring (exclusions, maxPercentage, namespace protection)
- [ ] Metrics and events for disk fill operations
- [ ] Sample YAML in `config/samples/`
- [ ] Scenario documentation in `docs/SCENARIOS.md`
- [ ] Unit and integration tests
- [ ] E2E test in Kind cluster

### Deferred
- [ ] Volume-based targeting (by PVC name)
- [ ] Gradual fill mode
- [ ] Multiple path filling
- [ ] Auto-detection of dangerous paths
- [ ] Fill verification metrics (actual vs requested)

## Future Enhancements

- **Volume targeting**: Fill specific PVCs by name
- **Gradual fill**: Slowly fill over time to simulate log growth
- **Inode exhaustion**: Create many small files instead of one large file
- **Path blacklist**: Automatically reject dangerous paths
- **Fill verification**: Report actual filled percentage vs requested
- **Write speed control**: Limit fill speed to avoid I/O saturation

## References

- [ADR 0003: pod-memory-stress-implementation](./0003-pod-memory-stress-implementation.md) - Similar ephemeral container approach
- [ADR 0005: pod-cpu-stress-implementation](./0005-pod-cpu-stress-implementation.md) - Ephemeral container patterns
- [Kubernetes Ephemeral Containers](https://kubernetes.io/docs/concepts/workloads/pods/ephemeral-containers/)
- [fallocate man page](https://man7.org/linux/man-pages/man1/fallocate.1.html)
- [df command](https://man7.org/linux/man-pages/man1/df.1.html)

## Notes

### Path Selection Guidelines

**Safe paths** (recommended):
- `/tmp` - Temporary directory, typically has space, auto-cleaned
- `/var/log` - Log directory (tests log rotation)
- `/data` or `/app/data` - Application data directories

**Dangerous paths** (avoid):
- `/` - Root filesystem (can crash pod)
- `/var` - System directory (can affect system services)
- `/etc` - Configuration directory (should never be filled)
- `/usr` - System binaries (read-only in many containers)
- `/sys`, `/proc` - Virtual filesystems (won't work)

### Cleanup Guarantees

The ephemeral container approach provides automatic cleanup:
1. Container exits after duration → file deletion command runs
2. If container crashes → file remains but pod can restart
3. If pod deleted → file deleted with pod
4. Manual cleanup: `kubectl exec` to remove `/tmp/chaos-disk-fill.img`

Unique filename format: `chaos-disk-fill-{timestamp}-{podname}.img`

---

**Review Checklist**:
- [ ] Discuss fillPercentage limits (50-95% reasonable?)
- [ ] Should we blacklist dangerous paths automatically?
- [ ] Do we need volume targeting in v1 or defer?
- [ ] Review safety implications with team
- [ ] Consider adding inode exhaustion variant
