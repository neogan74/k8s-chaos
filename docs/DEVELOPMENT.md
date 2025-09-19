# Local Development Guide

This guide helps you set up a complete local development environment for k8s-chaos.

## Quick Start

```bash
# Set up everything at once
make dev-setup

# Run the controller locally
make dev-run

# In another terminal, run chaos experiments
make demo-run
make demo-watch
```

## Prerequisites

### Required Tools
- **Go 1.24+**: For building the operator
- **Docker**: For container operations
- **kubectl**: For Kubernetes cluster access
- **Kind**: For local Kubernetes clusters

### Install Dependencies
```bash
# Install Kind
go install sigs.k8s.io/kind@latest

# Install kubectl (if not already installed)
# macOS
brew install kubectl

# Linux
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl

# Windows
curl.exe -LO "https://dl.k8s.io/release/v1.30.0/bin/windows/amd64/kubectl.exe"
```

## Development Workflow

### 1. Initial Setup

```bash
# Clone the repository
git clone https://github.com/neogan74/k8s-chaos.git
cd k8s-chaos

# Set up complete development environment
make dev-setup
```

This command will:
- ✅ Check and install development dependencies
- ✅ Create a Kind cluster named `k8s-chaos-dev`
- ✅ Install CRDs to the cluster
- ✅ Deploy demo environment with nginx pods

### 2. Development Loop

```bash
# Make code changes to controller or API types
# Then regenerate manifests
make manifests generate

# Run tests
make test

# Run linter
make lint

# Run controller locally
make dev-run
```

### 3. Testing Changes

```bash
# Check environment status
make dev-status

# Run a chaos experiment
make demo-run

# Watch pods being terminated and recreated
make demo-watch

# Check experiment status
make demo-status

# Stop experiments
make demo-stop
```

### 4. Development Commands

| Command | Description |
|---------|-------------|
| `make dev-setup` | Complete development environment setup |
| `make dev-dependencies` | Install development tools |
| `make dev-cluster` | Create Kind cluster |
| `make dev-install` | Install CRDs to cluster |
| `make dev-demo` | Deploy demo environment |
| `make dev-run` | Run controller locally |
| `make dev-status` | Show environment status |
| `make dev-clean` | Clean up everything |

### 5. Demo Commands

| Command | Description |
|---------|-------------|
| `make demo-run` | Start chaos experiment |
| `make demo-watch` | Watch pods in real-time |
| `make demo-status` | Show detailed status |
| `make demo-stop` | Stop all experiments |
| `make demo-reset` | Reset demo to clean state |

## Architecture Overview

### Project Structure
```
├── api/v1alpha1/           # CRD definitions
│   ├── chaosexperiment_types.go
│   └── groupversion_info.go
├── internal/controller/    # Controller logic
│   └── chaosexperiment_controller.go
├── config/                 # Deployment manifests
│   ├── crd/               # CRD manifests
│   ├── samples/           # Example resources
│   └── default/           # Default deployment
├── cmd/                   # Main entrypoint
└── test/                  # Test files
```

### Controller Flow
1. **Watch** ChaosExperiment resources
2. **Validate** experiment configuration
3. **Select** target pods using label selectors
4. **Execute** chaos actions (currently pod-kill)
5. **Update** experiment status
6. **Requeue** for continuous operation

## Making Changes

### Adding New API Fields

1. **Edit types**: Modify `api/v1alpha1/chaosexperiment_types.go`
2. **Regenerate**: Run `make manifests generate`
3. **Update samples**: Add examples in `config/samples/`
4. **Test**: Run `make test`

### Modifying Controller Logic

1. **Edit controller**: Modify `internal/controller/chaosexperiment_controller.go`
2. **Test locally**: Run `make dev-run`
3. **Run experiments**: Use `make demo-run`
4. **Verify**: Check logs and status

### Adding New Chaos Actions

1. **Update types**: Add new action to validation
2. **Implement logic**: Add action handling in controller
3. **Create samples**: Add example CRDs
4. **Update docs**: Document new action

## Testing

### Unit Tests
```bash
# Run all tests
make test

# Run specific package tests
go test ./internal/controller -v

# Run with coverage
make test && go tool cover -html=cover.out
```

### Integration Tests
```bash
# Run e2e tests (creates temporary cluster)
make test-e2e

# Manual integration testing
make dev-setup
make dev-run  # In one terminal
make demo-run # In another terminal
```

### Testing Different Scenarios

```bash
# Test with different selectors
kubectl apply -f config/samples/chaos_v1alpha1_chaosexperiment_multiple.yaml

# Test with StatefulSets
kubectl apply -f config/samples/chaos_v1alpha1_chaosexperiment_stateful.yaml

# Create custom test scenarios
kubectl create namespace test-ns
kubectl run test-pod --image=nginx -n test-ns --labels="app=test"
```

## Debugging

### Controller Logs
```bash
# When running locally
make dev-run  # Shows logs in console

# When deployed to cluster
kubectl logs -n k8s-chaos-system deployment/k8s-chaos-controller-manager -f
```

### Common Issues

#### "No pods found for selector"
```bash
# Check if pods exist with the selector
kubectl get pods -l app=nginx -n chaos-demo

# Verify selector syntax in CRD
kubectl describe chaosexperiment nginx-chaos-demo -n chaos-demo
```

#### "Permission denied"
```bash
# Check RBAC permissions
kubectl auth can-i delete pods --as=system:serviceaccount:k8s-chaos-system:k8s-chaos-controller-manager

# Verify service account
kubectl get serviceaccount -n k8s-chaos-system
```

#### Controller not starting
```bash
# Check cluster connection
kubectl cluster-info

# Verify CRDs are installed
kubectl get crd chaosexperiments.chaos.gushchin.dev

# Check for syntax errors
make manifests generate fmt vet
```

## Best Practices

### Code Quality
- Always run `make fmt vet lint` before committing
- Add unit tests for new functionality
- Update documentation for API changes
- Use structured logging with context

### Safety
- Test in isolated namespaces
- Use small pod counts initially
- Monitor cluster resources
- Have rollback procedures ready

### Git Workflow
```bash
# Create feature branch
git checkout -b feature/new-chaos-action

# Make changes and test
make dev-setup
make test
make lint

# Commit changes
git add .
git commit -m "feat: add new chaos action"

# Push and create PR
git push origin feature/new-chaos-action
```

## Advanced Topics

### Custom Kind Configuration
Create `kind-config.yaml`:
```yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
- role: worker
- role: worker
  extraMounts:
  - hostPath: /tmp
    containerPath: /tmp
```

Use with: `kind create cluster --config kind-config.yaml --name custom-cluster`

### Multiple Controller Instances
```bash
# Test leader election
make dev-run  # Terminal 1
make dev-run  # Terminal 2 (should wait as follower)
```

### Performance Testing
```bash
# Create many target pods
kubectl create deployment big-test --image=nginx --replicas=50 -n chaos-demo
kubectl label deployment big-test app=nginx -n chaos-demo

# Test with high count
# Edit config/samples/chaos_v1alpha1_chaosexperiment_demo.yaml
# Set count: 25
make demo-run
```

## Cleanup

```bash
# Clean up development environment
make dev-clean

# Or manually
kind delete cluster --name k8s-chaos-dev
```

## Getting Help

- **Issues**: Check [troubleshooting guide](../config/samples/README.md)
- **API Reference**: See `api/v1alpha1/chaosexperiment_types.go`
- **Examples**: Browse `config/samples/` directory
- **Controller Logic**: Read `internal/controller/chaosexperiment_controller.go`