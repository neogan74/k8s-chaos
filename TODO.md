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
- [x] **Implement Safety Checks** - COMPLETED (see docs/adr/0002-safety-features-implementation.md)
  - [x] Add dry-run mode to preview affected pods
  - [x] Implement maximum percentage limit (e.g., max 30% of pods)
  - [x] Add exclusion labels to protect critical pods
  - [x] Add confirmation/approval mechanism for production namespaces

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
  - [x] Add `affectedPods` list with pod names
  - [x] Add `startTime` and `completedAt` timestamps
  - [x] Add retry tracking fields (retryCount, nextRetryTime, lastError)
- [ ] **Kubernetes Events** - Emit events on ChaosExperiment and affected pods

## 🚀 New Chaos Actions

### Pod Chaos
- [x] **pod-delay** - Add network latency to pods
- [x] **pod-cpu-stress** - Consume CPU resources
- [x] **pod-memory-stress** - Consume memory resources
- [x] **pod-network-loss** - Simulate packet loss
- [x] **pod-disk-fill** - Fill pod disk space
- [ ] **pod-network-corruption** - Corrupt network packets
- [ ] **pod-restart** - Restart pods instead of delete

### Node Chaos
- [x] **node-drain** - Drain nodes temporarily
- [x] **node-uncordon** - Auto-uncordon nodes after drain experiments complete ✅
- [ ] **node-taint** - Add taints to nodes
- [ ] **node-cpu-stress** - Stress node CPU
- [ ] **node-disk-fill** - Fill node disk space

### Network Chaos
- [ ] **network-partition** - Simulate network splits
- [ ] **dns-chaos** - DNS resolution failures
- [ ] **http-chaos** - HTTP response manipulation

## ⏰ Scheduling & Duration

### Scheduling
- [x] **Cron Scheduling** - Add cron expression support for recurring experiments ✅ COMPLETED
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
- [x] **Getting Started Guide** - Step-by-step tutorial ✅ COMPLETED (docs/GETTING-STARTED.md)
- [x] **Example Scenarios** - Real-world use cases ✅ COMPLETED (docs/SCENARIOS.md)
- [x] **Best Practices** - Guidelines for safe chaos testing ✅ COMPLETED (docs/BEST-PRACTICES.md)
- [x] **Troubleshooting Guide** - Common issues and solutions ✅ COMPLETED (docs/TROUBLESHOOTING.md)

### Developer Documentation
- [x] **Architecture Decision Records (ADRs)** - Created ADR for pod-cpu-stress implementation
- [x] **API Documentation** - Detailed CRD field descriptions
- [x] **Contributing Guide** - How to add new chaos actions
- [x] **Code Comments** - Translate Russian comments to English

## 🔒 Security

### RBAC
- [ ] **Fine-grained Permissions** - Separate roles for different chaos levels
- [ ] **Namespace Isolation** - Restrict experiments to specific namespaces
- [x] **Audit Logging** - Track who runs experiments

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
- [x] **Helm Chart** - Create Helm chart for easier deployment ✅ COMPLETED (charts/k8s-chaos/)
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
---

## Recent Completions (2025-10-30)

### Safety Features Implementation ✅
- **Status**: Fully implemented and production-ready
- **Architecture**: Comprehensive ADR documented in `docs/adr/0002-safety-features-implementation.md`
- **Features Implemented**:
  - **Dry-Run Mode**: Preview affected resources without execution (`dryRun: true`)
    - Works for all actions: pod-kill, pod-delay, pod-cpu-stress, node-drain
    - Status message shows exact resources that would be affected
    - No requeueing for dry-run experiments
  - **Maximum Percentage Limit**: Prevent over-affecting resources (`maxPercentage: 1-100`)
    - Webhook validation with helpful error messages
    - Calculates actual percentage and suggests correct count values
    - Example: `maxPercentage: 30` ensures ≤30% of pods affected
  - **Production Namespace Protection**: Explicit approval required (`allowProduction: true`)
    - Multiple detection methods: annotations, labels, name patterns
    - Clear error messages guide users to add approval flag
    - Blocks unauthorized production experiments at webhook level
  - **Exclusion Labels**: Protect critical resources (`chaos.gushchin.dev/exclude: "true"`)
    - Pod-level exclusion via label
    - Namespace-level exclusion via annotation
    - Automatically filtered in all action handlers
    - Webhook warnings when pods excluded
- **Files**:
  - API types: Added 3 safety fields (dryRun, maxPercentage, allowProduction)
  - Webhook: Multi-layer safety validation pipeline
  - Controller: Safety helpers + updated all action handlers
  - Sample: `config/samples/chaos_v1alpha1_chaosexperiment_safety_demo.yaml`
- **RBAC**: Added namespace get/list permissions
- **Validation**: Code compiles successfully (`go vet` passed)
- **Impact**: Operator is now production-ready with multiple protection layers

## Recent Completions (2025-11-30)

### Quality-of-Life Improvements ✅
- **Status**: Fully implemented and tested
- **Features Implemented**:
  1. **Auto-Cleanup of Ephemeral Containers** ✅
     - Smart lifecycle management for CPU stress ephemeral containers
     - Checks container runtime status (not just existence)
     - Allows repeated experiments when previous containers complete
     - Prevents accumulation while providing audit trail
     - Location: `internal/controller/chaosexperiment_controller.go:484-556, 1590-1605`

  2. **Auto-Uncordon Nodes After Drain** ✅
     - Tracks nodes cordoned by experiments in `status.cordonedNodes`
     - Automatically uncordons when `experimentDuration` completes
     - Respects pre-existing cordoned state (only uncordons what we cordoned)
     - Graceful handling of individual node failures
     - New status field: `cordonedNodes []string`
     - New functions: `cordonNode()` returns bool, `uncordonNode()`
     - Location: `api/v1alpha1/chaosexperiment_types.go:208-211`
     - Location: `internal/controller/chaosexperiment_controller.go:812-857, 1158-1170`

  3. **Safety Metrics** ✅
     - Four new Prometheus metrics for observability:
       - `chaosexperiment_safety_dryrun_total` - Dry-run executions count
       - `chaosexperiment_safety_production_blocks_total` - Production blocks count
       - `chaosexperiment_safety_percentage_violations_total` - Percentage violations count
       - `chaosexperiment_safety_excluded_resources_total` - Excluded resources count (by type)
     - Tracked in webhook (production blocks, percentage violations)
     - Tracked in controller (dry-run, excluded resources)
     - Location: `internal/metrics/metrics.go:98-149`
     - Location: `api/v1alpha1/chaosexperiment_webhook.go:305, 349`
     - Location: `internal/controller/chaosexperiment_controller.go:1105, 1246-1259`

- **Testing**: All tests passing, code properly formatted
- **Impact**: Enhanced production readiness with better cleanup, automatic recovery, and comprehensive safety monitoring

## Recent Completions (2025-12-02)

### Comprehensive Documentation ✅
- **Status**: Complete documentation suite for users and contributors
- **User Documentation Completed**:
  - **Getting Started Guide** (`docs/GETTING-STARTED.md`) - Complete installation and first experiment tutorial
  - **Best Practices** (`docs/BEST-PRACTICES.md`) - Safety-first principles, progressive adoption, experiment design
  - **Troubleshooting** (`docs/TROUBLESHOOTING.md`) - Common issues and solutions with debug procedures
  - **Real-World Scenarios** (`docs/SCENARIOS.md`) - 13 ready-to-use scenarios covering web apps, microservices, databases, infrastructure

- **Labs Infrastructure Completed**:
  - Labs directory structure with README (`labs/README.md`)
  - Lab 01: Getting Started with hands-on exercises
  - Kind cluster configurations (single-node and multi-node)
  - Makefile targets for easy cluster management:
    - `make cluster-single-node` - Create 1-node cluster
    - `make cluster-multi-node` - Create 3-node cluster (1 control-plane + 2 workers)
    - `make labs-setup` - Complete automated setup
    - `make labs-teardown` - Clean teardown

- **Impact**: Users can now easily get started, learn best practices, troubleshoot issues, and use real-world examples. Labs provide hands-on learning experience.

### Production-Ready Helm Chart ✅
- **Status**: Complete and tested
- **Features Implemented**:
  - **Official Helm Chart** (`charts/k8s-chaos/`) - Production-ready installation
  - **Comprehensive values.yaml** - 50+ configurable parameters
  - **Complete Templates** - Deployment, RBAC, Service, Webhook, ServiceMonitor
  - **Multiple Installation Modes** - Dev, staging, production configurations
  - **Security Defaults** - Non-root, read-only filesystem, dropped capabilities
  - **Certificate Management** - Self-signed and cert-manager support
  - **Observability** - Optional ServiceMonitor for Prometheus Operator
  - **Helm README** (`charts/k8s-chaos/README.md`) - Comprehensive documentation with examples
  - **Post-install Notes** - Helpful NOTES.txt with next steps

- **Documentation Updates**:
  - Main README updated with Helm as primary installation method
  - GETTING-STARTED.md updated with Option A (Helm) and Option B (Manual)
  - ROADMAP.md marked Helm chart as complete in Q1 2026

- **Testing**: Helm lint passed, template rendering successful
- **Impact**: Major adoption barrier removed - one-command installation now available!
