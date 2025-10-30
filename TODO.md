# TODO - k8s-chaos Improvements

## 🔥 High Priority

### Core Functionality
- [x] **Complete Sample CRD** - Add a working example in `config/samples/chaos_v1alpha1_chaosexperiment.yaml`
- [x] **Add Validation** - Implement OpenAPI schema validation for CRD fields
  - [x] Validate action field against allowed values
  - [x] Ensure count is positive integer
  - [x] Validate selector is not empty
  - [x] Add admission webhook for advanced validation
  - [x] Validate namespace exists
  - [x] Validate selector matches pods
  - [x] Cross-field validation (e.g., duration required for pod-delay)
  - [x] Unit tests for validation logic
  - [x] Webhook tests with fake client
- [ ] **Implement Safety Checks**
  - [ ] Add dry-run mode to preview affected pods
  - [ ] Implement maximum percentage limit (e.g., max 30% of pods)
  - [ ] Add exclusion labels to protect critical pods
  - [ ] Add confirmation/approval mechanism for production namespaces

### Error Handling
- [x] **Improve Error Messages** - Add more descriptive error messages and status updates
- [x] **Add Retry Logic** - Implement exponential backoff for transient failures (completed with configurable strategies)
- [x] **Handle Edge Cases**
  - [x] What if namespace doesn't exist? - Webhook validates this
  - [ ] What if pods are already terminating?
  - [ ] Handle permission denied errors gracefully

## 📊 Observability

### Monitoring
- [x] **Add Prometheus Metrics** - Completed (see docs/METRICS.md)
  - [x] `chaos_experiments_total` - Total experiments run
  - [x] `chaos_experiments_failed` - Failed experiments (via errors metric)
  - [x] `chaos_pods_deleted_total` - Resources affected metric
  - [x] `chaos_experiment_duration_seconds` - Experiment execution time
  - [x] `chaos_active_experiments` - Currently running experiments
- [x] **Implement Structured Logging**
  - [x] Add correlation IDs for tracking experiments
  - [x] Log affected pod names and namespaces
  - [x] Add log levels (debug, info, warn, error)

### Status Reporting
- [x] **Enhance Status Fields**
  - [x] Add `phase` field (Pending, Running, Completed, Failed)
  - [ ] Add `affectedPods` list with pod names
  - [x] Add `startTime` and `completedAt` timestamps
  - [x] Add retry tracking fields (retryCount, nextRetryTime, lastError)
- [ ] **Kubernetes Events** - Emit events on ChaosExperiment and affected pods

## 🚀 New Chaos Actions

### Pod Chaos
- [x] **pod-delay** - Add network latency to pods
- [x] **pod-cpu-stress** - Consume CPU resources (implemented with ephemeral containers + stress-ng)
- [ ] **pod-memory-stress** - Consume memory resources
- [ ] **pod-network-loss** - Simulate packet loss
- [ ] **pod-network-corruption** - Corrupt network packets
- [ ] **pod-restart** - Restart pods instead of delete

### Node Chaos
- [x] **node-drain** - Drain nodes temporarily (implemented with cordon and eviction)
- [ ] **node-taint** - Add taints to nodes
- [ ] **node-cpu-stress** - Stress node CPU
- [ ] **node-disk-fill** - Fill node disk space

### Network Chaos
- [ ] **network-partition** - Simulate network splits
- [ ] **dns-chaos** - DNS resolution failures
- [ ] **http-chaos** - HTTP response manipulation

## ⏰ Scheduling & Duration

### Scheduling
- [ ] **Cron Scheduling** - Add cron expression support for recurring experiments
- [ ] **Time Windows** - Define maintenance windows for experiments
- [ ] **Dependency Management** - Wait for other experiments to complete

### Duration Control
- [x] **Experiment Duration** - Add `experimentDuration` field to auto-stop experiments (completed)
- [x] **Graceful Termination** - Clean up resources when experiment ends
- [ ] **Pause/Resume** - Allow pausing and resuming experiments

## 🧪 Testing

### Unit Tests
- [ ] **Increase Coverage** - Target 80% code coverage
- [ ] **Test Edge Cases** - Add tests for error conditions
- [ ] **Mock External Dependencies** - Better isolation in tests

### Integration Tests
- [ ] **E2E Test Scenarios**
  - [ ] Test with different selectors
  - [ ] Test with multiple namespaces
  - [ ] Test concurrent experiments
  - [ ] Test experiment cancellation
- [ ] **Chaos Testing the Chaos Operator** - Self-testing scenarios

## 📖 Documentation

### User Documentation
- [ ] **Getting Started Guide** - Step-by-step tutorial
- [ ] **Example Scenarios** - Real-world use cases
- [ ] **Best Practices** - Guidelines for safe chaos testing
- [ ] **Troubleshooting Guide** - Common issues and solutions

### Developer Documentation
- [x] **Architecture Decision Records (ADRs)** - Created ADR for pod-cpu-stress implementation
- [x] **API Documentation** - Detailed CRD field descriptions
- [ ] **Contributing Guide** - How to add new chaos actions
- [x] **Code Comments** - Translate Russian comments to English

## 🔒 Security

### RBAC
- [ ] **Fine-grained Permissions** - Separate roles for different chaos levels
- [ ] **Namespace Isolation** - Restrict experiments to specific namespaces
- [ ] **Audit Logging** - Track who runs experiments

### Policy
- [ ] **OPA Integration** - Policy-based experiment approval
- [ ] **Resource Quotas** - Limit resource consumption by experiments
- [ ] **Network Policies** - Isolate chaos experiments

## 🎯 Production Readiness

### High Availability
- [ ] **Leader Election** - Verify leader election works correctly
- [ ] **Horizontal Scaling** - Test with multiple controller replicas
- [ ] **Graceful Shutdown** - Clean shutdown on SIGTERM

### Performance
- [ ] **Resource Optimization** - Profile and optimize memory/CPU usage
- [ ] **Rate Limiting** - Prevent overwhelming the API server
- [ ] **Batch Operations** - Efficient handling of multiple experiments

### Operations
- [ ] **Helm Chart** - Create Helm chart for easier deployment
- [ ] **Operator Lifecycle Manager (OLM)** - Support for OLM
- [ ] **Multi-tenancy** - Support for multiple teams/projects
- [ ] **Backup/Restore** - Experiment history backup

## 🔄 CI/CD

### GitHub Actions
- [ ] **Release Automation** - Automated releases with changelogs
- [ ] **Security Scanning** - Container image vulnerability scanning
- [ ] **Multi-arch Builds** - Support ARM64 and other architectures
- [ ] **Benchmark Tests** - Performance regression testing

### Quality Gates
- [ ] **Coverage Threshold** - Fail builds if coverage drops
- [ ] **Linting Rules** - Stricter linting configuration
- [ ] **License Checking** - Ensure dependency license compliance

## 💡 Advanced Features

### Experiment Orchestration
- [ ] **Scenario Support** - Chain multiple chaos actions
- [ ] **Conditional Chaos** - Trigger based on metrics/alerts
- [ ] **Gradual Chaos** - Increase intensity over time
- [ ] **Chaos Workflows** - Argo Workflows integration

### Integrations
- [ ] **Prometheus Alerts** - Alert on experiment failures
- [ ] **Slack/PagerDuty** - Notifications for experiments
- [ ] **Grafana Dashboards** - Visualization of chaos metrics
- [ ] **Service Mesh Integration** - Istio/Linkerd chaos injection

### Analysis
- [ ] **Steady State Detection** - Verify system health before chaos
- [ ] **Impact Analysis** - Measure blast radius of experiments
- [ ] **Automated Reports** - Generate experiment reports
- [ ] **Learning Mode** - Suggest experiments based on system topology

## 🏗️ Technical Debt

### Code Quality
- [ ] **Refactor Controller** - Split into smaller, testable functions
- [ ] **Error Handling Consistency** - Standardize error handling
- [ ] **Configuration Management** - Externalize configuration
- [ ] **Remove TODOs** - Address inline TODO comments

### Dependencies
- [ ] **Update Dependencies** - Keep libraries up to date
- [ ] **Minimize Dependencies** - Remove unused dependencies
- [ ] **Vendor Dependencies** - Consider vendoring for stability

---

## Priority Levels
- 🔴 **Critical** - Blocks basic functionality
- 🟡 **High** - Important for production use
- 🟢 **Medium** - Nice to have features
- 🔵 **Low** - Future enhancements

## Getting Started
Pick items from the High Priority section first, then move to features that align with your use cases.

---

## Recent Completions (2025-10-29)

### pod-cpu-stress Implementation ✅
- **Status**: Fully implemented and tested
- **Approach**: Ephemeral containers with stress-ng
- **Features**:
  - Configurable CPU load percentage (1-100%)
  - Configurable CPU workers (1-32)
  - Duration-based stress testing
  - Resource limits to prevent node exhaustion
  - Automatic cleanup after duration expires
  - Full metrics and retry logic integration
- **Files**:
  - ADR: `docs/adr/0001-pod-cpu-stress-implementation.md`
  - Sample: `config/samples/chaos_v1alpha1_chaosexperiment_cpu_stress.yaml`
- **RBAC**: Added permissions for ephemeral containers
- **Validation**: Multi-layer validation (OpenAPI + webhooks)
- **Tests**: All tests passing with 19.2% controller coverage

### Retry Logic Implementation ✅
- **Status**: Fully implemented
- **Features**:
  - Configurable max retries (0-10, default: 3)
  - Two backoff strategies: exponential and fixed
  - Configurable initial retry delay
  - Status tracking with retry count, next retry time, and last error
  - Automatic retry on transient failures
  - Success resets retry counter

### Experiment Duration Lifecycle ✅
- **Status**: Fully implemented
- **Features**:
  - `experimentDuration` field for auto-stopping experiments
  - Automatic phase management (Pending → Running → Completed)
  - StartTime and CompletedAt timestamps
  - Graceful termination after duration expires

### Prometheus Metrics ✅
- **Status**: Fully implemented (see `docs/METRICS.md`)
- **Metrics Exported**:
  - `chaos_experiments_total` - Counter with action, namespace, status labels
  - `chaos_experiment_duration_seconds` - Histogram of execution times
  - `chaos_resources_affected` - Gauge of resources impacted
  - `chaos_experiment_errors_total` - Counter of errors by action/namespace
  - `chaos_active_experiments` - Gauge of currently running experiments