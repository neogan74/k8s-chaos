# Phase 3: Basic Selective Targeting Implementation

**Status**: ⏳ Not Started
**Effort**: 8-12 hours
**Risk**: Medium
**Prerequisites**: Phase 2 complete (API designed and validated)

## Context

API fields defined (targetIPs, targetCIDRs, targetPorts, targetProtocols) in Phase 2. Now implement iptables rule generation translating API fields to selective blocking rules using custom chains from Phase 1.

### Related Files
- `internal/controller/chaosexperiment_controller.go` (handler implementation)
- `api/v1alpha1/chaosexperiment_types.go` (API reference)
- `test/e2e/network_partition_test.go` (E2E tests)
- `config/samples/chaos_v1alpha1_chaosexperiment_network_partition.yaml`

### Related Docs
- Phase 1: Custom chain cleanup pattern
- Phase 2: API field definitions and validation
- reports/02-research-findings.md: iptables patterns

## Overview

Implement controller logic generating iptables rules from targetIPs, targetCIDRs, targetPorts, targetProtocols. Support combinations, apply direction correctly, maintain safety features (dry-run, maxPercentage, exclusions).

## Key Insights

**iptables Rule Generation**:
```bash
# No targets = full partition (current behavior)
iptables -A CHAOS_PARTITION -j DROP

# targetIPs
iptables -A CHAOS_PARTITION -d 10.96.100.50 -j DROP

# targetCIDRs
iptables -A CHAOS_PARTITION -d 10.96.0.0/12 -j DROP

# targetPorts (requires protocol)
iptables -A CHAOS_PARTITION -p tcp --dport 80 -j DROP

# Combined: IP + port + protocol
iptables -A CHAOS_PARTITION -d 10.96.100.50 -p tcp --dport 80 -j DROP

# Multiple targets = multiple rules
for ip in targetIPs:
    iptables -A CHAOS_PARTITION -d $ip -j DROP
```

**Direction Handling**:
- `ingress`: Apply rules to INPUT chain only
- `egress`: Apply rules to OUTPUT chain only
- `both`: Apply rules to both INPUT and OUTPUT chains

**Rule Priority**:
- Loopback ACCEPT rules first (highest priority)
- Selective DROP rules second
- No catch-all DROP if targets specified

## Requirements

### Functional
1. Generate iptables rules from API fields
2. Support targetIPs, targetCIDRs, targetPorts, targetProtocols
3. Support field combinations
4. Apply direction correctly (INPUT/OUTPUT chains)
5. Maintain backward compatibility (empty targets = full partition)
6. Preserve safety features (dry-run, maxPercentage)

### Non-Functional
1. Generated rules efficient (no duplicates)
2. Rule generation idempotent
3. Error handling for rule application failures
4. Logging for debugging
5. Metrics track selective targeting

## Architecture

### Rule Generation Logic

**Function Signature**:
```go
func (r *ChaosExperimentReconciler) generatePartitionScript(
    exp *chaosv1alpha1.ChaosExperiment,
    chainName string,
    timeoutSeconds int,
) string
```

**Script Structure**:
```bash
#!/bin/sh
set -e

CHAIN="%s"

# Create custom chain
iptables -N $CHAIN

# Attach to INPUT/OUTPUT based on direction
%s  # Placeholder for chain attachment

# Always allow loopback
iptables -A $CHAIN -i lo -j ACCEPT 2>/dev/null || true
iptables -A $CHAIN -o lo -j ACCEPT 2>/dev/null || true

# Apply selective rules or full partition
%s  # Placeholder for blocking rules

# Sleep for duration
sleep %d

# Cleanup
iptables -D INPUT -j $CHAIN 2>/dev/null || true
iptables -D OUTPUT -j $CHAIN 2>/dev/null || true
iptables -F $CHAIN 2>/dev/null || true
iptables -X $CHAIN 2>/dev/null || true
```

**Chain Attachment Logic**:
```go
func getChainAttachment(direction string, chainName string) string {
    switch direction {
    case "ingress":
        return fmt.Sprintf("iptables -I INPUT 1 -j %s", chainName)
    case "egress":
        return fmt.Sprintf("iptables -I OUTPUT 1 -j %s", chainName)
    case "both":
        return fmt.Sprintf("iptables -I INPUT 1 -j %s\niptables -I OUTPUT 1 -j %s",
            chainName, chainName)
    default:
        return fmt.Sprintf("iptables -I INPUT 1 -j %s\niptables -I OUTPUT 1 -j %s",
            chainName, chainName)
    }
}
```

**Blocking Rules Logic**:
```go
func generateBlockingRules(exp *chaosv1alpha1.ChaosExperiment, chainName string) string {
    var rules []string

    // If no selective targets, do full partition (backward compatible)
    if len(exp.Spec.TargetIPs) == 0 &&
       len(exp.Spec.TargetCIDRs) == 0 &&
       len(exp.Spec.TargetPorts) == 0 {
        rules = append(rules, fmt.Sprintf("iptables -A %s -j DROP", chainName))
        return strings.Join(rules, "\n")
    }

    // Generate rules for targetIPs
    for _, ip := range exp.Spec.TargetIPs {
        rules = append(rules, generateIPRule(chainName, ip, exp))
    }

    // Generate rules for targetCIDRs
    for _, cidr := range exp.Spec.TargetCIDRs {
        rules = append(rules, generateCIDRRule(chainName, cidr, exp))
    }

    // Generate rules for targetPorts
    for _, port := range exp.Spec.TargetPorts {
        rules = append(rules, generatePortRule(chainName, port, exp))
    }

    return strings.Join(rules, "\n")
}

func generateIPRule(chainName, ip string, exp *chaosv1alpha1.ChaosExperiment) string {
    // Base rule
    rule := fmt.Sprintf("iptables -A %s -d %s", chainName, ip)

    // Add protocol if specified
    if len(exp.Spec.TargetProtocols) > 0 {
        for _, proto := range exp.Spec.TargetProtocols {
            protoRule := fmt.Sprintf("%s -p %s", rule, proto)

            // Add ports if specified
            if len(exp.Spec.TargetPorts) > 0 {
                for _, port := range exp.Spec.TargetPorts {
                    return fmt.Sprintf("%s --dport %d -j DROP", protoRule, port)
                }
            }

            return fmt.Sprintf("%s -j DROP", protoRule)
        }
    }

    // No protocol specified, block all traffic to this IP
    return fmt.Sprintf("%s -j DROP", rule)
}

func generatePortRule(chainName string, port int32, exp *chaosv1alpha1.ChaosExperiment) string {
    // Default to TCP if no protocol specified
    protocols := exp.Spec.TargetProtocols
    if len(protocols) == 0 {
        protocols = []string{"tcp"}
    }

    var rules []string
    for _, proto := range protocols {
        rule := fmt.Sprintf("iptables -A %s -p %s --dport %d -j DROP",
            chainName, proto, port)
        rules = append(rules, rule)
    }

    return strings.Join(rules, "\n")
}
```

### Dry-Run Enhancement

**Selective Target Preview**:
```go
func (r *ChaosExperimentReconciler) handleDryRunWithTargets(
    ctx context.Context,
    exp *chaosv1alpha1.ChaosExperiment,
    eligiblePods []corev1.Pod,
) (ctrl.Result, error) {
    log := ctrl.LoggerFrom(ctx)

    targetSummary := r.buildTargetSummary(exp)

    exp.Status.Message = fmt.Sprintf(
        "DRY RUN: Would inject network partition into %d pod(s). Targets: %s",
        len(eligiblePods),
        targetSummary,
    )

    // List pods and targets
    podNames := []string{}
    for _, pod := range eligiblePods {
        podNames = append(podNames, pod.Name)
    }

    log.Info("Dry-run network partition",
        "pods", podNames,
        "targets", targetSummary,
        "direction", exp.Spec.Direction)

    return ctrl.Result{}, r.Status().Update(ctx, exp)
}

func (r *ChaosExperimentReconciler) buildTargetSummary(exp *chaosv1alpha1.ChaosExperiment) string {
    if len(exp.Spec.TargetIPs) == 0 &&
       len(exp.Spec.TargetCIDRs) == 0 &&
       len(exp.Spec.TargetPorts) == 0 {
        return "all traffic"
    }

    parts := []string{}

    if len(exp.Spec.TargetIPs) > 0 {
        parts = append(parts, fmt.Sprintf("IPs: %v", exp.Spec.TargetIPs))
    }
    if len(exp.Spec.TargetCIDRs) > 0 {
        parts = append(parts, fmt.Sprintf("CIDRs: %v", exp.Spec.TargetCIDRs))
    }
    if len(exp.Spec.TargetPorts) > 0 {
        parts = append(parts, fmt.Sprintf("Ports: %v", exp.Spec.TargetPorts))
    }
    if len(exp.Spec.TargetProtocols) > 0 {
        parts = append(parts, fmt.Sprintf("Protocols: %v", exp.Spec.TargetProtocols))
    }

    return strings.Join(parts, ", ")
}
```

## Related Code Files

**To Modify**:
- `internal/controller/chaosexperiment_controller.go`:
  - `handleNetworkPartition()` - use new script generation
  - `injectNetworkPartitionContainer()` - accept generated script
  - Add `generatePartitionScript()`
  - Add `generateBlockingRules()`
  - Add `buildTargetSummary()`

**To Update**:
- `config/samples/chaos_v1alpha1_chaosexperiment_network_partition.yaml` - Add selective targeting examples
- `test/e2e/network_partition_test.go` - Add selective targeting tests

**To Create**:
- `internal/controller/network_partition_rules_test.go` - Unit tests for rule generation

## Implementation Steps

### Step 1: Refactor Script Generation [2-3 hours]
1. Extract script generation into separate function
2. Implement `generatePartitionScript()` with templating
3. Implement `getChainAttachment()` for direction handling
4. Add error handling for invalid inputs
5. Add debug logging for generated scripts

### Step 2: Implement Rule Generation Logic [3-4 hours]
1. Implement `generateBlockingRules()` main dispatcher
2. Implement `generateIPRule()` with protocol/port support
3. Implement `generateCIDRRule()` similar to IP rules
4. Implement `generatePortRule()` with protocol defaults
5. Handle empty targets (backward compat - full partition)
6. Handle rule combinations (IP + port + protocol)

### Step 3: Update Handler Function [1-2 hours]
1. Modify `handleNetworkPartition()` to use new script gen
2. Pass exp to script generation functions
3. Update dry-run to show target summary
4. Update status messages to include targets
5. Update metrics labels with target types

### Step 4: Enhance Dry-Run [1 hour]
1. Implement `buildTargetSummary()` helper
2. Update dry-run handler to show targets
3. Add logging for selective targets
4. Test dry-run with various target combinations

### Step 5: Write Unit Tests [2-3 hours]
1. Test full partition generation (empty targets)
2. Test single IP rule generation
3. Test multiple IPs
4. Test CIDR rules
5. Test port rules (with/without protocol)
6. Test combined rules (IP + port + protocol)
7. Test direction application (ingress/egress/both)
8. Test script template rendering

### Step 6: Update E2E Tests [2 hours]
1. Add test for selective IP blocking
2. Add test for CIDR blocking
3. Add test for port blocking
4. Add test for combined targets
5. Add test verifying rules applied correctly
6. Add test for backward compat (empty targets)

### Step 7: Update Samples [30 minutes]
1. Add selective IP example
2. Add CIDR example
3. Add port blocking example
4. Add combined targets example
5. Add comments explaining each scenario

## Todo List

Refactoring:
- [ ] Extract script generation into separate function
- [ ] Create generatePartitionScript() function
- [ ] Create getChainAttachment() for directions
- [ ] Add script template with placeholders
- [ ] Add error handling for invalid inputs

Rule Generation:
- [ ] Implement generateBlockingRules() dispatcher
- [ ] Implement generateIPRule() with combinations
- [ ] Implement generateCIDRRule()
- [ ] Implement generatePortRule() with protocol defaults
- [ ] Handle empty targets (backward compatibility)
- [ ] Handle multiple targets of same type
- [ ] Handle cross-field combinations

Handler Updates:
- [ ] Modify handleNetworkPartition() to use new gen
- [ ] Pass experiment struct to generation functions
- [ ] Update container injection call
- [ ] Update status messages with target info
- [ ] Update metrics with target type labels

Dry-Run:
- [ ] Implement buildTargetSummary() helper
- [ ] Update dry-run handler with target preview
- [ ] Add logging for selective targets
- [ ] Test dry-run with all target combinations

Unit Testing:
- [ ] Test empty targets (full partition)
- [ ] Test single IP rule
- [ ] Test multiple IPs
- [ ] Test CIDR rules
- [ ] Test port rules (TCP default)
- [ ] Test port rules with protocol
- [ ] Test IP + port + protocol combo
- [ ] Test direction handling (3 cases)
- [ ] Test script rendering

E2E Testing:
- [ ] Test selective IP blocking works
- [ ] Test CIDR blocking works
- [ ] Test port blocking works
- [ ] Test combined targets work
- [ ] Verify iptables rules applied correctly
- [ ] Test backward compat (empty targets)
- [ ] Test dry-run shows targets

Samples:
- [ ] Add targetIPs example
- [ ] Add targetCIDRs example
- [ ] Add targetPorts example
- [ ] Add combined targets example
- [ ] Add comments explaining scenarios

## Success Criteria

**Implementation**:
- Generated scripts syntactically correct bash
- iptables rules apply without errors
- Direction handling correct (INPUT/OUTPUT)
- Backward compatible (empty targets work)
- All safety features preserved

**Testing**:
- Unit tests cover all target combinations (15+ cases)
- E2E tests verify actual network blocking
- Dry-run shows accurate target preview
- Backward compatibility verified

**Quality**:
- Code follows project patterns
- Error messages helpful and actionable
- Logging sufficient for debugging
- Metrics track selective targeting usage

## Risk Assessment

**Risk 1**: Generated iptables rules invalid
- **Probability**: Medium
- **Impact**: High
- **Mitigation**: Extensive unit tests, E2E verification
- **Detection**: E2E tests fail, pod logs show errors

**Risk 2**: Rule combinations unexpected behavior
- **Probability**: Medium
- **Impact**: Medium
- **Mitigation**: Test all combinations, document behavior
- **Detection**: E2E tests, manual verification

**Risk 3**: Direction handling incorrect
- **Probability**: Low
- **Impact**: High
- **Mitigation**: Unit tests for each direction
- **Detection**: E2E tests show wrong traffic blocked

**Risk 4**: Backward compatibility broken
- **Probability**: Low
- **Impact**: High
- **Mitigation**: Explicit test for empty targets
- **Detection**: Existing E2E tests fail

## Security Considerations

**Rule Injection Safety**:
- Sanitize all inputs (done in webhook Phase 2)
- Use parameterized rule generation (no string concat with user input)
- Validate generated script before injection

**Loopback Protection**:
- Always add loopback ACCEPT rules first
- Even if user specifies 127.0.0.1 in targets (webhook warns)
- Ensures pod can talk to itself

**Cleanup Safety**:
- Custom chains ensure only chaos rules removed
- `|| true` ensures cleanup doesn't fail experiment
- Idempotent cleanup for retries

## Next Steps

After Phase 3 completion:
1. Monitor user feedback on selective targeting
2. Identify common use cases for service-aware targeting
3. Proceed to Phase 4 if service/namespace targeting requested
4. Consider performance optimization if many rules generated
