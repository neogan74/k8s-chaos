# Deployment Methods

Multiple deployment options for k8s-chaos to suit different workflows and environments.

## Quick Comparison

| Method | Best For | Pros | Cons |
|--------|----------|------|------|
| **Helm** | Simple deployments, templating | Easy, widely adopted, configurable | Less GitOps-friendly |
| **ArgoCD** | GitOps, multi-cluster | Automated sync, UI, drift detection | Requires ArgoCD installed |
| **Flux** | GitOps, progressive delivery | Native Kubernetes, flexible | Steeper learning curve |
| **Kustomize** | Environment overlays, patching | No templating, native | More verbose than Helm |
| **kubectl** | Quick tests, CI/CD | Direct, no dependencies | Not declarative |

## Directory Structure

```
deploy/
├── argocd/                    # ArgoCD manifests
│   ├── application.yaml       # Single cluster deployment
│   ├── applicationset.yaml    # Multi-cluster deployment
│   └── README.md
├── flux/                      # Flux CD manifests
│   ├── gitrepository.yaml     # Git source
│   ├── kustomization.yaml     # Kustomize deployment
│   ├── helmrelease.yaml       # Helm deployment
│   └── README.md
├── kustomize/                 # Kustomize overlays
│   ├── base/                  # Base configuration
│   └── overlays/
│       ├── dev/               # Development
│       ├── staging/           # Staging
│       └── production/        # Production
└── README.md                  # This file
```

---

## Installation Methods

### 1. Helm (Recommended for Getting Started)

**Quick start**:
```bash
# Add repository (if published to Helm registry)
helm repo add k8s-chaos https://neogan74.github.io/k8s-chaos
helm repo update

# Install from local chart
helm install k8s-chaos charts/k8s-chaos \
  --namespace k8s-chaos-system \
  --create-namespace

# Or install with custom values
helm install k8s-chaos charts/k8s-chaos \
  --namespace k8s-chaos-system \
  --create-namespace \
  --values my-values.yaml
```

**Pros**:
- ✅ Simple and quick
- ✅ Widely adopted
- ✅ Good for templating
- ✅ Helm hooks for lifecycle management

**Cons**:
- ❌ Manual updates required
- ❌ Less GitOps-friendly
- ❌ Drift can occur

**When to use**: Local development, quick testing, traditional deployments

📚 [Full Helm Documentation](../charts/k8s-chaos/README.md)

---

### 2. ArgoCD (Recommended for GitOps)

**Quick start**:
```bash
# Install ArgoCD (if not already installed)
kubectl create namespace argocd
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml

# Deploy k8s-chaos
kubectl apply -f deploy/argocd/application.yaml

# Access ArgoCD UI
kubectl port-forward svc/argocd-server -n argocd 8080:443
# Login: admin / <password from secret>
```

**Pros**:
- ✅ Excellent UI
- ✅ Automatic sync from Git
- ✅ Drift detection and remediation
- ✅ Multi-cluster support
- ✅ RBAC and SSO

**Cons**:
- ❌ Requires ArgoCD installation
- ❌ Additional component to manage

**When to use**: Teams using GitOps, multi-cluster deployments, enterprises

📚 [Full ArgoCD Guide](argocd/README.md)

---

### 3. Flux CD (Recommended for Cloud-Native GitOps)

**Quick start**:
```bash
# Install Flux CLI
brew install fluxcd/tap/flux

# Bootstrap Flux
flux bootstrap github \
  --owner=neogan74 \
  --repository=k8s-chaos \
  --branch=main \
  --path=deploy/flux

# Or apply manually
kubectl apply -f deploy/flux/gitrepository.yaml
kubectl apply -f deploy/flux/kustomization.yaml
# OR
kubectl apply -f deploy/flux/helmrelease.yaml
```

**Pros**:
- ✅ Kubernetes-native
- ✅ Progressive delivery
- ✅ Image automation
- ✅ Multi-tenancy support
- ✅ Notification system

**Cons**:
- ❌ No built-in UI
- ❌ Steeper learning curve
- ❌ More YAML to manage

**When to use**: Cloud-native teams, advanced GitOps, image automation needed

📚 [Full Flux Guide](flux/README.md)

---

### 4. Kustomize (Recommended for Environment Overlays)

**Quick start**:
```bash
# Development
kubectl apply -k deploy/kustomize/overlays/dev

# Staging
kubectl apply -k deploy/kustomize/overlays/staging

# Production
kubectl apply -k deploy/kustomize/overlays/production
```

**Pros**:
- ✅ No templating (pure YAML)
- ✅ Native kubectl support
- ✅ Great for environment-specific configs
- ✅ Layered approach

**Cons**:
- ❌ Can be verbose
- ❌ Less powerful than Helm for complex scenarios
- ❌ Manual updates required

**When to use**: Multiple environments, teams preferring pure YAML, GitOps with Flux

📚 [Full Kustomize Guide](kustomize/README.md)

---

### 5. Plain kubectl

**Quick start**:
```bash
# Install CRDs
kubectl apply -f config/crd/bases/

# Install controller
kubectl apply -f config/default/

# Or install everything
kubectl apply -k config/default/
```

**Pros**:
- ✅ Direct and simple
- ✅ No additional tools
- ✅ Good for CI/CD

**Cons**:
- ❌ Manual management
- ❌ No version control integration
- ❌ Harder to track changes

**When to use**: Testing, CI/CD pipelines, learning

---

## Environment-Specific Deployments

### Development

**Characteristics**: Low resources, debug logging, latest images

```bash
# Helm
helm install k8s-chaos charts/k8s-chaos \
  --namespace k8s-chaos-dev \
  --create-namespace \
  --set image.tag=dev-latest \
  --set resources.limits.cpu=200m \
  --set replicaCount=1

# Kustomize
kubectl apply -k deploy/kustomize/overlays/dev

# ArgoCD
kubectl apply -f deploy/argocd/application-dev.yaml
```

### Staging

**Characteristics**: Moderate resources, HA, RC images

```bash
# Helm
helm install k8s-chaos charts/k8s-chaos \
  --namespace k8s-chaos-staging \
  --create-namespace \
  --set image.tag=v0.1.0-rc.1 \
  --set replicaCount=2

# Kustomize
kubectl apply -k deploy/kustomize/overlays/staging

# Flux
kubectl apply -f deploy/flux/helmrelease-staging.yaml
```

### Production

**Characteristics**: Production resources, HA, pinned versions, security hardening

```bash
# Helm
helm install k8s-chaos charts/k8s-chaos \
  --namespace k8s-chaos-system \
  --create-namespace \
  --set image.tag=v0.1.0 \
  --set replicaCount=2 \
  --set podSecurityPolicy.enabled=true

# Kustomize (recommended for production)
kubectl diff -k deploy/kustomize/overlays/production
kubectl apply -k deploy/kustomize/overlays/production

# ArgoCD (recommended for production)
kubectl apply -f deploy/argocd/application.yaml
```

---

## Multi-Cluster Deployments

### ArgoCD ApplicationSet

Deploy to multiple clusters automatically:

```bash
# Label clusters
kubectl label cluster dev chaos-enabled=true environment=dev
kubectl label cluster prod chaos-enabled=true environment=production

# Deploy ApplicationSet
kubectl apply -f deploy/argocd/applicationset.yaml

# k8s-chaos will be deployed to all labeled clusters
```

### Flux Multi-Cluster

```bash
# Create cluster-specific Kustomizations
flux create kustomization k8s-chaos-dev \
  --source=k8s-chaos \
  --path="./deploy/kustomize/overlays/dev" \
  --prune=true

flux create kustomization k8s-chaos-prod \
  --source=k8s-chaos \
  --path="./deploy/kustomize/overlays/production" \
  --prune=true
```

---

## Configuration Examples

### Enable Metrics

```bash
# Helm
helm upgrade k8s-chaos charts/k8s-chaos \
  --set metrics.enabled=true \
  --set metrics.serviceMonitor.enabled=true

# Kustomize patch
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: k8s-chaos-config
data:
  METRICS_ENABLED: "true"
EOF
```

### High Availability

```bash
# Helm
helm upgrade k8s-chaos charts/k8s-chaos \
  --set replicaCount=3 \
  --set podDisruptionBudget.enabled=true \
  --set podDisruptionBudget.minAvailable=2

# Kustomize
# Use production overlay (already configured for HA)
kubectl apply -k deploy/kustomize/overlays/production
```

### Custom Resources

```bash
# Helm
helm upgrade k8s-chaos charts/k8s-chaos \
  --set resources.limits.cpu=1000m \
  --set resources.limits.memory=1Gi \
  --set resources.requests.cpu=200m \
  --set resources.requests.memory=256Mi
```

---

## Upgrade Strategies

### Helm Upgrade

```bash
# Preview changes
helm diff upgrade k8s-chaos charts/k8s-chaos

# Upgrade
helm upgrade k8s-chaos charts/k8s-chaos

# Rollback if needed
helm rollback k8s-chaos
```

### ArgoCD Upgrade

```bash
# Update Git repository (commit new version)
git commit -m "Update k8s-chaos to v0.2.0"
git push

# ArgoCD syncs automatically
# Or sync manually:
argocd app sync k8s-chaos
```

### Flux Upgrade

```bash
# Update image tag in Git
# Flux syncs automatically

# Or use image automation:
# Flux will detect new images and update automatically
```

---

## Monitoring & Validation

### Health Checks

```bash
# Check pods
kubectl get pods -n k8s-chaos-system

# Check deployment
kubectl rollout status deployment/k8s-chaos-controller-manager -n k8s-chaos-system

# Check CRDs
kubectl get crd | grep chaos

# Check webhook
kubectl get validatingwebhookconfiguration | grep chaos
```

### Verify Installation

```bash
# Create test experiment
kubectl apply -f config/samples/chaos_v1alpha1_chaosexperiment.yaml

# Check status
kubectl get chaosexperiment

# View controller logs
kubectl logs -n k8s-chaos-system deployment/k8s-chaos-controller-manager -f
```

---

## Troubleshooting

### Common Issues

**Issue**: Pods not starting
```bash
# Check events
kubectl describe pod -n k8s-chaos-system

# Check logs
kubectl logs -n k8s-chaos-system deployment/k8s-chaos-controller-manager
```

**Issue**: Webhook not working
```bash
# Check webhook configuration
kubectl get validatingwebhookconfiguration

# Test webhook
kubectl apply -f config/samples/chaos_v1alpha1_chaosexperiment.yaml --dry-run=server
```

**Issue**: CRDs not installing
```bash
# Manually install CRDs
kubectl apply -f config/crd/bases/
```

---

## Uninstallation

### Helm

```bash
helm uninstall k8s-chaos -n k8s-chaos-system
kubectl delete namespace k8s-chaos-system
```

### ArgoCD

```bash
argocd app delete k8s-chaos
# Or
kubectl delete application k8s-chaos -n argocd
```

### Flux

```bash
flux delete kustomization k8s-chaos
flux delete source git k8s-chaos
```

### Kustomize/kubectl

```bash
kubectl delete -k deploy/kustomize/overlays/production
# Or
kubectl delete namespace k8s-chaos-system
```

---

## References

- [Helm Chart Documentation](../charts/k8s-chaos/README.md)
- [ArgoCD Deployment Guide](argocd/README.md)
- [Flux Deployment Guide](flux/README.md)
- [Kustomize Overlays Guide](kustomize/README.md)
- [Main Documentation](../docs/README.md)

---

## Support

For issues or questions:
- GitHub Issues: https://github.com/neogan74/k8s-chaos/issues
- Documentation: https://github.com/neogan74/k8s-chaos/tree/main/docs
