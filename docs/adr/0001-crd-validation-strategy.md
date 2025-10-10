# ADR 0001: CRD Validation Strategy

**Status:** Accepted

**Date:** 2025-10-10

**Authors:** k8s-chaos team

## Context

The ChaosExperiment CRD is the primary interface for users to define chaos engineering experiments in their Kubernetes clusters. Without proper validation, users could:

1. Specify invalid action types that the controller doesn't support
2. Set negative or extremely large count values that could cause unintended damage
3. Create experiments with empty selectors that could affect unintended pods
4. Target non-existent namespaces
5. Provide malformed duration strings

Invalid configurations can lead to:
- Runtime errors in the controller
- Unintended chaos affecting critical workloads
- Poor user experience with unclear error messages
- Increased load on the API server from invalid resource reconciliation

We need a robust validation strategy that provides early feedback to users and prevents dangerous configurations.

## Decision

We will implement **multi-layer validation** using Kubebuilder validation markers for compile-time OpenAPI schema generation, combined with admission webhooks for complex runtime validation.

### Layer 1: OpenAPI Schema Validation (Implemented)

Using Kubebuilder markers in `api/v1alpha1/chaosexperiment_types.go`:

```go
// Action validation - only allow supported chaos types
// +kubebuilder:validation:Required
// +kubebuilder:validation:Enum=pod-kill;pod-delay;node-drain
Action string `json:"action"`

// Namespace validation - must be non-empty
// +kubebuilder:validation:Required
// +kubebuilder:validation:MinLength=1
Namespace string `json:"namespace"`

// Selector validation - at least one label required
// +kubebuilder:validation:Required
// +kubebuilder:validation:MinProperties=1
Selector map[string]string `json:"selector"`

// Count validation - reasonable bounds
// +kubebuilder:validation:Minimum=1
// +kubebuilder:validation:Maximum=100
// +kubebuilder:default=1
Count int `json:"count,omitempty"`

// Duration validation - must match time format
// +kubebuilder:validation:Pattern="^([0-9]+(s|m|h))+$"
Duration string `json:"duration,omitempty"`
```

**Benefits:**
- Validation happens at API server level (before reaching controller)
- No custom code required - declarative markers
- Automatic OpenAPI schema generation
- Clear error messages to kubectl users
- No performance overhead in controller

**Limitations:**
- Cannot validate cross-field dependencies
- Cannot check if namespace exists
- Cannot validate against cluster state
- Limited to simple type/range/pattern checks

### Layer 2: Admission Webhooks (Future Enhancement)

For complex validation scenarios, we will implement ValidatingWebhookConfiguration:

**Planned validations:**
1. **Namespace existence check** - Verify target namespace exists before creating experiment
2. **Selector effectiveness** - Ensure selector matches at least one pod
3. **Safety limits** - Enforce percentage-based limits (e.g., max 30% of pods)
4. **Exclusion policies** - Prevent targeting pods with protection labels
5. **Cross-field validation** - Validate duration is required for pod-delay action

**Implementation plan:**
```go
// pkg/webhook/chaosexperiment_webhook.go
func (r *ChaosExperiment) ValidateCreate() error {
    // Check namespace exists
    // Validate selector matches pods
    // Apply safety policies
    // Check duration for delay actions
}
```

### Layer 3: Controller Runtime Validation

Additional safety checks in the reconciliation loop:

1. **Revalidate before execution** - Verify conditions haven't changed
2. **Graceful handling** - Log warnings instead of failing on validation errors
3. **Status updates** - Report validation issues in experiment status

## Alternatives Considered

### 1. Controller-only Validation
**Rejected** - Users would only discover errors after creating resources, leading to poor UX.

### 2. Admission Controller Only
**Rejected** - More complex to implement and deploy, harder to test, adds operational overhead.

### 3. No Validation (Trust Users)
**Rejected** - Chaos experiments can be destructive; we must prevent dangerous configurations.

### 4. External Policy Engine (OPA/Kyverno)
**Deferred** - Useful for organizational policies, but not a replacement for basic field validation. Can be added later for advanced use cases.

## Consequences

### Positive

1. **Better UX** - Users get immediate feedback on invalid configurations
2. **Safer operations** - Prevents dangerous experiments from being created
3. **Reduced controller complexity** - Less validation code in reconciliation loop
4. **Self-documenting** - OpenAPI schema shows valid values in kubectl explain
5. **Standard approach** - Follows Kubernetes best practices

### Negative

1. **Development overhead** - Need to maintain validation markers
2. **Schema updates** - Changes require `make manifests` and CRD redeployment
3. **Future webhook complexity** - Admission webhooks add deployment complexity
4. **Testing burden** - Need tests for both positive and negative validation cases

### Neutral

1. **API evolution** - Adding new actions requires updating enum validation
2. **Backward compatibility** - Must carefully version validation rules
3. **Documentation** - Need to document validation rules in user guides

## Implementation Status

### Completed
- [x] OpenAPI schema validation for all spec fields
- [x] Enum validation for action field
- [x] Range validation for count field
- [x] Pattern validation for duration field
- [x] MinProperties validation for selector
- [x] Required field markers

### Planned
- [ ] Admission webhook scaffolding
- [ ] Namespace existence validation
- [ ] Selector effectiveness check
- [ ] Safety policy enforcement (max percentage)
- [ ] Exclusion label support
- [ ] Duration requirement for delay actions
- [ ] Comprehensive validation tests

## References

- [Kubebuilder CRD Validation](https://book.kubebuilder.io/reference/markers/crd-validation.html)
- [Kubernetes API Conventions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md)
- [Admission Controllers](https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/)
- [OpenAPI v3 Schema](https://swagger.io/specification/)

## Notes

The current validation implementation (Layer 1) is sufficient for the v1alpha1 API. We should implement webhooks (Layer 2) before promoting to v1beta1 or v1, as they provide essential safety guarantees for production use.

Users should be made aware through documentation that:
1. The controller performs best-effort validation at runtime
2. Experiments may fail reconciliation if cluster state changes
3. RBAC permissions may prevent experiments from executing even if validation passes
