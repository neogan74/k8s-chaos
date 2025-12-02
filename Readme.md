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

# 2. Install k8s-chaos
make install deploy IMG=ghcr.io/neogan74/k8s-chaos:latest

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

### Install CRDs

```bash
make install
```

### Deploy Controller

```bash
# Using pre-built image
make deploy IMG=ghcr.io/neogan74/k8s-chaos:latest

# Or build and deploy your own
make docker-build docker-push IMG=<your-registry>/k8s-chaos:tag
make deploy IMG=<your-registry>/k8s-chaos:tag
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

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-chaos`)
3. Commit your changes (`git commit -m 'Add amazing chaos action'`)
4. Push to the branch (`git push origin feature/amazing-chaos`)
5. Open a Pull Request

### Development Workflow

1. **API Changes**: Modify types in `api/v1alpha1/`, then run `make manifests generate`
2. **Controller Logic**: Edit `internal/controller/chaosexperiment_controller.go`
3. **Testing**: Run `make test lint` before committing
4. **Documentation**: Update README and API docs as needed

## 📚 Documentation

- **[Getting Started](docs/GETTING-STARTED.md)** - Complete installation and first experiment tutorial
- **[Best Practices](docs/BEST-PRACTICES.md)** - Safety-first principles and progressive adoption
- **[Troubleshooting](docs/TROUBLESHOOTING.md)** - Common issues and solutions
- **[Real-World Scenarios](docs/SCENARIOS.md)** - 13 ready-to-use examples
- **[API Reference](docs/API.md)** - Complete CRD specification
- **[CLI Tool](docs/CLI.md)** - Command-line interface documentation
- **[Metrics Guide](docs/METRICS.md)** - Prometheus metrics and monitoring
- **[Grafana Dashboards](docs/GRAFANA.md)** - Dashboard setup and usage
- **[Experiment History](docs/HISTORY.md)** - Audit logging and history tracking
- **[Hands-on Labs](labs/README.md)** - Interactive learning tutorials
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