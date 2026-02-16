# Network-Partition Implementation Plan

**Date**: 2026-02-10
**Status**: Plan Complete - Ready for Implementation
**Total Effort**: 7-9 days
**Risk Level**: Low (Phase 1), Medium (Phases 2-3, 5), High (Phase 4)

## Quick Links

- **[Plan Overview](./plan.md)** - High-level plan with phases and timelines
- **[Summary](./SUMMARY.md)** - Executive summary and key decisions
- **[Current Analysis](./reports/01-current-implementation-analysis.md)** - Existing implementation review
- **[Research Findings](./reports/02-research-findings.md)** - Industry patterns and best practices

## Phase Documents

### Phase 1: Documentation & Stability [4-6 hours]
📄 **[Phase 1 Details](./phase-01-documentation-stability.md)**

**Focus**: Document existing implementation, improve cleanup safety

**Key Deliverables**:
- ADR 0011: Network Partition Implementation
- Sample YAML configurations
- Custom iptables chain cleanup
- Updated CLAUDE.md

**Risk**: Low | **Status**: ⏳ Not Started

---

### Phase 2: API Design for Selective Targeting [6-8 hours]
📄 **[Phase 2 Details](./phase-02-api-design-selective-targeting.md)**

**Focus**: Design API schema for selective partition capabilities

**Key Deliverables**:
- API fields: targetIPs, targetCIDRs, targetPorts, targetProtocols
- OpenAPI validation markers
- Webhook validation logic
- Comprehensive unit tests

**Risk**: Medium | **Status**: ⏳ Not Started | **Prerequisites**: Phase 1

---

### Phase 3: Basic Selective Targeting Implementation [8-12 hours]
📄 **[Phase 3 Details](./phase-03-basic-selective-targeting-implementation.md)**

**Focus**: Implement IP/CIDR/port-based selective blocking

**Key Deliverables**:
- iptables rule generation from API fields
- Support for target combinations
- Enhanced dry-run with target preview
- E2E tests for selective scenarios

**Risk**: Medium | **Status**: ⏳ Not Started | **Prerequisites**: Phase 2

---

### Phase 4: Service-Aware Partitions [12-16 hours]
📄 **[Phase 4 Details](./phase-04-service-aware-partitions.md)**

**Focus**: Service/namespace targeting with IP resolution

**Key Deliverables**:
- API fields: targetServices, targetNamespaces
- Service-to-IP resolution logic
- ipset integration for efficiency
- E2E tests for service-aware targeting

**Risk**: High | **Status**: ⏳ Not Started | **Prerequisites**: Phase 3

---

### Phase 5: Testing & Validation [8-10 hours]
📄 **[Phase 5 Details](./phase-05-testing-validation.md)**

**Focus**: Comprehensive test coverage and validation

**Key Deliverables**:
- 90%+ unit test coverage
- Integration tests for all handlers
- E2E tests for all features
- Performance tests for large-scale scenarios

**Risk**: Medium | **Status**: ⏳ Not Started | **Prerequisites**: Phases 1-4

---

## Implementation Status

Current implementation (commit 58615ef):
- ✅ Core iptables-based network partition
- ✅ Direction support (ingress/egress/both)
- ✅ Duration-based lifecycle
- ✅ Safety features (dry-run, maxPercentage, exclusions)
- ✅ E2E tests for basic scenarios
- ✅ Metrics and history recording

Missing:
- ❌ ADR documentation
- ❌ Sample YAML configurations
- ❌ Safe cleanup mechanism (custom chains)
- ❌ Selective targeting (IPs, CIDRs, services)
- ❌ Service-aware capabilities

## Key Decisions

1. **iptables Approach**: Industry-standard for network partitions (tc netem for degradation)
2. **Custom Chain Cleanup**: Safer than `iptables -F`, prevents CNI conflicts
3. **Phased Implementation**: Document first, then enhance with targeting
4. **Service Resolution**: At experiment start (not continuous watching)
5. **ipset for Efficiency**: Use for 10+ targets, fallback otherwise

## Success Criteria

- [ ] ADR 0011 documents design decisions
- [ ] Sample YAML demonstrates all scenarios
- [ ] Custom chain cleanup tested in production
- [ ] Selective targeting covers 80% of use cases
- [ ] Service-aware targeting enables split-brain tests
- [ ] Test coverage >90%
- [ ] Zero production incidents

## Risk Summary

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| iptables conflicts with CNI | Medium | High | Custom chains, E2E testing |
| PSA Restricted blocks NET_ADMIN | Medium | Medium | Document, pre-flight validation |
| Service IP staleness | High | Low | Document timing, consider watching |
| Large namespace performance | Medium | Medium | ipset, webhook warnings |
| Implementation complexity | Medium | Medium | Phased approach, incremental delivery |

## Timeline

**Recommended Approach**:
1. **Week 1**: Phase 1 (Documentation & Stability)
2. **Gather Feedback**: Assess user needs for advanced targeting
3. **Week 2-3**: Phases 2-3 (API Design & Basic Targeting) if needed
4. **Week 4-5**: Phase 4 (Service-Aware) if user demand warrants
5. **Week 6**: Phase 5 (Testing & Validation)

**Fast Track** (Phase 1 only): 1 day for documentation and safety improvements

## Files Overview

```
plans/20260210-network-partition-implementation/
├── README.md (this file)                                      # Navigation index
├── plan.md                                                    # High-level plan (150 lines)
├── SUMMARY.md                                                 # Executive summary (224 lines)
├── phase-01-documentation-stability.md                        # Phase 1 details (316 lines)
├── phase-02-api-design-selective-targeting.md                 # Phase 2 details (440 lines)
├── phase-03-basic-selective-targeting-implementation.md       # Phase 3 details (471 lines)
├── phase-04-service-aware-partitions.md                       # Phase 4 details (566 lines)
├── phase-05-testing-validation.md                             # Phase 5 details (598 lines)
└── reports/
    ├── 01-current-implementation-analysis.md                  # Existing code review (195 lines)
    └── 02-research-findings.md                                # Industry research (329 lines)

Total: 3,289 lines of comprehensive planning documentation
```

## Next Steps

1. **Review**: Present plan to team, gather feedback
2. **Prioritize**: Decide which phases required for MVP
3. **Execute Phase 1**: Start with documentation and custom chains (1 day)
4. **Assess Feedback**: Determine if Phases 2-4 needed
5. **Incremental Delivery**: Ship Phase 1, iterate based on user needs

## Questions for Team

1. Is Phase 1 sufficient or are advanced targeting features (Phases 2-4) required?
2. What network partition scenarios do users most commonly need?
3. Should service-aware targeting (Phase 4) be prioritized?
4. Are there PSA Restricted environments we need to support?
5. What's acceptable for service IP resolution staleness?

## Related Documentation

**Existing ADRs**:
- ADR 0007: Pod Network Loss (tc netem approach)
- ADR 0005: Pod CPU Stress (ephemeral container pattern)
- ADR 0002: Safety Features (validation patterns)

**Implementation Files**:
- `internal/controller/chaosexperiment_controller.go` (lines 2587-2766)
- `api/v1alpha1/chaosexperiment_types.go` (lines 49, 162-166)
- `test/e2e/network_partition_test.go`

**Codebase Docs**:
- `/Users/neogan/GitHub/k8s-chaos/CLAUDE.md`
- `/Users/neogan/GitHub/k8s-chaos/docs/adr/README.md`

## Contact

For questions about this plan, refer to:
- Plan creation date: 2026-02-10
- Implementation branch: network-partition (commit 58615ef)
- Related commit: "feat: Add network-partition chaos action..."
