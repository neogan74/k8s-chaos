# k8s-chaos: Kubernetes Chaos Engineering Operator

A lightweight, extensible Kubernetes Chaos Engineering operator built with Kubebuilder v4. This operator provides controlled chaos testing capabilities through Custom Resource Definitions (CRDs) to help identify weaknesses and improve the resilience of your Kubernetes applications.

## 🚀 Features

### Current (MVP)
- **Pod Chaos**: Randomly delete pods matching specific selectors
- **Flexible Targeting**: Use label selectors to target specific workloads
- **Status Tracking**: Monitor experiment execution through CRD status
- **Validation**: Built-in CRD validation for safe chaos experiments
- **RBAC**: Fine-grained permissions for chaos operations

### Planned
- **Pod Delay**: Introduce network latency to pods
- **Node Drain**: Simulate node failures
- **Network Chaos**: Packet loss, bandwidth limitations
- **Scheduling**: Cron-based experiment execution
- **Metrics**: Prometheus metrics for experiment tracking

## 📋 Prerequisites

- Kubernetes cluster (1.24+)
- kubectl configured to access your cluster
- Go 1.24.5+ (for development)
- Docker (for building images)

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

## 📊 Comparison with Other Solutions

| Feature | k8s-chaos | Chaos Mesh | Litmus Chaos |
|---------|-----------|------------|--------------|
| Lightweight | ✅ | ❌ | ❌ |
| Simple CRDs | ✅ | ❌ | ❌ |
| Pod Chaos | ✅ | ✅ | ✅ |
| Network Chaos | 🚧 | ✅ | ✅ |
| UI Dashboard | ❌ | ✅ | ✅ |
| Scheduling | 🚧 | ✅ | ✅ |
| Multi-tenancy | ✅ | ✅ | ✅ |

## 🐛 Known Issues

- Pod-delay action is not yet fully implemented
- Network chaos features are planned for future releases

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