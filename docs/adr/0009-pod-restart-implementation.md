# ADR 0009: Pod Restart Implementation

**Status**: Accepted

**Date**: 2025-12-27

**Authors**: k8s-chaos team

## Context

The k8s-chaos operator currently supports various pod-level chaos actions:
- `pod-kill`: Deletes pods entirely, testing rescheduling and pod recreation
- `pod-failure`: Kills PID 1 with SIGKILL, simulating immediate crashes
- `pod-delay`: Injects network latency
- `pod-cpu-stress` / `pod-memory-stress`: Resource exhaustion testing

However, there's a gap in testing **graceful container restarts**. In production environments, containers often restart due to:
- Application updates and rolling deployments
- Configuration changes triggering restarts
- Health check failures leading to graceful shutdown
- Manual restart requests for troubleshooting
- Sidecar container restarts (service mesh, log collectors)

These restarts differ from crashes because they:
1. Allow applications time to handle SIGTERM and shutdown gracefully
2. Close connections, flush buffers, and cleanup resources properly
3. Test the graceful shutdown code path, not crash recovery
4. Simulate controlled maintenance operations, not emergencies

We need a `pod-restart` action to test how applications handle graceful, controlled restarts without deleting the entire pod.

## Decision

We will implement a `pod-restart` action that triggers a graceful container restart by sending SIGTERM to the main process (PID 1), allowing applications to shutdown cleanly before Kubernetes restarts the container.

### Implementation Approach

**Method**: Graceful Termination via SIGTERM

1. Use Kubernetes pod exec API to run `kill -TERM 1` (or `kill -15 1`) in target containers
2. Send SIGTERM to PID 1, giving the application a chance to shutdown gracefully
3. The process receives SIGTERM and can:
   - Close database connections
   - Flush logs and metrics
   - Finish in-flight requests
   - Save state
   - Cleanup resources
4. After termination (or termination grace period), Kubernetes restarts the container
5. Tests the application's graceful shutdown and restart behavior

**Difference from pod-failure**:
- `pod-failure`: Uses SIGKILL (-9) for immediate crash, no cleanup possible
- `pod-restart`: Uses SIGTERM (-15) for graceful shutdown, cleanup allowed

**Target Selection**:
- Target the first container in each selected pod (main application container)
- Future enhancement: support targeting specific containers or all containers
- Respects all safety features (dry-run, exclusions, percentage limits)

### Operational Behavior

1. **Target Selection**: Randomly select `count` pods matching the selector
2. **Safety Checks**: Apply all standard safety features
3. **Graceful Termination**: Execute `kill -TERM 1` in the first container
4. **Shutdown Grace Period**: Application has time to cleanup (respects terminationGracePeriodSeconds)
5. **Container Restart**: Kubernetes automatically restarts container per restartPolicy
6. **Metrics Tracking**: Record successful restarts in Prometheus

### Example CRD

```yaml
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: pod-restart-test
  namespace: chaos-testing
spec:
  action: pod-restart
  namespace: production
  selector:
    app: web-api
    tier: backend
  count: 2

  # Optional: Control restart timing
  restartInterval: "30s"  # Wait 30s between restarting each pod (default: immediate)

  # Safety features
  dryRun: false
  maxPercentage: 25  # Only restart 25% of matching pods
  allowProduction: true  # Required for production namespaces
```

### Advanced Example: Staggered Restarts

```yaml
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: rolling-restart-test
  namespace: chaos-testing
spec:
  action: pod-restart
  namespace: production
  selector:
    app: microservice
  count: 5

  # Stagger restarts to avoid overwhelming the service
  restartInterval: "45s"  # Wait 45s between each pod restart

  # Experiment runs for 5 minutes total
  experimentDuration: "5m"

  # Safety: Only affect 20% of pods at a time
  maxPercentage: 20
```

## Alternatives Considered

### 1. Use Deployment Rollout Restart
**Description**: Use `kubectl rollout restart deployment/<name>` approach

**Pros**:
- Simple, built-in Kubernetes feature
- Well-tested and reliable
- Handles rolling restarts automatically

**Cons**:
- Works at Deployment level, not individual pod level
- Can't target specific pods by labels
- Doesn't work for standalone pods, StatefulSets, DaemonSets
- Less control over restart timing and selection

**Decision**: Rejected - We need pod-level control for fine-grained chaos testing

### 2. Update Pod Spec to Trigger Restart
**Description**: Modify pod annotations or environment variables to force restart

**Pros**:
- Uses standard Kubernetes update mechanisms
- Declarative approach

**Cons**:
- Pod specs are mostly immutable except for specific fields
- Doesn't reliably trigger container restarts
- May not work across different workload types
- Complex to implement correctly

**Decision**: Rejected - Too fragile and doesn't work reliably

### 3. Delete Container via CRI (Container Runtime Interface)
**Description**: Directly call container runtime to delete containers

**Pros**:
- Direct control over container lifecycle
- Fast and precise

**Cons**:
- Requires elevated privileges and CRI access
- Tightly coupled to container runtime implementation
- Security risk - direct runtime access
- Breaks abstraction layers
- Different for each runtime (containerd, CRI-O, Docker)

**Decision**: Rejected - Too invasive and requires excessive privileges

### 4. Use SIGTERM then Wait for Kubernetes
**Description**: Our chosen approach - send SIGTERM and let Kubernetes handle restart

**Pros**:
- Simple and portable - works with all container runtimes
- Uses existing RBAC permissions (pods/exec)
- Respects graceful shutdown semantics
- Tests real application shutdown code paths
- Kubernetes handles restart automatically per restartPolicy

**Cons**:
- Requires pod exec permissions
- Timing depends on terminationGracePeriodSeconds
- May not work in hardened environments that block exec

**Decision**: Accepted - Best balance of simplicity, safety, and effectiveness

### 5. Use SIGHUP for Reload Instead of Restart
**Description**: Send SIGHUP to trigger application reload without restart

**Pros**:
- Can test configuration reload
- No downtime

**Cons**:
- Application-specific behavior (not all apps handle SIGHUP)
- Doesn't test container restart behavior
- Different semantics than restart

**Decision**: Deferred - Could be future enhancement as `pod-reload` action

## Consequences

### Positive

1. **Tests Graceful Shutdown**: Validates applications handle SIGTERM correctly
2. **Production-Realistic**: Simulates planned maintenance restarts, not crashes
3. **Less Disruptive than pod-kill**: Container restarts in-place, pod keeps same IP
4. **Validates Cleanup Code**: Tests database connection cleanup, flush logic, etc.
5. **Tests Restart Policies**: Verifies restartPolicy configuration works as expected
6. **Simple Implementation**: Reuses existing pod exec mechanisms
7. **Works Everywhere**: Any container that supports SIGTERM (virtually all)
8. **Safety-First**: Integrates with all existing safety features

### Negative

1. **Requires Exec Permissions**: Controller needs RBAC for pods/exec
2. **Timing Uncertainty**: Restart timing depends on grace period and app shutdown time
3. **May Block in Hardened Environments**: Some security policies prevent pod exec
4. **Application-Dependent**: Effectiveness depends on app's SIGTERM handling
5. **No Guarantee of Cleanup**: App might ignore SIGTERM (though this reveals issues)

### Neutral

1. **Different from pod-failure**: Both restart containers but test different scenarios
2. **Complements pod-kill**: pod-kill tests rescheduling, pod-restart tests in-place recovery
3. **New Operational Pattern**: Teams can test planned vs unplanned restarts separately

### Risks and Mitigations

#### Risk 1: Application Hangs on SIGTERM
**Scenario**: Application doesn't handle SIGTERM, hangs indefinitely

**Mitigation**:
- Kubernetes terminationGracePeriodSeconds handles this (default 30s)
- After grace period, Kubernetes sends SIGKILL automatically
- This reveals application bugs in shutdown handling
- Dry-run mode allows testing without impact

#### Risk 2: Cascading Failures from Simultaneous Restarts
**Scenario**: Restarting many pods simultaneously overwhelms dependent services

**Mitigation**:
- `restartInterval` parameter staggers restarts
- `maxPercentage` limits how many pods are affected
- `count` parameter controls total pods restarted
- Dry-run shows which pods will be affected

#### Risk 3: Critical Pods Get Restarted
**Scenario**: System critical pods restart causing outages

**Mitigation**:
- Production namespace protection requires `allowProduction: true`
- Exclusion labels (`chaos.gushchin.dev/exclude: "true"`) protect critical pods
- Dry-run mode allows preview before execution
- Namespace-level exclusions via annotations

#### Risk 4: Long Shutdown Times
**Scenario**: Applications take long time to shutdown gracefully

**Mitigation**:
- This is valuable to discover - indicates potential issues
- Metrics track experiment duration
- Can adjust experiment scheduling based on observed timing
- Reveals if terminationGracePeriodSeconds needs tuning

## Implementation Details

### API Changes

#### ChaosExperimentSpec Enhancement
```go
type ChaosExperimentSpec struct {
    // ... existing fields ...

    // RestartInterval specifies delay between restarting each pod (pod-restart only)
    // Format: "30s", "1m", "2m30s"
    // Default: "" (restart all immediately)
    // +optional
    RestartInterval *string `json:"restartInterval,omitempty"`
}
```

#### Validation Rules

**OpenAPI Schema**:
- Add `pod-restart` to Action enum
- Add `restartInterval` field with regex validation: `^([0-9]+(s|m|h))+$`

**Admission Webhook**:
- Validate `restartInterval` format if provided
- Ensure `restartInterval` is only used with `pod-restart` action
- Standard checks: namespace exists, selector matches pods

### Controller Implementation

```go
func (r *ChaosExperimentReconciler) handlePodRestart(ctx context.Context, exp *chaosv1alpha1.ChaosExperiment) (ctrl.Result, error) {
    log := ctrl.LoggerFrom(ctx)

    // 1. Get eligible pods
    pods, err := r.getEligiblePods(ctx, exp)
    if err != nil {
        return ctrl.Result{}, err
    }

    // 2. Handle dry-run
    if exp.Spec.DryRun {
        return r.handleDryRun(ctx, exp, pods)
    }

    // 3. Limit by count
    selectedPods := r.selectRandomPods(pods, exp.Spec.Count)

    // 4. Parse restart interval
    interval := time.Duration(0)
    if exp.Spec.RestartInterval != nil {
        interval, err = parseDuration(*exp.Spec.RestartInterval)
        if err != nil {
            return ctrl.Result{}, fmt.Errorf("invalid restartInterval: %w", err)
        }
    }

    // 5. Restart each pod with optional delay
    affectedPods := []string{}
    for i, pod := range selectedPods {
        // Apply delay between restarts (except first)
        if i > 0 && interval > 0 {
            log.Info("Waiting before next restart", "interval", interval)
            time.Sleep(interval)
        }

        // Restart the container gracefully
        if err := r.gracefullyRestartContainer(ctx, &pod); err != nil {
            log.Error(err, "Failed to restart pod", "pod", pod.Name)
            // Continue with other pods even if one fails
            continue
        }

        affectedPods = append(affectedPods, pod.Name)
        log.Info("Successfully restarted pod", "pod", pod.Name, "namespace", pod.Namespace)
    }

    // 6. Update status
    exp.Status.AffectedPods = affectedPods
    exp.Status.Message = fmt.Sprintf("Successfully restarted %d pods", len(affectedPods))

    // 7. Emit metrics
    chaosmetrics.ExperimentsTotal.WithLabelValues(
        exp.Spec.Action,
        exp.Spec.Namespace,
        "success",
    ).Inc()

    chaosmetrics.ResourcesAffectedTotal.WithLabelValues(
        exp.Spec.Action,
        exp.Spec.Namespace,
    ).Add(float64(len(affectedPods)))

    return ctrl.Result{}, nil
}

func (r *ChaosExperimentReconciler) gracefullyRestartContainer(ctx context.Context, pod *corev1.Pod) error {
    log := ctrl.LoggerFrom(ctx)

    // Target the first container (main application container)
    if len(pod.Spec.Containers) == 0 {
        return fmt.Errorf("no containers found in pod")
    }

    containerName := pod.Spec.Containers[0].Name

    // Execute SIGTERM command
    // Using kill -15 (SIGTERM) instead of kill -9 (SIGKILL) for graceful shutdown
    cmd := []string{"/bin/sh", "-c", "kill -15 1 || kill -TERM 1"}

    req := r.ClientSet.CoreV1().RESTClient().
        Post().
        Namespace(pod.Namespace).
        Resource("pods").
        Name(pod.Name).
        SubResource("exec").
        VersionedParams(&corev1.PodExecOptions{
            Container: containerName,
            Command:   cmd,
            Stdout:    true,
            Stderr:    true,
        }, scheme.ParameterCodec)

    exec, err := remotecommand.NewSPDYExecutor(r.Config, "POST", req.URL())
    if err != nil {
        return fmt.Errorf("failed to create executor: %w", err)
    }

    var stdout, stderr bytes.Buffer
    err = exec.Stream(remotecommand.StreamOptions{
        Stdout: &stdout,
        Stderr: &stderr,
    })

    if err != nil {
        log.Info("Exec output", "stdout", stdout.String(), "stderr", stderr.String())
        return fmt.Errorf("failed to execute restart command: %w", err)
    }

    log.Info("Sent SIGTERM to container",
        "pod", pod.Name,
        "container", containerName,
        "namespace", pod.Namespace)

    return nil
}
```

### RBAC Requirements

Existing permissions are sufficient:
- `pods/exec`: create - already granted
- `pods`: get, list, watch - already granted
- No new ClusterRole changes needed

### Sample CRD Files

#### Basic Restart
```yaml
# config/samples/chaos_v1alpha1_chaosexperiment_pod_restart.yaml
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: chaosexperiment-pod-restart
  namespace: chaos-testing
spec:
  action: pod-restart
  namespace: default
  selector:
    app: nginx
  count: 2
```

#### Staggered Restart
```yaml
# config/samples/chaos_v1alpha1_chaosexperiment_pod_restart_staggered.yaml
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: chaosexperiment-staggered-restart
  namespace: chaos-testing
spec:
  action: pod-restart
  namespace: production
  selector:
    app: api-server
  count: 5
  restartInterval: "1m"  # 1 minute between each pod restart
  maxPercentage: 30
  allowProduction: true
```

## Integration with Existing Features

### Safety Features
- ✅ **Dry-run mode**: Shows which pods would restart without executing
- ✅ **Maximum percentage**: Limits % of pods that can restart
- ✅ **Production protection**: Requires explicit approval for prod namespaces
- ✅ **Exclusion labels**: Protects critical pods from restarts
- ✅ **Terminating pod filter**: Skips pods already terminating

### Operational Features
- ✅ **Retry logic**: Auto-retries if exec fails
- ✅ **Metrics tracking**: Records successes/failures in Prometheus
- ✅ **Scheduling**: Can run on cron schedule
- ✅ **Experiment duration**: Can limit how long chaos runs
- ✅ **History logging**: All restarts recorded in ChaosExperimentHistory

### New Capabilities
- ✅ **Staggered restarts**: `restartInterval` parameter for gradual chaos
- ✅ **Graceful shutdown testing**: Tests SIGTERM handling code paths
- ✅ **Rolling chaos**: Simulates rolling updates without actual deployment changes

## Testing Strategy

### Unit Tests
```go
func TestHandlePodRestart(t *testing.T) {
    // Test basic restart functionality
    // Test with restartInterval parameter
    // Test safety features (dry-run, maxPercentage)
    // Test error handling
}

func TestGracefullyRestartContainer(t *testing.T) {
    // Test SIGTERM execution
    // Test container selection
    // Test error cases (no containers, exec failure)
}
```

### E2E Tests
```go
It("should gracefully restart pods", func() {
    // Create test deployment
    // Apply pod-restart experiment
    // Verify containers restarted (check restart count)
    // Verify pods still exist (not deleted)
    // Check metrics recorded
})

It("should stagger restarts with restartInterval", func() {
    // Create multiple pods
    // Apply experiment with restartInterval: "30s"
    // Measure time between restarts
    // Verify proper spacing
})
```

### Manual Testing Checklist
- [ ] Basic pod-restart works on simple deployment
- [ ] Restarts are graceful (logs show SIGTERM handling)
- [ ] restartInterval parameter spaces out restarts correctly
- [ ] Dry-run mode shows correct preview
- [ ] Safety features work (maxPercentage, exclusions)
- [ ] Metrics are recorded correctly
- [ ] Works with different restart policies
- [ ] Handles exec failures gracefully
- [ ] Container restart count increments
- [ ] Pods maintain same IP address after restart

## Implementation Checklist

### Phase 1: Core Implementation
- [ ] Add `pod-restart` to action enum in ChaosExperimentSpec
- [ ] Add `restartInterval` field to ChaosExperimentSpec
- [ ] Update ValidActions list in validation
- [ ] Add restartInterval validation in OpenAPI schema
- [ ] Implement `handlePodRestart` in controller
- [ ] Implement `gracefullyRestartContainer` helper function
- [ ] Add case to switch statement in Reconcile

### Phase 2: Testing
- [ ] Write unit tests for handlePodRestart
- [ ] Write unit tests for gracefullyRestartContainer
- [ ] Create E2E tests for basic restart
- [ ] Create E2E tests for staggered restart
- [ ] Test with different restart policies
- [ ] Test safety features integration

### Phase 3: Documentation & Samples
- [ ] Create sample CRDs in config/samples/
- [ ] Update CLAUDE.md documentation
- [ ] Update API.md with new field
- [ ] Create user guide for pod-restart
- [ ] Add to SCENARIOS.md with examples
- [ ] Update BACKLOG.md to mark as completed

### Phase 4: Finalization
- [ ] Regenerate CRD manifests (`make manifests`)
- [ ] Run full test suite (`make test lint`)
- [ ] Update CHANGELOG.md
- [ ] Mark ADR as Accepted

## Metrics

New Prometheus metrics will track pod-restart actions:

```promql
# Total restarts by action
chaos_experiments_total{action="pod-restart",namespace="production",status="success"} 42

# Resources affected (pods restarted)
chaos_resources_affected_total{action="pod-restart",namespace="production"} 84

# Experiment duration (including restart intervals)
chaos_experiments_duration_seconds{action="pod-restart",namespace="production"}
```

## Future Enhancements

### 1. Multi-Container Support
Target specific containers or all containers in a pod:
```yaml
targetContainer: "app"  # Specific container name
targetAllContainers: true  # Restart all containers
```

### 2. Custom Signals
Support different signals for various testing scenarios:
```yaml
signal: "SIGTERM"  # Default, graceful shutdown
signal: "SIGHUP"   # Reload configuration
signal: "SIGUSR1"  # Application-specific behavior
```

### 3. Verify Restart Completion
Wait and verify containers actually restarted:
```yaml
verifyRestart: true  # Wait for container to restart
verifyTimeout: "2m"  # How long to wait
```

### 4. Health Check Validation
Verify pods become healthy after restart:
```yaml
waitForHealthy: true  # Wait for ready status
healthCheckTimeout: "5m"
```

### 5. Batch Restart Patterns
More sophisticated restart patterns:
```yaml
restartPattern: "sequential"  # One at a time
restartPattern: "parallel"    # All at once
restartPattern: "canary"      # One first, then rest
```

## Comparison with Related Actions

| Feature | pod-kill | pod-failure | pod-restart |
|---------|----------|-------------|-------------|
| **Pod Deleted** | Yes | No | No |
| **IP Address Changes** | Yes | No | No |
| **Signal Used** | N/A (deletion) | SIGKILL | SIGTERM |
| **Graceful Shutdown** | No | No | Yes |
| **Tests** | Rescheduling | Crash recovery | Graceful restart |
| **Cleanup Allowed** | Limited | No | Yes |
| **Use Case** | Pod recreation | Crash resilience | Planned maintenance |

## References

- [Kubernetes Pod Lifecycle](https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/)
- [Kubernetes Restart Policy](https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#restart-policy)
- [Container Termination](https://kubernetes.io/docs/concepts/containers/container-lifecycle-hooks/)
- [Graceful Shutdown in Kubernetes](https://cloud.google.com/blog/products/containers-kubernetes/kubernetes-best-practices-terminating-with-grace)
- [SIGTERM vs SIGKILL](https://www.gnu.org/software/libc/manual/html_node/Termination-Signals.html)
- [Pod Exec API](https://kubernetes.io/docs/tasks/debug/debug-application/get-shell-running-container/)
- [Rolling Updates](https://kubernetes.io/docs/tutorials/kubernetes-basics/update/update-intro/)

## Notes

### Why SIGTERM Instead of Other Signals?

SIGTERM (signal 15) is the standard signal for requesting graceful termination:
- Portable across all UNIX-like systems
- Expected by well-designed applications
- Allows cleanup before exit
- Standard practice in Kubernetes

### Relationship to kubectl rollout restart

While `kubectl rollout restart` triggers rolling updates at the workload level, `pod-restart` provides:
- Pod-level granularity
- Support for standalone pods, StatefulSets, DaemonSets
- Chaos-specific features (random selection, safety limits)
- Controlled timing with restartInterval
- Integration with chaos metrics and history

### Production Readiness Considerations

Before using pod-restart in production:
1. Verify applications handle SIGTERM gracefully
2. Set appropriate terminationGracePeriodSeconds
3. Start with dry-run mode
4. Use maxPercentage to limit blast radius
5. Test in staging first
6. Monitor restart counts and duration
7. Have rollback plan ready

---

**Last Updated**: 2025-12-27
**Next Review**: After implementation and initial production testing