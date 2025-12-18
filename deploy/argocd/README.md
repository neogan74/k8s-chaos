# ArgoCD Deployment

Deploy k8s-chaos using ArgoCD for GitOps-based continuous delivery.

## Prerequisites

- ArgoCD installed in the cluster
- kubectl access to the cluster
- Repository access (for private repos)

## Quick Start

### Single Cluster Deployment

Deploy k8s-chaos to a single cluster using the basic Application manifest:

```bash
# Apply the ArgoCD Application
kubectl apply -f application.yaml

# Watch the sync status
argocd app get k8s-chaos

# Or view in the UI
argocd app open k8s-chaos
```

### Multi-Cluster Deployment

Deploy k8s-chaos to multiple clusters using ApplicationSet:

```bash
# Label your clusters first
kubectl label cluster dev-cluster chaos-enabled=true environment=dev
kubectl label cluster staging-cluster chaos-enabled=true environment=staging
kubectl label cluster prod-cluster chaos-enabled=true environment=production

# Apply the ApplicationSet
kubectl apply -f applicationset.yaml

# List all generated applications
argocd app list | grep k8s-chaos
```

## Configuration

### Helm Values Override

Customize the deployment by editing `application.yaml`:

```yaml
spec:
  source:
    helm:
      values: |
        replicaCount: 2  # High availability

        resources:
          limits:
            cpu: 1000m
            memory: 1Gi

        metrics:
          enabled: true
          serviceMonitor:
            enabled: true  # Enable Prometheus monitoring
```

### Environment-Specific Configuration

For different environments, use separate Application files:

```bash
# Development
cp application.yaml application-dev.yaml
# Edit application-dev.yaml with dev-specific values

# Staging
cp application.yaml application-staging.yaml
# Edit application-staging.yaml with staging-specific values

# Production
cp application.yaml application-production.yaml
# Edit application-production.yaml with production-specific values
```

## Sync Policies

### Automatic Sync

The Application is configured with automatic sync enabled:

```yaml
syncPolicy:
  automated:
    prune: true      # Remove resources not in Git
    selfHeal: true   # Sync when cluster state drifts
```

To disable automatic sync:

```bash
argocd app set k8s-chaos --sync-policy none
```

### Manual Sync

Trigger manual sync:

```bash
# Full sync
argocd app sync k8s-chaos

# Sync specific resource
argocd app sync k8s-chaos --resource apps:Deployment:k8s-chaos-controller-manager

# Dry run
argocd app sync k8s-chaos --dry-run
```

## Monitoring

### View Application Status

```bash
# Get application details
argocd app get k8s-chaos

# View sync history
argocd app history k8s-chaos

# View logs
argocd app logs k8s-chaos
```

### Health Checks

ArgoCD automatically monitors these resources:

- Deployment health (replicas ready)
- CRD installation status
- Webhook configuration
- Service endpoints

## Troubleshooting

### Sync Failed

```bash
# Check sync status
argocd app get k8s-chaos

# View detailed error
argocd app logs k8s-chaos --kind Deployment

# Refresh app (re-fetch from Git)
argocd app refresh k8s-chaos
```

### Stuck in Syncing

```bash
# Terminate operation
argocd app terminate-op k8s-chaos

# Force sync
argocd app sync k8s-chaos --force
```

### Out of Sync

```bash
# Show differences
argocd app diff k8s-chaos

# Sync with prune
argocd app sync k8s-chaos --prune
```

## Advanced Usage

### App of Apps Pattern

Create a parent app that manages multiple chaos-related applications:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: chaos-stack
  namespace: argocd
spec:
  source:
    repoURL: https://github.com/neogan74/k8s-chaos
    path: deploy/argocd/apps
  destination:
    server: https://kubernetes.default.svc
    namespace: argocd
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
```

### Webhook Integration

Configure webhooks for faster sync:

```bash
# In GitHub repository settings
# Add webhook: https://argocd.example.com/api/webhook
# Content type: application/json
# Event: Push
```

### Progressive Sync

Use sync waves for ordered deployment:

```yaml
metadata:
  annotations:
    argocd.argoproj.io/sync-wave: "1"  # CRDs first
```

## Security

### Private Repository

Configure credentials:

```bash
# Add repository
argocd repo add https://github.com/neogan74/k8s-chaos \
  --username <username> \
  --password <token>

# Or use SSH
argocd repo add git@github.com:neogan74/k8s-chaos.git \
  --ssh-private-key-path ~/.ssh/id_rsa
```

### RBAC

Limit access to the application:

```yaml
# argocd-rbac-cm ConfigMap
policy.csv: |
  p, role:chaos-admin, applications, *, k8s-chaos/*, allow
  g, chaos-team, role:chaos-admin
```

## Uninstall

```bash
# Delete the Application (cascades to all resources)
argocd app delete k8s-chaos

# Or use kubectl
kubectl delete -f application.yaml
```

## References

- [ArgoCD Documentation](https://argo-cd.readthedocs.io/)
- [ApplicationSet Documentation](https://argo-cd.readthedocs.io/en/stable/user-guide/application-set/)
- [k8s-chaos Helm Chart](../../charts/k8s-chaos/)
