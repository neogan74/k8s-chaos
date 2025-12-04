# Getting Started with k8s-chaos

Welcome to k8s-chaos! This guide will walk you through everything you need to know to start practicing chaos engineering in your Kubernetes clusters.

## Table of Contents

- [What is k8s-chaos?](#what-is-k8s-chaos)
- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Your First Chaos Experiment](#your-first-chaos-experiment)
- [Understanding the CRD](#understanding-the-crd)
- [Next Steps](#next-steps)

---

## What is k8s-chaos?

k8s-chaos is a lightweight Kubernetes operator for chaos engineering. It helps you test your application's resilience by injecting controlled failures into your cluster.

**Key Features:**
- ✅ Pod chaos (kill, delay, CPU/memory stress, failure)
- ✅ Node chaos (drain, cordon)
- ✅ Safety features (dry-run, exclusions, production protection)
- ✅ Scheduling & duration control
- ✅ Retry logic with backoff strategies
- ✅ Prometheus metrics & Grafana dashboards
- ✅ Experiment history & audit logging

---

## Prerequisites

Before you begin, ensure you have:

### Required Tools
```bash
# Kubernetes cluster (v1.24+)
kubectl version --client

# Container runtime (Docker or Podman)
docker version
# or
podman version

# Make (for easier commands)
make --version

# Git (to clone the repository)
git --version
```

### Optional Tools
```bash
# Kind (for local testing)
kind version

# Helm (recommended for installation)
helm version
```

### Kubernetes Access
You need:
- A running Kubernetes cluster
- `kubectl` configured to access it
- Cluster-admin privileges (for CRD installation)

**Don't have a cluster?** Create one locally:
```bash
# Using Kind
kind create cluster --name chaos-testing

# Using Minikube
minikube start --cpus 4 --memory 8192
```

---

## Installation

You can install k8s-chaos using either Helm (recommended) or manually.

### Option A: Helm Installation (Recommended)

The easiest way to get started:

```bash
# Clone repository
git clone https://github.com/neogan74/k8s-chaos.git
cd k8s-chaos

# Install with Helm
helm install k8s-chaos charts/k8s-chaos \
  --namespace k8s-chaos-system \
  --create-namespace

# Verify installation
kubectl get pods -n k8s-chaos-system

# Wait for the operator to be ready
kubectl wait --for=condition=ready pod \
  -l control-plane=controller-manager \
  -n k8s-chaos-system \
  --timeout=120s
```

**That's it!** Skip to [Your First Chaos Experiment](#your-first-chaos-experiment).

**Custom Configuration:**
```bash
# Development setup
helm install k8s-chaos charts/k8s-chaos \
  -n k8s-chaos-system --create-namespace \
  --set controller.logLevel=debug

# Production setup with cert-manager
helm install k8s-chaos charts/k8s-chaos \
  -n k8s-chaos-system --create-namespace \
  --set webhook.certificate.certManager=true \
  --set metrics.serviceMonitor.enabled=true
```

See [Helm Chart Documentation](../charts/k8s-chaos/README.md) for all options.

---

### Option B: Manual Installation

For more control over the installation:

#### Step 1: Clone the Repository

```bash
git clone https://github.com/neogan74/k8s-chaos.git
cd k8s-chaos
```

#### Step 2: Install CRDs

Install the Custom Resource Definitions:

```bash
make install
```

Verify CRDs are installed:
```bash
kubectl get crds | grep chaos.gushchin.dev
```

You should see:
```
chaosexperimenthistories.chaos.gushchin.dev
chaosexperiments.chaos.gushchin.dev
```

### Step 3: Build and Deploy the Operator

Build the operator image:
```bash
make docker-build IMG=k8s-chaos-controller:latest
```

For Kind clusters, load the image:
```bash
kind load docker-image k8s-chaos-controller:latest --name chaos-testing
```

Deploy the operator:
```bash
make deploy IMG=k8s-chaos-controller:latest
```

### Step 4: Verify Installation

Check that the operator is running:
```bash
kubectl get pods -n k8s-chaos-system

# Wait for the pod to be Ready
kubectl wait --for=condition=ready pod \
  -l control-plane=controller-manager \
  -n k8s-chaos-system \
  --timeout=120s
```

Check the operator logs:
```bash
kubectl logs -n k8s-chaos-system \
  deployment/k8s-chaos-controller-manager \
  -f
```

---

## Your First Chaos Experiment

Let's run a simple chaos experiment to understand how k8s-chaos works.

### Step 1: Deploy a Test Application

Create a namespace and deploy nginx:

```bash
# Create namespace
kubectl create namespace chaos-demo

# Deploy nginx with 5 replicas
cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx
  namespace: chaos-demo
spec:
  replicas: 5
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx:1.25-alpine
        ports:
        - containerPort: 80
        resources:
          requests:
            cpu: 100m
            memory: 128Mi
          limits:
            cpu: 200m
            memory: 256Mi
EOF
```

Wait for pods to be ready:
```bash
kubectl wait --for=condition=ready pod \
  -l app=nginx \
  -n chaos-demo \
  --timeout=60s

kubectl get pods -n chaos-demo
```

### Step 2: Create Your First Experiment

Let's start with a **dry-run** to see what would happen:

```bash
cat <<EOF | kubectl apply -f -
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: my-first-experiment
  namespace: chaos-demo
spec:
  # Action to perform
  action: pod-kill

  # Target namespace
  namespace: chaos-demo

  # Pod selector
  selector:
    app: nginx

  # Number of pods to affect
  count: 1

  # DRY RUN - preview only!
  dryRun: true
EOF
```

Check the result:
```bash
kubectl get chaosexperiment my-first-experiment -n chaos-demo

# View the dry-run message
kubectl get chaosexperiment my-first-experiment -n chaos-demo \
  -o jsonpath='{.status.message}'
```

You'll see output like:
```
DRY RUN: Would delete 1 pod(s): [nginx-7d5c8f6d9-abc123]
```

### Step 3: Run a Real Experiment

Now let's run it for real! Remove the dry-run flag:

```bash
cat <<EOF | kubectl apply -f -
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: my-first-experiment
  namespace: chaos-demo
spec:
  action: pod-kill
  namespace: chaos-demo
  selector:
    app: nginx
  count: 1

  # Run for 3 minutes then auto-stop
  experimentDuration: "3m"
EOF
```

### Step 4: Watch the Chaos in Action

In one terminal, watch the pods:
```bash
kubectl get pods -n chaos-demo -w
```

In another terminal, watch the experiment:
```bash
watch -n 2 'kubectl get chaosexperiment my-first-experiment -n chaos-demo -o wide'
```

You should see:
1. **Pods being killed**: One pod terminates
2. **Deployment recreating pods**: New pod spins up
3. **Experiment repeating**: Every ~60 seconds, another pod is killed
4. **Auto-stop after 3 minutes**: Experiment completes

### Step 5: Examine the Results

Check experiment status:
```bash
kubectl describe chaosexperiment my-first-experiment -n chaos-demo
```

View experiment history (if history is enabled):
```bash
kubectl get chaosexperimenthistories -n k8s-chaos-system \
  -l chaos.gushchin.dev/experiment=my-first-experiment
```

### Step 6: Clean Up

Stop the experiment:
```bash
kubectl delete chaosexperiment my-first-experiment -n chaos-demo
```

Clean up the demo:
```bash
kubectl delete namespace chaos-demo
```

---

## Understanding the CRD

Let's break down the `ChaosExperiment` CRD:

### Basic Structure

```yaml
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: my-experiment           # Experiment name
  namespace: my-namespace        # Where to create the resource
spec:
  # REQUIRED FIELDS
  action: pod-kill               # What chaos to inject
  namespace: target-namespace    # Where to inject chaos
  selector:                      # Which pods to target
    app: my-app

  # OPTIONAL FIELDS
  count: 2                       # How many resources to affect (default: 1)
  dryRun: false                  # Preview mode (default: false)
  experimentDuration: "10m"      # Auto-stop after duration
  schedule: "*/30 * * * *"       # Cron schedule for recurring chaos

  # RETRY CONFIGURATION
  maxRetries: 3                  # Max retry attempts (default: 3)
  retryDelay: "30s"              # Initial retry delay (default: 30s)
  retryBackoff: "exponential"    # Backoff strategy: exponential|fixed

  # SAFETY FEATURES
  maxPercentage: 30              # Max % of pods to affect (1-100)
  allowProduction: false         # Required for production namespaces

  # ACTION-SPECIFIC OPTIONS
  duration: "5s"                 # For pod-delay, pod-cpu-stress, pod-memory-stress
  cpuLoad: 80                    # For pod-cpu-stress (1-100%)
  cpuWorkers: 2                  # For pod-cpu-stress (default: 1)
  memorySize: "256M"             # For pod-memory-stress
  memoryWorkers: 1               # For pod-memory-stress (default: 1)
```

### Available Actions

| Action | Description | Required Fields |
|--------|-------------|----------------|
| `pod-kill` | Delete pods | - |
| `pod-delay` | Inject network latency | `duration` |
| `pod-cpu-stress` | Stress CPU | `duration`, `cpuLoad` |
| `pod-memory-stress` | Stress memory | `duration`, `memorySize` |
| `pod-failure` | Kill main process | - |
| `node-drain` | Drain nodes | - |

### Status Fields

After creating an experiment, check its status:

```yaml
status:
  phase: Running                           # Pending, Running, Completed, Failed
  message: Successfully killed 1 pod(s)    # Human-readable message
  lastRunTime: "2025-12-02T10:00:00Z"     # Last execution time
  startTime: "2025-12-02T09:55:00Z"       # When experiment started
  completedAt: "2025-12-02T10:05:00Z"     # When it completed

  # Retry tracking
  retryCount: 0                            # Current retry attempt
  lastError: ""                            # Last error message
  nextRetryTime: null                      # When next retry will occur

  # Scheduling
  lastScheduledTime: "2025-12-02T10:00:00Z"   # Last scheduled run
  nextScheduledTime: "2025-12-02T10:30:00Z"   # Next scheduled run

  # Node drain specific
  cordonedNodes: ["worker-1", "worker-2"]  # Nodes we cordoned
```

---

## Next Steps

Congratulations! You've successfully:
- ✅ Installed k8s-chaos
- ✅ Created your first chaos experiment
- ✅ Used dry-run mode for safety
- ✅ Monitored chaos in action
- ✅ Understood the CRD structure

### Continue Learning

1. **Read Best Practices**: [docs/BEST-PRACTICES.md](BEST-PRACTICES.md)
2. **Explore All Actions**: [docs/API.md](API.md)
3. **Set Up Metrics**: [docs/METRICS.md](METRICS.md)
4. **Configure Grafana**: [docs/GRAFANA.md](GRAFANA.md)
5. **Try Hands-on Labs**: [labs/README.md](../labs/README.md)

### Real-World Scenarios

Check out [docs/SCENARIOS.md](SCENARIOS.md) for examples:
- Testing deployment resilience
- Network failure simulation
- Node failure handling
- Resource exhaustion testing

### Get Help

- **Troubleshooting**: [docs/TROUBLESHOOTING.md](TROUBLESHOOTING.md)
- **GitHub Issues**: https://github.com/neogan74/k8s-chaos/issues
- **Discussions**: https://github.com/neogan74/k8s-chaos/discussions

---

## Quick Reference Commands

```bash
# Installation
make install                    # Install CRDs
make deploy IMG=<image>         # Deploy operator
make uninstall                  # Remove CRDs
make undeploy                   # Remove operator

# Development
make build                      # Build operator binary
make docker-build IMG=<image>   # Build container image
make run                        # Run locally
make test                       # Run tests

# Labs
make cluster-single-node        # Create test cluster
make labs-setup                 # Complete lab setup
make labs-teardown              # Clean up labs

# Experiments
kubectl get chaosexperiments -A                    # List all experiments
kubectl describe chaosexperiment <name> -n <ns>    # Experiment details
kubectl delete chaosexperiment <name> -n <ns>      # Stop experiment

# History (if enabled)
kubectl get chaosexperimenthistories -n k8s-chaos-system
```

---

## Additional Resources

- **Main README**: [../Readme.md](../Readme.md)
- **API Documentation**: [API.md](API.md)
- **CLI Tool**: [CLI.md](CLI.md)
- **Metrics Guide**: [METRICS.md](METRICS.md)
- **Development Guide**: [DEVELOPMENT.md](DEVELOPMENT.md)

Happy chaos engineering! 🚀