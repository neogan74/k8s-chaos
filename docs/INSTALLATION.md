# Installation Guide

Complete installation guide for k8s-chaos on various environments.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [Installation Methods](#installation-methods)
  - [Helm (Recommended)](#helm-recommended)
  - [Manual Installation](#manual-installation)
  - [GitOps Deployment](#gitops-deployment)
  - [Local Development](#local-development)
- [Post-Installation](#post-installation)
- [Upgrading](#upgrading)
- [Uninstalling](#uninstalling)
- [Troubleshooting](#troubleshooting)

## Prerequisites

### Kubernetes Cluster

- **Kubernetes version**: 1.24+
- **kubectl**: Configured to access your cluster
- **Cluster permissions**: Cluster-admin or equivalent to create CRDs and ClusterRoles

### Optional Dependencies

- **Helm 3.8+**: For Helm installation method (recommended)
- **cert-manager**: For automatic webhook certificate management (production)
- **Prometheus**: For metrics collection (optional but recommended)
- **Grafana**: For dashboard visualization (optional)

### Local Development (Additional)

- **Go 1.24.5+**: For building from source
- **Docker**: For building container images
- **Kind** or **Minikube**: For local Kubernetes clusters

## Quick Start

The fastest way to get k8s-chaos running:

```bash
# Using Helm (recommended)
helm install k8s-chaos oci://ghcr.io/neogan74/k8s-chaos/charts/k8s-chaos \
  --namespace k8s-chaos-system \
  --create-namespace

# Verify installation
kubectl get pods -n k8s-chaos-system
```

Or for local development:

```bash
# Create local cluster and install everything
make dev-setup
make dev-run
```

## Installation Methods

### Helm (Recommended)

Helm is the recommended installation method for production environments.

#### 1. Basic Installation

```bash
# Install from OCI registry
helm install k8s-chaos oci://ghcr.io/neogan74/k8s-chaos/charts/k8s-chaos \
  --namespace k8s-chaos-system \
  --create-namespace
```

Or from local chart:

```bash
# Clone repository
git clone https://github.com/neogan74/k8s-chaos.git
cd k8s-chaos

# Install from local chart
helm install k8s-chaos charts/k8s-chaos \
  --namespace k8s-chaos-system \
  --create-namespace
```

#### 2. Development Environment

For development with debug logging and reduced resources:

```bash
helm install k8s-chaos charts/k8s-chaos \
  -n k8s-chaos-system --create-namespace \
  --set controller.logLevel=debug \
  --set controller.resources.limits.cpu=200m \
  --set controller.resources.limits.memory=256Mi \
  --set metrics.secure=false \
  --set history.retentionLimit=50
```

Or using a values file:

```yaml
# dev-values.yaml
controller:
  logLevel: debug
  resources:
    limits:
      cpu: 200m
      memory: 256Mi
    requests:
      cpu: 50m
      memory: 64Mi

metrics:
  enabled: true
  secure: false

history:
  enabled: true
  retentionLimit: 50

webhook:
  enabled: true
```

```bash
helm install k8s-chaos charts/k8s-chaos \
  -n k8s-chaos-system --create-namespace \
  -f dev-values.yaml
```

#### 3. Production Environment

For production with cert-manager and Prometheus:

```yaml
# prod-values.yaml
controller:
  logLevel: info
  replicaCount: 1
  resources:
    limits:
      cpu: 1000m
      memory: 1Gi
    requests:
      cpu: 500m
      memory: 512Mi
  affinity:
    podAntiAffinity:
      preferredDuringSchedulingIgnoredDuringExecution:
      - weight: 100
        podAffinityTerm:
          labelSelector:
            matchLabels:
              control-plane: controller-manager
          topologyKey: kubernetes.io/hostname

metrics:
  enabled: true
  secure: true
  serviceMonitor:
    enabled: true
    interval: 30s
    labels:
      prometheus: kube-prometheus

history:
  enabled: true
  retentionLimit: 200

webhook:
  enabled: true
  certificate:
    certManager: true
    generate: false
```

```bash
# Install cert-manager first (if not already installed)
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml

# Install k8s-chaos
helm install k8s-chaos charts/k8s-chaos \
  -n k8s-chaos-system --create-namespace \
  -f prod-values.yaml
```

#### 4. Custom Configuration Options

Key Helm values you can customize:

| Parameter | Description | Default |
|-----------|-------------|---------|
| `image.repository` | Controller image repository | `ghcr.io/neogan74/k8s-chaos` |
| `image.tag` | Controller image tag | Chart appVersion |
| `controller.replicaCount` | Number of replicas | `1` |
| `controller.logLevel` | Log level (debug/info/warn/error) | `info` |
| `metrics.enabled` | Enable Prometheus metrics | `true` |
| `metrics.secure` | Use HTTPS for metrics | `true` |
| `metrics.serviceMonitor.enabled` | Create ServiceMonitor | `false` |
| `history.enabled` | Enable experiment history | `true` |
| `history.namespace` | History storage namespace | `k8s-chaos-system` |
| `history.retentionLimit` | Max records per experiment | `100` |
| `webhook.enabled` | Enable admission webhook | `true` |
| `webhook.certificate.certManager` | Use cert-manager | `false` |
| `webhook.certificate.generate` | Auto-generate certificates | `true` |

See [charts/k8s-chaos/README.md](../charts/k8s-chaos/README.md) for complete values documentation.

### Manual Installation

For advanced users or when Helm is not available.

#### 1. Install CRDs

```bash
# Clone repository
git clone https://github.com/neogan74/k8s-chaos.git
cd k8s-chaos

# Install Custom Resource Definitions
kubectl apply -f config/crd/bases/
```

#### 2. Install RBAC Resources

```bash
# Create namespace
kubectl create namespace k8s-chaos-system

# Install RBAC (ServiceAccount, ClusterRole, ClusterRoleBinding)
kubectl apply -f config/rbac/
```

#### 3. Deploy Controller

```bash
# Deploy the controller
kubectl apply -f config/manager/

# Or build and deploy custom image
make docker-build IMG=myregistry/k8s-chaos:latest
make docker-push IMG=myregistry/k8s-chaos:latest
make deploy IMG=myregistry/k8s-chaos:latest
```

#### 4. Configure Webhooks (Optional)

If you want admission webhook validation:

```bash
# Generate certificates
./hack/generate-webhook-certs.sh

# Install webhook
kubectl apply -f config/webhook/
```

### GitOps Deployment

k8s-chaos supports GitOps workflows with ArgoCD, Flux, and Kustomize.

#### ArgoCD

```yaml
# argocd-application.yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: k8s-chaos
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://github.com/neogan74/k8s-chaos.git
    targetRevision: main
    path: deploy/argocd
  destination:
    server: https://kubernetes.default.svc
    namespace: k8s-chaos-system
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
```

```bash
kubectl apply -f argocd-application.yaml
```

See [deploy/argocd/README.md](../deploy/argocd/README.md) for detailed configuration.

#### Flux

```bash
# Add Helm repository
flux create source helm k8s-chaos \
  --url=oci://ghcr.io/neogan74/k8s-chaos/charts \
  --interval=1h

# Create HelmRelease
flux create helmrelease k8s-chaos \
  --source=HelmRepository/k8s-chaos \
  --chart=k8s-chaos \
  --target-namespace=k8s-chaos-system \
  --create-target-namespace=true \
  --values=./prod-values.yaml
```

See [deploy/flux/README.md](../deploy/flux/README.md) for detailed configuration.

#### Kustomize

```bash
# Base installation
kubectl apply -k deploy/kustomize/base

# Or with overlays
kubectl apply -k deploy/kustomize/overlays/production
```

See [deploy/kustomize/README.md](../deploy/kustomize/README.md) for overlay options.

### Local Development

For local development and testing.

#### Method 1: Automated Setup

```bash
# Clone repository
git clone https://github.com/neogan74/k8s-chaos.git
cd k8s-chaos

# Complete automated setup
make dev-setup
```

This will:
- Install development dependencies (Kind, kubectl)
- Create a Kind cluster named `k8s-chaos-dev`
- Install CRDs
- Deploy demo environment with nginx pods

Then run the controller locally:

```bash
make dev-run
```

#### Method 2: Manual Setup

```bash
# 1. Install Kind
go install sigs.k8s.io/kind@latest

# 2. Create cluster
kind create cluster --name k8s-chaos-dev

# 3. Install CRDs
make install

# 4. Run controller locally
make run
```

#### Method 3: Single-Node Cluster

For minimal resource usage:

```bash
make cluster-single-node
make install
make run
```

See [docs/DEVELOPMENT.md](DEVELOPMENT.md) for complete development guide.

## Post-Installation

### Verify Installation

```bash
# Check controller pod
kubectl get pods -n k8s-chaos-system
# Should show: k8s-chaos-controller-manager-xxxxx Running

# Check CRDs
kubectl get crds | grep chaos
# Should show:
#   chaosexperiments.chaos.gushchin.dev
#   chaosexperimenthistories.chaos.gushchin.dev

# Check webhook (if enabled)
kubectl get validatingwebhookconfigurations | grep k8s-chaos

# View controller logs
kubectl logs -n k8s-chaos-system deployment/k8s-chaos-controller-manager -f
```

### Configure Metrics (Optional)

If using Prometheus:

```bash
# Check metrics endpoint
kubectl port-forward -n k8s-chaos-system svc/k8s-chaos-metrics-service 8443:8443
curl -k https://localhost:8443/metrics
```

Deploy ServiceMonitor:

```bash
kubectl apply -f config/prometheus/monitor.yaml
```

### Install Grafana Dashboards (Optional)

```bash
# Deploy Grafana (if needed)
kubectl apply -k config/grafana/

# Import dashboards
cd docs/grafana
./import-dashboards.sh http://localhost:3000 admin:admin
```

See [docs/GRAFANA.md](GRAFANA.md) for complete setup.

### Run First Experiment

Create a test namespace and deployment:

```bash
kubectl create namespace chaos-test
kubectl create deployment nginx --image=nginx --replicas=3 -n chaos-test
kubectl label deployment nginx app=nginx -n chaos-test
```

Create a simple chaos experiment:

```yaml
# first-experiment.yaml
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: my-first-chaos
  namespace: chaos-test
spec:
  action: pod-kill
  namespace: chaos-test
  selector:
    app: nginx
  count: 1
```

```bash
kubectl apply -f first-experiment.yaml
kubectl get chaosexperiment -n chaos-test
kubectl describe chaosexperiment my-first-chaos -n chaos-test
```

See [docs/GETTING-STARTED.md](GETTING-STARTED.md) for detailed tutorial.

## Upgrading

### Helm Upgrade

```bash
# Upgrade to latest version
helm upgrade k8s-chaos charts/k8s-chaos \
  -n k8s-chaos-system

# Upgrade with new values
helm upgrade k8s-chaos charts/k8s-chaos \
  -n k8s-chaos-system \
  -f new-values.yaml

# Check upgrade status
helm list -n k8s-chaos-system
helm history k8s-chaos -n k8s-chaos-system
```

### Manual Upgrade

```bash
# Pull latest changes
git pull origin main

# Update CRDs
kubectl apply -f config/crd/bases/

# Update controller
make deploy IMG=ghcr.io/neogan74/k8s-chaos:latest
```

### Rollback

```bash
# Helm rollback
helm rollback k8s-chaos -n k8s-chaos-system

# Or rollback to specific revision
helm rollback k8s-chaos 1 -n k8s-chaos-system
```

## Uninstalling

### Helm Uninstall

```bash
# Uninstall release
helm uninstall k8s-chaos -n k8s-chaos-system

# Remove CRDs (careful: this deletes all experiments and history)
kubectl delete crd chaosexperiments.chaos.gushchin.dev
kubectl delete crd chaosexperimenthistories.chaos.gushchin.dev

# Remove namespace
kubectl delete namespace k8s-chaos-system
```

### Manual Uninstall

```bash
# Remove controller
kubectl delete -f config/manager/

# Remove RBAC
kubectl delete -f config/rbac/

# Remove CRDs (this deletes all experiments)
kubectl delete -f config/crd/bases/

# Remove namespace
kubectl delete namespace k8s-chaos-system
```

### Clean Up Experiments

Before uninstalling, you may want to clean up running experiments:

```bash
# List all experiments across namespaces
kubectl get chaosexperiments --all-namespaces

# Delete specific experiment
kubectl delete chaosexperiment <name> -n <namespace>

# Delete all experiments (be careful!)
kubectl delete chaosexperiments --all --all-namespaces
```

## Troubleshooting

### Controller Not Starting

**Symptom**: Pod stuck in CrashLoopBackOff or ImagePullBackOff

```bash
# Check pod status
kubectl get pods -n k8s-chaos-system
kubectl describe pod -n k8s-chaos-system <pod-name>

# Check logs
kubectl logs -n k8s-chaos-system <pod-name>

# Common causes:
# 1. Image not found - check image repository/tag
# 2. CRDs not installed - run: make install
# 3. RBAC issues - verify ClusterRole and bindings
```

### Webhook Failures

**Symptom**: Cannot create ChaosExperiment resources

```bash
# Check webhook configuration
kubectl get validatingwebhookconfigurations k8s-chaos-validating-webhook-configuration

# Check certificate
kubectl get secret k8s-chaos-webhook-cert -n k8s-chaos-system

# If using cert-manager
kubectl get certificate -n k8s-chaos-system
kubectl describe certificate k8s-chaos-serving-cert -n k8s-chaos-system

# Temporary workaround: disable webhook
helm upgrade k8s-chaos charts/k8s-chaos \
  -n k8s-chaos-system \
  --set webhook.enabled=false
```

### RBAC Permission Errors

**Symptom**: Errors about insufficient permissions in logs

```bash
# Check ClusterRole
kubectl get clusterrole | grep k8s-chaos
kubectl describe clusterrole k8s-chaos-manager-role

# Check ClusterRoleBinding
kubectl get clusterrolebinding | grep k8s-chaos
kubectl describe clusterrolebinding k8s-chaos-manager-rolebinding

# Test permissions
kubectl auth can-i delete pods \
  --as=system:serviceaccount:k8s-chaos-system:k8s-chaos-controller-manager
```

### Metrics Not Working

**Symptom**: Cannot access /metrics endpoint

```bash
# Check metrics service
kubectl get svc -n k8s-chaos-system k8s-chaos-metrics-service

# Port-forward and test
kubectl port-forward -n k8s-chaos-system svc/k8s-chaos-metrics-service 8443:8443
curl -k https://localhost:8443/metrics

# If using HTTP (dev mode)
curl http://localhost:8080/metrics
```

### CRD Version Conflicts

**Symptom**: Error about CRD version mismatch

```bash
# Remove old CRDs
kubectl delete crd chaosexperiments.chaos.gushchin.dev
kubectl delete crd chaosexperimenthistories.chaos.gushchin.dev

# Reinstall
make install

# Or with Helm
helm upgrade k8s-chaos charts/k8s-chaos \
  -n k8s-chaos-system \
  --force
```

### Local Development Issues

See [docs/DEVELOPMENT.md](DEVELOPMENT.md) troubleshooting section.

## Next Steps

- [Getting Started Guide](GETTING-STARTED.md) - Complete tutorial
- [Best Practices](BEST-PRACTICES.md) - Production recommendations
- [API Reference](API.md) - CRD specification
- [Real-World Scenarios](SCENARIOS.md) - Example experiments
- [Metrics Guide](METRICS.md) - Monitoring setup
- [Grafana Dashboards](GRAFANA.md) - Visualization

## Support

- Documentation: [/docs](/docs)
- Issues: [GitHub Issues](https://github.com/neogan74/k8s-chaos/issues)
- Discussions: [GitHub Discussions](https://github.com/neogan74/k8s-chaos/discussions)