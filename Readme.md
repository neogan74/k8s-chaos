# k8s-chaos: Kubernetes Chaos Engineering Operator

 [![Go Version](https://img.shields.io/badge/Go-1.24.5+-blue.svg)](https://golang.org)
[![Kubernetes](https://img.shields.io/badge/Kubernetes-1.24+-blue.svg)](https://kubernetes.io)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

A **production-ready**, lightweight Kubernetes Chaos Engineering operator built with Kubebuilder v4. Test your application's resilience through controlled chaos injection with comprehensive safety features.

## ✨ Highlights

- 🛡️ **Safety First**: Dry-run mode, percentage limits, exclusion labels, production protection
- 🎯 **6 Chaos Actions**: Pod kill, delay, CPU/memory stress, failure, node drain
- ⏰ **Smart Scheduling**: Cron-based recurring experiments with duration control
- 📊 **Full Observability**: Prometheus metrics, Grafana dashboards, audit history
- 🔄 **Automatic Retry**: Configurable backoff strategies for transient failures
- 📚 **Comprehensive Docs**: Getting started guide, best practices, real-world scenarios
- 🧪 **Hands-on Labs**: Interactive learning environment with automated setup

## 🚀 Features

### Chaos Actions

**Pod Chaos**
- ✅ **pod-kill**: Delete pods to test deployment resilience
- ✅ **pod-delay**: Inject network latency (50ms-5s)
- ✅ **pod-cpu-stress**: Consume CPU resources (1-100%)
- ✅ **pod-memory-stress**: Consume memory resources
- ✅ **pod-failure**: Kill main process to test restart behavior

**Node Chaos**
- ✅ **node-drain**: Drain nodes with automatic uncordon

### Safety & Control

- ✅ **Dry-Run Mode**: Preview affected resources without execution
- ✅ **Max Percentage Limits**: Prevent affecting too many resources (e.g., max 30%)
- ✅ **Production Protection**: Explicit approval required for production namespaces
- ✅ **Exclusion Labels**: Protect critical pods/namespaces
- ✅ **Experiment Duration**: Auto-stop after specified time
- ✅ **Cron Scheduling**: Recurring experiments (`*/30 * * * *`)
- ✅ **Retry Logic**: Exponential or fixed backoff strategies

### Observability

- ✅ **Prometheus Metrics**: Experiments, duration, resources affected, errors, safety metrics
- ✅ **Grafana Dashboards**: 3 comprehensive dashboards (overview, detailed, safety)
- ✅ **Experiment History**: Full audit trail with configurable retention
- ✅ **Safety Metrics**: Track dry-runs, production blocks, percentage violations

### Developer Experience

- ✅ **CLI Tool**: Rich commands for listing, describing, stats, and top experiments
- ✅ **Comprehensive Docs**: Getting Started, Best Practices, Troubleshooting, Scenarios
- ✅ **Hands-on Labs**: Step-by-step tutorials with automated cluster setup
- ✅ **Validation**: Multi-layer validation (OpenAPI + admission webhooks)

## 🚀 Quick Start

**New to k8s-chaos?** Follow our [Getting Started Guide](docs/GETTING-STARTED.md) for a complete tutorial.

```bash
# 1. Create a local cluster (optional)
make cluster-single-node

# 2. Install k8s-chaos with Helm
helm install k8s-chaos charts/k8s-chaos -n k8s-chaos-system --create-namespace

# 3. Try Lab 01
cd labs/01-getting-started
make setup
kubectl apply -f experiments/01-simple-pod-kill.yaml
```

## 📋 Prerequisites

- Kubernetes cluster (1.24+)
- kubectl configured to access your cluster
- Go 1.24.5+ (for development)
- Docker (for building images)
- Kind or Minikube (for local testing)

## 🛠️ Installation

### Helm (Recommended)

The easiest way to install k8s-chaos is using Helm:

```bash
# Install from local chart
helm install k8s-chaos charts/k8s-chaos \
  --namespace k8s-chaos-system \
  --create-namespace

# Verify installation
kubectl get pods -n k8s-chaos-system
```

**Custom Configuration:**
```bash
# Development setup
helm install k8s-chaos charts/k8s-chaos \
  -n k8s-chaos-system --create-namespace \
  --set controller.logLevel=debug \
  --set history.retentionLimit=50

# Production setup with cert-manager
helm install k8s-chaos charts/k8s-chaos \
  -n k8s-chaos-system --create-namespace \
  --set webhook.certificate.certManager=true \
  --set metrics.serviceMonitor.enabled=true
```

See [Helm Chart Documentation](charts/k8s-chaos/README.md) for all configuration options.

### Manual Installation (Alternative)

If you prefer to install manually:

```bash
# Install CRDs
make install

# Deploy controller
make deploy IMG=ghcr.io/neogan74/k8s-chaos:latest
```

## 📝 Usage

### CLI Tool

k8s-chaos includes a powerful command-line tool for managing and monitoring chaos experiments:

```bash
# Build and install the CLI
make build-cli
make install-cli

# List all experiments
k8s-chaos list

# View experiment details
k8s-chaos describe nginx-chaos-demo -n chaos-testing

# Show statistics
k8s-chaos stats

# Show top experiments by metrics
k8s-chaos top
```

See the [CLI documentation](docs/CLI.md) for complete usage details.

### Create a ChaosExperiment

```yaml
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: nginx-chaos
  namespace: default
spec:
  action: pod-kill        # Action to perform
  namespace: production   # Target namespace
  selector:               # Label selector for targets
    app: nginx
  count: 2               # Number of pods to affect (default: 1)
```

Apply the experiment:

```bash
kubectl apply -f config/samples/chaos_v1alpha1_chaosexperiment.yaml
```

### Monitor Experiment Status

```bash
# List experiments
kubectl get chaosexperiments

# Get detailed status
kubectl describe chaosexperiment nginx-chaos

# Watch status updates
kubectl get chaosexperiment nginx-chaos -w
```

### Delete Experiment

```bash
kubectl delete chaosexperiment nginx-chaos
```

## 🔧 Development

### Project Structure

```
.
├── api/v1alpha1/          # API types and CRD definitions
├── internal/controller/    # Reconciliation logic
├── config/                # Kustomize deployment manifests
├── cmd/main.go            # Controller entrypoint
└── hack/                  # Build scripts and tools
```

### Local Development

```bash
# Clone repository
git clone https://github.com/neogan74/k8s-chaos.git
cd k8s-chaos

# Install dependencies
go mod download

# Generate code after API changes
make generate manifests

# Run locally against cluster
make run

# Run tests
make test

# Run linter
make lint
```

### Testing

```bash
# Unit tests with coverage
make test

# E2E tests (creates Kind cluster)
make test-e2e

# Specific test package
go test ./internal/controller/... -v
```

### Building

```bash
# Build binary
make build

# Build container image
make docker-build IMG=myrepo/k8s-chaos:tag

# Push to registry
make docker-push IMG=myrepo/k8s-chaos:tag
```

## 🎯 ChaosExperiment Specification

### Spec Fields

| Field | Type | Description | Required | Default |
|-------|------|-------------|----------|---------|
| `action` | string | Chaos action to perform (`pod-kill`, `pod-delay`, `node-drain`) | Yes | - |
| `namespace` | string | Target namespace for experiments | Yes | - |
| `selector` | map[string]string | Label selector for target resources | Yes | - |
| `count` | int | Number of resources to affect (1-100) | No | 1 |
| `duration` | string | Duration for time-based actions (e.g., "30s", "5m") | No | - |

### Status Fields

| Field | Type | Description |
|-------|------|-------------|
| `lastRunTime` | Time | Timestamp of last execution |
| `message` | string | Human-readable status message |
| `phase` | string | Current phase (`Pending`, `Running`, `Completed`, `Failed`) |

## 🔒 Security Considerations

- **RBAC**: The controller requires specific permissions to manage pods and other resources
- **Namespace Isolation**: Experiments are namespace-scoped by design
- **Validation**: All inputs are validated to prevent malicious configurations
- **Audit**: All chaos actions are logged for audit purposes

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for detailed information on:

- **Code of Conduct**: Standards for community interaction
- **Development Setup**: Setting up your environment
- **Contribution Process**: How to submit changes
- **Code Standards**: Coding conventions and best practices
- **Testing Requirements**: Writing and running tests
- **Documentation Guidelines**: Updating documentation

### Quick Start for Contributors

```bash
# 1. Fork and clone
git clone https://github.com/YOUR_USERNAME/k8s-chaos.git
cd k8s-chaos

# 2. Set up development environment
make dev-setup

# 3. Create a branch
git checkout -b feature/your-feature

# 4. Make changes, test, and commit
make test lint
git commit -m "feat: your feature description"

# 5. Push and create PR
git push origin feature/your-feature
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for complete guidelines.

## 📚 Documentation

### Getting Started
- **[Quick Start](docs/QUICKSTART.md)** - Get running in 5 minutes with video demo guides
- **[Installation Guide](docs/INSTALLATION.md)** - Complete installation for all environments
- **[Getting Started Tutorial](docs/GETTING-STARTED.md)** - First experiment walkthrough
- **[Hands-on Labs](labs/README.md)** - Interactive learning tutorials

### User Guides
- **[Best Practices](docs/BEST-PRACTICES.md)** - Safety-first principles and progressive adoption
- **[Real-World Scenarios](docs/SCENARIOS.md)** - 13 ready-to-use examples
- **[Troubleshooting](docs/TROUBLESHOOTING.md)** - Common issues and solutions
- **[CLI Tool](docs/CLI.md)** - Command-line interface documentation

### Technical Reference
- **[Architecture Overview](docs/ARCHITECTURE.md)** - System design and components
- **[API Reference](docs/API.md)** - Complete CRD specification
- **[Metrics Guide](docs/METRICS.md)** - Prometheus metrics and monitoring
- **[Grafana Dashboards](docs/GRAFANA.md)** - Dashboard setup and usage
- **[Experiment History](docs/HISTORY.md)** - Audit logging and history tracking

### Contributing
- **[Contributing Guide](CONTRIBUTING.md)** - How to contribute to k8s-chaos
- **[Development Guide](docs/DEVELOPMENT.md)** - Local development setup
- **[Roadmap](ROADMAP.md)** - Future development plans

## 📊 Comparison with Other Solutions

| Feature | k8s-chaos | Chaos Mesh | Litmus Chaos |
|---------|-----------|------------|--------------|
| Lightweight | ✅ | ❌ | ❌ |
| Simple CRDs | ✅ | ❌ | ❌ |
| Pod Chaos | ✅ | ✅ | ✅ |
| Node Chaos | ✅ | ✅ | ✅ |
| Network Chaos | 🚧 Planned | ✅ | ✅ |
| Scheduling | ✅ Cron | ✅ | ✅ |
| Safety Features | ✅ Comprehensive | ✅ | ✅ |
| Metrics & Dashboards | ✅ | ✅ | ✅ |
| Audit History | ✅ | ✅ | ✅ |
| UI Dashboard | 🚧 Planned | ✅ | ✅ |
| Learning Curve | Easy | Moderate | Moderate |

**k8s-chaos** excels at being lightweight, simple to deploy, and production-ready with comprehensive safety features while maintaining an easy learning curve.

## 📄 License

Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

## 🙏 Acknowledgments

- Built with [Kubebuilder](https://kubebuilder.io/)
- Inspired by [Chaos Mesh](https://chaos-mesh.org/) and [Litmus Chaos](https://litmuschaos.io/)
- Thanks to the Kubernetes SIG API Machinery community