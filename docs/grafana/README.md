# K8s Chaos Grafana Dashboards

This directory contains Grafana dashboard JSON files and an import script for the k8s-chaos operator.

## Files

- `chaos-experiments-overview.json` - High-level overview dashboard
- `chaos-experiments-detailed.json` - Detailed analysis with filters
- `chaos-safety-monitoring.json` - Safety and error monitoring
- `import-dashboards.sh` - Automated import script

## Quick Start

### 1. Deploy Grafana (if needed)

```bash
kubectl apply -k ../../config/grafana/
kubectl port-forward svc/grafana -n monitoring 3000:3000
```

### 2. Import Dashboards

```bash
./import-dashboards.sh http://localhost:3000 admin:admin
```

### 3. Access Dashboards

Open http://localhost:3000/dashboards and look for "Chaos Engineering" folder.

## Prerequisites

- Prometheus configured and scraping k8s-chaos controller metrics
- jq installed (for import script)

## Dashboard URLs

After import, access dashboards at:

- Overview: `http://localhost:3000/d/k8s-chaos-overview`
- Detailed: `http://localhost:3000/d/k8s-chaos-detailed`
- Safety: `http://localhost:3000/d/k8s-chaos-safety`

## Detailed Documentation

See [docs/GRAFANA.md](../GRAFANA.md) for complete setup instructions, customization guide, and troubleshooting.
