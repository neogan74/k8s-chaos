# Current Network-Partition Implementation Analysis

**Date**: 2026-02-10
**Status**: Analysis Complete
**Commit**: 58615ef (network-partition branch)

## Executive Summary

Network-partition chaos action already implemented but lacks documentation, sample YAML, and advanced targeting features. Current implementation uses iptables-based complete network isolation. Gaps identified: ADR missing, no sample config, limited targeting (full partition only, no selective IP/service blocking).

## Current Implementation Status

### What's Implemented

**Core Functionality** (internal/controller/chaosexperiment_controller.go:2587-2766):
- Action handler: `handleNetworkPartition()`
- Injection mechanism: Ephemeral containers with NET_ADMIN capability
- Network isolation tool: iptables (via nicolaka/netshoot image)
- Direction support: both, ingress, egress
- Duration-based lifecycle with automatic cleanup
- Safety features: dry-run, maxPercentage, exclusions, production protection
- Metrics tracking and history recording

**API Definition** (api/v1alpha1/chaosexperiment_types.go:49,162-166):
- Action enum includes "network-partition"
- Direction field: Enum(both, ingress, egress), default="both"
- Duration field: Required (validated)
- Reuses existing selector/count/namespace fields

**Validation** (api/v1alpha1/chaosexperiment_webhook.go:257-261):
- Duration requirement enforced
- Direction enum validated in CRD schema
- Safety constraints applied (namespace existence, selector effectiveness, production protection)

**E2E Tests** (test/e2e/network_partition_test.go):
- Basic injection test
- Ephemeral container verification
- Duration requirement validation
- Complete test coverage for happy path

### Implementation Mechanism

**Technical Approach**:
1. Injects ephemeral container into target pods
2. Container runs shell script executing iptables rules:
   - Allow loopback (127.0.0.1) traffic
   - Drop INPUT/OUTPUT based on direction
   - Sleep for duration
   - Flush iptables rules (cleanup)
3. Automatic cleanup after duration expires
4. Tracked in status.affectedPods for lifecycle management

**iptables Rules Applied**:
```bash
# Always allow loopback
iptables -A INPUT -i lo -j ACCEPT
iptables -A OUTPUT -o lo -j ACCEPT

# Direction=both or ingress
iptables -A INPUT -j DROP

# Direction=both or egress
iptables -A OUTPUT -j DROP

# Cleanup after duration
iptables -F
```

**Container Spec**:
- Image: nicolaka/netshoot (public, includes iptables)
- Capability: NET_ADMIN
- Lifecycle: runs for duration then exits
- Unique name: network-partition-{timestamp}

## Gaps Identified

### Documentation Gaps

**Missing ADR**: No ADR documenting design decisions, alternatives considered, consequences
- Should follow template: docs/adr/0000-adr-template.md
- Should explain why iptables vs tc netem vs network policies
- Should document NET_ADMIN requirement and cluster compatibility

**Missing Sample YAML**: No config/samples/chaos_v1alpha1_chaosexperiment_network_partition.yaml
- Users have no reference configuration
- Different from pod-network-loss (packet loss) - needs clear differentiation

**Incomplete CLAUDE.md**: Action listed in enum but no detailed docs
- Missing from "Supported actions" description
- No usage examples
- No troubleshooting guidance

### Functional Gaps

**Limited Targeting Capabilities**:
- Current: Complete network partition (all traffic blocked except loopback)
- Missing: Selective blocking by target IP/CIDR/service/namespace
- Use case: Simulate partition between specific services (e.g., frontend → backend)
- Implementation impact: Would require additional API fields + iptables rules

**No Advanced Options**:
- Cannot specify protocols (TCP/UDP/ICMP)
- Cannot specify ports
- Cannot specify source/destination filters
- All-or-nothing approach limits realistic failure scenarios

**Cleanup Concerns**:
- iptables -F flushes ALL rules, may be too aggressive if pod has existing rules
- No verification that rules were actually removed
- No handling of partial cleanup failures
- Risk of lingering rules if container crashes before cleanup

### Comparison with pod-network-loss

| Feature | pod-network-loss | network-partition |
|---------|------------------|-------------------|
| Mechanism | tc netem | iptables DROP |
| Granularity | Packet loss percentage | Complete isolation |
| Direction | egress/ingress/both | egress/ingress/both |
| Use case | Flaky network simulation | Split-brain, partition testing |
| Cleanup | tc qdisc del | iptables -F |
| Risk level | Medium (packets delayed) | High (complete isolation) |

**Differentiation**: Both are network chaos but serve different purposes
- network-loss: Gradual degradation (5-40% loss)
- network-partition: Binary failure (complete isolation)

## Strengths

1. **Follows established patterns**: Reuses ephemeral container injection from cpu/memory-stress
2. **Safety-first design**: All safety features (dry-run, maxPercentage, exclusions) implemented
3. **Clean lifecycle**: Automatic cleanup via sleep + flush
4. **Good observability**: Metrics, events, history tracking all present
5. **Proper validation**: Multi-layer (OpenAPI + webhook)
6. **Complete E2E coverage**: Tests verify actual behavior

## Risks & Limitations

**iptables -F Aggressiveness**:
- Flushes all filter table rules, not just chaos-injected ones
- May conflict with network policies, service meshes, CNI plugins
- Mitigation: Use specific chain deletion or mark rules for selective cleanup

**NET_ADMIN Requirement**:
- Not all clusters allow NET_ADMIN in ephemeral containers
- PSP/PSA may block injection
- Documentation should warn about cluster requirements

**No Partial Partition**:
- Cannot simulate realistic scenarios like "partition from service X but allow service Y"
- Current design is all-or-nothing
- Enhancement needed for advanced chaos testing

**Cleanup Reliability**:
- Depends on container running to completion
- If pod deleted before duration expires, rules may linger
- Need graceful termination handling

## Recommendations

### Immediate (Documentation)
1. Create ADR 0011: Network Partition Implementation
2. Create sample YAML with multiple scenarios
3. Update CLAUDE.md with detailed action description
4. Add troubleshooting section for NET_ADMIN issues

### Short-term (Enhancements)
1. Add targetIPs/targetCIDRs for selective blocking
2. Implement safer cleanup (use custom iptables chain)
3. Add validation for cluster PSP/PSA compatibility
4. Document differences vs pod-network-loss clearly

### Long-term (Advanced Features)
1. Protocol/port filtering support
2. Service-to-service partition simulation
3. Alternative implementations (tc netem, network policies)
4. Automatic detection and preservation of existing rules

## Related Files

**Implementation**:
- internal/controller/chaosexperiment_controller.go:2587-2766
- api/v1alpha1/chaosexperiment_types.go:49,162-166
- api/v1alpha1/chaosexperiment_webhook.go:257-261

**Testing**:
- test/e2e/network_partition_test.go

**Documentation** (missing):
- docs/adr/0011-network-partition-implementation.md (to create)
- config/samples/chaos_v1alpha1_chaosexperiment_network_partition.yaml (to create)

## Conclusion

Implementation is production-ready for basic network partition scenarios. Core functionality solid, follows project patterns, includes safety features. Main gaps are documentation and advanced targeting capabilities. Recommend documenting first, then enhancing targeting as Phase 2 based on user feedback.
