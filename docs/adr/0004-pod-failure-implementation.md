# ADR 0004: Pod Failure Implementation

**Status**: Accepted
**Date**: 2025-11-19
**Author**: k8s-chaos team

## Context

The k8s-chaos operator currently supports various chaos actions including pod-kill (deleting pods), pod-delay (network latency), node-drain (infrastructure failures), and resource stress testing (CPU/memory). However, there's a gap in testing application resilience to container crashes and restarts.

While `pod-kill` deletes entire pods and tests rescheduling behavior, it doesn't simulate the common scenario where containers crash due to application bugs, OOM conditions, or signal handling issues. We need a way to test:
- Application restart behavior and recovery mechanisms
- Proper signal handling and graceful shutdown
- Restart policy effectiveness
- Crash loop backoff behavior
- Health check and readiness probe reliability

## Decision

We will implement a `pod-failure` action that kills the main process (PID 1) in containers to cause container crashes, triggering Kubernetes restart mechanisms without deleting the pod.

### Implementation Approach

**Method**: Process Termination via Exec
- Use Kubernetes pod exec API to run `kill -9 1` in target containers
- Kill PID 1 (the main process) to cause an immediate container crash
- Kubernetes detects the crash and restarts the container based on restartPolicy
- Pod remains on the same node with the same IP, testing in-place recovery

**Target Selection**:
- Target the first container in each selected pod
- This simulates the most common failure scenario (main application container crash)
- Future enhancement could allow targeting specific containers

### Operational Behavior

1. **Target Selection**: Randomly select `count` pods matching the selector
2. **Safety Checks**: Apply all standard safety features (dry-run, exclusions, percentage limits)
3. **Process Kill**: Execute `kill -9 1` in the first container of each selected pod
4. **Container Restart**: Kubernetes automatically restarts the container per restartPolicy
5. **Metrics Tracking**: Record successful failures in Prometheus metrics

### Example CRD

```yaml
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: pod-failure-test
  namespace: chaos-testing
spec:
  action: pod-failure
  namespace: default
  selector:
    app: web-server
  count: 2
  # No duration needed - crash is instantaneous
  # Container restart is handled by Kubernetes
```

### Behavior with Restart Policies

The action's effect varies based on the pod's restart policy:

- **`restartPolicy: Always`** (default for Deployments): Container crashes and immediately restarts
- **`restartPolicy: OnFailure`** (common for Jobs): Container restarts on crash (exit code 137)
- **`restartPolicy: Never`** (some batch jobs): Container crashes and stays failed

## Alternatives Considered

### 1. SIGTERM Followed by SIGKILL
**Rejected**: We want to simulate an immediate crash, not graceful shutdown. SIGTERM would test different behavior (graceful termination).

### 2. Simulate OOM by Memory Exhaustion
**Rejected**: This is covered by pod-memory-stress action. pod-failure should be simpler and faster.

### 3. Delete Container via CRI (Container Runtime Interface)
**Rejected**: Requires elevated privileges and direct CRI access. Using kill command is simpler and works across all container runtimes.

### 4. Inject Segfault or Panic into Application
**Rejected**: Application-specific, requires code injection, too complex. Process kill is universal.

### 5. Use SIGKILL Instead of `kill -9 1`
**Considered**: `kill -9 1` and `kill -KILL 1` are equivalent. Using `-9` is more explicit and universally recognized.

## Consequences

### Positive
- **Simple and reliable**: Just kills PID 1, no complex logic
- **Tests real failure scenarios**: Simulates crashes from bugs, OOM, or unhandled signals
- **Fast execution**: Crash happens immediately, no waiting for duration
- **Tests in-place recovery**: Pod stays on same node, tests restart behavior not rescheduling
- **Works everywhere**: No special container requirements (unlike bash-based approaches)
- **Non-destructive to infrastructure**: Pod isn't deleted, just container restarts
- **Integrates with all safety features**: Dry-run, exclusions, percentage limits all work

### Negative
- **Requires pod exec permissions**: Controller needs RBAC to exec into pods
- **May not work with hardened containers**: Some security policies block exec
- **Assumes PID 1 exists**: Should always be true, but edge cases possible
- **Only tests crash, not graceful shutdown**: This is intentional, but worth noting

### Risks and Mitigations

1. **Risk**: Killing PID 1 in critical system pods could cause node issues
   - **Mitigation**: Production namespace protection requires `allowProduction: true`
   - **Mitigation**: Exclusion labels prevent affecting system pods
   - **Mitigation**: Dry-run mode allows previewing impact

2. **Risk**: Rapid container restarts could trigger crash loop backoff
   - **Mitigation**: This is actually desired behavior to test backoff mechanisms
   - **Mitigation**: Experiment duration controls how long chaos runs
   - **Mitigation**: Status tracking shows which pods were affected

3. **Risk**: Restart might take longer than expected due to image pull
   - **Mitigation**: This tests realistic failure recovery time
   - **Mitigation**: Metrics track experiment duration for monitoring

4. **Risk**: Application might not handle crashes gracefully
   - **Mitigation**: This is exactly what we're testing! Discovering this is valuable

## Validation Requirements

### OpenAPI Schema
No additional fields required for pod-failure action. It reuses existing fields:
- `action: pod-failure` (added to enum)
- `namespace`: target namespace
- `selector`: pod label selector
- `count`: number of pods to affect

### Admission Webhook
No special validation needed beyond standard checks:
- Namespace existence
- Selector effectiveness
- Safety constraints (maxPercentage, production protection)

### RBAC
Existing permissions are sufficient:
- `pods/exec`: create - already granted for pod-delay action
- `pods`: get, list - already granted

## Integration with Existing Features

### Safety Features
- ✅ **Dry-run mode**: Shows which pods would crash without executing
- ✅ **Maximum percentage**: Limits % of pods that can crash
- ✅ **Production protection**: Requires explicit approval for prod namespaces
- ✅ **Exclusion labels**: Protects critical pods from crashes

### Operational Features
- ✅ **Retry logic**: Auto-retries if exec fails
- ✅ **Metrics tracking**: Records successes/failures in Prometheus
- ✅ **Scheduling**: Can run on cron schedule for continuous resilience testing
- ✅ **Experiment duration**: Can limit how long chaos runs

## Testing Strategy

### Unit Tests
- Test handlePodFailure function with mock clients
- Verify correct pod selection and safety filtering
- Test error handling when exec fails

### Integration Tests
- Deploy test pod with simple app
- Trigger pod-failure action
- Verify container restarts
- Check metrics are recorded

### Manual Testing Checklist
- [ ] Basic pod-failure works on simple deployment
- [ ] Dry-run mode shows correct preview
- [ ] Safety features block unsafe operations
- [ ] Metrics are recorded correctly
- [ ] Works with different restart policies
- [ ] Handles exec failures gracefully

## Implementation Checklist

- [x] Add pod-failure to action enum in ChaosExperimentSpec
- [x] Update ValidActions list in validation helpers
- [x] Implement handlePodFailure in controller
- [x] Implement killContainerProcess helper function
- [x] Add case to switch statement in Reconcile
- [x] Create sample CRDs in config/samples/
- [x] Regenerate CRD manifests
- [x] Update CLAUDE.md documentation
- [x] Run tests and verify build
- [x] Create this ADR document

## Future Enhancements

### Multi-Container Support
Currently targets first container only. Could add:
```yaml
targetContainer: "app"  # Optional: specific container name
targetAllContainers: true  # Optional: crash all containers
```

### Signal Customization
Currently uses SIGKILL (-9). Could support:
```yaml
signal: "SIGTERM"  # Allow different signals for testing graceful shutdown
```

### Crash Patterns
Could add more sophisticated crash patterns:
```yaml
crashPattern: "random"  # random, sequential, simultaneous
crashDelay: "30s"  # delay between crashes
```

## References

- [Kubernetes Restart Policy](https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#restart-policy)
- [Kubernetes Pod Exec](https://kubernetes.io/docs/tasks/debug/debug-application/get-shell-running-container/)
- [Linux Kill Command](https://man7.org/linux/man-pages/man1/kill.1.html)
- [Chaos Engineering Principles](https://principlesofchaos.org/)
- [Container Crash Loop Backoff](https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#container-restart-policy)
- [PID 1 and Container Init](https://cloud.google.com/architecture/best-practices-for-building-containers#signal-handling)
