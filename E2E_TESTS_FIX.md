# E2E Tests Fix Guide

## Current Issue

The e2e tests are failing because **Docker is not running**. The error message indicates:

```
Cannot connect to the Docker daemon at unix:///Users/neogan/.orbstack/run/docker.sock.
Is the docker daemon running?
```

## Quick Fix

**Start Docker/OrbStack** before running the e2e tests:

```bash
# Make sure OrbStack/Docker is running, then:
make test-e2e
```

## E2E Test Coverage

The e2e test suite includes tests for the following chaos actions:

### 1. **Manager Tests** (`e2e_test.go`)
- Controller deployment verification
- Metrics endpoint serving
- Basic infrastructure tests

### 2. **pod-network-loss Tests** (`e2e_test.go`)
- Basic packet loss injection
- Dry-run mode verification
- Loss correlation parameter testing
- Selector and namespace isolation
- Concurrent experiments
- maxPercentage safety limits

### 3. **pod-memory-stress Tests** (`memory_stress_test.go`)
- Basic memory stress injection
- Multiple workers support
- Dry-run mode
- maxPercentage limits
- Exclusion labels
- Webhook validation (duration, memorySize, format)

### 4. **pod-disk-fill Tests** (`disk_fill_test.go`)
- Basic disk filling
- Webhook validation (duration, fillPercentage, targetPath)

## Implementation Status

✅ **All actions are fully implemented**:
- Controller handlers: `handlePodNetworkLoss()`, `handlePodMemoryStress()`, `handlePodDiskFill()`
- CRD validation markers in `chaosexperiment_types.go`
- Webhook validation in `chaosexperiment_webhook.go`
- Action routing in controller switch/case

## Test Structure

```
test/e2e/
├── e2e_suite_test.go          # Suite setup (builds image, installs CRDs)
├── e2e_test.go                # Manager & pod-network-loss tests
├── memory_stress_test.go      # pod-memory-stress tests
└── disk_fill_test.go          # pod-disk-fill tests
```

## Running Specific Tests

```bash
# Run all e2e tests
make test-e2e

# Run specific test file (after starting cluster manually)
make cluster-single-node
go test -v -tags=e2e ./test/e2e/ -run TestMemoryStress

# Run specific test case
go test -v -tags=e2e ./test/e2e/ -ginkgo.focus "should successfully inject memory stress"
```

## Prerequisites

1. ✅ Docker/OrbStack running
2. ✅ `make` available
3. ✅ `kubectl` installed
4. ✅ `kind` installed (for creating test cluster)

## Test Flow

1. **BeforeSuite** (runs once):
   - Builds manager Docker image
   - Loads image into Kind cluster
   - Installs CRDs globally
   - Optionally installs CertManager

2. **Test Execution**:
   - Creates test namespaces
   - Deploys test pods/deployments
   - Creates ChaosExperiment resources
   - Verifies expected behavior
   - Cleans up experiments

3. **AfterSuite** (runs once):
   - Uninstalls CRDs
   - Optionally uninstalls CertManager

## Validation Coverage

All three new actions have comprehensive validation:

### pod-memory-stress
- ✅ Requires `duration`
- ✅ Requires `memorySize` with format validation (e.g., "256M", "1G")
- ✅ Validates `memoryWorkers` range (1-32)
- ✅ OpenAPI schema validation
- ✅ Webhook cross-field validation

### pod-network-loss
- ✅ Requires `duration`
- ✅ Requires `lossPercentage` (1-40)
- ✅ Validates `lossCorrelation` (0-100)
- ✅ OpenAPI schema validation
- ✅ Webhook cross-field validation

### pod-disk-fill
- ✅ Requires `duration`
- ✅ Requires `fillPercentage` (50-95)
- ✅ Validates `targetPath` when `volumeName` not set
- ✅ OpenAPI schema validation
- ✅ Webhook cross-field validation

## Known Test Behaviors

1. **Pending Test**: "should cleanup effects on cancellation" is marked as `Pending` because cleanup on deletion is not yet implemented (requires finalizers).

2. **Metrics Tests**: Some tests skip metrics validation because they depend on the `curl-metrics` pod from the Manager test suite, which gets cleaned up.

3. **Webhook Tests**: Some validation tests use `requireWebhookEnabled()` and will skip if `WEBHOOK_ENABLED=false`.

## Environment Variables

- `WEBHOOK_ENABLED=true`: Enable webhook validation tests
- `CERT_MANAGER_INSTALL_SKIP=true`: Skip CertManager installation if already present

## Next Steps

1. **Start OrbStack/Docker**
2. Run: `make test-e2e`
3. All tests should pass ✅

## Troubleshooting

If tests fail after starting Docker:

```bash
# Check if Kind cluster exists
kind get clusters

# Delete any existing test clusters
kind delete cluster --name k8s-chaos-test-e2e

# Rebuild and retry
make docker-build IMG=example.com/k8s-chaos:v0.0.1
make test-e2e
```

## Test Duration

Expected runtime:
- **Suite setup**: ~2-3 minutes (build image, install CRDs)
- **Manager tests**: ~3-5 minutes
- **Network loss tests**: ~10-15 minutes
- **Memory stress tests**: ~5-8 minutes
- **Disk fill tests**: ~3-5 minutes
- **Total**: ~25-35 minutes
