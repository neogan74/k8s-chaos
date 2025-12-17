# K8s Chaos Engineering Labs

This directory contains hands-on labs for learning and practicing chaos engineering with k8s-chaos operator.

## Lab Structure

Each lab is in its own directory with:
- `README.md` - Lab instructions and objectives
- `setup/` - YAML manifests and scripts
- `experiments/` - ChaosExperiment CRDs
- `solutions/` - Example solutions (if applicable)

## Available Labs

### 1. **Getting Started** (`01-getting-started/`)
- Install k8s-chaos operator
- Create your first chaos experiment
- Understand CRD structure
- Monitor experiment execution

### 2. **Pod Chaos Basics** (`02-pod-chaos-basics/`)
- Pod kill experiments
- Network delay injection
- CPU stress testing
- Pod failure scenarios

### 3. **Safety Features** (`03-safety-features/`)
- Dry-run mode
- Maximum percentage limits
- Production namespace protection
- Exclusion labels

### 4. **Node Chaos** (`04-node-chaos/`)
- Node drain experiments
- Auto-uncordon behavior
- Multi-node scenarios

### 5. **Scheduling & Duration** (`05-scheduling-duration/`)
- Cron-based scheduling
- Experiment duration control
- Automated chaos testing

### 6. **Retry Logic** (`06-retry-logic/`)
- Retry configuration
- Exponential vs fixed backoff
- Failure handling

### 7. **Observability** (`07-observability/`)
- Prometheus metrics
- Grafana dashboards
- Experiment history tracking

### 8. **Advanced Scenarios** (`08-advanced-scenarios/`)
- Multi-experiment orchestration
- Complex selectors
- Production-like testing

## Prerequisites

Before starting the labs, ensure you have:

```bash
# Required tools
- kubectl (v1.24+)
- kind or minikube (for local clusters)
- docker
- make

# Verify installation
kubectl version --client
kind version  # or: minikube version
docker version
make --version
```

## Quick Start

```bash
# Navigate to labs directory
cd labs

# Option 1: Full setup in one command (recommended)
make all                    # Creates cluster + installs CRDs + deploys operator

# Option 2: Step by step
make cluster-single-node    # Create single-node Kind cluster
make install                # Install CRDs
make deploy                 # Deploy operator

# Check status
make status

# Start with Lab 01
cd 01-getting-started
make setup
cat README.md
```

For node chaos experiments (Lab 04), use multi-node cluster:
```bash
make all-multi              # Creates 3-node cluster + full setup
```

## Lab Setup Commands

```bash
# Infrastructure
make cluster-single-node    # Create 1-node Kind cluster
make cluster-multi-node     # Create 3-node Kind cluster
make cluster-delete         # Delete Kind cluster

# Operator
make install                # Install CRDs
make deploy                 # Deploy operator
make undeploy               # Remove operator
make uninstall              # Remove CRDs

# Lab-specific
cd labs/<lab-name>
make setup                  # Setup lab environment
make teardown               # Cleanup lab resources
```

## Learning Path

**Beginner**: Labs 01 → 02 → 03
**Intermediate**: Labs 04 → 05 → 06 → 07
**Advanced**: Lab 08

## Lab Completion Checklist

Each lab includes a checklist of objectives. Mark them as you complete:
- [ ] Understand the concept
- [ ] Complete hands-on exercises
- [ ] Review results
- [ ] Clean up resources

## Getting Help

- Check lab-specific README for detailed instructions
- Review main project docs in `/docs`
- Open GitHub issues for questions

## Contributing Labs

Want to contribute a new lab? See `CONTRIBUTING.md` for guidelines.