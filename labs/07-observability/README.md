# Lab 07: Observability

## Objectives
After completing this lab, you will be able to:
- [ ] Access Prometheus metrics from the operator
- [ ] Understand available chaos metrics
- [ ] Deploy and configure Grafana dashboards
- [ ] Query experiment history records
- [ ] Monitor chaos activity in real-time

## Prerequisites
- Completed Labs 01-06
- k8s-chaos operator installed and running
- kubectl configured

## Lab Duration
Estimated time: 30-35 minutes

---

## Overview: Observability Stack

k8s-chaos provides three observability mechanisms:

| Component | Purpose | Access |
|-----------|---------|--------|
| **Prometheus Metrics** | Real-time operational data | `/metrics` endpoint |
| **Grafana Dashboards** | Visualization and alerting | Pre-built dashboards |
| **Experiment History** | Audit trail and analysis | ChaosExperimentHistory CRD |

---

## Step 1: Setup Lab Environment

```bash
cd labs/07-observability
make setup
```

This deploys:
- Demo application
- Prometheus (if not already installed)
- Grafana with pre-configured dashboards

---

## Step 2: Access Prometheus Metrics

The operator exposes metrics on port 8080:

```bash
# Port-forward to metrics endpoint
kubectl port-forward -n k8s-chaos-system deployment/k8s-chaos-controller-manager 8080:8080 &

# Query raw metrics
curl http://localhost:8080/metrics | grep chaos
```

**Key metrics:**

| Metric | Type | Description |
|--------|------|-------------|
| `chaos_experiments_total` | Counter | Total experiments by action/status |
| `chaos_experiment_duration_seconds` | Histogram | Execution duration |
| `chaos_resources_affected_total` | Counter | Resources affected by chaos |
| `chaos_experiment_errors_total` | Counter | Errors by type |
| `chaos_experiments_active` | Gauge | Currently running experiments |

Example queries:
```bash
# Total successful pod-kills
curl -s http://localhost:8080/metrics | grep 'chaos_experiments_total.*action="pod-kill".*status="success"'

# Current active experiments
curl -s http://localhost:8080/metrics | grep chaos_experiments_active
```

---

## Step 3: Understanding Metrics

### Experiments Total
```promql
chaos_experiments_total{action="pod-kill", status="success"}
```
Tracks how many experiments of each type completed successfully or failed.

### Duration Histogram
```promql
histogram_quantile(0.99, rate(chaos_experiment_duration_seconds_bucket[5m]))
```
P99 execution time - useful for SLOs and alerting.

### Resources Affected
```promql
sum(rate(chaos_resources_affected_total[1h])) by (action)
```
How many pods/nodes were affected per hour by action type.

### Errors
```promql
sum(rate(chaos_experiment_errors_total[1h])) by (error_type)
```
Error frequency - useful for debugging and alerting.

---

## Step 4: Access Grafana

```bash
# Port-forward to Grafana
kubectl port-forward -n monitoring svc/grafana 3000:3000 &

# Open in browser
echo "Open http://localhost:3000"
echo "Default credentials: admin / admin"
```

Navigate to Dashboards > Browse > k8s-chaos folder.

---

## Step 5: Explore Dashboards

### Overview Dashboard
High-level chaos activity:
- Total experiments executed
- Success rate percentage
- Active experiments count
- Experiments by action type (pie chart)
- Duration percentiles over time

### Detailed Analysis Dashboard
Deep-dive with filters:
- Filter by action type (pod-kill, pod-delay, etc.)
- Filter by namespace
- Execution rate by status
- Resources affected per experiment
- Error breakdown by type

### Safety Monitoring Dashboard
Focus on safety and impact:
- Error rate and trends
- Resources currently affected
- Namespace-level activity
- Active experiments table
- Health gauge

---

## Step 6: Run Experiments and Watch Metrics

Open two terminals:

**Terminal 1**: Watch Grafana dashboard (keep port-forward running)

**Terminal 2**: Run experiments:
```bash
# Apply pod-kill experiment
kubectl apply -f experiments/01-metrics-demo.yaml

# Wait a few seconds, then apply more
kubectl apply -f experiments/02-cpu-stress-demo.yaml

# Watch experiments
kubectl get chaosexperiments -n chaos-lab -w
```

Watch the Grafana dashboard update in real-time.

Clean up:
```bash
kubectl delete chaosexperiments --all -n chaos-lab
```

---

## Step 7: Query Experiment History

ChaosExperimentHistory records provide an audit trail:

```bash
# List all history records
kubectl get chaosexperimenthistory -n k8s-chaos-system

# Get details of a specific record
kubectl describe chaosexperimenthistory <name> -n k8s-chaos-system

# Query by experiment name
kubectl get cehist -n k8s-chaos-system -l chaos.gushchin.dev/experiment=metrics-demo

# Query by action type
kubectl get cehist -n k8s-chaos-system -l chaos.gushchin.dev/action=pod-kill

# Query by status
kubectl get cehist -n k8s-chaos-system -l chaos.gushchin.dev/status=success
```

### History Record Fields

```yaml
spec:
  experimentName: metrics-demo
  experimentNamespace: chaos-lab
  action: pod-kill
  targetNamespace: chaos-lab
  executionTime: "2024-01-15T10:30:00Z"
  duration: "2.5s"
  status: success
  affectedResources:
    - kind: Pod
      name: nginx-demo-abc123
      namespace: chaos-lab
  auditInfo:
    triggeredBy: schedule
    operatorVersion: v0.1.0
```

---

## Step 8: Create Custom Alerts

Example Prometheus alert rules:

```yaml
# Alert on high error rate
- alert: ChaosExperimentHighErrorRate
  expr: rate(chaos_experiment_errors_total[5m]) > 0.1
  for: 10m
  labels:
    severity: warning
  annotations:
    summary: "High chaos experiment error rate"

# Alert on long-running experiments
- alert: ChaosExperimentRunningTooLong
  expr: chaos_experiments_active > 0 and time() - chaos_experiment_start_time > 3600
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "Chaos experiment running for over 1 hour"
```

---

## Step 9: Useful PromQL Queries

```promql
# Success rate over last hour
sum(rate(chaos_experiments_total{status="success"}[1h])) / sum(rate(chaos_experiments_total[1h]))

# Experiments per action type
sum by (action) (increase(chaos_experiments_total[24h]))

# Average duration by action
histogram_quantile(0.5, sum by (action, le) (rate(chaos_experiment_duration_seconds_bucket[1h])))

# Pods affected per hour
sum(increase(chaos_resources_affected_total{resource_type="pod"}[1h]))

# Error trend
sum(rate(chaos_experiment_errors_total[5m])) by (error_type)
```

---

## Step 10: Cleanup

```bash
make teardown
```

---

## What You Learned

- Operator exposes Prometheus metrics on `/metrics`
- Key metrics: experiments_total, duration, resources_affected, errors
- Three Grafana dashboards: Overview, Detailed, Safety
- ChaosExperimentHistory provides audit trail
- Label-based queries enable flexible history analysis

## Next Steps

- **Lab 08**: Advanced multi-experiment scenarios
- Integrate with your existing monitoring stack
- Create custom dashboards and alerts

## Troubleshooting

**No metrics appearing?**
- Verify operator is running: `kubectl get pods -n k8s-chaos-system`
- Check metrics endpoint: `curl http://localhost:8080/metrics`
- Ensure Prometheus is scraping the target

**Grafana dashboards empty?**
- Verify Prometheus datasource is configured
- Check dashboard time range
- Run some experiments to generate data

**History records not created?**
- Check operator logs for history errors
- Verify history is enabled (--history-enabled flag)
- Check k8s-chaos-system namespace for records