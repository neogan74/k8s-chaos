# k8s-chaos v1.0 Release Notes

**Release Date:** December 2, 2025

We're excited to announce the first production-ready release of k8s-chaos - a lightweight Kubernetes Chaos Engineering operator built with Kubebuilder v4. This release provides comprehensive chaos testing capabilities with a strong focus on safety, observability, and developer experience.

## 🎉 What is k8s-chaos?

k8s-chaos is a Kubernetes operator that enables controlled chaos engineering experiments to test application resilience and reliability. It provides a simple, declarative approach through Custom Resource Definitions (CRDs) while maintaining production-grade safety features and comprehensive observability.

## ✨ Highlights

- **Production-Ready Safety Features**: Comprehensive protection mechanisms for running chaos experiments safely in any environment
- **6 Chaos Actions**: Full suite of pod and node chaos capabilities
- **Smart Scheduling**: Cron-based recurring experiments with automatic retry logic
- **Full Observability**: Prometheus metrics, Grafana dashboards, and audit history
- **Excellent DX**: CLI tool, comprehensive documentation, and hands-on labs
- **Helm Chart**: Official Helm chart for easy installation and management

---

## 🚀 Core Features

### Chaos Actions

**Pod Chaos**
- **pod-kill**: Randomly delete pods to test deployment resilience and restart behavior
- **pod-delay**: Inject network latency (50ms-5s) using traffic control
- **pod-cpu-stress**: Consume CPU resources (1-100%) using ephemeral containers with stress-ng
- **pod-memory-stress**: Consume memory resources to test OOM handling and resource limits
- **pod-failure**: Kill main process (PID 1) to cause container crashes and test restart policies

**Node Chaos**
- **node-drain**: Drain nodes with cordon and eviction to test infrastructure failures and pod rescheduling

### Safety & Control Features

**Dry-Run Mode**
- Preview affected resources without executing chaos
- Perfect for validation and understanding impact before real experiments
- Works with all chaos actions

**Percentage Limits**
- Prevent affecting too many resources simultaneously
- Set `maxPercentage: 30` to limit impact to 30% of matching pods
- Automatic validation with helpful error messages

**Production Protection**
- Requires explicit `allowProduction: true` flag for production namespaces
- Production namespaces detected via:
  - Annotations: `chaos.gushchin.dev/production: "true"`
  - Labels: `environment: production` or `env: prod`
  - Name patterns: `production`, `prod-*`, `*-production`, `*-prod`

**Exclusion Labels**
- Protect critical pods: `chaos.gushchin.dev/exclude: "true"` (pod label)
- Protect entire namespaces: `chaos.gushchin.dev/exclude: "true"` (namespace annotation)
- Automatically filtered from all experiments

**Experiment Duration Control**
- Set `experimentDuration: "5m"` to auto-stop experiments after specified time
- Prevents runaway experiments and ensures time-bounded testing

### Scheduling & Reliability

**Cron-Based Scheduling**
- Automatic recurring experiments using standard cron syntax
- Supports predefined schedules: `@hourly`, `@daily`, `@weekly`, `@monthly`, `@yearly`
- Examples:
  - `"*/30 * * * *"` - Every 30 minutes
  - `"0 2 * * *"` - Daily at 2 AM
  - `"0 9 * * 1"` - Every Monday at 9 AM
  - `"*/15 9-17 * * 1-5"` - Every 15 minutes during business hours
- Status tracks `lastScheduledTime` and `nextScheduledTime`

**Automatic Retry Logic**
- Configurable retry attempts (0-10, default: 3)
- Two backoff strategies:
  - **Exponential**: Delay doubles with each retry (30s → 1m → 2m → 4m → 8m, max 10m)
  - **Fixed**: Constant delay between retries
- Status tracks `retryCount`, `lastError`, and `nextRetryTime`
- Automatic reset on success

### Observability & Monitoring

**Prometheus Metrics**
- `chaos_experiments_total`: Total experiments executed (by action, status, target_namespace)
- `chaos_experiments_duration_seconds`: Experiment execution duration (p50, p95, p99)
- `chaos_resources_affected_total`: Resources affected by chaos (pods, nodes)
- `chaos_experiments_errors_total`: Errors encountered (by action, error_type)
- `chaos_active_experiments`: Currently active experiments
- Safety metrics: dry-run count, production blocks, percentage violations

**Grafana Dashboards**
Three comprehensive dashboards included:
1. **Overview Dashboard**: High-level executive view with success rates, experiment trends, and resource impact
2. **Detailed Analysis Dashboard**: Deep dive with filtering by action/namespace, duration percentiles, error tracking
3. **Safety Monitoring Dashboard**: Focus on errors, resource impact, and safety threshold monitoring

**Experiment History & Audit Logging**
- Automatic recording of all experiment executions via `ChaosExperimentHistory` CRD
- Complete audit trail with experiment config, execution details, and affected resources
- Configurable retention limits (default: 100 records per experiment)
- Label-based indexing for efficient querying by experiment, action, status, namespace
- Perfect for compliance, debugging, and post-incident analysis

### Developer Experience

**CLI Tool**
Rich command-line interface for managing experiments:
```bash
k8s-chaos list                    # List all experiments
k8s-chaos describe <name> -n <ns> # View detailed status
k8s-chaos stats                   # Show statistics
k8s-chaos top                     # Top experiments by metrics
```

**Comprehensive Documentation**
- [Getting Started Guide](docs/GETTING-STARTED.md) - Complete tutorial for first-time users
- [Best Practices](docs/BEST-PRACTICES.md) - Safety principles and progressive adoption
- [Troubleshooting](docs/TROUBLESHOOTING.md) - Common issues and solutions
- [Real-World Scenarios](docs/SCENARIOS.md) - 13 ready-to-use chaos experiments
- [API Reference](docs/API.md) - Complete CRD specification
- [CLI Documentation](docs/CLI.md) - Command-line interface guide
- [Metrics Guide](docs/METRICS.md) - Prometheus metrics reference
- [Grafana Setup](docs/GRAFANA.md) - Dashboard installation and configuration
- [History & Audit](docs/HISTORY.md) - Audit logging and query patterns

**Hands-On Labs**
Interactive learning environment with automated setup:
- Lab 01: Getting Started - Basic pod-kill experiments
- Lab 02: Network Chaos - Pod-delay injection
- Lab 03: Resource Stress - CPU and memory stress testing
- Lab 04: Node Chaos - Node drain experiments
- Lab 05: Safety Features - Dry-run, exclusions, production protection
- Lab 06: Scheduled Experiments - Cron-based automation

**Validation & Testing**
- Multi-layer validation (OpenAPI schema + admission webhooks)
- Namespace existence checks
- Selector effectiveness validation
- Cross-field constraints (e.g., duration required for certain actions)
- Comprehensive test suite with unit, integration, and e2e tests

### Installation & Deployment

**Helm Chart**
Official Helm chart with:
- Flexible configuration options
- Support for all operator features (history, metrics, scheduling)
- Service monitor integration for Prometheus
- Grafana dashboard ConfigMaps
- Production-ready defaults

**Quick Installation**
```bash
# Install using Helm
helm install k8s-chaos ./charts/k8s-chaos -n chaos-system --create-namespace

# Or using kubectl
make install deploy IMG=ghcr.io/neogan74/k8s-chaos:v0.1.0
```

---

## 📋 Requirements

- **Kubernetes**: 1.24+
- **kubectl**: Configured to access your cluster
- **Go**: 1.24.5+ (for development)
- **Docker**: For building images
- **Helm**: 3.x (optional, for Helm installation)

---

## 🎯 Example Usage

### Basic Pod Kill Experiment
```yaml
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: nginx-chaos
  namespace: default
spec:
  action: pod-kill
  namespace: production
  selector:
    app: nginx
  count: 2
  experimentDuration: "5m"
```

### Scheduled CPU Stress
```yaml
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: scheduled-cpu-stress
  namespace: default
spec:
  action: pod-cpu-stress
  namespace: production
  selector:
    app: api-server
  count: 1
  cpuLoad: 80
  cpuWorkers: 4
  duration: "2m"
  schedule: "0 */4 * * *"  # Every 4 hours
  maxRetries: 3
  retryBackoff: exponential
```

### Safe Production Testing
```yaml
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: safe-production-test
  namespace: default
spec:
  action: pod-failure
  namespace: production
  selector:
    app: web-frontend
  maxPercentage: 25  # Limit to 25% of pods
  dryRun: true      # Preview first
  allowProduction: true
```

---

## 📊 Architecture Decisions

This release includes comprehensive Architecture Decision Records (ADRs) documenting key design choices:

- **ADR-0001**: CRD Validation Strategy - Multi-layer validation approach
- **ADR-0002**: Safety Features Implementation - Production protection mechanisms
- **ADR-0003**: Pod Memory Stress Implementation - Memory chaos using stress-ng
- **ADR-0004**: Pod Failure Implementation - Process-kill approach for container crashes
- **ADR-0005**: Pod CPU Stress Implementation - Ephemeral container strategy
- **ADR-0006**: Experiment History and Audit Logging - Immutable audit trail design

All ADRs are available in the `docs/adr/` directory.

---

## 🔧 What's Changed

### Added
- ✅ Complete chaos operator with 6 actions (pod-kill, pod-delay, pod-cpu-stress, pod-memory-stress, pod-failure, node-drain)
- ✅ Comprehensive safety features (dry-run, maxPercentage, exclusion labels, production protection)
- ✅ Cron-based scheduling for automated recurring experiments
- ✅ Automatic retry logic with exponential and fixed backoff strategies
- ✅ Experiment duration control with auto-stop functionality
- ✅ Full Prometheus metrics integration with 5+ metric types
- ✅ Three Grafana dashboards (overview, detailed analysis, safety monitoring)
- ✅ Experiment history and audit logging with ChaosExperimentHistory CRD
- ✅ CLI tool with list, describe, stats, and top commands
- ✅ Official Helm chart with production-ready configuration
- ✅ Multi-layer validation (OpenAPI + admission webhooks)
- ✅ Comprehensive documentation (8 guides + API reference)
- ✅ Hands-on labs with 6 interactive tutorials
- ✅ Complete test suite (unit, integration, e2e)

### Fixed
- 🐛 Controller logic bug: inverted condition for pod-kill action routing (commit 427f4a4)
- 🐛 Removed deprecated `rand.Seed()` call
- 🐛 Fixed JSON tag typo from `omitemptly` to `omitempty`
- 🐛 Updated GitHub Actions golangci-lint-action from v8 to v6
- 🐛 Translated Russian comments to English for better accessibility

---

## 📚 Documentation

Complete documentation is available in the repository:

- **[README.md](README.md)** - Project overview and quick start
- **[Getting Started](docs/GETTING-STARTED.md)** - Step-by-step first experiment
- **[Best Practices](docs/BEST-PRACTICES.md)** - Production usage guidelines
- **[Troubleshooting](docs/TROUBLESHOOTING.md)** - Common issues and solutions
- **[Real-World Scenarios](docs/SCENARIOS.md)** - 13 ready-to-use examples
- **[API Reference](docs/API.md)** - Complete CRD specification
- **[CLI Tool](docs/CLI.md)** - Command-line interface guide
- **[Metrics Guide](docs/METRICS.md)** - Prometheus metrics reference
- **[Grafana Setup](docs/GRAFANA.md)** - Dashboard configuration
- **[History & Audit](docs/HISTORY.md)** - Audit logging guide
- **[Labs](labs/README.md)** - Interactive learning tutorials
- **[Roadmap](ROADMAP.md)** - Future development plans
- **[ADRs](docs/adr/)** - Architecture decision records

---

## 🎓 Learning Resources

### Quick Start
```bash
# 1. Create a local cluster
make cluster-single-node

# 2. Install k8s-chaos
helm install k8s-chaos ./charts/k8s-chaos -n chaos-system --create-namespace

# 3. Try Lab 01
cd labs/01-getting-started
make setup
kubectl apply -f experiments/01-simple-pod-kill.yaml
```

### Hands-On Labs
Interactive tutorials with automated setup:
- `labs/01-getting-started/` - Your first chaos experiment
- `labs/02-network-chaos/` - Network latency injection
- `labs/03-resource-stress/` - CPU and memory testing
- `labs/04-node-chaos/` - Node drain experiments
- `labs/05-safety-features/` - Production safety mechanisms
- `labs/06-scheduled-experiments/` - Automated chaos testing

---

## 🔐 Security Considerations

- **RBAC**: Controller requires specific permissions (get/list/watch pods, delete pods, create pod execs, etc.)
- **Namespace Isolation**: Experiments are namespace-scoped by design
- **Multi-Layer Validation**: OpenAPI schema + admission webhooks prevent malicious configurations
- **Audit Logging**: All chaos actions recorded in experiment history for audit trails
- **Production Protection**: Explicit approval required for production namespace experiments
- **Exclusion Mechanisms**: Critical resources can be protected with exclusion labels

---

## 🤝 Contributing

We welcome contributions! Here's how to get started:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes and add tests
4. Run `make test lint` to verify
5. Commit your changes (`git commit -m 'Add amazing feature'`)
6. Push to the branch (`git push origin feature/amazing-feature`)
7. Open a Pull Request

See [CONTRIBUTING.md](CONTRIBUTING.md) for detailed guidelines.

---

## 🗺️ Roadmap

Future development plans (see [ROADMAP.md](ROADMAP.md) for details):

**Near Term (v1.1.0)**
- Advanced network chaos: packet loss, bandwidth limitation, network partitioning
- Auto-uncordon nodes after drain experiments complete
- Auto-cleanup of ephemeral containers after CPU stress experiments
- Additional safety metrics (dry-run count, production blocks, violations)

**Medium Term (v1.2.0)**
- Disk I/O chaos: disk-fill, disk-latency
- HTTP chaos: request delay, error injection, rate limiting
- Container restart chaos
- ConfigMap/Secret chaos

**Long Term (v2.0.0)**
- Web UI dashboard for experiment management
- Experiment templates and scenarios library
- Multi-cluster chaos orchestration
- Integration with service mesh (Istio, Linkerd)
- Chaos workflows and pipelines

---

## 📦 Installation Assets

### Helm Chart
```bash
# Add repository (when published to Helm registry)
helm repo add k8s-chaos https://neogan74.github.io/k8s-chaos

# Install
helm install k8s-chaos k8s-chaos/k8s-chaos -n chaos-system --create-namespace

# Or install from local chart
helm install k8s-chaos ./charts/k8s-chaos -n chaos-system --create-namespace
```

### Container Images
- **ghcr.io/neogan74/k8s-chaos:v1.0.0** - Main operator image
- **ghcr.io/neogan74/k8s-chaos:latest** - Latest stable release

### Kubernetes Manifests
```bash
# Install CRDs
kubectl apply -f https://raw.githubusercontent.com/neogan74/k8s-chaos/main/config/crd/bases/chaos.gushchin.dev_chaosexperiments.yaml
kubectl apply -f https://raw.githubusercontent.com/neogan74/k8s-chaos/main/config/crd/bases/chaos.gushchin.dev_chaosexperimenthistories.yaml

# Deploy operator
kubectl apply -k https://github.com/neogan74/k8s-chaos/config/default?ref=v1.0.0
```

---

## 🙏 Acknowledgments

- Built with [Kubebuilder](https://kubebuilder.io/) v4
- Inspired by [Chaos Mesh](https://chaos-mesh.org/) and [Litmus Chaos](https://litmuschaos.io/)
- Thanks to the Kubernetes SIG API Machinery community
- Special thanks to all contributors and early adopters

---

## 📄 License

k8s-chaos is licensed under the Apache License 2.0. See [LICENSE](LICENSE) for details.

Copyright 2025.

---

## 📞 Support & Community

- **Issues**: [GitHub Issues](https://github.com/neogan74/k8s-chaos/issues)
- **Discussions**: [GitHub Discussions](https://github.com/neogan74/k8s-chaos/discussions)
- **Documentation**: [docs/](docs/)
- **Examples**: [config/samples/](config/samples/)

---

## 🎯 What's Next?

After installing k8s-chaos v1.0.0:

1. **Start Learning**: Follow the [Getting Started Guide](docs/GETTING-STARTED.md)
2. **Try Labs**: Complete hands-on tutorials in `labs/`
3. **Setup Monitoring**: Configure Prometheus and Grafana dashboards
4. **Read Best Practices**: Learn safe chaos engineering in [Best Practices](docs/BEST-PRACTICES.md)
5. **Explore Scenarios**: Try real-world examples from [Scenarios Guide](docs/SCENARIOS.md)

Happy chaos testing! 🎉