# ADR 0006: Experiment History and Audit Logging

**Status**: Accepted
**Date**: 2025-11-21
**Author**: k8s-chaos team

## Context

The k8s-chaos operator currently tracks experiment state in the ChaosExperiment status, but this information is ephemeral and limited:

1. **No historical record**: Once an experiment completes or is deleted, all execution history is lost
2. **Limited observability**: Cannot answer questions like "Which experiments ran last week?" or "What resources were affected yesterday?"
3. **No audit trail**: Compliance and security teams need to know who ran what experiments and when
4. **Debugging challenges**: Difficult to troubleshoot issues when historical execution data is unavailable
5. **No trending**: Cannot analyze experiment success rates or failure patterns over time

### User Stories

**As a chaos engineer**, I want to:
- See a history of all experiments that ran in the past month
- Understand which resources were affected by previous experiments
- Analyze failure patterns to improve experiment design

**As a compliance officer**, I need to:
- Audit all chaos experiments for regulatory compliance
- Track who initiated which experiments
- Prove that chaos testing follows approved procedures

**As a platform engineer**, I want to:
- Troubleshoot issues by reviewing past experiment executions
- Generate reports on chaos testing coverage
- Set up alerts based on experiment failure trends

## Decision

We will implement **experiment history and audit logging** using a dedicated CRD called `ChaosExperimentHistory` that automatically records each experiment execution with comprehensive metadata.

### Implementation Approach

**Method**: Dedicated CRD for History Records
- Create `ChaosExperimentHistory` CRD to store execution records
- Controller automatically creates history record after each experiment execution
- History records are immutable (write-once, read-many)
- Support configurable retention policies via TTL

### ChaosExperimentHistory CRD Structure

```yaml
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperimentHistory
metadata:
  name: experiment-name-20251121-143022-abc123
  namespace: chaos-system
  labels:
    chaos.gushchin.dev/experiment: experiment-name
    chaos.gushchin.dev/action: pod-kill
    chaos.gushchin.dev/target-namespace: default
    chaos.gushchin.dev/status: success
spec:
  # Reference to the original experiment
  experimentRef:
    name: experiment-name
    namespace: chaos-system
    uid: "abc-123-def"

  # Experiment configuration at time of execution
  experimentSpec:
    action: pod-kill
    namespace: default
    selector:
      app: web-server
    count: 2
    # ... full spec snapshot

  # Execution details
  execution:
    startTime: "2025-11-21T14:30:22Z"
    endTime: "2025-11-21T14:30:25Z"
    duration: "3s"
    status: "success"  # success, failure, partial
    message: "Successfully killed 2 pod(s)"

  # Resources affected
  affectedResources:
    - kind: Pod
      name: web-server-abc123
      namespace: default
      action: deleted
    - kind: Pod
      name: web-server-def456
      namespace: default
      action: deleted

  # Metadata for auditing
  audit:
    initiatedBy: "system:serviceaccount:chaos-system:chaos-controller"
    scheduledExecution: true  # vs manual trigger
    dryRun: false
    retryCount: 0

  # Error information (if failed)
  error:
    message: ""
    code: ""
    lastError: ""
```

### Operational Behavior

1. **Automatic Recording**: Controller creates history record after each experiment execution
2. **Immutable Records**: Once created, history records cannot be modified
3. **Label-based Indexing**: Labels enable efficient querying by experiment, action, status, etc.
4. **Retention Management**: Optional TTL controller deletes old history records
5. **Namespace Isolation**: History records stored in operator namespace (chaos-system)

### Configuration Options

Add to operator configuration:

```yaml
# Enable/disable history recording
historyEnabled: true

# Number of history records to retain per experiment
historyLimit: 100

# Time-based retention (optional, overrides historyLimit)
historyTTL: "30d"  # Delete records older than 30 days

# History namespace (where to store records)
historyNamespace: "chaos-system"
```

### Querying History

Users can query history using standard kubectl:

```bash
# List all history records
kubectl get chaosexperimenthistory -n chaos-system

# Get history for specific experiment
kubectl get chaosexperimenthistory -n chaos-system \
  -l chaos.gushchin.dev/experiment=my-experiment

# Get failed experiments
kubectl get chaosexperimenthistory -n chaos-system \
  -l chaos.gushchin.dev/status=failure

# Get experiments for specific namespace
kubectl get chaosexperimenthistory -n chaos-system \
  -l chaos.gushchin.dev/target-namespace=production
```

## Alternatives Considered

### Alternative 1: Store History in ChaosExperiment Status
**Approach**: Add `history[]` array to ChaosExperiment status field

**Pros**:
- No new CRD needed
- History co-located with experiment
- Simple to implement

**Cons**:
- Status field size limits (etcd has 1.5MB limit)
- History lost when experiment is deleted
- Difficult to query across experiments
- Performance impact on status updates

**Decision**: Rejected - Too limiting for long-term history

### Alternative 2: Kubernetes Events
**Approach**: Use native Kubernetes Event objects

**Pros**:
- Native Kubernetes mechanism
- Built-in retention and cleanup
- Well-understood by operators

**Cons**:
- Events are ephemeral (1 hour default retention)
- Limited metadata structure
- Not designed for audit logging
- Difficult to query and aggregate

**Decision**: Rejected - Events too short-lived for meaningful history

### Alternative 3: External Logging System
**Approach**: Send history to external system (Loki, Elasticsearch, S3)

**Pros**:
- Unlimited retention
- Advanced querying and analytics
- No etcd storage pressure
- Centralized logging

**Cons**:
- Requires external dependencies
- More complex setup and maintenance
- Not Kubernetes-native
- Network dependency for history recording

**Decision**: Rejected for core feature, but could be future enhancement

### Alternative 4: ConfigMaps for History
**Approach**: Store history in ConfigMaps

**Pros**:
- No new CRD
- Simple to implement

**Cons**:
- ConfigMap size limits (1MB)
- Not semantically correct use of ConfigMaps
- Poor query performance
- Manual management required

**Decision**: Rejected - ConfigMaps not designed for this use case

### Alternative 5: Combined Approach - CRD + Events
**Approach**: Create detailed CRD records AND emit Kubernetes Events

**Pros**:
- Best of both worlds
- Events for real-time monitoring
- CRD for long-term audit

**Cons**:
- More implementation complexity
- Potential consistency issues

**Decision**: Accepted - Implement CRD now, add Events in future phase

## Consequences

### Positive
- **Complete audit trail**: Full history of all experiment executions
- **Compliance ready**: Meets regulatory requirements for audit logging
- **Debugging enabled**: Historical data helps troubleshoot issues
- **Kubernetes-native**: Uses CRDs, no external dependencies
- **Queryable**: Standard kubectl commands for accessing history
- **Flexible retention**: Configurable via TTL and limits
- **Performance isolated**: History recording doesn't block experiment execution
- **Immutable records**: Cannot be tampered with, increasing trust

### Negative
- **etcd storage**: History records consume etcd storage
- **Controller complexity**: Additional reconciliation logic needed
- **Resource overhead**: One history record per execution
- **Cleanup required**: Need retention management to prevent unbounded growth
- **Query performance**: Large result sets may be slow without pagination

### Risks and Mitigations

**Risk**: History records consume too much etcd storage
- **Mitigation**: Implement configurable retention limits
- **Mitigation**: Use TTL to automatically clean up old records
- **Mitigation**: Compress spec by omitting default values
- **Mitigation**: Make history recording optional (disable in dev environments)

**Risk**: High-frequency experiments create too many history records
- **Mitigation**: Implement aggregation for scheduled experiments (daily summary)
- **Mitigation**: Allow per-experiment history opt-out via annotation
- **Mitigation**: Configurable sampling rate (e.g., record every 10th execution)

**Risk**: Query performance degrades with thousands of records
- **Mitigation**: Efficient label indexing
- **Mitigation**: Document pagination best practices
- **Mitigation**: Consider future API server with pagination support

**Risk**: Controller crashes before creating history record
- **Mitigation**: Create history record in same transaction as status update
- **Mitigation**: Controller can reconcile missing history on restart
- **Mitigation**: Not critical - occasional loss acceptable for non-safety-critical feature

## Implementation Details

### API Types

New file: `api/v1alpha1/chaosexperimenthistory_types.go`

```go
type ChaosExperimentHistory struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    Spec   ChaosExperimentHistorySpec   `json:"spec"`
    Status ChaosExperimentHistoryStatus `json:"status,omitempty"`
}

type ChaosExperimentHistorySpec struct {
    ExperimentRef  ObjectReference       `json:"experimentRef"`
    ExperimentSpec ChaosExperimentSpec   `json:"experimentSpec"`
    Execution      ExecutionDetails      `json:"execution"`
    AffectedResources []ResourceReference `json:"affectedResources"`
    Audit          AuditMetadata         `json:"audit"`
    Error          *ErrorDetails         `json:"error,omitempty"`
}

type ExecutionDetails struct {
    StartTime metav1.Time `json:"startTime"`
    EndTime   metav1.Time `json:"endTime"`
    Duration  string      `json:"duration"`
    Status    string      `json:"status"`
    Message   string      `json:"message"`
}

type ResourceReference struct {
    Kind      string `json:"kind"`
    Name      string `json:"name"`
    Namespace string `json:"namespace"`
    Action    string `json:"action"`
}

type AuditMetadata struct {
    InitiatedBy        string `json:"initiatedBy"`
    ScheduledExecution bool   `json:"scheduledExecution"`
    DryRun            bool   `json:"dryRun"`
    RetryCount        int    `json:"retryCount"`
}
```

### Controller Changes

1. **Create history record function**:
   ```go
   func (r *ChaosExperimentReconciler) createHistoryRecord(
       ctx context.Context,
       exp *ChaosExperiment,
       affectedResources []ResourceReference,
   ) error
   ```

2. **Call after each experiment execution**:
   - In `handlePodKill()`, `handlePodDelay()`, etc.
   - Before returning from reconciliation
   - Include all affected resources

3. **Add retention cleanup**:
   - Periodic reconciliation to delete old records
   - Based on historyLimit and historyTTL config

### RBAC Requirements

```yaml
# History CRD management
- apiGroups: ["chaos.gushchin.dev"]
  resources: ["chaosexperimenthistories"]
  verbs: ["create", "get", "list", "watch", "delete"]
```

### Metrics Integration

Add Prometheus metrics:
```go
chaos_history_records_total{action,status}
chaos_history_cleanup_total{reason}
chaos_history_storage_bytes
```

## Validation Requirements

### OpenAPI Schema
- ExperimentRef: Required, valid object reference
- Execution: Required with valid timestamps
- Status: Enum (success, failure, partial)
- AffectedResources: Array of valid resource references

### Controller Validation
- Verify history namespace exists
- Validate retention config values
- Check storage capacity before creating records

## Testing Strategy

### Unit Tests
- Test history record creation
- Test retention policy enforcement
- Test label assignment
- Test query performance with large datasets

### Integration Tests
- Verify history created for each experiment type
- Test history survives experiment deletion
- Verify retention cleanup works
- Test concurrent history creation

### Manual Testing
- Query history with various label selectors
- Verify audit metadata accuracy
- Test with high-frequency experiments
- Check etcd storage impact

## Implementation Checklist

- [x] Create ChaosExperimentHistory CRD types
- [x] Add OpenAPI validation markers
- [x] Generate CRD manifests (make manifests)
- [x] Implement createHistoryRecord() function
- [x] Add history recording to all action handlers
- [x] Implement retention/cleanup logic
- [x] Add history configuration to operator config
- [x] Update RBAC with history permissions
- [x] Add Prometheus metrics
- [x] Create sample history queries documentation (docs/HISTORY.md)
- [ ] Add unit tests
- [ ] Add integration tests
- [x] Update CLAUDE.md (marked as completed in project instructions)
- [x] Create usage examples (config/samples/chaos_v1alpha1_chaosexperimenthistory_examples.yaml)

## Migration Path

- Fully backward compatible
- History recording disabled by default initially
- Can be enabled via operator configuration
- No impact on existing experiments
- No schema changes to ChaosExperiment CRD

## Future Enhancements

### Phase 2: Advanced Querying
- Custom API endpoint for complex queries
- Aggregation and analytics API
- History export to CSV/JSON

### Phase 3: External Integration
- Export to external logging systems (Loki, Elasticsearch)
- Webhook notifications on experiment completion
- Slack/email notifications with history links

### Phase 4: Visualization
- Web UI for browsing history
- Timeline visualization
- Experiment success rate dashboard

### Phase 5: Advanced Retention
- Intelligent retention (keep failures longer than successes)
- Compression for old records
- Archival to external storage

## References

- [Kubernetes API Conventions - Status](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties)
- [etcd Storage Limits](https://etcd.io/docs/v3.5/dev-guide/limit/)
- [Kubernetes Events](https://kubernetes.io/docs/reference/kubernetes-api/cluster-resources/event-v1/)
- [Audit Logging Best Practices](https://kubernetes.io/docs/tasks/debug/debug-cluster/audit/)
- [Time-to-Live (TTL) Controller](https://kubernetes.io/docs/concepts/workloads/controllers/ttlafterfinished/)
- [CRD Validation](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#validation)
