# Network-Partition Implementation Plan - Summary

**Date**: 2026-02-10
**Status**: Plan Complete - Ready for Implementation
**Branch**: network-partition (commit 58615ef)

## Executive Summary

Network-partition chaos action already implemented in commit 58615ef but lacks documentation, safe cleanup mechanism, and advanced targeting capabilities. This plan provides comprehensive roadmap for documenting existing implementation and enhancing with selective IP/service/namespace targeting.

## Current State

**What's Working**:
- Core iptables-based network partition via ephemeral containers
- Direction support (ingress/egress/both)
- Duration-based lifecycle with automatic cleanup
- All safety features (dry-run, maxPercentage, exclusions, production protection)
- E2E tests for basic scenarios
- Metrics and history recording

**What's Missing**:
- ADR documentation
- Sample YAML configurations
- Safe cleanup (current iptables -F too aggressive)
- Selective targeting (IPs, CIDRs, services, namespaces)
- Service-aware partition capabilities

## Implementation Plan

### Phase 1: Documentation & Stability [4-6 hours, Low Risk]
**Focus**: Document existing implementation, improve cleanup safety

**Deliverables**:
- ADR 0011: Network Partition Implementation
- Sample YAML with multiple scenarios
- Custom iptables chain cleanup (prevents CNI conflicts)
- Updated CLAUDE.md with action description

**Impact**: Production-ready documentation, safer cleanup mechanism

### Phase 2: API Design for Selective Targeting [6-8 hours, Medium Risk]
**Focus**: Design API schema for selective partition capabilities

**Deliverables**:
- API fields: targetIPs, targetCIDRs, targetPorts, targetProtocols
- OpenAPI validation markers
- Webhook validation logic
- Unit tests for validation
- API documentation

**Impact**: Foundation for advanced targeting, backward compatible

### Phase 3: Basic Selective Targeting Implementation [8-12 hours, Medium Risk]
**Focus**: Implement IP/CIDR/port-based selective blocking

**Deliverables**:
- iptables rule generation from API fields
- Support for target combinations
- Enhanced dry-run with target preview
- Unit tests for rule generation
- E2E tests for selective scenarios

**Impact**: Realistic failure injection (block specific IPs/ports)

### Phase 4: Service-Aware Partitions [12-16 hours, High Risk]
**Focus**: Service/namespace targeting with IP resolution

**Deliverables**:
- API fields: targetServices, targetNamespaces
- Service-to-IP resolution logic
- ipset integration for efficiency
- Webhook validation for services/namespaces
- E2E tests for service-aware targeting

**Impact**: User-friendly targeting ("block redis-service" vs "block 10.96.100.50")

### Phase 5: Testing & Validation [8-10 hours, Medium Risk]
**Focus**: Comprehensive test coverage and validation

**Deliverables**:
- 90%+ unit test coverage
- Integration tests for all handlers
- E2E tests for all features
- Negative tests for error conditions
- Performance tests for large-scale scenarios
- Documentation validation

**Impact**: Production-ready quality, confidence in implementation

## Total Effort

**Timeline**: 7-9 days for complete implementation
- Phase 1: 1 day
- Phase 2: 1 day
- Phase 3: 2 days
- Phase 4: 2-3 days
- Phase 5: 1-2 days

**Recommended Approach**: Implement Phase 1 immediately for documentation and safety improvements. Gather user feedback before investing in Phases 2-4 enhancements.

## Key Technical Decisions

**iptables vs tc netem**: iptables chosen for complete isolation (correct for partitions), tc netem used for degradation (pod-network-loss)

**Custom Chain Cleanup**: Safer than `iptables -F`, prevents removing CNI/mesh rules

**Service Resolution Timing**: Resolve at experiment start, not continuously (simplicity, document staleness)

**ipset for Efficiency**: Use ipset for 10+ targets, fallback to multiple rules otherwise

**Backward Compatibility**: All new fields optional, empty targets = full partition (current behavior)

## Success Metrics

- [ ] ADR 0011 approved and published
- [ ] Sample YAML demonstrates all scenarios
- [ ] Custom chain cleanup tested in production
- [ ] Selective targeting (Phase 3) covers 80% of user needs
- [ ] Service-aware targeting (Phase 4) enables realistic split-brain tests
- [ ] Test coverage >90% for network partition code
- [ ] Zero production incidents from network-partition action

## Risk Mitigation

**Risk 1**: iptables conflicts with CNI
- **Mitigation**: Custom chains (Phase 1), E2E testing

**Risk 2**: PSA Restricted blocks NET_ADMIN
- **Mitigation**: Document requirements, pre-flight validation

**Risk 3**: Service IP staleness
- **Mitigation**: Document resolution timing, consider watching in future

**Risk 4**: Large namespace performance
- **Mitigation**: ipset for efficiency, warn in webhook

## Files Created

**Documentation**:
- `plans/20260210-network-partition-implementation/plan.md`
- `plans/20260210-network-partition-implementation/SUMMARY.md`
- `plans/20260210-network-partition-implementation/phase-01-documentation-stability.md`
- `plans/20260210-network-partition-implementation/phase-02-api-design-selective-targeting.md`
- `plans/20260210-network-partition-implementation/phase-03-basic-selective-targeting-implementation.md`
- `plans/20260210-network-partition-implementation/phase-04-service-aware-partitions.md`
- `plans/20260210-network-partition-implementation/phase-05-testing-validation.md`

**Reports**:
- `plans/20260210-network-partition-implementation/reports/01-current-implementation-analysis.md`
- `plans/20260210-network-partition-implementation/reports/02-research-findings.md`

## Implementation Checklist

Phase 1 (Immediate):
- [ ] Create ADR 0011
- [ ] Create sample YAML
- [ ] Implement custom chain cleanup
- [ ] Update CLAUDE.md
- [ ] Run E2E tests

Phase 2 (After Phase 1):
- [ ] Define API fields
- [ ] Add validation markers
- [ ] Implement webhook validation
- [ ] Write unit tests
- [ ] Update documentation

Phase 3 (After Phase 2):
- [ ] Implement rule generation
- [ ] Update handler
- [ ] Enhance dry-run
- [ ] Write unit tests
- [ ] Add E2E tests

Phase 4 (Optional, based on feedback):
- [ ] Define service API fields
- [ ] Implement service resolution
- [ ] Integrate ipset
- [ ] Add webhook validation
- [ ] Write comprehensive tests

Phase 5 (Final):
- [ ] Complete unit test suite
- [ ] Complete integration tests
- [ ] Complete E2E test suite
- [ ] Run performance tests
- [ ] Validate documentation
- [ ] Generate coverage report

## Next Actions

1. **Review Plan**: Present plan to team, gather feedback
2. **Start Phase 1**: Begin with documentation and custom chains
3. **User Research**: Identify selective targeting use cases (before Phase 2)
4. **Incremental Delivery**: Ship Phase 1, gather feedback, then continue

## Questions for Stakeholders

1. Is Phase 1 sufficient for current needs or are Phases 2-4 required?
2. What are the most common network partition scenarios users need?
3. Should service-aware targeting (Phase 4) be prioritized?
4. Are there PSA Restricted environments we need to support?
5. What's acceptable staleness for service IP resolution?

## References

**Related ADRs**:
- ADR 0007: Pod Network Loss (tc netem approach)
- ADR 0005: Pod CPU Stress (ephemeral container pattern)
- ADR 0002: Safety Features (validation patterns)

**External Resources**:
- [Chaos Mesh Network Chaos](https://deepwiki.com/chaos-mesh/chaos-mesh/3.2-network-chaos)
- [Harness Chaos Faults](https://developer.harness.io/docs/chaos-engineering/faults/chaos-faults/kubernetes/)
- [DigitalOcean iptables Essentials](https://www.digitalocean.com/community/tutorials/iptables-essentials-common-firewall-rules-and-commands)

**Implementation Files**:
- `internal/controller/chaosexperiment_controller.go` (lines 2587-2766)
- `api/v1alpha1/chaosexperiment_types.go` (lines 49, 162-166)
- `test/e2e/network_partition_test.go`

## Conclusion

Network-partition implementation solid foundation, production-ready for basic scenarios. Plan provides clear roadmap for documentation, safety improvements, and advanced targeting capabilities. Phased approach allows incremental delivery and feedback collection. Recommended: implement Phase 1 immediately, assess user needs before Phases 2-4.
