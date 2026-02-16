# E2E Tests Fixes - Summary

## Issues Found

### 1. History Namespace Configuration Issue
**Problem**: Controller was trying to create history records in "system" namespace, which doesn't exist in e2e cluster (actual namespace is "k8s-chaos-system").

**Root Cause**: The `--history-namespace=system` flag in manager.yaml was not being substituted by kustomize because it's inside an args array.

**Solution**: Use Kubernetes downward API to inject the actual namespace:
```yaml
args:
  - --history-namespace=$(POD_NAMESPACE)
env:
- name: POD_NAMESPACE
  valueFrom:
    fieldRef:
      fieldPath: metadata.namespace
```

**Files Changed**:
- `config/manager/manager.yaml`

### 2. Concurrent Ephemeral Container Injection Conflicts
**Problem**: Multiple reconciliation loops were trying to inject ephemeral containers simultaneously, causing "object has been modified" errors.

**Root Cause**: The controller Update() call didn't handle conflict errors. When multiple retr files attempted to update the same pod concurrently, conflicts occurred.

**Solution**: Implemented retry logic with exponential backoff:
- Created `updatePodWithEphemeralContainer()` helper function
- Retries up to 5 times with exponential backoff (100ms → 200ms → 400ms → 800ms → 1600ms)
- Fetches latest pod version before each retry attempt
- Applied to all 4 chaos actions: pod-cpu-stress, pod-memory-stress, pod-network-loss, pod-disk-fill

**Files Changed**:
- `internal/controller/chaosexperiment_controller.go`

### 3. Missing RBAC Permission for Namespaces
**Problem**: Controller couldn't watch namespaces resource.

**Solution**: Added `watch` verb to namespaces RBAC permission.

**Files Changed**:
- `internal/controller/chaosexperiment_controller.go` (RBAC marker)
- `config/rbac/role.yaml` (regenerated)

## Implementation Details

### Helper Function: updatePodWithEphemeralContainer()

```go
func (r *ChaosExperimentReconciler) updatePodWithEphemeralContainer(ctx context.Context, pod *corev1.Pod, ephemeralContainer corev1.EphemeralContainer) error {
	maxRetries := 5
	backoff := time.Millisecond * 100

	for attempt := 0; attempt < maxRetries; attempt++ {
		// Get latest pod version
		currentPod := &corev1.Pod{}
		if err := r.Get(ctx, client.ObjectKeyFromObject(pod), currentPod); err != nil {
			return fmt.Errorf("failed to get current pod state: %w", err)
		}

		// Append ephemeral container
		currentPod.Spec.EphemeralContainers = append(currentPod.Spec.EphemeralContainers, ephemeralContainer)

		// Try to update
		err := r.Client.SubResource("ephemeralcontainers").Update(ctx, currentPod)
		if err == nil {
			return nil // Success
		}

		// Retry on conflict errors
		if strings.Contains(err.Error(), "the object has been modified") ||
		   strings.Contains(err.Error(), "Operation cannot be fulfilled") {
			if attempt < maxRetries-1 {
				log.Info("Conflict detected, retrying", "attempt", attempt+1)
				time.Sleep(backoff)
				backoff *= 2 // Exponential backoff
				continue
			}
		}

		return fmt.Errorf("failed to inject ephemeral container after %d attempts: %w", attempt+1, err)
	}

	return fmt.Errorf("failed to inject ephemeral container: max retries exceeded")
}
```

### Updated Injection Functions

All four injection functions now use the helper:

1. **injectCPUStressContainer()** - line ~590
2. **injectMemoryStressContainer()** - line ~1536
3. **injectNetworkLossContainer()** - line ~2006
4. **injectDiskFillContainer()** - line ~2060

Before:
```go
currentPod.Spec.EphemeralContainers = append(currentPod.Spec.EphemeralContainers, ephemeralContainer)
if err := r.Client.SubResource("ephemeralcontainers").Update(ctx, currentPod); err != nil {
	return "", fmt.Errorf("failed to inject ephemeral container: %w", err)
}
```

After:
```go
if err := r.updatePodWithEphemeralContainer(ctx, pod, ephemeralContainer); err != nil {
	return "", err
}
```

## Testing

### E2E Test Coverage

Tests validate:
- ✅ Pod-network-loss action (basic injection, dry-run, correlation, maxPercentage, selectors)
- ✅ Pod-memory-stress action (basic injection, multiple workers, dry-run, exclusion labels)
- ✅ Pod-disk-fill action (basic injection, validation)
- ✅ Manager deployment and metrics
- ✅ Safety features (maxPercentage, exclusion labels, dry-run)
- ✅ Concurrent experiments
- ✅ Selector and namespace isolation

### Expected Results

With these fixes:
1. History records should be created successfully in k8s-chaos-system namespace
2. Ephemeral container injection should succeed even with concurrent reconciliations
3. No RBAC permission errors for namespace watching
4. All chaos actions should transition to "Running" phase correctly
5. Tests should pass without timing out

## Benefits

1. **Reliability**: Handles concurrent reconciliations gracefully
2. **Flexibility**: Works regardless of deployment namespace name
3. **Robustness**: Automatic retry with backoff prevents transient failures
4. **Production-Ready**: Follows Kubernetes best practices for resource updates

## Related Files

- `config/manager/manager.yaml` - Manager deployment configuration
- `config/rbac/role.yaml` - RBAC permissions
- `internal/controller/chaosexperiment_controller.go` - Controller logic
- `test/e2e/*.go` - E2E test suites

## Next Steps

1. ✅ Run `make test-e2e` to verify all fixes
2. ⏳ Review test results
3. ⏳ Commit changes if tests pass
4. ⏳ Update documentation
