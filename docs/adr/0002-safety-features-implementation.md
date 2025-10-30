# ADR 0002: Safety Features Implementation

**Status**: Accepted
**Date**: 2025-10-30
**Author**: k8s-chaos team

## Context

The k8s-chaos operator currently provides powerful chaos engineering capabilities (pod-kill, pod-delay, node-drain, pod-cpu-stress), but lacks safety mechanisms to prevent accidental damage in production environments. Without proper safeguards, operators could:
- Accidentally affect too many pods, causing service outages
- Impact critical system pods (kube-system, monitoring, etc.)
- Run chaos experiments in production without proper controls
- Execute destructive actions without preview/confirmation

## Decision

We will implement a comprehensive multi-layer safety system with the following features:

### 1. Dry-Run Mode

**Purpose**: Preview experiment impact before execution

**Implementation**:
- Add `dryRun: bool` field to ChaosExperimentSpec
- When enabled, controller lists affected resources and updates status without executing chaos
- Status message shows: "DRY RUN: Would affect N pods: [pod1, pod2, ...]"
- No actual chaos actions performed
- Requeue disabled to prevent repeated dry-runs

**Example**:
```yaml
spec:
  action: pod-kill
  dryRun: true  # Preview mode
  namespace: production
  selector:
    app: web-server
```

### 2. Maximum Percentage Limit

**Purpose**: Prevent affecting too many resources simultaneously

**Implementation**:
- Add `maxPercentage: int` field to ChaosExperimentSpec (1-100)
- Validate at webhook level: calculate affected percentage and reject if exceeded
- Formula: `(count / totalMatchingPods) * 100 <= maxPercentage`
- Default: no limit (backward compatible)
- Warning if count would exceed limit

**Example**:
```yaml
spec:
  action: pod-kill
  namespace: production
  selector:
    app: web-server
  count: 10
  maxPercentage: 30  # Fail if would affect >30% of matching pods
```

### 3. Exclusion Labels

**Purpose**: Protect critical pods from chaos experiments

**Implementation**:
- Support `chaos.gushchin.dev/exclude: "true"` label on pods
- Support `chaos.gushchin.dev/exclude: "true"` annotation on namespaces
- Controller filters out excluded pods before selection
- Webhook validates at least one non-excluded pod exists
- Add metrics for excluded pods

**Label Examples**:
```yaml
# On Pod
metadata:
  labels:
    chaos.gushchin.dev/exclude: "true"

# On Namespace
metadata:
  annotations:
    chaos.gushchin.dev/exclude: "true"
```

### 4. Production Namespace Protection

**Purpose**: Require explicit opt-in for sensitive namespaces

**Implementation**:
- Add `allowProduction: bool` field to ChaosExperimentSpec (default: false)
- Identify production namespaces via:
  - Annotation: `chaos.gushchin.dev/production: "true"`
  - Label: `environment: production` or `env: prod`
  - Namespace name patterns: `production`, `prod-*`, `*-production`, `*-prod`
- Webhook rejects experiments in production namespaces unless `allowProduction: true`
- Error message guides users to add allowProduction flag

**Example**:
```yaml
# Namespace marked as production
apiVersion: v1
kind: Namespace
metadata:
  name: production
  annotations:
    chaos.gushchin.dev/production: "true"

---
# Experiment requires explicit approval
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: prod-experiment
spec:
  action: pod-kill
  namespace: production
  allowProduction: true  # Required for production namespaces
  selector:
    app: web-server
```

### 5. Combined Safety Example

All safety features work together:

```yaml
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: safe-prod-experiment
spec:
  action: pod-cpu-stress
  namespace: production
  selector:
    app: web-server
    chaos.gushchin.dev/exclude: "false"  # Explicitly not excluded
  count: 5
  maxPercentage: 20          # Max 20% of pods
  allowProduction: true      # Explicit production approval
  dryRun: false              # First run with dryRun: true to preview
  cpuLoad: 80
  duration: "5m"
```

## Alternatives Considered

### Alternative 1: Require External Approval System
**Approach**: Integrate with external approval systems (Slack, PagerDuty, etc.)
**Rejected**: Too complex, requires external dependencies, slows down development cycles

### Alternative 2: Time-based Protection Windows
**Approach**: Block chaos during business hours or peak times
**Deferred**: Good feature but can be added later via cron scheduling integration

### Alternative 3: Blast Radius Calculation
**Approach**: Calculate impact on service availability before running
**Deferred**: Requires service topology awareness, too complex for initial implementation

### Alternative 4: Rollback Capability
**Approach**: Automatically revert changes if problems detected
**Rejected**: Not feasible for destructive actions (pod deletion), better to prevent than rollback

## Implementation Details

### API Changes

**ChaosExperimentSpec additions**:
```go
// DryRun mode previews affected resources without executing chaos
// +kubebuilder:default=false
// +optional
DryRun bool `json:"dryRun,omitempty"`

// MaxPercentage limits the percentage of matching resources that can be affected
// +kubebuilder:validation:Minimum=1
// +kubebuilder:validation:Maximum=100
// +optional
MaxPercentage int `json:"maxPercentage,omitempty"`

// AllowProduction explicitly allows experiments in production namespaces
// +kubebuilder:default=false
// +optional
AllowProduction bool `json:"allowProduction,omitempty"`
```

### Webhook Validation

```go
func (w *ChaosExperimentWebhook) validateSafetyConstraints(ctx context.Context, exp *ChaosExperiment) error {
    // 1. Check production namespace protection
    if err := w.validateProductionNamespace(ctx, exp); err != nil {
        return err
    }

    // 2. Filter excluded pods
    eligiblePods := w.filterExcludedPods(ctx, exp)

    // 3. Check maximum percentage limit
    if err := w.validateMaxPercentage(exp, eligiblePods); err != nil {
        return err
    }

    return nil
}
```

### Controller Changes

```go
func (r *ChaosExperimentReconciler) handleWithSafety(ctx context.Context, exp *ChaosExperiment) {
    // 1. Filter excluded pods
    eligiblePods := r.filterExcludedPods(ctx, exp)

    // 2. Handle dry-run mode
    if exp.Spec.DryRun {
        return r.handleDryRun(ctx, exp, eligiblePods)
    }

    // 3. Execute chaos action
    return r.executeAction(ctx, exp, eligiblePods)
}
```

## Consequences

### Positive
- **Production-ready**: Safe to deploy in production environments
- **Prevent accidents**: Multiple layers of protection against mistakes
- **Developer-friendly**: Dry-run mode enables testing without risk
- **Audit compliance**: Production approval flags create audit trail
- **Backward compatible**: All new fields are optional, existing CRDs work unchanged

### Negative
- **Additional complexity**: More validation logic and fields
- **Performance overhead**: Additional namespace/pod lookups for validation
- **User friction**: Production experiments require extra flags (intentional)

### Risks and Mitigations

**Risk**: Users bypass safety features
- **Mitigation**: Make safety the default (opt-out, not opt-in), clear documentation

**Risk**: Performance impact from additional validation
- **Mitigation**: Cache namespace annotations, optimize pod filtering

**Risk**: Exclusion labels can be forgotten
- **Mitigation**: Add metrics to track excluded vs affected pods, warn in logs

## Metrics

New Prometheus metrics:
- `chaos_safety_dry_runs_total` - Count of dry-run experiments
- `chaos_safety_production_blocks_total` - Experiments blocked by production protection
- `chaos_safety_percentage_blocks_total` - Experiments blocked by percentage limit
- `chaos_safety_excluded_resources_total` - Resources excluded by labels

## Testing Strategy

1. **Unit Tests**:
   - Test exclusion label filtering
   - Test percentage calculations
   - Test production namespace detection

2. **Webhook Tests**:
   - Test production namespace validation
   - Test maxPercentage validation with various scenarios
   - Test error messages are clear and actionable

3. **E2E Tests**:
   - Test dry-run mode shows correct preview
   - Test exclusion labels work across experiments
   - Test production protection blocks unauthorized experiments

## Migration Path

**Existing experiments**: No changes required, all safety fields are optional

**Gradual adoption**:
1. Deploy updated operator with safety features disabled by default
2. Add exclusion labels to critical pods
3. Mark production namespaces with annotations
4. Enable maxPercentage limits gradually
5. Require allowProduction flag via policy (optional)

## Future Enhancements

- **Automatic exclusion**: Auto-exclude system namespaces (kube-system, kube-public)
- **Time-based protection**: Block experiments during business hours
- **Blast radius scoring**: Calculate and display potential impact
- **Approval workflows**: Integrate with GitOps/approval systems
- **Resource quotas**: Limit concurrent experiments per namespace
- **Rollback hooks**: Pre/post experiment validation hooks

## References

- [Chaos Engineering Principles](https://principlesofchaos.org/)
- [Kubernetes Production Best Practices](https://kubernetes.io/docs/setup/best-practices/)
- [SRE: Practicing Chaos Engineering](https://sre.google/workbook/managing-load/)
