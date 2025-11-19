# Grafana Dashboards for K8s Chaos

This document describes the Grafana dashboards available for monitoring chaos experiments and provides setup instructions.

## Overview

The k8s-chaos operator exports Prometheus metrics that can be visualized using Grafana. Three comprehensive dashboards are provided:

1. **K8s Chaos - Overview**: High-level view of all chaos experiments
2. **K8s Chaos - Detailed Analysis**: Deep dive with filtering by action and namespace
3. **K8s Chaos - Safety Monitoring**: Focus on errors, resource impact, and safety

## Prerequisites

- Kubernetes cluster with k8s-chaos operator deployed
- Prometheus Operator installed (for metrics collection)
- Grafana instance (provided manifests or your own)

## Dashboard Features

### 1. K8s Chaos - Overview

**Purpose**: Executive dashboard showing overall chaos engineering activity

**Key Panels**:
- **Total Experiments Executed**: Cumulative count of all experiments
- **Active Experiments**: Current running experiments
- **Experiment Success Rate**: Real-time success percentage over time
- **Experiments by Action Type**: Stacked area chart showing distribution
- **Experiments Distribution**: Donut chart of action types
- **Experiment Duration (p50 & p95)**: Performance percentiles by action
- **Resources Affected**: Number of pods/nodes currently impacted
- **Error Rate by Action and Type**: Stacked error trends

**Best For**: Teams leads, managers, stakeholders wanting high-level visibility

**Refresh Rate**: 30 seconds

---

### 2. K8s Chaos - Detailed Analysis

**Purpose**: Detailed analysis with filtering capabilities for troubleshooting

**Key Panels**:
- **Filtered Statistics**: Total executions, success rate, active experiments (based on filters)
- **Execution Rate by Status**: Success vs failure trends
- **Resources Affected by Experiment**: Individual experiment tracking
- **Duration Percentiles**: p99, p95, p50 bar chart
- **Errors by Type**: Detailed error breakdowns
- **Experiments Summary Table**: Complete list with metrics

**Filters**:
- **Action**: Filter by pod-kill, pod-delay, node-drain, pod-cpu-stress, etc.
- **Namespace**: Filter by target namespace

**Best For**: Engineers troubleshooting issues, analyzing specific experiments

**Refresh Rate**: 30 seconds

---

### 3. K8s Chaos - Safety Monitoring

**Purpose**: Monitor safety constraints, errors, and resource impact

**Key Panels**:
- **Error Rate**: Overall error rate across all experiments
- **Total Resources Currently Affected**: Real-time resource impact
- **Namespaces with Chaos Activity**: Spread of chaos testing
- **Overall Success Rate**: Health gauge (red < 90%, green > 95%)
- **Errors by Type Over Time**: Trend analysis of error categories
- **Error Distribution**: Pie chart of error types
- **Resources Affected by Namespace**: Namespace-level impact tracking
- **Experiment Activity by Namespace**: Where chaos is happening
- **Active Experiments - Resources Impact**: Table of current experiments
- **Errors Summary**: Detailed error table by namespace and action
- **Maximum Resources Affected**: Safety threshold monitoring with alerts

**Best For**: SREs, platform teams ensuring safe chaos testing practices

**Refresh Rate**: 30 seconds

---

## Quick Start

### Option 1: Deploy Grafana with K8s Manifests

If you don't have Grafana already:

```bash
# Deploy Grafana to the monitoring namespace
kubectl apply -k config/grafana/

# Wait for Grafana to be ready
kubectl wait --for=condition=available --timeout=300s deployment/grafana -n monitoring

# Port-forward to access Grafana
kubectl port-forward svc/grafana -n monitoring 3000:3000
```

Access Grafana at http://localhost:3000
- Username: `admin`
- Password: `admin`

### Option 2: Use Existing Grafana

If you already have Grafana:

1. Ensure Prometheus is configured as a datasource
2. Import the dashboards (see below)

---

## Importing Dashboards

### Method 1: Automated Script (Recommended)

```bash
# Navigate to dashboard directory
cd docs/grafana/

# Import all dashboards
./import-dashboards.sh http://localhost:3000 admin:admin

# Or with API key
./import-dashboards.sh https://grafana.example.com $GRAFANA_API_KEY
```

### Method 2: Manual Import via Grafana UI

1. Open Grafana UI
2. Navigate to **Dashboards** → **Import**
3. Click **Upload JSON file**
4. Select a dashboard file from `docs/grafana/`:
   - `chaos-experiments-overview.json`
   - `chaos-experiments-detailed.json`
   - `chaos-safety-monitoring.json`
5. Select the Prometheus datasource
6. Click **Import**

### Method 3: Grafana API

```bash
# Set variables
GRAFANA_URL="http://localhost:3000"
GRAFANA_AUTH="admin:admin"
DASHBOARD_FILE="docs/grafana/chaos-experiments-overview.json"

# Import using curl
curl -X POST \
  -H "Content-Type: application/json" \
  -u "$GRAFANA_AUTH" \
  -d @<(jq -n --argjson dashboard "$(cat $DASHBOARD_FILE)" '{dashboard: $dashboard, overwrite: true}') \
  "$GRAFANA_URL/api/dashboards/db"
```

---

## Prometheus Configuration

### Ensure Metrics are Being Scraped

The k8s-chaos controller exports metrics on port 8080 at `/metrics`.

**ServiceMonitor Example** (if using Prometheus Operator):

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: k8s-chaos-controller
  namespace: k8s-chaos-system
spec:
  selector:
    matchLabels:
      control-plane: controller-manager
  endpoints:
    - port: metrics
      path: /metrics
      interval: 30s
```

**Manual Prometheus Config**:

```yaml
scrape_configs:
  - job_name: 'k8s-chaos'
    kubernetes_sd_configs:
      - role: pod
        namespaces:
          names:
            - k8s-chaos-system
    relabel_configs:
      - source_labels: [__meta_kubernetes_pod_label_control_plane]
        regex: controller-manager
        action: keep
      - source_labels: [__meta_kubernetes_pod_container_port_name]
        regex: metrics
        action: keep
```

### Verify Metrics

Check that Prometheus is scraping metrics:

```bash
# Port-forward to Prometheus
kubectl port-forward svc/prometheus-operated -n monitoring 9090:9090

# Open browser to http://localhost:9090
# Query: chaosexperiment_executions_total
```

You should see metrics with labels like `action`, `namespace`, `status`.

---

## Available Metrics

The k8s-chaos operator exports the following Prometheus metrics:

| Metric Name | Type | Labels | Description |
|-------------|------|--------|-------------|
| `chaosexperiment_executions_total` | Counter | `action`, `namespace`, `status` | Total number of chaos experiments executed |
| `chaosexperiment_duration_seconds` | Histogram | `action`, `namespace` | Duration of experiment execution in seconds |
| `chaosexperiment_resources_affected` | Gauge | `action`, `namespace`, `experiment` | Number of resources (pods/nodes) affected |
| `chaosexperiment_errors_total` | Counter | `action`, `namespace`, `error_type` | Total number of errors during experiments |
| `chaosexperiment_active` | Gauge | `action` | Number of currently active experiments |

---

## Customization

### Modifying Dashboards

1. Open the dashboard in Grafana
2. Make your changes in the UI
3. Click **Dashboard settings** (gear icon)
4. Click **JSON Model**
5. Copy the JSON
6. Save to `docs/grafana/<dashboard-name>.json`
7. Commit changes to version control

### Creating Alerts

Example alert for high error rate:

```yaml
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: k8s-chaos-alerts
  namespace: monitoring
spec:
  groups:
    - name: chaos-experiments
      interval: 30s
      rules:
        - alert: ChaosExperimentHighErrorRate
          expr: |
            rate(chaosexperiment_errors_total[5m]) > 0.1
          for: 5m
          labels:
            severity: warning
          annotations:
            summary: "High error rate in chaos experiments"
            description: "Chaos experiments are experiencing {{ $value }} errors per second"

        - alert: ChaosExperimentFailureRate
          expr: |
            (
              sum(rate(chaosexperiment_executions_total{status="failure"}[5m]))
              /
              sum(rate(chaosexperiment_executions_total[5m]))
            ) > 0.2
          for: 10m
          labels:
            severity: critical
          annotations:
            summary: "Chaos experiment failure rate above 20%"
            description: "{{ $value | humanizePercentage }} of chaos experiments are failing"

        - alert: TooManyResourcesAffected
          expr: |
            sum(chaosexperiment_resources_affected) > 100
          for: 5m
          labels:
            severity: warning
          annotations:
            summary: "Too many resources affected by chaos experiments"
            description: "{{ $value }} resources are currently affected by chaos experiments"
```

---

## Troubleshooting

### Dashboard shows "No data"

1. **Check Prometheus datasource**:
   - Grafana → Configuration → Data sources
   - Test the Prometheus connection
   - Verify URL: `http://prometheus-operated.monitoring.svc:9090`

2. **Check metrics are being exported**:
   ```bash
   # Access controller metrics directly
   kubectl port-forward -n k8s-chaos-system \
     deployment/k8s-chaos-controller-manager 8080:8080

   # Check metrics endpoint
   curl http://localhost:8080/metrics | grep chaosexperiment
   ```

3. **Check Prometheus is scraping**:
   - Open Prometheus UI
   - Status → Targets
   - Look for k8s-chaos-controller target
   - Should show state "UP"

4. **Check time range**:
   - Dashboards default to last 1-6 hours
   - If no experiments ran recently, extend time range
   - Or run a test experiment

### Panels show "Error"

1. **Check PromQL query syntax**:
   - Click panel title → Edit
   - Check query for syntax errors
   - Test query in Prometheus UI first

2. **Check label names**:
   - Metrics labels may have changed
   - Verify actual label names in Prometheus

3. **Check Grafana version compatibility**:
   - Dashboards tested with Grafana 10.x
   - Older versions may have different panel options

### Performance Issues

1. **Reduce refresh rate**:
   - Change from 30s to 1m or 5m
   - Dashboard settings → Auto refresh

2. **Limit time range**:
   - Use shorter time windows (last 1h instead of 24h)

3. **Optimize queries**:
   - Use recording rules in Prometheus for complex queries
   - Pre-aggregate frequently used queries

---

## Best Practices

1. **Dashboard Organization**:
   - Use folders to organize dashboards
   - Default folder: "Chaos Engineering"

2. **Access Control**:
   - Limit edit permissions to chaos engineering team
   - Give view access to wider audience

3. **Alerting**:
   - Create alerts in Prometheus, not Grafana
   - Use PrometheusRule CRDs for GitOps

4. **Variables**:
   - Use Grafana variables for dynamic filtering
   - Add team/environment variables as needed

5. **Annotations**:
   - Add annotations for incidents or changes
   - Correlate chaos experiments with system behavior

---

## Examples

### Tracking a Specific Experiment

1. Open **K8s Chaos - Detailed Analysis**
2. Set filters:
   - Action: `pod-kill`
   - Namespace: `demo`
3. Observe:
   - Execution rate trends
   - Success/failure ratio
   - Resources affected over time

### Safety Review

1. Open **K8s Chaos - Safety Monitoring**
2. Check:
   - Error distribution (should be low)
   - Resources affected (should be within limits)
   - Namespace spread (not concentrated in production)
3. If issues found:
   - Review error types
   - Adjust experiment count or maxPercentage
   - Consider using dryRun mode first

### Performance Analysis

1. Open **K8s Chaos - Overview**
2. Check duration percentiles:
   - p95 should be consistent
   - Spikes indicate controller issues
3. Compare actions:
   - pod-kill should be fastest
   - pod-cpu-stress may take longer (by design)

---

## Additional Resources

- [Prometheus Operator Documentation](https://prometheus-operator.dev/)
- [Grafana Provisioning Docs](https://grafana.com/docs/grafana/latest/administration/provisioning/)
- [PromQL Basics](https://prometheus.io/docs/prometheus/latest/querying/basics/)
- [K8s Chaos Metrics Documentation](METRICS.md)

---

## Support

For issues or questions:
- GitHub Issues: https://github.com/neogan74/k8s-chaos/issues
- Documentation: https://github.com/neogan74/k8s-chaos/docs/
