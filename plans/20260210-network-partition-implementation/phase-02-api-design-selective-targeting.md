# Phase 2: API Design for Selective Targeting

**Status**: ⏳ Not Started
**Effort**: 6-8 hours
**Risk**: Medium
**Prerequisites**: Phase 1 complete

## Context

Current network-partition is all-or-nothing: blocks ALL traffic except loopback. Users need selective targeting for realistic scenarios:
- "Partition frontend from backend but allow database"
- "Block traffic to specific IP ranges"
- "Simulate partition from specific service"

### Related Files
- `api/v1alpha1/chaosexperiment_types.go` (CRD spec)
- `api/v1alpha1/chaosexperiment_webhook.go` (validation)
- `api/v1alpha1/validation_helpers.go` (helper functions)
- `internal/controller/chaosexperiment_controller.go` (implementation prep)

### Related Docs
- ADR 0011: Network Partition Implementation (Phase 1)
- reports/02-research-findings.md (iptables patterns)
- ADR 0002: Safety Features (maxPercentage, validation patterns)

## Overview

Design API schema for selective network partition targeting supporting IPs, CIDRs, ports, and protocols. Define validation rules, default behaviors, and safety constraints. Prepare foundation for Phase 3 implementation.

## Key Insights

**User Requirements** (from research):
1. Block traffic to specific IPs/CIDRs (e.g., "10.96.0.0/12" - K8s services)
2. Block specific ports (e.g., HTTP but not HTTPS)
3. Block specific protocols (TCP vs UDP)
4. Combine targets (IP range + port)

**API Design Principles**:
- Optional fields (backward compatible)
- Empty = full partition (current behavior)
- Explicit validation (fail fast)
- Composable targets (combine IP + port + protocol)

**Safety Considerations**:
- Don't block loopback (always allow 127.0.0.1)
- Don't block kubelet (10.96.0.1 in most clusters)
- Validate CIDR format
- Warn about blocking cluster DNS (could break pod)

## Requirements

### Functional
1. API fields support IP, CIDR, port, protocol targeting
2. Validation enforces correct formats (CIDR notation, port ranges)
3. Backward compatible (empty = current behavior)
4. Combinable targets (IP + port works)
5. Direction field works with selective targets

### Non-Functional
1. API changes backward compatible
2. CRD validation catches invalid inputs
3. Webhook provides helpful error messages
4. No breaking changes to existing experiments
5. Clear documentation of each field

## Architecture

### API Schema Design

**New Fields** (add to ChaosExperimentSpec):
```go
// TargetIPs specifies exact IP addresses to block (for network-partition)
// Examples: ["10.96.0.50", "192.168.1.100"]
// If empty, blocks all traffic (current behavior)
// +optional
TargetIPs []string `json:"targetIPs,omitempty"`

// TargetCIDRs specifies IP ranges to block using CIDR notation (for network-partition)
// Examples: ["10.96.0.0/12", "192.168.0.0/16"]
// Validates CIDR format (x.x.x.x/y where y is 0-32)
// +kubebuilder:validation:Pattern="^([0-9]{1,3}\\.){3}[0-9]{1,3}/[0-9]{1,2}$"
// +optional
TargetCIDRs []string `json:"targetCIDRs,omitempty"`

// TargetPorts specifies ports to block (for network-partition)
// Can be combined with targetIPs/targetCIDRs
// Examples: [80, 443, 8080]
// +kubebuilder:validation:Minimum=1
// +kubebuilder:validation:Maximum=65535
// +optional
TargetPorts []int32 `json:"targetPorts,omitempty"`

// TargetProtocols specifies protocols to block (for network-partition)
// +kubebuilder:validation:Enum=tcp;udp;icmp
// +optional
TargetProtocols []string `json:"targetProtocols,omitempty"`
```

**Field Interactions**:
```yaml
# Example 1: Block specific IP
targetIPs: ["10.96.100.50"]

# Example 2: Block IP range
targetCIDRs: ["10.96.0.0/12"]

# Example 3: Block specific port on all IPs
targetPorts: [80, 8080]

# Example 4: Combine - block HTTP to specific service
targetIPs: ["10.96.100.50"]
targetPorts: [80]
targetProtocols: ["tcp"]

# Example 5: Empty = full partition (backward compatible)
# No target fields = block all traffic
```

**iptables Translation**:
```bash
# Example 1: targetIPs
iptables -A OUTPUT -d 10.96.100.50 -j DROP

# Example 2: targetCIDRs
iptables -A OUTPUT -d 10.96.0.0/12 -j DROP

# Example 3: targetPorts
iptables -A OUTPUT -p tcp --dport 80 -j DROP
iptables -A OUTPUT -p tcp --dport 8080 -j DROP

# Example 4: Combined
iptables -A OUTPUT -d 10.96.100.50 -p tcp --dport 80 -j DROP

# Example 5: Empty (current behavior)
iptables -A OUTPUT -j DROP
```

### Validation Rules

**CRD-Level Validation** (OpenAPI markers):
```go
// TargetCIDRs pattern validation
// +kubebuilder:validation:Pattern="^([0-9]{1,3}\\.){3}[0-9]{1,3}/[0-9]{1,2}$"

// TargetPorts range validation
// +kubebuilder:validation:Minimum=1
// +kubebuilder:validation:Maximum=65535

// TargetProtocols enum validation
// +kubebuilder:validation:Enum=tcp;udp;icmp
```

**Webhook-Level Validation**:
```go
func validateNetworkPartitionTargets(spec *ChaosExperimentSpec) error {
    // Only validate if action is network-partition
    if spec.Action != "network-partition" {
        return nil
    }

    // Validate CIDR notation
    for _, cidr := range spec.TargetCIDRs {
        if _, _, err := net.ParseCIDR(cidr); err != nil {
            return fmt.Errorf("invalid CIDR notation '%s': %w", cidr, err)
        }
    }

    // Validate IP addresses
    for _, ip := range spec.TargetIPs {
        if net.ParseIP(ip) == nil {
            return fmt.Errorf("invalid IP address '%s'", ip)
        }
    }

    // Validate port ranges
    for _, port := range spec.TargetPorts {
        if port < 1 || port > 65535 {
            return fmt.Errorf("invalid port %d: must be 1-65535", port)
        }
    }

    // Validate protocol combinations
    if len(spec.TargetPorts) > 0 && len(spec.TargetProtocols) == 0 {
        // Default to TCP if ports specified but no protocol
        // Or warn user?
    }

    // Check for dangerous configurations
    if containsDangerousTarget(spec) {
        return admission.Warnings{
            "Warning: Blocking cluster IPs may break pod functionality",
        }
    }

    return nil
}

func containsDangerousTarget(spec *ChaosExperimentSpec) bool {
    // Check for loopback (should never block, but warn)
    for _, ip := range spec.TargetIPs {
        if ip == "127.0.0.1" {
            return true
        }
    }

    // Check for cluster DNS (10.96.0.10 in most clusters)
    // Check for kubelet API
    // etc.

    return false
}
```

### Default Behaviors

**Empty targets** = Full partition (current behavior)
```yaml
spec:
  action: network-partition
  # No target fields = block ALL traffic (backward compatible)
```

**Direction applies to targets**:
```yaml
spec:
  action: network-partition
  direction: egress  # Only apply targets to outgoing traffic
  targetIPs: ["10.96.100.50"]
  # Result: Block outgoing traffic to 10.96.100.50, allow incoming
```

**Protocol defaults**:
- If targetPorts specified but no targetProtocols → assume TCP
- If targetProtocols specified but no ports → apply to all ports of that protocol

## Related Code Files

**To Modify**:
- `api/v1alpha1/chaosexperiment_types.go` - Add new fields to ChaosExperimentSpec
- `api/v1alpha1/chaosexperiment_webhook.go` - Add validation functions
- `api/v1alpha1/validation_helpers.go` - Add CIDR/IP validation helpers

**To Create**:
- `api/v1alpha1/chaosexperiment_validation_test.go` - Unit tests for validation
- Unit tests in `api/v1alpha1/chaosexperiment_webhook_test.go`

**To Prepare** (not modify yet, Phase 3):
- `internal/controller/chaosexperiment_controller.go` - Implementation prep
- `config/samples/chaos_v1alpha1_chaosexperiment_network_partition.yaml` - Update samples

## Implementation Steps

### Step 1: Define API Fields [1 hour]
1. Add new fields to ChaosExperimentSpec in types.go
2. Add OpenAPI validation markers (Pattern, Enum, Min/Max)
3. Add detailed godoc comments explaining each field
4. Add examples in comments
5. Run `make generate manifests` to regenerate CRDs

### Step 2: Create Validation Helpers [1-2 hours]
1. Add `ValidateCIDR(cidr string) error` to validation_helpers.go
2. Add `ValidateIP(ip string) error`
3. Add `ValidatePortRange(port int32) error`
4. Add `IsDangerousTarget(ip string) bool` with cluster IP checks
5. Add comprehensive unit tests for each helper

### Step 3: Implement Webhook Validation [2 hours]
1. Add `validateNetworkPartitionTargets()` function to webhook
2. Integrate into existing ValidateCreate/ValidateUpdate
3. Add CIDR format validation
4. Add IP address validation
5. Add port range validation
6. Add warnings for dangerous targets (DNS, kubelet)
7. Add validation for target field combinations

### Step 4: Write Validation Unit Tests [2 hours]
1. Test valid CIDR formats
2. Test invalid CIDR formats
3. Test valid IP addresses
4. Test invalid IP addresses
5. Test port range boundaries (1, 65535, 0, 65536)
6. Test protocol enum validation
7. Test combined target scenarios
8. Test backward compatibility (empty targets)
9. Test dangerous target warnings

### Step 5: Update CRD Manifests [30 minutes]
1. Run `make manifests` to regenerate CRDs
2. Verify new fields appear in CRD YAML
3. Verify validation markers translated correctly
4. Test CRD applies to cluster without errors
5. Test invalid values rejected by OpenAPI schema

### Step 6: Documentation [1 hour]
1. Add field documentation to ADR 0011
2. Create API reference section
3. Add examples for each target type
4. Document validation rules
5. Document dangerous target warnings
6. Update CLAUDE.md with new fields

### Step 7: Create Sample Configurations [1 hour]
1. Update network_partition.yaml with selective targeting examples
2. Add commented examples for each field
3. Add combination examples
4. Add dangerous target example with warning comment
5. Validate samples with `kubectl apply --dry-run=server`

## Todo List

API Definition:
- [ ] Add TargetIPs field with validation
- [ ] Add TargetCIDRs field with CIDR pattern validation
- [ ] Add TargetPorts field with range validation
- [ ] Add TargetProtocols field with enum validation
- [ ] Add godoc comments with examples
- [ ] Run make generate manifests

Validation Helpers:
- [ ] Create ValidateCIDR helper function
- [ ] Create ValidateIP helper function
- [ ] Create ValidatePortRange helper function
- [ ] Create IsDangerousTarget helper function
- [ ] Add unit tests for each helper (10+ test cases)

Webhook Validation:
- [ ] Implement validateNetworkPartitionTargets
- [ ] Integrate with ValidateCreate
- [ ] Integrate with ValidateUpdate
- [ ] Add CIDR format checks
- [ ] Add IP address checks
- [ ] Add port range checks
- [ ] Add dangerous target warnings
- [ ] Add combination validation logic

Testing:
- [ ] Test valid CIDR formats (5 cases)
- [ ] Test invalid CIDR formats (5 cases)
- [ ] Test valid IPs (IPv4)
- [ ] Test invalid IPs
- [ ] Test port boundaries (0, 1, 65535, 65536)
- [ ] Test protocol enums
- [ ] Test combined targets
- [ ] Test empty targets (backward compat)
- [ ] Test dangerous targets trigger warnings

Documentation:
- [ ] Update ADR 0011 with API reference
- [ ] Document each field purpose and format
- [ ] Add validation rule documentation
- [ ] Add examples for each target type
- [ ] Update CLAUDE.md with selective targeting
- [ ] Create/update sample YAML

Verification:
- [ ] CRD manifests regenerate cleanly
- [ ] Invalid values rejected by OpenAPI
- [ ] Webhook validation catches edge cases
- [ ] All unit tests pass
- [ ] Sample YAML validates successfully

## Success Criteria

**API Design**:
- All new fields have OpenAPI validation markers
- Field names follow Kubernetes API conventions
- Godoc comments explain purpose and format
- Examples provided for each field

**Validation**:
- Invalid CIDR notation rejected
- Invalid IP addresses rejected
- Out-of-range ports rejected
- Dangerous targets trigger warnings
- Backward compatible (empty = full partition)

**Testing**:
- 30+ unit test cases covering validation
- All edge cases tested (boundaries, combinations)
- Backward compatibility verified
- Dangerous target detection works

**Documentation**:
- ADR 0011 updated with API reference
- Sample YAML demonstrates all target types
- CLAUDE.md explains when to use each field
- Validation errors are clear and actionable

## Risk Assessment

**Risk 1**: API changes break existing experiments
- **Probability**: Low
- **Impact**: High
- **Mitigation**: All new fields optional, backward compatible
- **Detection**: E2E tests with old experiment configs

**Risk 2**: CIDR validation too strict/lenient
- **Probability**: Medium
- **Impact**: Medium
- **Mitigation**: Use Go's net.ParseCIDR (standard library)
- **Detection**: Unit tests with diverse CIDR formats

**Risk 3**: Dangerous target warnings too aggressive
- **Probability**: Medium
- **Impact**: Low
- **Mitigation**: Only warn, don't block; clear message
- **Detection**: Manual testing, user feedback

**Risk 4**: Protocol defaults unclear
- **Probability**: Medium
- **Impact**: Low
- **Mitigation**: Document defaults clearly, validate combinations
- **Detection**: Integration tests, user feedback

## Security Considerations

**Loopback Protection**:
- Validate users don't target 127.0.0.1
- Warn if targeting loopback CIDR (127.0.0.0/8)
- Controller should always allow loopback regardless

**Cluster Infrastructure**:
- Warn if targeting cluster IP ranges (10.96.0.0/12)
- Warn if targeting DNS (10.96.0.10 typical)
- Warn if targeting kubelet API
- Don't block these automatically (user may want to test DNS failure)

**Input Validation**:
- Sanitize CIDR input (prevent injection)
- Validate IP format strictly
- Enforce port ranges (1-65535)
- Enum validation for protocols

## Next Steps

After Phase 2 completion:
1. Review API design with team/users
2. Gather feedback on field names and defaults
3. Proceed to Phase 3 (implementation)
4. Consider adding TargetServices/TargetNamespaces (Phase 4 preview)
