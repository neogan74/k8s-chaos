# Phase 4: Service-Aware Partitions

**Status**: ⏳ Not Started
**Effort**: 12-16 hours
**Risk**: High
**Prerequisites**: Phase 3 complete (basic targeting working)

## Context

Phase 3 enables IP/CIDR/port targeting but users must manually resolve service names to IPs. Phase 4 adds service-aware targeting: specify service/namespace names, controller resolves to IPs, handles IP changes.

### Related Files
- `internal/controller/chaosexperiment_controller.go` (service resolution)
- `api/v1alpha1/chaosexperiment_types.go` (new API fields)
- `api/v1alpha1/chaosexperiment_webhook.go` (service validation)

### Related Docs
- Phase 2: API design patterns
- Phase 3: IP/CIDR targeting implementation
- reports/02-research-findings.md: ipset for efficient IP lists

## Overview

Add API fields for service/namespace targeting. Implement service-to-IP resolution. Use ipset for efficient large IP list management. Handle service IP changes and pod churn.

## Key Insights

**User Experience Goal**:
```yaml
# Before (Phase 3): Manual IP resolution
targetIPs: ["10.96.100.50"]  # What's this IP? Redis? MySQL?

# After (Phase 4): Service-aware
targetServices:
  - name: redis-service
    namespace: backend
    ports: [6379]  # Optional: block specific ports only

targetNamespaces:
  - backend  # Block all pods in this namespace
  - database
```

**Implementation Challenges**:
1. **Service IP resolution**: ClusterIP may change (rare but possible)
2. **Namespace pod listing**: Pods come and go (churn)
3. **IP list efficiency**: Namespace may have 100+ pods
4. **Staleness**: Resolved IPs may become stale

**Solution Approach**:
- Resolve at experiment start (not continuously watch)
- Use ipset for efficiency with many IPs
- Document staleness (IPs resolved at start, not updated)
- Consider future enhancement: continuous watching

## Requirements

### Functional
1. API fields: targetServices (name, namespace, ports)
2. API fields: targetNamespaces (list of namespace names)
3. Resolve services to ClusterIP at experiment start
4. Resolve namespaces to pod IPs at experiment start
5. Use ipset for efficient IP list management
6. Combine with existing targetIPs/targetCIDRs
7. Validate service/namespace existence (webhook)

### Non-Functional
1. IP resolution fast (<5s for typical clusters)
2. Efficient iptables rules (use ipset, not 100+ rules)
3. Clear error messages for nonexistent services/namespaces
4. Dry-run shows resolved IPs for transparency
5. Metrics track service-aware targeting usage

## Architecture

### API Schema Design

**New Structures**:
```go
// TargetService represents a Kubernetes service to block
type TargetService struct {
    // Name of the service
    // +kubebuilder:validation:Required
    Name string `json:"name"`

    // Namespace of the service
    // +kubebuilder:validation:Required
    Namespace string `json:"namespace"`

    // Ports to block (optional, blocks all if empty)
    // +optional
    Ports []int32 `json:"ports,omitempty"`
}

// Add to ChaosExperimentSpec:
// TargetServices specifies Kubernetes services to partition from (network-partition)
// Controller resolves service names to ClusterIPs at experiment start
// +optional
TargetServices []TargetService `json:"targetServices,omitempty"`

// TargetNamespaces specifies namespaces to partition from (network-partition)
// Controller resolves to all pod IPs in those namespaces at experiment start
// Warning: Large namespaces may result in many iptables rules
// +optional
TargetNamespaces []string `json:"targetNamespaces,omitempty"`
```

**Webhook Validation**:
```go
func (w *ChaosExperimentWebhook) validateServiceTargets(
    ctx context.Context,
    exp *ChaosExperiment,
) (admission.Warnings, error) {
    var warnings admission.Warnings

    // Validate services exist
    for _, svc := range exp.Spec.TargetServices {
        service := &corev1.Service{}
        key := types.NamespacedName{
            Name:      svc.Name,
            Namespace: svc.Namespace,
        }
        if err := w.Client.Get(ctx, key, service); err != nil {
            return warnings, fmt.Errorf(
                "target service '%s/%s' not found: %w",
                svc.Namespace, svc.Name, err,
            )
        }

        // Warn if service has no ClusterIP
        if service.Spec.ClusterIP == "" || service.Spec.ClusterIP == "None" {
            warnings = append(warnings, fmt.Sprintf(
                "Service '%s/%s' is headless (no ClusterIP), will resolve to pod IPs instead",
                svc.Namespace, svc.Name,
            ))
        }
    }

    // Validate namespaces exist
    for _, ns := range exp.Spec.TargetNamespaces {
        namespace := &corev1.Namespace{}
        if err := w.Client.Get(ctx, types.NamespacedName{Name: ns}, namespace); err != nil {
            return warnings, fmt.Errorf("target namespace '%s' not found: %w", ns, err)
        }

        // Warn if namespace has many pods
        podList := &corev1.PodList{}
        if err := w.Client.List(ctx, podList, client.InNamespace(ns)); err == nil {
            if len(podList.Items) > 50 {
                warnings = append(warnings, fmt.Sprintf(
                    "Namespace '%s' has %d pods, may generate many iptables rules",
                    ns, len(podList.Items),
                ))
            }
        }
    }

    return warnings, nil
}
```

### Service Resolution Logic

**Service to IP**:
```go
func (r *ChaosExperimentReconciler) resolveServiceTargets(
    ctx context.Context,
    exp *chaosv1alpha1.ChaosExperiment,
) ([]ServiceTarget, error) {
    log := ctrl.LoggerFrom(ctx)
    var resolved []ServiceTarget

    for _, svc := range exp.Spec.TargetServices {
        service := &corev1.Service{}
        key := types.NamespacedName{
            Name:      svc.Name,
            Namespace: svc.Namespace,
        }

        if err := r.Get(ctx, key, service); err != nil {
            log.Error(err, "Failed to resolve service", "service", key)
            return nil, fmt.Errorf("service %s not found", key)
        }

        // Handle headless services (resolve to endpoints)
        if service.Spec.ClusterIP == "None" {
            endpoints, err := r.resolveHeadlessService(ctx, svc)
            if err != nil {
                return nil, err
            }
            resolved = append(resolved, endpoints...)
            continue
        }

        // Regular service with ClusterIP
        resolved = append(resolved, ServiceTarget{
            IP:    service.Spec.ClusterIP,
            Ports: svc.Ports,
        })

        log.Info("Resolved service to ClusterIP",
            "service", key,
            "clusterIP", service.Spec.ClusterIP)
    }

    return resolved, nil
}

func (r *ChaosExperimentReconciler) resolveHeadlessService(
    ctx context.Context,
    svc TargetService,
) ([]ServiceTarget, error) {
    // Get endpoints for headless service
    endpoints := &corev1.Endpoints{}
    key := types.NamespacedName{
        Name:      svc.Name,
        Namespace: svc.Namespace,
    }

    if err := r.Get(ctx, key, endpoints); err != nil {
        return nil, err
    }

    var targets []ServiceTarget
    for _, subset := range endpoints.Subsets {
        for _, addr := range subset.Addresses {
            targets = append(targets, ServiceTarget{
                IP:    addr.IP,
                Ports: svc.Ports,
            })
        }
    }

    return targets, nil
}

type ServiceTarget struct {
    IP    string
    Ports []int32
}
```

**Namespace to Pod IPs**:
```go
func (r *ChaosExperimentReconciler) resolveNamespaceTargets(
    ctx context.Context,
    namespaces []string,
) ([]string, error) {
    log := ctrl.LoggerFrom(ctx)
    var allIPs []string

    for _, ns := range namespaces {
        podList := &corev1.PodList{}
        if err := r.List(ctx, podList, client.InNamespace(ns)); err != nil {
            return nil, fmt.Errorf("failed to list pods in namespace %s: %w", ns, err)
        }

        nsIPs := []string{}
        for _, pod := range podList.Items {
            if pod.Status.PodIP != "" && pod.Status.Phase == corev1.PodRunning {
                nsIPs = append(nsIPs, pod.Status.PodIP)
            }
        }

        log.Info("Resolved namespace to pod IPs",
            "namespace", ns,
            "podCount", len(nsIPs))

        allIPs = append(allIPs, nsIPs...)
    }

    return allIPs, nil
}
```

### ipset Integration

**Why ipset**:
- Efficient for large IP lists (O(1) lookup vs O(n) iptables rules)
- Single iptables rule referencing set
- Dynamic updates possible (add/remove IPs without rule changes)

**Implementation**:
```bash
# Create ipset (in ephemeral container script)
ipset create chaos_blocked hash:net

# Add IPs from resolved services/namespaces
{% for ip in resolved_ips %}
ipset add chaos_blocked {{ip}}
{% endfor %}

# Single iptables rule using the set
iptables -A CHAOS_PARTITION -m set --match-set chaos_blocked dst -j DROP

# Cleanup
iptables -D CHAOS_PARTITION -m set --match-set chaos_blocked dst -j DROP || true
ipset destroy chaos_blocked || true
```

**Fallback** (if ipset not available):
```bash
# Detect ipset availability
if ! command -v ipset &> /dev/null; then
    # Fallback: multiple iptables rules
    {% for ip in resolved_ips %}
    iptables -A CHAOS_PARTITION -d {{ip}} -j DROP
    {% endfor %}
fi
```

### Script Generation Updates

```go
func (r *ChaosExperimentReconciler) generatePartitionScript(
    exp *chaosv1alpha1.ChaosExperiment,
    chainName string,
    timeoutSeconds int,
) (string, error) {
    // Resolve service-aware targets
    resolvedIPs := []string{}

    if len(exp.Spec.TargetServices) > 0 {
        serviceTargets, err := r.resolveServiceTargets(ctx, exp)
        if err != nil {
            return "", err
        }
        for _, target := range serviceTargets {
            resolvedIPs = append(resolvedIPs, target.IP)
        }
    }

    if len(exp.Spec.TargetNamespaces) > 0 {
        nsIPs, err := r.resolveNamespaceTargets(ctx, exp.Spec.TargetNamespaces)
        if err != nil {
            return "", err
        }
        resolvedIPs = append(resolvedIPs, nsIPs...)
    }

    // Combine with explicit targetIPs/targetCIDRs
    allIPs := append(resolvedIPs, exp.Spec.TargetIPs...)
    allCIDRs := exp.Spec.TargetCIDRs

    // Use ipset if many IPs (threshold: 10+)
    useIPSet := len(allIPs) > 10

    script := r.buildScriptWithIPSet(chainName, allIPs, allCIDRs, useIPSet, timeoutSeconds)
    return script, nil
}
```

## Related Code Files

**To Modify**:
- `api/v1alpha1/chaosexperiment_types.go` - Add TargetService struct and fields
- `api/v1alpha1/chaosexperiment_webhook.go` - Add service/namespace validation
- `internal/controller/chaosexperiment_controller.go`:
  - Add `resolveServiceTargets()`
  - Add `resolveNamespaceTargets()`
  - Update `generatePartitionScript()` to call resolution
  - Update script template for ipset

**To Create**:
- `internal/controller/service_resolution.go` - Service resolution helpers
- `internal/controller/service_resolution_test.go` - Unit tests

**To Update**:
- `config/samples/chaos_v1alpha1_chaosexperiment_network_partition.yaml` - Add service examples
- `test/e2e/network_partition_test.go` - Add service-aware tests

## Implementation Steps

### Step 1: Define API [2 hours]
1. Add TargetService struct
2. Add targetServices field to ChaosExperimentSpec
3. Add targetNamespaces field to ChaosExperimentSpec
4. Add OpenAPI validation markers
5. Run make generate manifests

### Step 2: Implement Service Resolution [3-4 hours]
1. Create service_resolution.go file
2. Implement resolveServiceTargets()
3. Handle ClusterIP services
4. Handle headless services (via Endpoints)
5. Implement resolveNamespaceTargets()
6. Add error handling and logging

### Step 3: Implement ipset Integration [2-3 hours]
1. Create ipset script template
2. Add ipset creation commands
3. Add IP addition loop
4. Add iptables rule with match-set
5. Add ipset cleanup
6. Add fallback for no-ipset environments

### Step 4: Update Script Generation [2 hours]
1. Call resolution functions in generatePartitionScript()
2. Combine resolved IPs with explicit targets
3. Decide ipset vs multiple rules (threshold logic)
4. Pass resolved IPs to script template
5. Update dry-run to show resolved IPs

### Step 5: Add Webhook Validation [2 hours]
1. Implement validateServiceTargets()
2. Check service existence
3. Check namespace existence
4. Warn about headless services
5. Warn about large namespaces (50+ pods)

### Step 6: Write Tests [3-4 hours]
1. Unit test service resolution (ClusterIP)
2. Unit test headless service resolution
3. Unit test namespace resolution
4. Unit test ipset script generation
5. E2E test service targeting
6. E2E test namespace targeting
7. E2E test combined targeting

### Step 7: Update Samples [1 hour]
1. Add service targeting example
2. Add namespace targeting example
3. Add combined example (services + namespaces + IPs)
4. Add comments explaining resolution behavior

## Todo List

API Definition:
- [ ] Create TargetService struct
- [ ] Add targetServices field
- [ ] Add targetNamespaces field
- [ ] Add validation markers
- [ ] Run make generate manifests

Service Resolution:
- [ ] Create service_resolution.go
- [ ] Implement resolveServiceTargets()
- [ ] Handle ClusterIP services
- [ ] Handle headless services
- [ ] Implement resolveNamespaceTargets()
- [ ] Filter non-running pods
- [ ] Add comprehensive logging

ipset Integration:
- [ ] Create ipset script template
- [ ] Add ipset create command
- [ ] Add IP addition loop
- [ ] Add iptables match-set rule
- [ ] Add ipset destroy cleanup
- [ ] Add fallback for no-ipset
- [ ] Test ipset availability detection

Script Generation:
- [ ] Call resolution functions
- [ ] Combine resolved with explicit IPs
- [ ] Implement ipset threshold logic (10+ IPs)
- [ ] Pass IPs to template rendering
- [ ] Update dry-run with resolved IPs

Webhook Validation:
- [ ] Implement validateServiceTargets()
- [ ] Check service existence
- [ ] Check namespace existence
- [ ] Warn about headless services
- [ ] Warn about large namespaces

Testing:
- [ ] Unit test: ClusterIP service resolution
- [ ] Unit test: Headless service resolution
- [ ] Unit test: Namespace resolution
- [ ] Unit test: ipset script generation
- [ ] Unit test: Fallback script generation
- [ ] E2E test: Service targeting works
- [ ] E2E test: Namespace targeting works
- [ ] E2E test: Combined targets work
- [ ] E2E test: ipset applied correctly

Samples:
- [ ] Add service targeting example
- [ ] Add namespace targeting example
- [ ] Add combined example
- [ ] Add headless service example
- [ ] Document resolution timing (start-time)

## Success Criteria

**Service Resolution**:
- ClusterIP services resolve correctly
- Headless services resolve to endpoints
- Namespaces resolve to running pod IPs
- Resolution errors handled gracefully

**ipset Integration**:
- ipset used for 10+ IPs
- Fallback works without ipset
- Single iptables rule when using ipset
- Cleanup removes ipset

**Validation**:
- Nonexistent services rejected
- Nonexistent namespaces rejected
- Helpful warnings for edge cases
- Dry-run shows resolved IPs

**Testing**:
- Unit tests cover all resolution paths
- E2E tests verify actual network blocking
- Combined targeting works correctly

## Risk Assessment

**Risk 1**: Service IP changes after resolution
- **Probability**: Low
- **Impact**: Medium
- **Mitigation**: Document behavior, consider future watch enhancement
- **Detection**: Experiment targets old IP, user reports

**Risk 2**: Namespace with many pods (100+)
- **Probability**: Medium
- **Impact**: Medium
- **Mitigation**: ipset for efficiency, warn in webhook
- **Detection**: Slow injection, many iptables rules

**Risk 3**: ipset not available in container image
- **Probability**: Medium
- **Impact**: Low
- **Mitigation**: Fallback to multiple rules, detect availability
- **Detection**: ipset command not found, fallback used

**Risk 4**: Headless service resolution complexity
- **Probability**: Low
- **Impact**: Medium
- **Mitigation**: Use Endpoints API, test thoroughly
- **Detection**: E2E tests, unit tests

**Risk 5**: Staleness of resolved IPs
- **Probability**: High
- **Impact**: Low
- **Mitigation**: Document clearly, resolve at start only
- **Detection**: User confusion, documentation questions

## Security Considerations

**Service Discovery**:
- Controller needs list/get services permission (already has)
- Controller needs list/get endpoints (for headless services)
- Validate user has permission to target services in other namespaces

**Namespace Listing**:
- Controller needs list pods in any namespace (already has)
- Large namespace may expose many pod IPs
- Document that namespace targeting reveals pod IPs

**ipset Security**:
- ipset isolated to ephemeral container
- Custom set name (chaos_blocked) avoids conflicts
- Destroyed on cleanup

## Next Steps

After Phase 4 completion:
1. Gather user feedback on service-aware targeting
2. Monitor staleness issues (IP changes)
3. Consider continuous watching enhancement (Phase 6)
4. Consider TargetSelector alternative (pod labels across namespaces)
5. Proceed to Phase 5 (testing and validation)
