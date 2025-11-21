# ADR 0005: Pod CPU Stress Implementation

**Status**: Accepted
**Date**: 2025-10-28
**Author**: k8s-chaos team

## Context

The k8s-chaos operator currently supports pod-kill, pod-delay, and node-drain chaos actions. To expand chaos testing capabilities, we need to implement CPU resource stress testing to simulate high CPU load scenarios and test application behavior under resource contention.

## Decision

We will implement a `pod-cpu-stress` action that consumes CPU resources on target pods using the `stress-ng` utility injected via ephemeral containers.

### Implementation Approach

**Method**: Ephemeral Container Injection
- Use Kubernetes ephemeral containers feature to inject a `stress-ng` container into target pods
- Ephemeral containers are temporary, non-restarting containers ideal for debugging and testing
- Cleanup is automatic when the experiment completes or the pod is restarted

**CPU Stress Tool**: stress-ng
- Industry-standard stress testing tool
- Lightweight Alpine-based image: `alexeiled/stress-ng:latest-alpine`
- Supports precise CPU load control via `--cpu` and `--cpu-load` parameters
- Active maintenance and wide adoption in chaos engineering

### Configuration Parameters

The following fields will be added to the ChaosExperiment spec:

```yaml
cpuLoad: integer        # Percentage of CPU to consume (1-100), required for pod-cpu-stress
cpuWorkers: integer     # Number of CPU workers (default: 1, range: 1-32)
```

### Operational Behavior

1. **Target Selection**: Randomly select `count` pods matching the selector
2. **Container Injection**: Add ephemeral container with stress-ng to each selected pod
3. **Duration Control**: stress-ng runs for the specified `duration` (reuses existing duration field)
4. **Cleanup**: Controller tracks ephemeral containers and ensures cleanup after duration expires
5. **Resource Limits**: Ephemeral container will have CPU limits set to prevent node exhaustion

### Example CRD

```yaml
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: cpu-stress-test
  namespace: chaos-testing
spec:
  action: pod-cpu-stress
  namespace: default
  selector:
    app: web-server
  count: 2
  duration: "5m"
  cpuLoad: 80              # Consume 80% CPU
  cpuWorkers: 2            # Use 2 CPU workers
```

## Alternatives Considered

### 1. Direct Pod Exec with Bash CPU Burn
**Rejected**: Requires bash/shell in target container, not always available (especially in distroless images)

### 2. Sidecar Container Injection via Pod Mutation
**Rejected**: Requires pod restart, too disruptive for chaos testing goals

### 3. DaemonSet with cgroup manipulation
**Rejected**: Overly complex, requires elevated privileges, harder to scope to specific pods

## Consequences

### Positive
- Non-destructive testing: pods continue running during stress
- Realistic simulation of CPU contention scenarios
- Clean separation from application containers
- Automatic cleanup when pod restarts or experiment ends
- Works with any container image (including distroless)

### Negative
- Requires Kubernetes 1.23+ for stable ephemeral containers support
- Ephemeral containers cannot be removed without pod restart (tracked in status for manual cleanup if needed)
- Network overhead from pulling stress-ng image (mitigated by small Alpine image ~10MB)

### Risks
- Node CPU exhaustion if too many experiments run simultaneously
  - **Mitigation**: Implement resource limits on ephemeral containers
  - **Mitigation**: Add validation warnings if total cpuLoad * count exceeds node capacity
- Zombie ephemeral containers if controller crashes during experiment
  - **Mitigation**: Controller uses status field to track active experiments and cleanup on restart

## Validation Requirements

1. **OpenAPI Schema**:
   - `cpuLoad`: integer, range 1-100, required when action=pod-cpu-stress
   - `cpuWorkers`: integer, range 1-32, default 1

2. **Admission Webhook**:
   - Verify duration is specified for pod-cpu-stress
   - Warn if cpuLoad * cpuWorkers * count may impact node stability
   - Check Kubernetes version supports ephemeral containers (1.23+)

3. **RBAC**:
   - Add permission to patch pods (for ephemeral container injection)
   - Permission to get pod/ephemeralcontainers subresource

## Implementation Checklist

- [ ] Update ChaosExperiment API with cpuLoad and cpuWorkers fields
- [ ] Add OpenAPI validation markers
- [ ] Implement pod-cpu-stress case in controller reconciliation loop
- [ ] Add admission webhook validation for CPU stress parameters
- [ ] Update RBAC manifests with ephemeral container permissions
- [ ] Create sample CRD in config/samples/
- [ ] Add unit tests for CPU stress logic
- [ ] Add metrics for CPU stress experiments
- [ ] Update CLAUDE.md and README.md

## References

- [Kubernetes Ephemeral Containers](https://kubernetes.io/docs/concepts/workloads/pods/ephemeral-containers/)
- [stress-ng Documentation](https://wiki.ubuntu.com/Kernel/Reference/stress-ng)
- [Chaos Engineering Best Practices](https://principlesofchaos.org/)
