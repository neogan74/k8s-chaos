# Network-Partition Implementation Plan

**Created**: 2026-02-10
**Status**: Ready for Implementation
**Branch**: network-partition
**Implementation Commit**: 58615ef

## Overview

Network-partition chaos action ALREADY IMPLEMENTED but lacks documentation and advanced targeting features. Plan covers: documenting existing implementation via ADR, creating sample configs, improving cleanup safety, and phased enhancement roadmap for selective IP/service targeting.

## Context

Commit 58615ef introduced network-partition action using iptables-based complete network isolation via ephemeral containers. Implementation follows project patterns (safety features, metrics, E2E tests) but missing ADR, sample YAML, and selective targeting capabilities users need for realistic split-brain scenarios.

## Scope

**In Scope**:
- Document current iptables implementation (ADR 0011)
- Create sample YAML configurations
- Improve cleanup safety (custom iptables chains)
- Add selective targeting (IPs, CIDRs, services, namespaces)
- PSA compatibility validation
- Enhanced E2E tests

**Out of Scope**:
- NetworkPolicy alternative implementation (future)
- Service mesh integration (deferred)
- Protocol/port filtering (Phase 4)
- eBPF-based advanced filtering (not planned)

## Key Decisions

1. **Keep iptables approach**: Industry-standard for partitions, correct choice vs tc netem
2. **Custom chain cleanup**: Safer than `iptables -F`, prevents CNI/mesh conflicts
3. **Phased enhancement**: Document first, then basic targeting, then service-aware
4. **PSA validation**: Detect Restricted environments, fail fast with helpful errors

## Implementation Phases

### Phase 1: Documentation & Stability [COMPLETED]
**Status**: ✅ Completed (2026-02-12)
**Effort**: 4-6 hours (actual: ~5 hours)
**Risk**: Low
**Files**: 3 new, 2 modified

Document existing implementation, create samples, improve cleanup safety.

**Deliverables**:
- ✅ ADR 0011 created documenting iptables approach and custom chains
- ✅ Sample YAML with 8 comprehensive examples
- ✅ Custom chain implementation for safe cleanup
- ✅ CLAUDE.md updated with network-partition documentation
- ✅ Enhanced E2E tests with custom chain verification
- ✅ All tests passing (unit tests, fmt, vet)

### Phase 2: API Design for Selective Targeting [READY]
**Status**: ⏳ Not Started
**Effort**: 6-8 hours
**Risk**: Medium
**Files**: 4 modified, 1 new

Design and validate API schema for targetIPs, targetCIDRs, targetPorts fields.

### Phase 3: Basic Selective Targeting Implementation [READY]
**Status**: ⏳ Not Started
**Effort**: 8-12 hours
**Risk**: Medium
**Files**: 5 modified, 3 new

Implement iptables rules for IP/CIDR/port-based selective blocking.

### Phase 4: Service-Aware Partitions [READY]
**Status**: ⏳ Not Started
**Effort**: 12-16 hours
**Risk**: High
**Files**: 6 modified, 4 new

Service/namespace targeting with IP resolution and ipset management.

### Phase 5: Testing & Validation [READY]
**Status**: ⏳ Not Started
**Effort**: 8-10 hours
**Risk**: Medium
**Files**: 4 new, 2 modified

Comprehensive E2E tests, integration tests, validation scenarios.

## Success Criteria

### Phase 1 (Completed)
- [x] ADR 0011 documents iptables approach and design decisions
- [x] Sample YAML demonstrates common partition scenarios
- [x] Custom chain cleanup prevents CNI rule conflicts
- [x] Documentation explains differences vs pod-network-loss
- [x] E2E tests enhanced with custom chain verification

### Phase 2-5 (Pending)
- [ ] Selective targeting by IP/CIDR works in E2E tests
- [ ] Service-aware targeting resolves service IPs correctly
- [ ] PSA validation detects incompatible environments
- [ ] All safety features work with new targeting options
- [ ] E2E tests cover realistic split-brain scenarios
- [ ] Grafana metrics track partition targets

## Dependencies

**External**:
- nicolaka/netshoot image (already in use)
- iptables in ephemeral containers (cluster must allow NET_ADMIN)
- Kubernetes 1.24+ (ephemeral containers GA)

**Internal**:
- Existing ephemeral container injection logic
- Safety feature pipeline (dry-run, maxPercentage, exclusions)
- Metrics and history recording infrastructure
- Webhook validation framework

## Risks & Mitigations

**Risk 1**: iptables -F conflicts with CNI rules
- **Impact**: High - could break pod networking
- **Mitigation**: Custom chains (Phase 1), tested in E2E

**Risk 2**: PSA Restricted blocks NET_ADMIN
- **Impact**: Medium - won't work in hardened namespaces
- **Mitigation**: Pre-flight validation, clear error messages

**Risk 3**: Service IP resolution adds complexity
- **Impact**: Medium - requires service watching, IP tracking
- **Mitigation**: Phased approach, Phase 4 only if Phase 3 successful

**Risk 4**: ipset not available in all images
- **Impact**: Low - fallback to multiple iptables rules
- **Mitigation**: Detect ipset, use alternative if missing

## Related Documentation

- Current implementation: internal/controller/chaosexperiment_controller.go:2587-2766
- E2E tests: test/e2e/network_partition_test.go
- Analysis: reports/01-current-implementation-analysis.md
- Research: reports/02-research-findings.md
- Similar actions: ADR 0007 (pod-network-loss), ADR 0005 (pod-cpu-stress)

## Timeline Estimate

**Phase 1**: 1 day (documentation & stability)
**Phase 2**: 1 day (API design)
**Phase 3**: 2 days (basic targeting)
**Phase 4**: 2-3 days (service-aware)
**Phase 5**: 1-2 days (testing)

**Total**: 7-9 days for complete implementation

**Recommended Approach**: Implement Phase 1 immediately, gather user feedback before Phases 2-4.

## Notes

- Implementation already production-ready for basic partition scenarios
- Main value add is selective targeting for realistic failure injection
- Consider user feedback after Phase 1 before investing in Phases 3-4
- Service-aware targeting (Phase 4) high value but high complexity
