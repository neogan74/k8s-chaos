# Phase 5: Testing & Validation

**Status**: ⏳ Not Started
**Effort**: 8-10 hours
**Risk**: Medium
**Prerequisites**: Phases 1-4 complete (all features implemented)

## Context

Phases 1-4 implemented network-partition action with documentation, custom chains, selective targeting, and service-aware partitions. Phase 5 creates comprehensive test suite validating all features, edge cases, and failure scenarios.

### Related Files
- `test/e2e/network_partition_test.go` (E2E tests)
- `internal/controller/chaosexperiment_controller_test.go` (integration tests)
- `api/v1alpha1/chaosexperiment_webhook_test.go` (validation tests)
- All implementation files (to verify)

### Related Docs
- All phase documentation (1-4)
- ADR 0011: Network Partition Implementation
- Test patterns from other actions (pod-cpu-stress, pod-network-loss)

## Overview

Create comprehensive test coverage including unit tests, integration tests, E2E tests, negative tests, performance tests, and documentation validation. Ensure all features work correctly and edge cases handled gracefully.

## Key Insights

**Test Pyramid**:
```
       /\
      /E2E\       10% - Full cluster integration (slow)
     /------\
    /  INT  \     20% - Controller integration (medium)
   /----------\
  /   UNIT     \  70% - Pure logic (fast)
 /--------------\
```

**Test Categories**:
1. **Unit Tests**: Rule generation, validation, resolution logic
2. **Integration Tests**: Controller with fake client, webhook with envtest
3. **E2E Tests**: Real Kind cluster, actual network effects
4. **Negative Tests**: Error handling, invalid inputs
5. **Performance Tests**: Large namespaces, many targets
6. **Documentation Tests**: Samples validate, examples work

**Coverage Goals**:
- Unit tests: 90%+ coverage
- Integration tests: All handler paths
- E2E tests: Happy paths + critical errors
- Negative tests: All error conditions

## Requirements

### Functional
1. Unit tests for all rule generation functions
2. Integration tests for service resolution
3. E2E tests for all targeting modes
4. Negative tests for error conditions
5. Performance tests for large-scale scenarios
6. Documentation validation tests

### Non-Functional
1. Tests run in <5 minutes total
2. E2E tests isolated (no side effects)
3. Tests deterministic (no flakiness)
4. Clear test failure messages
5. CI-friendly (parallel execution)

## Architecture

### Test Organization

```
test/
├── e2e/
│   ├── network_partition_basic_test.go          [Complete basic scenarios]
│   ├── network_partition_selective_test.go      [IP/CIDR/port targeting]
│   ├── network_partition_service_test.go        [Service-aware]
│   └── network_partition_advanced_test.go       [Edge cases, combinations]
│
internal/controller/
├── network_partition_rules_test.go              [Rule generation unit tests]
├── service_resolution_test.go                   [Service resolution unit tests]
└── chaosexperiment_controller_test.go           [Integration tests]

api/v1alpha1/
└── chaosexperiment_webhook_test.go              [Validation tests]
```

### Test Scenarios

**Unit Tests (internal/controller)**:
1. Rule generation - empty targets (full partition)
2. Rule generation - single IP
3. Rule generation - multiple IPs
4. Rule generation - CIDR
5. Rule generation - ports with protocol
6. Rule generation - combined (IP + port + protocol)
7. Rule generation - direction (ingress/egress/both)
8. Service resolution - ClusterIP service
9. Service resolution - headless service
10. Namespace resolution - single namespace
11. Namespace resolution - multiple namespaces
12. ipset script generation - threshold logic
13. ipset script generation - fallback
14. Script template rendering
15. Target summary building

**Integration Tests (internal/controller)**:
1. Handler with dry-run enabled
2. Handler with selective targets
3. Handler with service targets
4. Handler with namespace targets
5. Handler with maxPercentage limit
6. Handler with exclusion labels
7. Handler with production namespace
8. Handler cleanup on completion
9. Handler retry on failure
10. History record creation

**E2E Tests (test/e2e)**:
1. Basic full partition (both directions)
2. Ingress-only partition
3. Egress-only partition
4. Selective IP targeting
5. CIDR targeting
6. Port targeting
7. Combined targeting
8. Service targeting
9. Namespace targeting
10. Dry-run mode
11. maxPercentage enforcement
12. Exclusion labels respected
13. Production protection
14. Custom chain cleanup
15. Experiment duration lifecycle

**Negative Tests**:
1. Invalid CIDR format
2. Invalid IP address
3. Out-of-range port
4. Nonexistent service
5. Nonexistent namespace
6. PSA Restricted namespace (if possible)
7. No eligible pods
8. Count exceeds maxPercentage
9. Missing duration
10. Invalid direction value

**Performance Tests**:
1. Namespace with 50+ pods
2. 100+ target IPs
3. Multiple experiments concurrent
4. Rapid create/delete cycles

## Related Code Files

**To Create**:
- `test/e2e/network_partition_selective_test.go`
- `test/e2e/network_partition_service_test.go`
- `test/e2e/network_partition_advanced_test.go`
- `internal/controller/network_partition_rules_test.go`
- `internal/controller/service_resolution_test.go`

**To Modify**:
- `test/e2e/network_partition_test.go` - Expand existing tests
- `internal/controller/chaosexperiment_controller_test.go` - Add integration tests
- `api/v1alpha1/chaosexperiment_webhook_test.go` - Add validation tests

**To Verify**:
- `config/samples/chaos_v1alpha1_chaosexperiment_network_partition.yaml` - Validate samples

## Implementation Steps

### Step 1: Create Unit Test Suite [3-4 hours]

**Rule Generation Tests** (network_partition_rules_test.go):
```go
func TestGenerateBlockingRules(t *testing.T) {
    tests := []struct {
        name     string
        exp      *chaosv1alpha1.ChaosExperiment
        expected []string  // Expected iptables rules
    }{
        {
            name: "empty targets - full partition",
            exp:  &chaosv1alpha1.ChaosExperiment{
                Spec: chaosv1alpha1.ChaosExperimentSpec{},
            },
            expected: []string{
                "iptables -A CHAOS_PARTITION -j DROP",
            },
        },
        {
            name: "single IP target",
            exp:  &chaosv1alpha1.ChaosExperiment{
                Spec: chaosv1alpha1.ChaosExperimentSpec{
                    TargetIPs: []string{"10.96.100.50"},
                },
            },
            expected: []string{
                "iptables -A CHAOS_PARTITION -d 10.96.100.50 -j DROP",
            },
        },
        // ... more test cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            rules := generateBlockingRules(tt.exp, "CHAOS_PARTITION")
            // Assert rules match expected
        })
    }
}
```

**Service Resolution Tests** (service_resolution_test.go):
```go
func TestResolveServiceTargets(t *testing.T) {
    // Use fake client
    client := fake.NewClientBuilder().WithObjects(
        &corev1.Service{
            ObjectMeta: metav1.ObjectMeta{
                Name:      "test-service",
                Namespace: "default",
            },
            Spec: corev1.ServiceSpec{
                ClusterIP: "10.96.100.50",
            },
        },
    ).Build()

    reconciler := &ChaosExperimentReconciler{Client: client}

    targets, err := reconciler.resolveServiceTargets(ctx, exp)

    assert.NoError(t, err)
    assert.Len(t, targets, 1)
    assert.Equal(t, "10.96.100.50", targets[0].IP)
}
```

### Step 2: Create Integration Tests [2-3 hours]

**Handler Integration** (chaosexperiment_controller_test.go):
```go
var _ = Describe("Network Partition Handler", func() {
    It("should handle dry-run mode", func() {
        exp := createExperiment("network-partition", map[string]string{
            "dryRun": "true",
            "targetIPs": "10.96.100.50",
        })

        result, err := reconciler.handleNetworkPartition(ctx, exp)

        Expect(err).NotTo(HaveOccurred())
        Expect(exp.Status.Message).To(ContainSubstring("DRY RUN"))
        Expect(exp.Status.Message).To(ContainSubstring("10.96.100.50"))
    })

    It("should resolve service targets", func() {
        // Create service
        // Create experiment with targetServices
        // Verify resolution
    })

    It("should respect maxPercentage", func() {
        // Create 10 pods
        // Set maxPercentage: 20 (allows 2 pods)
        // Set count: 5 (would affect 5)
        // Verify only 2 pods affected
    })
})
```

### Step 3: Create E2E Test Suite [3-4 hours]

**Selective Targeting E2E** (network_partition_selective_test.go):
```go
var _ = Describe("Selective Network Partition", func() {
    It("should block traffic to specific IP", func() {
        By("creating experiment with targetIPs")
        // Create experiment YAML
        // Apply with kubectl

        By("verifying traffic blocked to target IP")
        // Exec into pod, try to reach target
        // Verify connection fails

        By("verifying traffic allowed to other IPs")
        // Try to reach non-target IP
        // Verify connection succeeds
    })

    It("should block specific ports only", func() {
        By("creating experiment with targetPorts: [80]")

        By("verifying HTTP blocked")
        // curl http://service should fail

        By("verifying HTTPS allowed")
        // curl https://service should succeed
    })
})
```

**Service-Aware E2E** (network_partition_service_test.go):
```go
var _ = Describe("Service-Aware Partition", func() {
    It("should block traffic to service by name", func() {
        By("creating redis service")
        // kubectl apply redis service

        By("creating experiment targeting redis service")
        // targetServices: [{name: redis, namespace: default}]

        By("verifying traffic to redis blocked")
        // Try to connect to redis from another pod
        // Should fail

        By("verifying traffic to other services allowed")
        // Connect to different service
        // Should succeed
    })

    It("should handle headless services", func() {
        By("creating headless service (ClusterIP: None)")

        By("creating experiment targeting headless service")

        By("verifying traffic to all endpoints blocked")
        // Get endpoint IPs
        // Try each endpoint
        // All should be blocked
    })
})
```

**Advanced Scenarios E2E** (network_partition_advanced_test.go):
```go
var _ = Describe("Advanced Network Partition", func() {
    It("should combine multiple target types", func() {
        // targetIPs + targetServices + targetNamespaces
        // Verify all resolved and blocked
    })

    It("should use ipset for many targets", func() {
        // targetNamespaces with 50+ pods
        // Verify ipset created
        // Verify single iptables rule with match-set
    })

    It("should cleanup custom chains properly", func() {
        // Run experiment
        // Wait for completion
        // Exec into pod, check iptables -L
        // Verify CHAOS_PARTITION chain removed
        // Verify other chains intact
    })
})
```

### Step 4: Create Negative Tests [1-2 hours]

**Validation Error Tests**:
```go
var _ = Describe("Network Partition Validation", func() {
    It("should reject invalid CIDR", func() {
        exp := createExperiment(map[string]string{
            "targetCIDRs": "10.96.0.0/99",  // Invalid: /99
        })

        err := webhook.ValidateCreate(ctx, exp)
        Expect(err).To(HaveOccurred())
        Expect(err.Error()).To(ContainSubstring("invalid CIDR"))
    })

    It("should reject nonexistent service", func() {
        exp := createExperiment(map[string]string{
            "targetServices": "[{name: nonexistent, namespace: default}]",
        })

        err := webhook.ValidateCreate(ctx, exp)
        Expect(err).To(HaveOccurred())
        Expect(err.Error()).To(ContainSubstring("not found"))
    })

    It("should reject when count exceeds maxPercentage", func() {
        // 10 pods, maxPercentage: 20 (allows 2), count: 5
        err := webhook.ValidateCreate(ctx, exp)
        Expect(err).To(HaveOccurred())
        Expect(err.Error()).To(ContainSubstring("maxPercentage"))
    })
})
```

### Step 5: Create Performance Tests [1 hour]

**Large-Scale Tests**:
```go
func TestLargeNamespaceResolution(t *testing.T) {
    // Create namespace with 100 pods
    // Measure resolution time
    // Should complete in <5 seconds
    // Verify ipset used (not 100 iptables rules)
}

func TestConcurrentExperiments(t *testing.T) {
    // Create 10 experiments simultaneously
    // Verify all succeed
    // Verify no race conditions
    // Verify custom chains don't conflict
}
```

### Step 6: Validate Documentation [30 minutes]

**Sample Validation**:
```bash
# Validate all sample YAMLs
for yaml in config/samples/chaos_v1alpha1_chaosexperiment_network_partition*.yaml; do
    kubectl apply --dry-run=server -f $yaml
done

# Verify samples match documentation
# Check CLAUDE.md examples
# Verify ADR 0011 code examples
```

### Step 7: CI Integration [30 minutes]

**GitHub Actions Workflow**:
```yaml
- name: Run network partition tests
  run: |
    make test  # Unit + integration
    make test-e2e  # E2E in Kind cluster

- name: Check test coverage
  run: |
    go test ./internal/controller/... -coverprofile=coverage.out
    go tool cover -func=coverage.out | grep network_partition
    # Verify >80% coverage
```

## Todo List

Unit Tests:
- [ ] Test empty targets (full partition)
- [ ] Test single IP rule generation
- [ ] Test multiple IPs
- [ ] Test CIDR rules
- [ ] Test port rules
- [ ] Test combined rules (IP + port + protocol)
- [ ] Test direction handling (3 cases)
- [ ] Test service resolution (ClusterIP)
- [ ] Test service resolution (headless)
- [ ] Test namespace resolution
- [ ] Test ipset script generation
- [ ] Test fallback script generation
- [ ] Test target summary building
- [ ] Verify 90%+ coverage

Integration Tests:
- [ ] Test dry-run handler
- [ ] Test selective targeting handler
- [ ] Test service targeting handler
- [ ] Test namespace targeting handler
- [ ] Test maxPercentage enforcement
- [ ] Test exclusion labels
- [ ] Test production protection
- [ ] Test cleanup logic
- [ ] Test retry logic
- [ ] Test history creation

E2E Tests:
- [ ] Test basic full partition (both)
- [ ] Test ingress-only partition
- [ ] Test egress-only partition
- [ ] Test selective IP targeting
- [ ] Test CIDR targeting
- [ ] Test port targeting
- [ ] Test combined targeting
- [ ] Test service targeting
- [ ] Test namespace targeting
- [ ] Test dry-run mode
- [ ] Test maxPercentage
- [ ] Test exclusion labels
- [ ] Test custom chain cleanup
- [ ] Test experiment duration

Negative Tests:
- [ ] Test invalid CIDR format
- [ ] Test invalid IP address
- [ ] Test out-of-range port
- [ ] Test nonexistent service
- [ ] Test nonexistent namespace
- [ ] Test no eligible pods
- [ ] Test count exceeds maxPercentage
- [ ] Test missing duration
- [ ] Test invalid direction

Performance Tests:
- [ ] Test namespace with 50+ pods
- [ ] Test 100+ target IPs
- [ ] Test concurrent experiments
- [ ] Verify ipset used for large lists

Documentation:
- [ ] Validate all sample YAMLs
- [ ] Verify CLAUDE.md examples
- [ ] Verify ADR 0011 code examples
- [ ] Check inline documentation

CI:
- [ ] Add network partition tests to workflow
- [ ] Add coverage reporting
- [ ] Verify E2E tests run in Kind
- [ ] Ensure tests deterministic

## Success Criteria

**Coverage**:
- Unit tests: 90%+ coverage for network partition code
- Integration tests: All handler code paths covered
- E2E tests: All major features tested end-to-end
- Negative tests: All error conditions tested

**Quality**:
- All tests pass consistently (no flakiness)
- Test execution time <5 minutes total
- Clear failure messages
- Tests isolated (no side effects)

**Documentation**:
- All sample YAMLs validate successfully
- CLAUDE.md examples work
- ADR 0011 code examples accurate

**CI**:
- Tests run automatically on PR
- Coverage reported
- E2E tests reliable in CI environment

## Risk Assessment

**Risk 1**: E2E tests flaky in CI
- **Probability**: Medium
- **Impact**: High
- **Mitigation**: Proper timeouts, retries, resource cleanup
- **Detection**: CI failures, manual verification

**Risk 2**: Performance tests too slow
- **Probability**: Medium
- **Impact**: Medium
- **Mitigation**: Parallel execution, mock large datasets
- **Detection**: Test execution time >5min

**Risk 3**: Coverage gaps in edge cases
- **Probability**: Medium
- **Impact**: Medium
- **Mitigation**: Systematic test case enumeration, code review
- **Detection**: Bugs in production, missing scenarios

**Risk 4**: ipset not available in CI
- **Probability**: Low
- **Impact**: Low
- **Mitigation**: Test fallback path, check image has ipset
- **Detection**: E2E test failures

## Security Considerations

**Test Isolation**:
- Each E2E test uses unique namespace
- Cleanup ensures no leftover resources
- No tests affect cluster infrastructure

**Sensitive Data**:
- No real credentials in tests
- Mock services for testing
- No production data

**Resource Limits**:
- Tests respect cluster quotas
- Large-scale tests use reasonable limits (100 pods max)
- Cleanup prevents resource exhaustion

## Next Steps

After Phase 5 completion:
1. Run full test suite, verify 100% pass
2. Generate coverage report, identify gaps
3. Fix any bugs discovered during testing
4. Update plan.md to mark all phases complete
5. Create final summary report
6. Prepare for PR review and merge
