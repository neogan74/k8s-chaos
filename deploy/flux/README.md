# Flux CD Deployment

Deploy k8s-chaos using Flux CD for GitOps-based continuous delivery.

## Prerequisites

- Flux CD installed in the cluster ([Installation Guide](https://fluxcd.io/docs/installation/))
- kubectl access to the cluster
- GitHub/GitLab repository access

## Quick Start

### Option 1: Using Kustomize (Recommended)

Deploy using Kustomize manifests:

```bash
# Apply GitRepository
kubectl apply -f gitrepository.yaml

# Apply Kustomization
kubectl apply -f kustomization.yaml

# Watch reconciliation
flux get kustomizations --watch
```

### Option 2: Using Helm Chart

Deploy using the Helm chart via HelmRelease:

```bash
# Apply GitRepository
kubectl apply -f gitrepository.yaml

# Apply HelmRelease
kubectl apply -f helmrelease.yaml

# Watch reconciliation
flux get helmreleases --watch
```

## Bootstrap from Scratch

Bootstrap Flux and k8s-chaos together:

```bash
# Bootstrap Flux on the cluster
flux bootstrap github \
  --owner=neogan74 \
  --repository=k8s-chaos \
  --branch=main \
  --path=./deploy/flux \
  --personal

# Flux will automatically deploy k8s-chaos
```

## Configuration

### Using Kustomize

Customize via Kustomize patches:

```yaml
# kustomization-patch.yaml
apiVersion: kustomize.toolkit.fluxcd.io/v1
kind: Kustomization
metadata:
  name: k8s-chaos
  namespace: flux-system
spec:
  patches:
    - patch: |
        - op: replace
          path: /spec/replicas
          value: 2
      target:
        kind: Deployment
        name: k8s-chaos-controller-manager
```

### Using Helm Values

Create a ConfigMap with custom values:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: k8s-chaos-values
  namespace: flux-system
data:
  values.yaml: |
    replicaCount: 2
    resources:
      limits:
        cpu: 1000m
        memory: 1Gi
```

Then reference it in `helmrelease.yaml`:

```yaml
spec:
  valuesFrom:
    - kind: ConfigMap
      name: k8s-chaos-values
```

## Multi-Environment Setup

### Directory Structure

```
deploy/flux/
├── base/
│   ├── gitrepository.yaml
│   └── kustomization.yaml
└── environments/
    ├── dev/
    │   ├── kustomization.yaml
    │   └── values.yaml
    ├── staging/
    │   ├── kustomization.yaml
    │   └── values.yaml
    └── production/
        ├── kustomization.yaml
        └── values.yaml
```

### Environment-Specific Kustomizations

```bash
# Development
flux create kustomization k8s-chaos-dev \
  --source=GitRepository/k8s-chaos \
  --path="./deploy/flux/environments/dev" \
  --prune=true \
  --interval=10m

# Staging
flux create kustomization k8s-chaos-staging \
  --source=GitRepository/k8s-chaos \
  --path="./deploy/flux/environments/staging" \
  --prune=true \
  --interval=10m

# Production
flux create kustomization k8s-chaos-production \
  --source=GitRepository/k8s-chaos \
  --path="./deploy/flux/environments/production" \
  --prune=true \
  --interval=10m \
  --depends-on=k8s-chaos-staging
```

## Monitoring

### Check Status

```bash
# List all Flux resources
flux get all

# Check Kustomization
flux get kustomization k8s-chaos

# Check HelmRelease
flux get helmrelease k8s-chaos

# View logs
flux logs --kind=Kustomization --name=k8s-chaos
```

### Suspend/Resume

```bash
# Suspend reconciliation
flux suspend kustomization k8s-chaos

# Resume reconciliation
flux resume kustomization k8s-chaos
```

## Troubleshooting

### Reconciliation Failed

```bash
# Check events
flux events --for Kustomization/k8s-chaos

# Force reconciliation
flux reconcile kustomization k8s-chaos --with-source

# View detailed logs
kubectl logs -n flux-system deployment/kustomize-controller -f
```

### Helm Release Failed

```bash
# Check Helm release status
flux get helmrelease k8s-chaos

# View Helm logs
kubectl logs -n flux-system deployment/helm-controller -f

# Debug values
flux diff helmrelease k8s-chaos
```

### Source Not Found

```bash
# Check GitRepository
flux get sources git

# Force fetch
flux reconcile source git k8s-chaos

# Verify credentials
kubectl get secret -n flux-system
```

## Advanced Features

### Image Automation

Automatically update image tags:

```yaml
apiVersion: image.toolkit.fluxcd.io/v1beta1
kind: ImageRepository
metadata:
  name: k8s-chaos
  namespace: flux-system
spec:
  image: ghcr.io/neogan74/k8s-chaos
  interval: 5m

---
apiVersion: image.toolkit.fluxcd.io/v1beta1
kind: ImagePolicy
metadata:
  name: k8s-chaos
  namespace: flux-system
spec:
  imageRepositoryRef:
    name: k8s-chaos
  policy:
    semver:
      range: '>=0.1.0 <1.0.0'

---
apiVersion: image.toolkit.fluxcd.io/v1beta1
kind: ImageUpdateAutomation
metadata:
  name: k8s-chaos
  namespace: flux-system
spec:
  interval: 30m
  sourceRef:
    kind: GitRepository
    name: k8s-chaos
  git:
    commit:
      author:
        name: fluxcdbot
        email: flux@users.noreply.github.com
      messageTemplate: |
        Update k8s-chaos to {{range .Updated.Images}}{{println .}}{{end}}
  update:
    path: ./deploy/flux
    strategy: Setters
```

### Secret Management (SOPS)

Encrypt secrets:

```bash
# Install SOPS
brew install sops age

# Generate key
age-keygen -o age.key

# Create Kubernetes secret with key
cat age.key | kubectl create secret generic sops-age \
  --namespace=flux-system \
  --from-file=age.agekey=/dev/stdin

# Encrypt file
sops --age=$(cat age.key | grep public | cut -d: -f2) \
  --encrypt --encrypted-regex '^(data|stringData)$' \
  --in-place secret.yaml

# Configure decryption in Kustomization
spec:
  decryption:
    provider: sops
    secretRef:
      name: sops-age
```

### Notifications

Configure alerts:

```yaml
apiVersion: notification.toolkit.fluxcd.io/v1beta1
kind: Alert
metadata:
  name: k8s-chaos
  namespace: flux-system
spec:
  providerRef:
    name: slack
  eventSeverity: info
  eventSources:
    - kind: Kustomization
      name: k8s-chaos
    - kind: HelmRelease
      name: k8s-chaos

---
apiVersion: notification.toolkit.fluxcd.io/v1beta1
kind: Provider
metadata:
  name: slack
  namespace: flux-system
spec:
  type: slack
  channel: chaos-engineering
  secretRef:
    name: slack-webhook
```

## Uninstall

```bash
# Delete Kustomization (cascades to all resources)
flux delete kustomization k8s-chaos

# Or delete HelmRelease
flux delete helmrelease k8s-chaos

# Delete GitRepository
flux delete source git k8s-chaos
```

## References

- [Flux Documentation](https://fluxcd.io/docs/)
- [Kustomize Controller](https://fluxcd.io/docs/components/kustomize/)
- [Helm Controller](https://fluxcd.io/docs/components/helm/)
- [k8s-chaos Helm Chart](../../charts/k8s-chaos/)
