# Kustomize Overlays

Environment-specific Kustomize configurations for k8s-chaos deployment.

## Structure

```
kustomize/
├── base/                       # Base configuration
│   └── kustomization.yaml      # References config/default
└── overlays/
    ├── dev/                    # Development environment
    │   └── kustomization.yaml
    ├── staging/                # Staging environment
    │   └── kustomization.yaml
    └── production/             # Production environment
        ├── kustomization.yaml
        └── network-policy.yaml
```

## Quick Start

### Development

```bash
# Preview changes
kubectl kustomize deploy/kustomize/overlays/dev

# Apply
kubectl apply -k deploy/kustomize/overlays/dev

# Verify
kubectl get all -n k8s-chaos-dev
```

### Staging

```bash
# Preview
kubectl kustomize deploy/kustomize/overlays/staging

# Apply
kubectl apply -k deploy/kustomize/overlays/staging

# Verify
kubectl get all -n k8s-chaos-staging
```

### Production

```bash
# Preview (always preview first in production!)
kubectl kustomize deploy/kustomize/overlays/production

# Apply
kubectl apply -k deploy/kustomize/overlays/production

# Verify
kubectl get all -n k8s-chaos-system
```

## Environment Configurations

### Development

**Namespace**: `k8s-chaos-dev`

**Characteristics**:
- Single replica
- Lower resource limits (200m CPU, 256Mi memory)
- Debug logging enabled
- Latest/dev image tags
- Faster iteration

**Use Case**: Local testing, feature development

### Staging

**Namespace**: `k8s-chaos-staging`

**Characteristics**:
- 2 replicas (HA)
- Moderate resources (400m CPU, 384Mi memory)
- Info logging
- RC (release candidate) tags
- PodDisruptionBudget

**Use Case**: Pre-production testing, QA validation

### Production

**Namespace**: `k8s-chaos-system`

**Characteristics**:
- 2 replicas with anti-affinity (HA)
- Production resources (500m CPU, 512Mi memory)
- Pinned semantic versions (v0.1.0)
- Enhanced security (NetworkPolicy, securityContext)
- PodDisruptionBudget
- Resource quotas

**Use Case**: Production chaos engineering

## Customization

### Override Image Tag

```bash
# Using kustomize edit
cd deploy/kustomize/overlays/dev
kustomize edit set image controller=ghcr.io/neogan74/k8s-chaos:v0.2.0

# Or using kubectl
kubectl kustomize deploy/kustomize/overlays/dev | \
  kubectl set image -f - controller=ghcr.io/neogan74/k8s-chaos:v0.2.0 --local -o yaml
```

### Add Custom Patches

Create a patch file:

```yaml
# custom-patch.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: k8s-chaos-controller-manager
spec:
  template:
    spec:
      containers:
        - name: manager
          env:
            - name: CUSTOM_VAR
              value: "custom-value"
```

Reference in `kustomization.yaml`:

```yaml
patches:
  - path: custom-patch.yaml
```

### Change Namespace

```bash
cd deploy/kustomize/overlays/dev
kustomize edit set namespace my-custom-namespace
```

### Add Labels

```bash
cd deploy/kustomize/overlays/dev
kustomize edit add label team:platform-engineering
```

## Advanced Usage

### Multi-Environment Deployment

Deploy to all environments:

```bash
#!/bin/bash
for env in dev staging production; do
  echo "Deploying to $env..."
  kubectl apply -k deploy/kustomize/overlays/$env
done
```

### Diff Before Apply

```bash
# Show what would change
kubectl diff -k deploy/kustomize/overlays/production

# Or use kustomize directly
kustomize build deploy/kustomize/overlays/production | kubectl diff -f -
```

### Validate Before Deploy

```bash
# Kubeval validation
kustomize build deploy/kustomize/overlays/production | kubeval

# Kube-score
kustomize build deploy/kustomize/overlays/production | kube-score score -

# Conftest (policy as code)
kustomize build deploy/kustomize/overlays/production | conftest test -
```

### Remote Base

Use remote base for version pinning:

```yaml
# kustomization.yaml
resources:
  - https://github.com/neogan74/k8s-chaos//config/default?ref=v0.1.0
```

## Integration with CI/CD

### GitHub Actions

```yaml
name: Deploy to Staging
on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Setup kubectl
        uses: azure/setup-kubectl@v3

      - name: Deploy
        run: |
          kubectl apply -k deploy/kustomize/overlays/staging
        env:
          KUBECONFIG: ${{ secrets.KUBECONFIG_STAGING }}
```

### GitLab CI

```yaml
deploy:staging:
  stage: deploy
  script:
    - kubectl apply -k deploy/kustomize/overlays/staging
  environment:
    name: staging
  only:
    - main
```

## Secrets Management

### Using Sealed Secrets

```bash
# Create secret
kubectl create secret generic my-secret \
  --from-literal=password=secret123 \
  --dry-run=client -o yaml | \
  kubeseal -o yaml > sealed-secret.yaml

# Add to kustomization
resources:
  - sealed-secret.yaml
```

### Using External Secrets Operator

```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: k8s-chaos-secrets
spec:
  secretStoreRef:
    name: vault-backend
    kind: SecretStore
  target:
    name: k8s-chaos-secrets
  data:
    - secretKey: api-key
      remoteRef:
        key: chaos/api-key
```

## Troubleshooting

### Build Errors

```bash
# Validate syntax
kustomize build deploy/kustomize/overlays/dev

# Debug with verbose output
kustomize build --load-restrictor LoadRestrictionsNone deploy/kustomize/overlays/dev
```

### Resource Not Found

```bash
# Check if base exists
ls -la deploy/kustomize/base/

# Verify paths
kustomize cfg tree deploy/kustomize/overlays/dev
```

### Patch Not Applied

```bash
# Check patch format
kustomize cfg grep kind=Deployment deploy/kustomize/overlays/dev

# Verify target selector
kustomize build --enable-alpha-plugins deploy/kustomize/overlays/dev
```

## Best Practices

1. **Always preview in production**: `kubectl diff -k` before `kubectl apply -k`
2. **Pin versions**: Use semantic versioning in production
3. **Test overlays**: Validate with kubeval/kube-score
4. **Document patches**: Add comments explaining why patches exist
5. **Use namespaces**: Separate environments with namespaces
6. **Version control**: Commit kustomization.yaml changes
7. **Security**: Enable NetworkPolicy and PodSecurityPolicy in production

## Migration from Helm

If migrating from Helm:

```bash
# Export current Helm values
helm get values k8s-chaos > current-values.yaml

# Convert to Kustomize patches
# (manual process - review each value)

# Test side-by-side
helm template k8s-chaos charts/k8s-chaos > helm-output.yaml
kustomize build deploy/kustomize/overlays/production > kustomize-output.yaml
diff helm-output.yaml kustomize-output.yaml
```

## References

- [Kustomize Documentation](https://kubectl.docs.kubernetes.io/guides/introduction/kustomize/)
- [Kubernetes Kustomization](https://kubernetes.io/docs/tasks/manage-kubernetes-objects/kustomization/)
- [k8s-chaos Base Config](../../config/default/)
