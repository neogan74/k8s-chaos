# k8s-chaos Helm Chart

A Helm chart for deploying the k8s-chaos Kubernetes chaos engineering operator.

## TL;DR

```bash
helm repo add k8s-chaos https://neogan74.github.io/k8s-chaos
helm install k8s-chaos k8s-chaos/k8s-chaos --namespace k8s-chaos-system --create-namespace
```

## Introduction

This chart bootstraps a [k8s-chaos](https://github.com/neogan74/k8s-chaos) deployment on a [Kubernetes](https://kubernetes.io) cluster using the [Helm](https://helm.sh) package manager.

k8s-chaos is a production-ready, lightweight chaos engineering operator with comprehensive safety features including dry-run mode, percentage limits, exclusion labels, and production namespace protection.

## Prerequisites

- Kubernetes 1.24+
- Helm 3.8+
- (Optional) cert-manager for webhook certificates

## Installing the Chart

### Basic Installation

```bash
helm install k8s-chaos k8s-chaos/k8s-chaos \
  --namespace k8s-chaos-system \
  --create-namespace
```

### Installation with Custom Values

```bash
helm install k8s-chaos k8s-chaos/k8s-chaos \
  --namespace k8s-chaos-system \
  --create-namespace \
  --values custom-values.yaml
```

### Installation with cert-manager

If you have cert-manager installed:

```bash
helm install k8s-chaos k8s-chaos/k8s-chaos \
  --namespace k8s-chaos-system \
  --create-namespace \
  --set webhook.certificate.certManager=true \
  --set webhook.certificate.generate=false
```

## Uninstalling the Chart

```bash
helm uninstall k8s-chaos --namespace k8s-chaos-system
```

**Note:** By default, CRDs are kept on uninstall for safety. To remove them:

```bash
kubectl delete crd chaosexperiments.chaos.gushchin.dev
kubectl delete crd chaosexperimenthistories.chaos.gushchin.dev
```

## Configuration

### Key Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `image.repository` | Controller image repository | `ghcr.io/neogan74/k8s-chaos` |
| `image.tag` | Controller image tag | `Chart.appVersion` |
| `controller.replicaCount` | Number of controller replicas | `1` |
| `controller.logLevel` | Log level (debug, info, warn, error) | `info` |
| `webhook.enabled` | Enable admission webhook | `true` |
| `metrics.enabled` | Enable Prometheus metrics | `true` |
| `history.enabled` | Enable experiment history | `true` |
| `history.retentionLimit` | Max history records per experiment | `100` |

### Resource Configuration

```yaml
controller:
  resources:
    limits:
      cpu: 500m
      memory: 512Mi
    requests:
      cpu: 100m
      memory: 128Mi
```

### Security Configuration

```yaml
controller:
  podSecurityContext:
    runAsNonRoot: true
    runAsUser: 65532
    fsGroup: 65532
  securityContext:
    allowPrivilegeEscalation: false
    readOnlyRootFilesystem: true
    runAsNonRoot: true
    capabilities:
      drop:
        - ALL
```

### Metrics and ServiceMonitor

Enable Prometheus ServiceMonitor:

```yaml
metrics:
  enabled: true
  serviceMonitor:
    enabled: true
    interval: 30s
    labels:
      prometheus: kube-prometheus
```

### High Availability

```yaml
controller:
  replicaCount: 1  # Only 1 supported with leader election
  affinity:
    podAntiAffinity:
      preferredDuringSchedulingIgnoredDuringExecution:
      - weight: 100
        podAffinityTerm:
          labelSelector:
            matchLabels:
              control-plane: controller-manager
          topologyKey: kubernetes.io/hostname
```

## Examples

### Development Installation

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
  secure: false

history:
  retentionLimit: 50
```

```bash
helm install k8s-chaos k8s-chaos/k8s-chaos \
  -n k8s-chaos-system --create-namespace \
  -f dev-values.yaml
```

### Production Installation

```yaml
# prod-values.yaml
controller:
  logLevel: info
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

history:
  enabled: true
  retentionLimit: 200

webhook:
  certificate:
    certManager: true
    generate: false
```

```bash
helm install k8s-chaos k8s-chaos/k8s-chaos \
  -n k8s-chaos-system --create-namespace \
  -f prod-values.yaml
```

## Upgrade

### Upgrade to Latest Version

```bash
helm repo update
helm upgrade k8s-chaos k8s-chaos/k8s-chaos \
  -n k8s-chaos-system
```

### Upgrade with New Values

```bash
helm upgrade k8s-chaos k8s-chaos/k8s-chaos \
  -n k8s-chaos-system \
  -f new-values.yaml
```

## Troubleshooting

### Check Installation Status

```bash
# Check Helm release
helm list -n k8s-chaos-system

# Check pods
kubectl get pods -n k8s-chaos-system

# Check logs
kubectl logs -n k8s-chaos-system deployment/k8s-chaos-controller-manager -f
```

### Common Issues

**1. Webhook Certificate Issues**

If webhook is failing:

```bash
# Check certificate
kubectl get secret k8s-chaos-webhook-cert -n k8s-chaos-system

# If using cert-manager
kubectl get certificate -n k8s-chaos-system
kubectl describe certificate k8s-chaos-serving-cert -n k8s-chaos-system
```

**2. RBAC Issues**

```bash
# Check RBAC
kubectl get clusterrole | grep k8s-chaos
kubectl get clusterrolebinding | grep k8s-chaos
```

**3. CRD Installation Issues**

```bash
# Check CRDs
kubectl get crds | grep chaos

# Reinstall CRDs
helm upgrade k8s-chaos k8s-chaos/k8s-chaos \
  -n k8s-chaos-system \
  --force
```

## Parameters Reference

See `values.yaml` for complete parameters documentation with comments.

## Additional Resources

- **Main Repository**: https://github.com/neogan74/k8s-chaos
- **Documentation**: https://github.com/neogan74/k8s-chaos/tree/main/docs
- **Getting Started**: https://github.com/neogan74/k8s-chaos/blob/main/docs/GETTING-STARTED.md
- **Best Practices**: https://github.com/neogan74/k8s-chaos/blob/main/docs/BEST-PRACTICES.md
- **Issue Tracker**: https://github.com/neogan74/k8s-chaos/issues

## License

Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0