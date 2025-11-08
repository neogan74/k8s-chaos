# E2E Tests for k8s-chaos

This directory contains end-to-end (E2E) tests for the k8s-chaos operator. These tests run in an isolated Kind cluster to validate the full functionality of chaos experiments.

## Test Coverage

### Manager Tests (`e2e_test.go`)
- Controller deployment and startup
- Metrics endpoint availability
- Basic infrastructure validation

### Memory Stress Tests (`memory_stress_test.go`)
Comprehensive tests for the `pod-memory-stress` action:

#### Basic Functionality
- **Memory stress injection**: Verifies ephemeral containers are injected with stress-ng
- **Multiple workers**: Tests memory stress with multiple worker processes
- **Duration handling**: Validates experiments run for the specified duration

#### Safety Features
- **Dry-run mode**: Verifies no actual stress is applied in dry-run mode
- **Max percentage limits**: Tests the maxPercentage safety constraint
- **Exclusion labels**: Validates pods with `chaos.gushchin.dev/exclude=true` are protected

#### Validation
- **Required fields**: Tests webhook rejection when duration or memorySize is missing
- **Format validation**: Tests rejection of invalid memorySize formats (e.g., missing M/G suffix)

#### Observability
- **Metrics exposure**: Verifies Prometheus metrics are exported correctly

## Running E2E Tests

### Quick Start

Run all E2E tests (creates temporary Kind cluster):
```bash
make test-e2e
```

This command will:
1. Create a temporary Kind cluster (`k8s-chaos-test-e2e`)
2. Build and load the controller image
3. Install CRDs and deploy the controller
4. Run all E2E test suites
5. Clean up the cluster after tests complete

### Manual Test Execution

For development and debugging, you can run tests manually:

```bash
# 1. Set up the Kind cluster
make setup-test-e2e

# 2. Run tests manually
KIND=kind KIND_CLUSTER=k8s-chaos-test-e2e go test -tags=e2e ./test/e2e/ -v -ginkgo.v

# 3. Run specific test suites
KIND=kind KIND_CLUSTER=k8s-chaos-test-e2e go test -tags=e2e ./test/e2e/ -v -ginkgo.focus="Memory Stress"

# 4. Clean up when done
make cleanup-test-e2e
```

### Running Specific Test Cases

Use Ginkgo's focus feature to run specific tests:

```bash
# Run only basic memory stress tests
go test -tags=e2e ./test/e2e/ -v -ginkgo.focus="Basic Memory Stress"

# Run only safety feature tests
go test -tags=e2e ./test/e2e/ -v -ginkgo.focus="Safety Features"

# Run only validation tests
go test -tags=e2e ./test/e2e/ -v -ginkgo.focus="Validation Tests"

# Run only dry-run tests
go test -tags=e2e ./test/e2e/ -v -ginkgo.focus="Dry-Run"

# Run only metrics tests
go test -tags=e2e ./test/e2e/ -v -ginkgo.focus="Metrics Validation"
```

## Prerequisites

### Required Tools
- **Go**: 1.24.5 or later
- **Docker**: For building container images
- **Kind**: For creating test clusters
  ```bash
  go install sigs.k8s.io/kind@latest
  ```
- **kubectl**: For interacting with the cluster

### Optional
- **CertManager**: Installed automatically by default
  - Skip installation: `CERT_MANAGER_INSTALL_SKIP=true make test-e2e`

## Test Environment

### Test Namespaces
- **k8s-chaos-system**: Controller deployment namespace
- **memory-stress-test**: Dedicated namespace for memory stress tests

### Test Resources
Memory stress tests create:
- A test deployment with 3 nginx pods
- Various ChaosExperiment resources
- Ephemeral stress-ng containers (automatically injected)

### Resource Requirements
Each test pod is configured with:
- **Memory requests**: 64Mi
- **Memory limits**: 2Gi (allows testing memory stress up to 2GB per pod)
- **CPU requests**: 100m
- **CPU limits**: 500m

## Debugging Failed Tests

### View Controller Logs
```bash
kubectl logs -n k8s-chaos-system deployment/k8s-chaos-controller-manager
```

### View Test Pods
```bash
kubectl get pods -n memory-stress-test
```

### Check ChaosExperiment Status
```bash
kubectl get chaosexperiment -n memory-stress-test -o yaml
```

### View Ephemeral Containers
```bash
kubectl describe pod <pod-name> -n memory-stress-test | grep -A 10 "Ephemeral Containers"
```

### Check Metrics
```bash
kubectl logs curl-metrics -n k8s-chaos-system
```

## Test Output

### Successful Test Run
```
Running Suite: e2e suite
========================
• [dd.ddd seconds]

Manager
  should run successfully
  /path/to/test.go:xxx

  should ensure the metrics endpoint is serving metrics
  /path/to/test.go:xxx

Memory Stress Chaos Experiments
  Basic Memory Stress Tests
    should successfully inject memory stress into pods
    /path/to/test.go:xxx

    should handle multiple workers correctly
    /path/to/test.go:xxx

  Dry-Run Mode Tests
    should preview affected pods without injecting stress in dry-run mode
    /path/to/test.go:xxx

  Safety Features Tests
    should respect maxPercentage limits
    /path/to/test.go:xxx

    should exclude pods with exclusion label
    /path/to/test.go:xxx

  Metrics Validation Tests
    should expose memory stress experiment metrics
    /path/to/test.go:xxx

  Validation Tests
    should reject memory stress experiment without duration
    /path/to/test.go:xxx

    should reject memory stress experiment without memorySize
    /path/to/test.go:xxx

    should reject invalid memorySize format
    /path/to/test.go:xxx

Ran 12 of 12 Specs in 180.234 seconds
SUCCESS! -- 12 Passed | 0 Failed | 0 Pending | 0 Skipped
```

## CI Integration

These tests are designed to run in CI environments:

```yaml
# Example GitHub Actions workflow
- name: Run E2E Tests
  run: make test-e2e
```

## Troubleshooting

### Tests Timeout
- Increase timeout: `SetDefaultEventuallyTimeout(5 * time.Minute)`
- Check Kind cluster resources: `docker stats`

### Image Pull Errors
- Ensure stress-ng image is available: `ghcr.io/neogan74/stress-ng:latest`
- Check image pull policy in controller

### Webhook Validation Failures
- Verify CertManager is installed: `kubectl get pods -n cert-manager`
- Check webhook configuration: `kubectl get validatingwebhookconfiguration`

### Ephemeral Containers Not Injected
- Verify Kubernetes version supports ephemeral containers (1.23+)
- Check RBAC permissions: `kubectl get clusterrole manager-role -o yaml`
- Ensure `pods/ephemeralcontainers` subresource permission exists

## Contributing

When adding new E2E tests:

1. Use descriptive test names
2. Add cleanup in `AfterEach` hooks
3. Use `Eventually` for async operations
4. Document test purpose in comments
5. Follow existing test patterns
6. Update this README with new test coverage

## References

- [Ginkgo Testing Framework](https://onsi.github.io/ginkgo/)
- [Gomega Matcher Library](https://onsi.github.io/gomega/)
- [Kind Documentation](https://kind.sigs.k8s.io/)
- [Ephemeral Containers](https://kubernetes.io/docs/concepts/workloads/pods/ephemeral-containers/)
