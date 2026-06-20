# Metrics

k8s-chaos exports Prometheus metrics for monitoring chaos experiments. These metrics help you track experiment execution, success rates, resource impact, and potential issues.

## Available Metrics

### Experiment Execution Metrics

#### `chaosexperiment_executions_total`
**Type:** Counter
**Labels:**
- `action`: Type of chaos action (pod-kill, pod-delay, node-drain)
- `namespace`: Target namespace
- `status`: Experiment result (success, failure)

**Description:** Total number of chaos experiments executed.

**Example queries:**
```promql
# Total experiments executed
sum(chaosexperiment_executions_total)

# Success rate by action
rate(chaosexperiment_executions_total{status="success"}[5m]) /
rate(chaosexperiment_executions_total[5m])

# Failed experiments in the last hour
increase(chaosexperiment_executions_total{status="failure"}[1h])
```

#### `chaosexperiment_duration_seconds`
**Type:** Histogram
**Labels:**
- `action`: Type of chaos action
- `namespace`: Target namespace

**Description:** Duration of chaos experiment execution in seconds.

**Example queries:**
```promql
# Average experiment duration
avg(rate(chaosexperiment_duration_seconds_sum[5m]) /
    rate(chaosexperiment_duration_seconds_count[5m]))

# P95 duration by action
histogram_quantile(0.95,
  sum(rate(chaosexperiment_duration_seconds_bucket[5m])) by (action, le))

# Slow experiments (>10s)
chaosexperiment_duration_seconds_bucket{le="10"} > 0
```

#### `chaosexperiment_resources_affected`
**Type:** Gauge
**Labels:**
- `action`: Type of chaos action
- `namespace`: Target namespace
- `experiment`: Name of the ChaosExperiment resource

**Description:** Number of resources (pods/nodes) currently affected by chaos experiments.

**Example queries:**
```promql
# Total resources currently affected
sum(chaosexperiment_resources_affected)

# Resources affected by action type
sum(chaosexperiment_resources_affected) by (action)

# Resources affected in specific namespace
chaosexperiment_resources_affected{namespace="production"}
```

#### `chaosexperiment_errors_total`
**Type:** Counter
**Labels:**
- `action`: Type of chaos action
- `namespace`: Target namespace
- `error_type`: Type of error encountered

**Description:** Total number of errors during chaos experiments.

**Error Type Values:**
- `permission` - RBAC or authentication failures (403 Forbidden, 401 Unauthorized)
- `execution` - Runtime errors during chaos injection
- `validation` - Invalid experiment configuration
- `timeout` - Operation timeouts
- `unknown` - Uncategorized errors

**Example queries:**
```promql
# Error rate
rate(chaosexperiment_errors_total[5m])

# Errors by type
sum(chaosexperiment_errors_total) by (error_type)

# Permission errors by action
sum(chaosexperiment_errors_total{error_type="permission"}) by (action)

# Permission error rate
rate(chaosexperiment_errors_total{error_type="permission"}[5m])

# Actions with highest permission failures
topk(5, sum(chaosexperiment_errors_total{error_type="permission"}) by (action, namespace))

# Errors in last 24 hours
increase(chaosexperiment_errors_total[24h])

# Distribution of error types
sum(chaosexperiment_errors_total) by (error_type)
```

**Common Use Cases:**

*Monitoring RBAC Issues:*
```promql
# Alert on permission errors
sum(increase(chaosexperiment_errors_total{error_type="permission"}[5m])) > 0
```

*Identifying Problematic Actions:*
```promql
# Actions with most errors
topk(3, sum(rate(chaosexperiment_errors_total[1h])) by (action))
```

*Error Distribution Analysis:*
```promql
# Percentage of errors by type
(
  sum(chaosexperiment_errors_total) by (error_type)
  / ignoring(error_type) group_left
  sum(chaosexperiment_errors_total)
) * 100
```

#### `chaosexperiment_active`
**Type:** Gauge
**Labels:**
- `action`: Type of chaos action

**Description:** Number of currently active (running) chaos experiments.

**Example queries:**
```promql
# Currently running experiments
sum(chaosexperiment_active)

# Active experiments by action
chaosexperiment_active

# Alert on too many concurrent experiments
chaosexperiment_active > 10
```

## Enabling Metrics

The metrics endpoint is configured via command-line flags when starting the controller:

### HTTP (Insecure - Development Only)
```bash
./manager --metrics-bind-address=:8080 --metrics-secure=false
```

Metrics will be available at: `http://localhost:8080/metrics`

### HTTPS (Secure - Production)
```bash
./manager --metrics-bind-address=:8443 --metrics-secure=true
```

Metrics will be available at: `https://localhost:8443/metrics` (requires authentication)

### Disable Metrics
```bash
./manager --metrics-bind-address=0
```

## Prometheus Configuration

### ServiceMonitor (with Prometheus Operator)

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: k8s-chaos-controller-manager
  namespace: k8s-chaos-system
spec:
  endpoints:
  - path: /metrics
    port: https
    scheme: https
    bearerTokenFile: /var/run/secrets/kubernetes.io/serviceaccount/token
    tlsConfig:
      insecureSkipVerify: true
  selector:
    matchLabels:
      control-plane: controller-manager
```

### Scrape Config (Standalone Prometheus)

```yaml
scrape_configs:
  - job_name: 'k8s-chaos'
    kubernetes_sd_configs:
    - role: endpoints
      namespaces:
        names:
        - k8s-chaos-system
    relabel_configs:
    - source_labels: [__meta_kubernetes_service_label_control_plane]
      action: keep
      regex: controller-manager
    - source_labels: [__meta_kubernetes_endpoint_port_name]
      action: keep
      regex: https
```

## Grafana Dashboards

### Example Dashboard Panels

**Experiments Over Time:**
```promql
sum(rate(chaosexperiment_executions_total[5m])) by (action)
```

**Success Rate:**
```promql
sum(rate(chaosexperiment_executions_total{status="success"}[5m])) /
sum(rate(chaosexperiment_executions_total[5m])) * 100
```

**Average Duration:**
```promql
avg(rate(chaosexperiment_duration_seconds_sum[5m]) /
    rate(chaosexperiment_duration_seconds_count[5m])) by (action)
```

**Active Experiments:**
```promql
sum(chaosexperiment_active)
```

**Resources Under Chaos:**
```promql
sum(chaosexperiment_resources_affected) by (namespace)
```

## Alerting Rules

### Example Prometheus Alerts

```yaml
groups:
- name: chaos_experiments
  rules:
  # High failure rate
  - alert: ChaosExperimentHighFailureRate
    expr: |
      rate(chaosexperiment_executions_total{status="failure"}[5m]) /
      rate(chaosexperiment_executions_total[5m]) > 0.5
    for: 10m
    labels:
      severity: warning
    annotations:
      summary: "High chaos experiment failure rate"
      description: "More than 50% of chaos experiments are failing"

  # Slow experiments
  - alert: ChaosExperimentSlowExecution
    expr: |
      histogram_quantile(0.95,
        sum(rate(chaosexperiment_duration_seconds_bucket[5m])) by (le)
      ) > 30
    for: 10m
    labels:
      severity: warning
    annotations:
      summary: "Chaos experiments running slowly"
      description: "P95 experiment duration is above 30 seconds"

  # Too many active experiments
  - alert: TooManyConcurrentChaosExperiments
    expr: sum(chaosexperiment_active) > 5
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "Too many concurrent chaos experiments"
      description: "{{ $value }} chaos experiments are running concurrently"

  # Experiment errors
  - alert: ChaosExperimentErrors
    expr: increase(chaosexperiment_errors_total[5m]) > 10
    labels:
      severity: warning
    annotations:
      summary: "High number of chaos experiment errors"
      description: "{{ $value }} errors in the last 5 minutes"
```

## Accessing Metrics

### Via kubectl port-forward

```bash
# For HTTP metrics
kubectl port-forward -n k8s-chaos-system \
  deployment/k8s-chaos-controller-manager 8080:8080

curl http://localhost:8080/metrics

# For HTTPS metrics (requires auth token)
kubectl port-forward -n k8s-chaos-system \
  deployment/k8s-chaos-controller-manager 8443:8443

TOKEN=$(kubectl create token -n k8s-chaos-system k8s-chaos-controller-manager)
curl -k -H "Authorization: Bearer $TOKEN" https://localhost:8443/metrics
```

### Via Service

Create a service to expose metrics:
```yaml
apiVersion: v1
kind: Service
metadata:
  name: k8s-chaos-metrics
  namespace: k8s-chaos-system
spec:
  selector:
    control-plane: controller-manager
  ports:
  - name: https
    port: 8443
    targetPort: 8443
```

## Troubleshooting

### Metrics Not Available

1. Check if metrics server is enabled:
   ```bash
   kubectl logs -n k8s-chaos-system deployment/k8s-chaos-controller-manager | grep metrics
   ```

2. Verify metrics bind address flag:
   ```bash
   kubectl get deployment -n k8s-chaos-system k8s-chaos-controller-manager -o yaml | grep metrics-bind-address
   ```

3. Check service and endpoint:
   ```bash
   kubectl get svc,ep -n k8s-chaos-system
   ```

### Authentication Issues (HTTPS)

If you get authentication errors when accessing secure metrics:

1. Ensure you're using a valid service account token
2. Check RBAC permissions for the metrics endpoint
3. Verify TLS certificates are valid

### No Custom Metrics Visible

If Prometheus default metrics work but custom chaos metrics don't appear:

1. Check controller logs for metric registration errors
2. Verify the metrics package is imported in main.go
3. Ensure experiments are actually running to generate metrics
