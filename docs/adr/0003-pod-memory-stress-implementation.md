# ADR 0003: Pod Memory Stress Implementation

**Status**: Accepted
**Date**: 2025-10-30
**Author**: k8s-chaos team

## Context

Following the successful implementation of pod-cpu-stress (ADR 0005), we need to complete the resource stress testing suite by adding memory stress capabilities. This enables testing application behavior under memory pressure, OOMKiller scenarios, and memory leak simulations.

## Decision

Implement `pod-memory-stress` action using the same proven ephemeral container approach with stress-ng.

### Implementation Approach

**Method**: Ephemeral Container Injection (same as pod-cpu-stress)
- Use Kubernetes ephemeral containers feature
- Inject stress-ng container into target pods
- Non-destructive: pods continue running during stress
- Automatic cleanup when experiment completes

**Memory Stress Tool**: stress-ng
- Same tool as CPU stress for consistency
- Supports precise memory allocation control
- Options for different memory stress patterns

### Configuration Parameters

Add to ChaosExperimentSpec:

```go
// MemorySize specifies the amount of memory to consume (for pod-memory-stress)
// Format: "256M", "1G", "512M", etc.
// +kubebuilder:validation:Pattern="^[0-9]+[MG]$"
// +optional
MemorySize string `json:"memorySize,omitempty"`

// MemoryWorkers specifies the number of memory workers (for pod-memory-stress)
// +kubebuilder:validation:Minimum=1
// +kubebuilder:validation:Maximum=8
// +kubebuilder:default=1
// +optional
MemoryWorkers int `json:"memoryWorkers,omitempty"`
```

### Example Usage

```yaml
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: memory-stress-test
spec:
  action: pod-memory-stress
  namespace: default
  selector:
    app: web-server
  count: 2
  duration: "5m"
  memorySize: "512M"      # Allocate 512MB per worker
  memoryWorkers: 2         # Use 2 workers = 1GB total

  # Safety features
  dryRun: false
  maxPercentage: 30
```

### stress-ng Command

```bash
stress-ng --vm 2 --vm-bytes 512M --timeout 300s --metrics-brief
```

Parameters:
- `--vm N`: Number of memory workers
- `--vm-bytes SIZE`: Memory per worker
- `--timeout Ns`: Duration in seconds
- `--metrics-brief`: Output metrics

## Alternatives Considered

### Alternative 1: Memory Leak Simulation
**Approach**: Gradually increase memory consumption
**Rejected**: More complex, harder to predict behavior, less controlled

### Alternative 2: Different per-worker sizes
**Approach**: Allow specifying different sizes for each worker
**Rejected**: Unnecessary complexity, uniform size is sufficient

### Alternative 3: Memory Access Patterns
**Approach**: Support different access patterns (sequential, random)
**Rejected**: Too specialized, basic allocation is sufficient for most chaos testing

## Implementation Details

### Similarities to pod-cpu-stress
- Same ephemeral container injection mechanism
- Same validation patterns (duration required, webhook checks)
- Same safety features integration (dry-run, exclusion, maxPercentage)
- Same metrics patterns

### Differences from pod-cpu-stress
- Memory parameter instead of CPU percentage
- Memory size validation (regex pattern for M/G suffix)
- Different resource limits (memory instead of CPU)
- Different stress-ng flags

### Resource Limits

```go
Resources: corev1.ResourceRequirements{
    Limits: corev1.ResourceList{
        corev1.ResourceMemory: resource.MustParse(totalMemory),
    },
    Requests: corev1.ResourceList{
        corev1.ResourceMemory: resource.MustParse("64M"),
    },
},
```

Where `totalMemory = memorySize * memoryWorkers`

## Consequences

### Positive
- **Complete resource testing**: CPU + Memory coverage
- **Consistent implementation**: Reuses proven patterns from CPU stress
- **Safe testing**: All safety features automatically apply
- **Flexible**: Can simulate various memory pressure scenarios
- **Realistic**: Tests actual OOMKiller behavior

### Negative
- **Node impact**: Can affect node memory availability
- **Pod eviction risk**: May trigger pod evictions if limits exceeded
- **Container restart**: OOMKill if memory exceeds pod limits

### Risks and Mitigations

**Risk**: Memory stress causes node memory pressure
- **Mitigation**: Resource limits prevent unbounded allocation
- **Mitigation**: maxPercentage safety feature limits blast radius

**Risk**: Pods get OOMKilled
- **Mitigation**: Set appropriate resource limits
- **Mitigation**: Document best practices for memory stress testing
- **Mitigation**: Warn users in status if memorySize exceeds pod limits

**Risk**: Ephemeral container memory isn't counted in pod limits
- **Mitigation**: Actually, ephemeral containers DO count toward pod limits
- **Mitigation**: Resource limits ensure proper accounting

## Validation Requirements

1. **OpenAPI Schema**:
   - `memorySize`: Pattern validation for M/G suffix
   - `memoryWorkers`: Range 1-8

2. **Admission Webhook**:
   - Verify duration is specified
   - Validate memorySize format and reasonable values
   - Warn if total memory (size * workers) seems excessive

3. **Controller**:
   - Parse memorySize correctly
   - Calculate total memory allocation
   - Set appropriate resource limits

## Testing Strategy

1. **Unit Tests**:
   - Test memory size parsing
   - Test worker count validation
   - Test total memory calculation

2. **Integration Tests**:
   - Test with different memory sizes
   - Verify resource limits are set correctly
   - Test safety features integration

3. **Manual Testing**:
   - Test OOMKill behavior with excessive allocation
   - Verify memory is actually consumed
   - Check node memory metrics

## Migration Path

- Fully backward compatible
- No changes to existing experiments
- All new fields optional

## Future Enhancements

- Memory access patterns (sequential, random)
- Gradual memory increase over time
- Memory leak simulation mode
- Swap stress (if enabled on nodes)

## References

- [ADR 0005: pod-cpu-stress-implementation](./0005-pod-cpu-stress-implementation.md)
- [stress-ng memory stressors](https://wiki.ubuntu.com/Kernel/Reference/stress-ng)
- [Kubernetes Resource Management](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/)
- [Ephemeral Containers](https://kubernetes.io/docs/concepts/workloads/pods/ephemeral-containers/)
