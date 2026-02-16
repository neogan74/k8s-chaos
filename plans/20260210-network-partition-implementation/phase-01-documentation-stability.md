# Phase 1: Documentation & Stability

**Status**: ⏳ Not Started
**Effort**: 4-6 hours
**Risk**: Low
**Prerequisites**: None

## Context

Network-partition action implemented (commit 58615ef) but lacks:
- ADR documenting design decisions
- Sample YAML for user reference
- Safe cleanup mechanism (current `iptables -F` too aggressive)
- Clear differentiation from pod-network-loss action

### Related Files
- `internal/controller/chaosexperiment_controller.go` (lines 2587-2766)
- `api/v1alpha1/chaosexperiment_types.go` (lines 49, 162-166)
- `api/v1alpha1/chaosexperiment_webhook.go` (lines 257-261)
- `test/e2e/network_partition_test.go`

### Related Docs
- ADR 0007: Pod Network Loss (similar pattern)
- ADR 0005: Pod CPU Stress (ephemeral containers)
- reports/01-current-implementation-analysis.md
- reports/02-research-findings.md

## Overview

Document existing network-partition implementation through ADR, create sample configurations, improve cleanup safety using custom iptables chains, update project documentation.

## Key Insights

**Current Cleanup Risk**:
- `iptables -F` flushes ALL rules in filter table
- May remove CNI plugin rules (Calico, Cilium, etc.)
- May remove service mesh rules (Istio, Linkerd)
- Risk of pod networking failure after experiment completes

**Safe Cleanup Pattern** (from research):
```bash
# Setup: Create custom chain
iptables -N CHAOS_PARTITION_<ID>
iptables -I INPUT 1 -j CHAOS_PARTITION_<ID>
iptables -I OUTPUT 1 -j CHAOS_PARTITION_<ID>

# Apply rules to custom chain
iptables -A CHAOS_PARTITION_<ID> -i lo -j ACCEPT  # Allow loopback
iptables -A CHAOS_PARTITION_<ID> -j DROP          # Drop others

# Cleanup: Remove only chaos chain
iptables -D INPUT -j CHAOS_PARTITION_<ID>
iptables -D OUTPUT -j CHAOS_PARTITION_<ID>
iptables -F CHAOS_PARTITION_<ID>
iptables -X CHAOS_PARTITION_<ID>
```

**Benefits**:
- Isolated from other iptables rules
- No impact on CNI/mesh networking
- Deterministic cleanup verification
- Easy debugging (inspect chain separately)

## Requirements

### Functional
1. ADR 0011 documents iptables approach, alternatives, consequences
2. Sample YAML shows common scenarios (full partition, directions)
3. Custom chain cleanup prevents CNI conflicts
4. CLAUDE.md updated with action description
5. Differentiate from pod-network-loss clearly

### Non-Functional
1. No breaking changes to existing API
2. Backward compatible with existing experiments
3. Zero downtime deployment
4. E2E tests still pass

## Architecture

### File Structure
```
docs/adr/
  0011-network-partition-implementation.md  [NEW]

config/samples/
  chaos_v1alpha1_chaosexperiment_network_partition.yaml  [NEW]

internal/controller/
  chaosexperiment_controller.go  [MODIFY: injectNetworkPartitionContainer]

CLAUDE.md  [MODIFY: add network-partition description]

test/e2e/
  network_partition_test.go  [MODIFY: test custom chain cleanup]
```

### Component Changes

**1. ADR 0011 Structure**:
- Context: Network partition for split-brain testing
- Decision: iptables with ephemeral containers
- Alternatives: tc netem, NetworkPolicy, service mesh
- Consequences: NET_ADMIN required, PSA impact
- Implementation status: Core complete, enhancements planned

**2. Sample YAML Scenarios**:
- Basic full partition (both directions)
- Ingress-only partition
- Egress-only partition
- With dry-run enabled
- With maxPercentage limit
- With experimentDuration

**3. Custom Chain Implementation**:
```go
func (r *ChaosExperimentReconciler) injectNetworkPartitionContainer(
    ctx context.Context,
    pod *corev1.Pod,
    direction string,
    timeoutSeconds int
) (string, error) {
    chainName := fmt.Sprintf("CHAOS_PARTITION_%d", time.Now().Unix())

    script := fmt.Sprintf(`
# Create custom chain
iptables -N %s

# Insert jump to custom chain (high priority)
iptables -I INPUT 1 -j %s
iptables -I OUTPUT 1 -j %s

# Allow loopback in custom chain
iptables -A %s -i lo -j ACCEPT
iptables -A %s -o lo -j ACCEPT

# Block traffic based on direction in custom chain
if [ "%s" = "both" ] || [ "%s" = "ingress" ]; then
  iptables -A %s -j DROP
fi
if [ "%s" = "both" ] || [ "%s" = "egress" ]; then
  iptables -A %s -j DROP
fi

# Wait for duration
sleep %d

# Safe cleanup: Remove only chaos chain
iptables -D INPUT -j %s || true
iptables -D OUTPUT -j %s || true
iptables -F %s || true
iptables -X %s || true
`, chainName, chainName, chainName, chainName, chainName,
   direction, direction, chainName,
   direction, direction, chainName,
   timeoutSeconds,
   chainName, chainName, chainName, chainName)

    // Rest of implementation unchanged...
}
```

## Related Code Files

**To Create**:
- `docs/adr/0011-network-partition-implementation.md`
- `config/samples/chaos_v1alpha1_chaosexperiment_network_partition.yaml`

**To Modify**:
- `internal/controller/chaosexperiment_controller.go` (injectNetworkPartitionContainer function)
- `CLAUDE.md` (add network-partition to supported actions)
- `test/e2e/network_partition_test.go` (add custom chain verification)

## Implementation Steps

### Step 1: Create ADR 0011 [1-2 hours]
1. Copy ADR template
2. Fill in context: network partition use cases, requirements
3. Document decision: iptables with ephemeral containers, custom chains
4. List alternatives: tc netem (for degradation), NetworkPolicy (less precise), service mesh (env-specific)
5. Document consequences: NET_ADMIN requirement, PSA compatibility, cleanup safety
6. Mark implementation status: Core complete, targeting enhancements planned

### Step 2: Create Sample YAML [30 minutes]
1. Create base example with full partition (direction: both)
2. Add commented examples for ingress/egress only
3. Include safety feature examples (dryRun, maxPercentage)
4. Add experimentDuration example
5. Add inline comments explaining each field
6. Reference ADR 0011 in header comment

### Step 3: Implement Custom Chain Cleanup [1-2 hours]
1. Update `injectNetworkPartitionContainer` function
2. Generate unique chain name with timestamp
3. Replace direct iptables rules with custom chain approach
4. Add cleanup error handling (`|| true` for idempotency)
5. Add logging for chain creation/deletion
6. Test locally with Kind cluster

### Step 4: Update CLAUDE.md [30 minutes]
1. Add network-partition to "Supported actions" list
2. Add description: "Simulate network partitions using iptables"
3. Add to action comparison table vs pod-network-loss
4. Add note about NET_ADMIN requirement
5. Add troubleshooting section for PSA issues

### Step 5: Enhance E2E Tests [1 hour]
1. Add test case verifying custom chain creation
2. Verify chain cleanup after experiment completes
3. Test that CNI rules remain intact
4. Add negative test for PSA Restricted namespace (if possible)

### Step 6: Update Documentation Links [30 minutes]
1. Update README.md to reference ADR 0011
2. Add network-partition to features list
3. Link sample YAML from README
4. Update docs/METRICS.md with network-partition metrics

## Todo List

Documentation:
- [ ] Create ADR 0011 following template structure
- [ ] Document iptables approach and design rationale
- [ ] List alternatives (tc netem, NetworkPolicy, mesh)
- [ ] Document consequences (NET_ADMIN, PSA, cleanup)
- [ ] Create sample YAML with 6 scenarios
- [ ] Add inline comments explaining fields
- [ ] Update CLAUDE.md supported actions section
- [ ] Add troubleshooting section for PSA issues

Implementation:
- [ ] Modify injectNetworkPartitionContainer for custom chains
- [ ] Generate unique chain name per experiment
- [ ] Add chain creation before rules
- [ ] Update cleanup to remove chain safely
- [ ] Add error handling with `|| true` for idempotency
- [ ] Add debug logging for chain operations

Testing:
- [ ] Add E2E test verifying custom chain exists
- [ ] Add E2E test verifying chain cleanup
- [ ] Test CNI rules unaffected after experiment
- [ ] Run full E2E suite to ensure no regressions
- [ ] Manual test in Kind cluster

Final Steps:
- [ ] Update README.md with network-partition
- [ ] Update docs/METRICS.md if needed
- [ ] Commit with message referencing ADR 0011
- [ ] Update plan.md Phase 1 status to complete

## Success Criteria

**Documentation**:
- ADR 0011 approved and merged
- Sample YAML demonstrates all common scenarios
- CLAUDE.md clearly differentiates network-partition from network-loss
- Troubleshooting section addresses PSA compatibility

**Implementation**:
- Custom chains prevent CNI rule conflicts
- E2E tests verify chain creation and cleanup
- No regressions in existing network-partition tests
- Backward compatible with experiments created before changes

**Quality**:
- ADR follows template structure completely
- Sample YAML passes `kubectl apply --dry-run=server`
- Code changes have unit test coverage
- E2E tests pass in CI

## Risk Assessment

**Risk 1**: Custom chain changes break existing experiments
- **Probability**: Low
- **Impact**: Medium
- **Mitigation**: Backward compatible, only changes cleanup logic
- **Detection**: E2E tests fail

**Risk 2**: Chain name collisions between experiments
- **Probability**: Very Low
- **Impact**: Low
- **Mitigation**: Timestamp-based unique names
- **Detection**: iptables returns error on duplicate chain

**Risk 3**: Documentation incomplete or unclear
- **Probability**: Low
- **Impact**: Low
- **Mitigation**: Follow ADR template, peer review
- **Detection**: User questions, confusion

## Security Considerations

**NET_ADMIN Capability**:
- Required for iptables manipulation
- Document in ADR and sample YAML
- Add warning in CLAUDE.md
- Include PSA compatibility check recommendation

**Rule Isolation**:
- Custom chains prevent accidental rule removal
- Reduces risk of breaking pod networking
- Easier to audit (inspect chain separately)

**Cleanup Verification**:
- `|| true` ensures cleanup doesn't fail experiment
- Idempotent cleanup safe for retries
- Logging helps debug cleanup issues

## Next Steps

After Phase 1 completion:
1. Gather user feedback on documentation
2. Identify common use cases for selective targeting
3. Proceed to Phase 2 (API design) if targeting needed
4. Consider NetworkPolicy alternative if PSA blocking adoption
