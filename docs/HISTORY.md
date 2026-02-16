# Experiment History and Audit Logging

The k8s-chaos operator automatically records detailed history of all chaos experiment executions through the `ChaosExperimentHistory` CRD. This provides an immutable audit trail for compliance, debugging, and analysis.

## Overview

Every time a chaos experiment runs, the operator creates a `ChaosExperimentHistory` record containing:
- Complete experiment configuration at execution time
- Start/end timestamps and duration
- List of affected resources (pods, nodes)
- Execution status (success, failure, partial)
- Audit metadata (who initiated, scheduled vs manual)
- Error details if the experiment failed

History records are:
- **Immutable** - Cannot be modified after creation
- **Labeled** - Efficiently queryable using Kubernetes labels
- **Retained** - Configurable retention policies prevent unbounded growth
- **Namespace-scoped** - Stored in a dedicated namespace (default: `chaos-system`)

## Configuration

Configure history recording via operator flags:

```bash
# Enable/disable history recording (default: true)
--history-enabled=true

# Namespace to store history records (default: chaos-system)
--history-namespace=chaos-system

# Maximum records per experiment (default: 100)
--history-retention-limit=100

# Time-to-live for history records (default: 720h / 30 days, 0 = disabled)
--history-ttl=720h
```

Example deployment with custom history configuration:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: chaos-controller-manager
spec:
  template:
    spec:
      containers:
      - name: manager
        args:
        - --history-enabled=true
        - --history-namespace=chaos-history
        - --history-retention-limit=200
        - --history-ttl=2160h  # 90 days for compliance
```

## Querying History

### Basic Queries

List all history records:
```bash
kubectl get chaosexperimenthistory -n chaos-system
# Short form:
kubectl get cehist -n chaos-system
```

Get history for specific experiment:
```bash
kubectl get cehist -n chaos-system \
  -l chaos.gushchin.dev/experiment=my-experiment
```

### Query by Status

Find failed experiments:
```bash
kubectl get cehist -n chaos-system \
  -l chaos.gushchin.dev/status=failure
```

Find successful experiments:
```bash
kubectl get cehist -n chaos-system \
  -l chaos.gushchin.dev/status=success
```

### Query by Action Type

Pod kill experiments:
```bash
kubectl get cehist -n chaos-system \
  -l chaos.gushchin.dev/action=pod-kill
```

CPU stress experiments:
```bash
kubectl get cehist -n chaos-system \
  -l chaos.gushchin.dev/action=pod-cpu-stress
```

### Query by Target Namespace

Experiments affecting production:
```bash
kubectl get cehist -n chaos-system \
  -l chaos.gushchin.dev/target-namespace=production
```

### Combined Queries

Failed pod-kill experiments in staging:
```bash
kubectl get cehist -n chaos-system \
  -l chaos.gushchin.dev/action=pod-kill,\
chaos.gushchin.dev/target-namespace=staging,\
chaos.gushchin.dev/status=failure
```

## Viewing Details

Get full details of a history record:
```bash
kubectl get cehist <history-name> -n chaos-system -o yaml
```

Key sections in the history record:

### Experiment Reference
```yaml
spec:
  experimentRef:
    name: my-experiment
    namespace: chaos-system
    uid: "550e8400-e29b-41d4-a716-446655440000"
```

### Execution Details
```yaml
spec:
  execution:
    startTime: "2025-11-21T14:30:22Z"
    endTime: "2025-11-21T14:30:25Z"
    duration: "3.2s"
    status: "success"
    message: "Successfully killed 2 pod(s)"
    phase: "Completed"
```

### Affected Resources
```yaml
spec:
  affectedResources:
  - kind: Pod
    name: web-server-abc123
    namespace: default
    action: deleted
  - kind: Pod
    name: web-server-def456
    namespace: default
    action: deleted
```

### Audit Information
```yaml
spec:
  audit:
    initiatedBy: "system:serviceaccount:chaos-system:chaos-controller"
    scheduledExecution: true
    dryRun: false
    retryCount: 0
```

### Error Details (if failed)
```yaml
spec:
  error:
    message: "No eligible pods found matching selector"
    code: "NO_PODS_FOUND"
    failureReason: "ResourceNotFound"
```

## Analyzing History

### Count experiments by status
```bash
kubectl get cehist -n chaos-system \
  -l chaos.gushchin.dev/experiment=my-experiment \
  -o jsonpath='{range .items[*]}{.spec.execution.status}{"\n"}{end}' | \
  sort | uniq -c
```

### List recent failures
```bash
kubectl get cehist -n chaos-system \
  -l chaos.gushchin.dev/status=failure \
  --sort-by='.metadata.creationTimestamp' | tail -n 10
```

### Get affected resources from last run
```bash
kubectl get cehist -n chaos-system \
  -l chaos.gushchin.dev/experiment=my-experiment \
  --sort-by='.metadata.creationTimestamp' | tail -n 1 | \
  kubectl get -o jsonpath='{.spec.affectedResources[*].name}'
```

## Retention and Cleanup

The operator implements dual-strategy automatic cleanup for history records:

### Count-Based Cleanup (Retention Limit)

Deletes oldest records when they exceed the per-experiment limit:

1. Runs after each experiment execution
2. Counts records for the specific experiment
3. Deletes oldest records if count exceeds `--history-retention-limit`
4. Tracks cleanup via `chaosexperiment_history_cleanup_total{reason="retention_limit"}` metric

### TTL-Based Cleanup (Time-To-Live)

Deletes records older than the configured TTL duration:

1. Runs periodically every hour in the background
2. Also runs after each experiment execution
3. Deletes all records (across all experiments) older than `--history-ttl`
4. Tracks cleanup via `chaosexperiment_history_cleanup_total{reason="ttl_expired"}` metric

**Both strategies work independently** - a record is deleted if it violates either limit.

### TTL Configuration Examples

```bash
# Default: 30 days TTL
--history-ttl=720h

# Compliance: Keep records for 90 days
--history-ttl=2160h

# Aggressive cleanup: Keep only 7 days
--history-ttl=168h

# Disable TTL cleanup (count-based only)
--history-ttl=0
```

**Minimum TTL:** 1 hour (values below 1h are rejected)
**Warning:** TTL values less than 24h may cause aggressive cleanup

### Manual Cleanup

For ad-hoc cleanup needs:

```bash
# Delete all history for an experiment
kubectl delete cehist -n chaos-system \
  -l chaos.gushchin.dev/experiment=old-experiment

# Delete old records (older than 30 days)
kubectl get cehist -n chaos-system -o json | \
  jq -r '.items[] | select(.metadata.creationTimestamp < (now - 2592000 | todateiso8601)) | .metadata.name' | \
  xargs kubectl delete cehist -n chaos-system
```

## Prometheus Metrics

The operator exports metrics for history operations:

- `chaosexperiment_history_records_total{action,status}` - Total history records created
- `chaosexperiment_history_cleanup_total{reason}` - Total records deleted by retention policies
  - `reason="retention_limit"` - Deleted due to count-based cleanup
  - `reason="ttl_expired"` - Deleted due to TTL-based cleanup
- `chaosexperiment_history_records_count{experiment,namespace}` - Current count per experiment

Query examples (PromQL):

```promql
# History creation rate
rate(chaosexperiment_history_records_total[5m])

# Overall cleanup rate
rate(chaosexperiment_history_cleanup_total[1h])

# TTL cleanup rate specifically
rate(chaosexperiment_history_cleanup_total{reason="ttl_expired"}[1h])

# Count-based cleanup rate
rate(chaosexperiment_history_cleanup_total{reason="retention_limit"}[1h])

# Current history count per experiment
chaosexperiment_history_records_count

# Total records cleaned up by reason
sum by (reason) (chaosexperiment_history_cleanup_total)
```

## Use Cases

### Compliance Auditing
```bash
# Generate audit report for last 7 days
kubectl get cehist -n chaos-system \
  --field-selector 'metadata.creationTimestamp>2025-11-14T00:00:00Z' \
  -o custom-columns=\
TIME:.metadata.creationTimestamp,\
EXPERIMENT:.spec.experimentRef.name,\
ACTION:.spec.experimentSpec.action,\
TARGET:.spec.experimentSpec.namespace,\
STATUS:.spec.execution.status,\
INITIATED:.spec.audit.initiatedBy
```

### Debugging Failed Experiments
```bash
# Find pattern in recent failures
kubectl get cehist -n chaos-system \
  -l chaos.gushchin.dev/status=failure \
  -o jsonpath='{range .items[*]}{.spec.error.failureReason}{"\t"}{.spec.error.message}{"\n"}{end}' | \
  sort | uniq -c | sort -rn
```

### Resource Impact Analysis
```bash
# Count total pods affected by an experiment
kubectl get cehist -n chaos-system \
  -l chaos.gushchin.dev/experiment=my-experiment \
  -o json | jq '[.items[].spec.affectedResources | length] | add'
```

## Best Practices

1. **Retention Configuration**: Set retention limits based on compliance requirements and storage capacity
2. **Namespace Isolation**: Use dedicated namespace for history to separate concerns
3. **Regular Audits**: Periodically review history for unusual patterns
4. **Metric Monitoring**: Set up alerts on history cleanup rate to detect issues
5. **Backup Strategy**: Consider backing up critical history records to external storage

## Troubleshooting

### History not being created

Check if history is enabled:
```bash
kubectl logs -n chaos-system deployment/chaos-controller-manager | grep history
```

Verify RBAC permissions:
```bash
kubectl get clusterrole chaos-manager-role -o yaml | grep -A 5 chaosexperimenthistories
```

### History namespace not found

Ensure the history namespace exists:
```bash
kubectl create namespace chaos-system
```

Or configure a different namespace:
```bash
--history-namespace=my-custom-namespace
```

### Too many history records

Reduce retention limit:
```bash
--history-retention-limit=50
```

Or manually clean up old records as shown above.

## Related Documentation

- [ADR 0006: Experiment History and Audit Logging](adr/0006-experiment-history-and-audit-logging.md)
- [Metrics Documentation](METRICS.md)
- [ChaosExperiment API Reference](../api/v1alpha1/chaosexperiment_types.go)
