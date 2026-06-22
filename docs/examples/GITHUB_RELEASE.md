# k8s-chaos v1.0.0 🎉

We're excited to announce the first stable release of **k8s-chaos** - a production-ready, lightweight Kubernetes Chaos Engineering operator!

## 🚀 What's New

This release brings a complete, production-ready chaos engineering platform with comprehensive safety features and excellent observability.

### ✨ Key Features

**6 Chaos Actions**
- `pod-kill` - Test deployment resilience and restart behavior
- `pod-delay` - Inject network latency (50ms-5s)
- `pod-cpu-stress` - Consume CPU resources (1-100%)
- `pod-memory-stress` - Test OOM handling and memory limits
- `pod-failure` - Kill main process to test crash recovery
- `node-drain` - Test infrastructure failures and rescheduling

**Safety First**
- 🛡️ **Dry-Run Mode** - Preview impact before execution
- 📊 **Percentage Limits** - Prevent affecting too many resources (`maxPercentage`)
- 🔒 **Production Protection** - Explicit approval required for prod namespaces
- 🏷️ **Exclusion Labels** - Protect critical pods/namespaces
- ⏱️ **Duration Control** - Auto-stop experiments after specified time

**Smart Scheduling & Reliability**
- ⏰ **Cron Scheduling** - Automated recurring experiments (`"0 2 * * *"`, `@hourly`, etc.)
- 🔄 **Automatic Retry** - Configurable backoff strategies (exponential/fixed)
- 📈 **Status Tracking** - Full visibility into experiment execution

**Complete Observability**
- 📊 **Prometheus Metrics** - 5+ metric types for comprehensive monitoring
- 📉 **3 Grafana Dashboards** - Overview, detailed analysis, and safety monitoring
- 📝 **Audit Logging** - Complete experiment history with `ChaosExperimentHistory` CRD
- 🔍 **Label-based Querying** - Efficient filtering by experiment, action, status

**Excellent Developer Experience**
- 🛠️ **CLI Tool** - Rich commands (`list`, `describe`, `stats`, `top`)
- 📚 **Comprehensive Docs** - 8 guides + API reference + ADRs
- 🧪 **Hands-On Labs** - 6 interactive tutorials with automated setup
- ✅ **Multi-Layer Validation** - OpenAPI schema + admission webhooks
- ⚙️ **Helm Chart** - Official chart with production-ready configuration

## 📦 Installation

### Using Helm
```bash
helm install k8s-chaos ./charts/k8s-chaos -n chaos-system --create-namespace
```

### Using kubectl
```bash
# Install CRDs
kubectl apply -f https://raw.githubusercontent.com/neogan74/k8s-chaos/v1.0.0/config/crd/bases/chaos.gushchin.dev_chaosexperiments.yaml
kubectl apply -f https://raw.githubusercontent.com/neogan74/k8s-chaos/v1.0.0/config/crd/bases/chaos.gushchin.dev_chaosexperimenthistories.yaml

# Deploy operator
kubectl apply -k https://github.com/neogan74/k8s-chaos/config/default?ref=v1.0.0
```

### Container Images
```
ghcr.io/neogan74/k8s-chaos:v1.0.0
```

## 🎯 Quick Start Example

```yaml
apiVersion: chaos.gushchin.dev/v1.0.0
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
  maxPercentage: 30
  allowProduction: true
```

## 📋 What's Included

### Core Operator
- ✅ Complete chaos operator with 6 actions
- ✅ Comprehensive safety features
- ✅ Cron-based scheduling
- ✅ Automatic retry logic
- ✅ Experiment duration control
- ✅ Multi-layer validation (OpenAPI + webhooks)

### Observability
- ✅ Full Prometheus metrics integration
- ✅ Three Grafana dashboards
- ✅ Experiment history and audit logging (ChaosExperimentHistory CRD)
- ✅ Safety metrics tracking

### Developer Tools
- ✅ CLI tool with rich commands
- ✅ Official Helm chart
- ✅ Comprehensive test suite (unit, integration, e2e)

### Documentation
- ✅ 8 comprehensive guides
- ✅ 6 hands-on labs with automated setup
- ✅ 6 Architecture Decision Records (ADRs)
- ✅ 13 real-world scenario examples
- ✅ Complete API reference

## 🐛 Bug Fixes

- Fixed controller logic bug: inverted condition for pod-kill action routing (427f4a4)
- Removed deprecated `rand.Seed()` call
- Fixed JSON tag typo from `omitemptly` to `omitempty`
- Updated GitHub Actions golangci-lint-action from v8 to v6
- Translated Russian comments to English for accessibility

## 📚 Documentation

- **[Getting Started](https://github.com/neogan74/k8s-chaos/blob/main/docs/GETTING-STARTED.md)** - Complete tutorial
- **[Best Practices](https://github.com/neogan74/k8s-chaos/blob/main/docs/BEST-PRACTICES.md)** - Production guidelines
- **[Troubleshooting](https://github.com/neogan74/k8s-chaos/blob/main/docs/TROUBLESHOOTING.md)** - Common issues
- **[Real-World Scenarios](https://github.com/neogan74/k8s-chaos/blob/main/docs/SCENARIOS.md)** - 13 ready-to-use examples
- **[CLI Documentation](https://github.com/neogan74/k8s-chaos/blob/main/docs/CLI.md)** - Command reference
- **[Metrics Guide](https://github.com/neogan74/k8s-chaos/blob/main/docs/METRICS.md)** - Prometheus integration
- **[Grafana Setup](https://github.com/neogan74/k8s-chaos/blob/main/docs/GRAFANA.md)** - Dashboard configuration
- **[History & Audit](https://github.com/neogan74/k8s-chaos/blob/main/docs/HISTORY.md)** - Audit logging guide

## 📊 Architecture Decision Records

This release includes comprehensive ADRs documenting key design choices:

- **[ADR-0001](https://github.com/neogan74/k8s-chaos/blob/main/docs/adr/0001-crd-validation-strategy.md)** - CRD Validation Strategy
- **[ADR-0002](https://github.com/neogan74/k8s-chaos/blob/main/docs/adr/0002-safety-features-implementation.md)** - Safety Features Implementation
- **[ADR-0003](https://github.com/neogan74/k8s-chaos/blob/main/docs/adr/0003-pod-memory-stress-implementation.md)** - Pod Memory Stress Implementation
- **[ADR-0004](https://github.com/neogan74/k8s-chaos/blob/main/docs/adr/0004-pod-failure-implementation.md)** - Pod Failure Implementation
- **[ADR-0005](https://github.com/neogan74/k8s-chaos/blob/main/docs/adr/0005-pod-cpu-stress-implementation.md)** - Pod CPU Stress Implementation
- **[ADR-0006](https://github.com/neogan74/k8s-chaos/blob/main/docs/adr/0006-experiment-history-and-audit-logging.md)** - Experiment History and Audit Logging

## 🎓 Learning Resources

Get started quickly with our hands-on labs:

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

### Available Labs
- `labs/01-getting-started/` - Your first chaos experiment
- `labs/02-network-chaos/` - Network latency injection
- `labs/03-resource-stress/` - CPU and memory testing
- `labs/04-node-chaos/` - Node drain experiments
- `labs/05-safety-features/` - Production safety mechanisms
- `labs/06-scheduled-experiments/` - Automated chaos testing

## 🔒 Security Considerations

- **RBAC**: Controller requires specific permissions for pod/node management
- **Namespace Isolation**: Experiments are namespace-scoped by design
- **Multi-Layer Validation**: OpenAPI + admission webhooks prevent malicious configs
- **Audit Logging**: All chaos actions recorded for compliance
- **Production Protection**: Explicit approval required for production experiments

## 🗺️ Roadmap

**Near Term (v1.1.0)**
- Advanced network chaos (packet loss, bandwidth limits, partitioning)
- Auto-uncordon nodes after drain experiments
- Auto-cleanup of ephemeral containers
- Additional safety metrics

**Medium Term (v1.2.0)**
- Disk I/O chaos (disk-fill, disk-latency)
- HTTP chaos (delay, error injection, rate limiting)
- Container restart chaos
- ConfigMap/Secret chaos

**Long Term (v2.0.0)**
- Web UI dashboard
- Experiment templates library
- Multi-cluster orchestration
- Service mesh integration (Istio, Linkerd)

## 📋 Requirements

- **Kubernetes**: 1.24+
- **kubectl**: Configured to access your cluster
- **Helm**: 3.x (optional, for Helm installation)
- **Go**: 1.24.5+ (for development only)

## 🙏 Acknowledgments

- Built with [Kubebuilder](https://kubebuilder.io/) v4
- Inspired by [Chaos Mesh](https://chaos-mesh.org/) and [Litmus Chaos](https://litmuschaos.io/)
- Thanks to the Kubernetes SIG API Machinery community
- Special thanks to all contributors and early adopters

## 📄 License

k8s-chaos is licensed under the Apache License 2.0.

---

## 🎯 What's Next?

1. ⭐ **Star the repository** to support the project
2. 📖 **Read the [Getting Started Guide](https://github.com/neogan74/k8s-chaos/blob/main/docs/GETTING-STARTED.md)**
3. 🧪 **Try the hands-on labs** in the `labs/` directory
4. 📊 **Setup monitoring** with Prometheus and Grafana
5. 💬 **Join discussions** and share your feedback

Happy chaos testing! 🎉

---

**Full Changelog**: https://github.com/neogan74/k8s-chaos/commits/v1.0.0