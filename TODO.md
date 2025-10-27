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
- [x] **Add Retry Logic** - Implement exponential backoff for transient failures
- [x] **Handle Edge Cases**
  - [x] What if namespace doesn't exist? - Webhook validates this
  - [ ] What if pods are already terminating?
  - [ ] Handle permission denied errors gracefully

## 📊 Observability

### Monitoring
- [x] **Add Prometheus Metrics**
  - [x] `chaos_experiments_total` - Total experiments run
  - [x] `chaos_experiments_failed` - Failed experiments
  - [x] `chaos_pods_deleted_total` - Total pods deleted
  - [x] `chaos_experiment_duration_seconds` - Experiment execution time
- [x] **Implement Structured Logging**
  - [x] Add correlation IDs for tracking experiments
  - [x] Log affected pod names and namespaces
  - [x] Add log levels (debug, info, warn, error)

### Status Reporting
- [x] **Enhance Status Fields**
  - [x] Add `phase` field (Pending, Running, Completed, Failed)
  - [ ] Add `affectedPods` list with pod names
  - [x] Add `startTime` and `endTime` timestamps
  - [ ] Add `conditions` array for detailed status
- [ ] **Kubernetes Events** - Emit events on ChaosExperiment and affected pods

## 🚀 New Chaos Actions

### Pod Chaos
- [x] **pod-delay** - Add network latency to pods
- [ ] **pod-cpu-stress** - Consume CPU resources
- [ ] **pod-memory-stress** - Consume memory resources
- [ ] **pod-network-loss** - Simulate packet loss
- [ ] **pod-network-corruption** - Corrupt network packets
- [ ] **pod-restart** - Restart pods instead of delete

### Node Chaos
- [x] **node-drain** - Drain nodes temporarily
- [ ] **node-taint** - Add taints to nodes
- [ ] **node-cpu-stress** - Stress node CPU
- [ ] **node-disk-fill** - Fill node disk space
- [ ] **node-uncordon** - Auto-uncordon nodes after drain experiments complete

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
- [x] **Experiment Duration** - Add `experimentDuration` field to auto-stop experiments
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
- [x] **Architecture Decision Records (ADRs)**
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

## 🖥️ CLI Tool

### Core Commands
- [x] **list** - List all chaos experiments with compact/wide output
- [x] **describe** - Show detailed experiment information
- [x] **delete** - Delete experiments with confirmation prompt
- [x] **stats** - Display aggregate statistics (success rates, action breakdown)
- [x] **top** - Show top experiments by retries, age, and failures

### Advanced Commands
- [ ] **create** - Interactive wizard for creating experiments
  - [ ] Guided prompts for action, namespace, selector
  - [ ] Validation before creation
  - [ ] Template selection
- [ ] **validate** - Validate experiment YAML files
  - [ ] Schema validation
  - [ ] Cross-field validation
  - [ ] Namespace and selector checks
- [ ] **check** - Health check for cluster readiness
  - [ ] Verify CRD installation
  - [ ] Check RBAC permissions
  - [ ] Test API connectivity
- [ ] **logs** - Show experiment execution history
  - [ ] View past executions
  - [ ] Filter by date/status
  - [ ] Export to file

### Enhancements
- [ ] **Watch mode** - Real-time updates with `--watch` flag
- [ ] **Export formats** - JSON/CSV export for stats
- [ ] **Dashboard** - Web-based UI integration
- [ ] **Shell completion** - Bash/Zsh/Fish autocompletion
- [ ] **Config file** - Support for `.k8s-chaos.yaml` config file

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