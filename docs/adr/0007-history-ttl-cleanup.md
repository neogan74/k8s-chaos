# ADR 0007: History TTL Cleanup

## Status

Proposed

## Context

The ChaosExperimentHistory feature currently supports retention limit-based cleanup (e.g., keep last 100 records per experiment). However, this approach has limitations:

1. **Storage Growth**: Old records remain indefinitely until count threshold is exceeded
2. **Compliance**: Some organizations require time-based retention policies (e.g., "keep audit logs for 90 days")
3. **Predictability**: TTL provides predictable storage usage independent of experiment frequency

We need to add time-based cleanup (TTL) while maintaining the existing count-based retention limit.

## Decision

Implement TTL-based cleanup for ChaosExperimentHistory records with the following design:

### 1. API Changes

Add optional TTL field to operator configuration (flags):

```go
--history-ttl=720h  // Default: 30 days (720h), 0 = disabled
```

Add timestamp metadata to ChaosExperimentHistory:
- Use existing `metadata.creationTimestamp` (no CRD changes needed)

### 2. Cleanup Strategy

Implement dual-strategy cleanup in the history controller:

**Strategy 1: Count-based (existing)**
- Keep last N records per experiment (controlled by `--history-retention-limit`)

**Strategy 2: TTL-based (new)**
- Delete records older than TTL duration
- Independent of count-based cleanup
- Both strategies run concurrently - records deleted if they violate either limit

**Cleanup Triggers:**
- On every experiment completion (piggyback on existing cleanup)
- Periodic reconciliation every 1 hour for orphaned records

### 3. Implementation Details

**Cleanup Logic:**
```go
func (r *HistoryReconciler) cleanupExpiredHistory(ctx context.Context) error {
    if r.HistoryTTL == 0 {
        return nil // TTL disabled
    }

    expirationTime := time.Now().Add(-r.HistoryTTL)

    // List all history records older than TTL
    listOpts := []client.ListOption{
        client.InNamespace(r.HistoryNamespace),
    }

    var historyList chaosv1alpha1.ChaosExperimentHistoryList
    if err := r.List(ctx, &historyList, listOpts...); err != nil {
        return err
    }

    for _, history := range historyList.Items {
        if history.CreationTimestamp.Time.Before(expirationTime) {
            if err := r.Delete(ctx, &history); err != nil {
                // Log but continue with other records
                continue
            }
            // Update metrics
        }
    }

    return nil
}
```

**Integration Points:**
- Call from existing `cleanupOldHistory()` function after count-based cleanup
- Add periodic reconciliation for TTL cleanup independent of experiment execution

### 4. Metrics Updates

Add new Prometheus metric:

```go
chaos_experiment_history_ttl_deleted_total{reason="ttl_expired"}
```

Update existing metric labels to distinguish cleanup reasons:
- `reason="retention_limit"` - Deleted due to count limit
- `reason="ttl_expired"` - Deleted due to age

### 5. Configuration Validation

- TTL must be >= 1h or 0 (disabled)
- Warning if TTL < 24h (may cause aggressive cleanup)
- Default: 30 days (720h)

### 6. Backward Compatibility

- TTL is optional and disabled by default (0)
- No CRD changes required (uses existing creationTimestamp)
- Existing history records are subject to TTL if enabled
- Both cleanup strategies work independently

## Consequences

### Positive

1. **Compliance**: Supports time-based retention policies required by regulations
2. **Predictable Storage**: Automatic cleanup prevents unbounded growth
3. **Flexibility**: Dual-strategy allows fine-grained control (count + time)
4. **No Breaking Changes**: Fully backward compatible, opt-in feature

### Negative

1. **Complexity**: Two cleanup strategies to maintain and test
2. **Clock Skew**: Relies on system time (k8s timestamps are UTC)
3. **Deleted Data**: Old records deleted automatically (expected behavior, but users must be aware)

### Neutral

1. **Default Behavior**: No change unless TTL explicitly configured
2. **Performance**: Minimal impact (cleanup is efficient label-based query)

## Alternatives Considered

### Alternative 1: TTL Only (Remove Count Limit)

**Rejected**: Count-based limits are useful for low-frequency experiments where time-based cleanup may not be sufficient.

### Alternative 2: Use Kubernetes TTL Controller

**Rejected**: Requires annotation on each resource, less flexible, harder to configure dynamically.

### Alternative 3: External Cleanup Job

**Rejected**: Adds operational complexity, prefer built-in solution.

## Implementation Plan

1. Add `--history-ttl` flag to operator configuration
2. Implement `cleanupExpiredHistory()` function
3. Integrate TTL cleanup with existing cleanup logic
4. Add periodic reconciliation for TTL cleanup
5. Update metrics to track TTL-based deletions
6. Add unit tests for TTL cleanup logic
7. Update documentation (HISTORY.md, CLAUDE.md)
8. Add example configurations

## Testing Strategy

**Unit Tests:**
- TTL parsing and validation
- Cleanup logic with various TTL values
- Interaction between count and TTL cleanup
- Edge cases (TTL=0, very small TTL, very large TTL)

**E2E Tests:**
- Create histories with backdated timestamps
- Verify TTL cleanup triggers correctly
- Verify metrics are updated
- Verify both strategies work together

## Documentation Updates

- `docs/HISTORY.md`: Add TTL configuration and examples
- `CLAUDE.md`: Update configuration flags section
- `README.md`: Mention TTL feature in history section

## References

- ADR 0006: Experiment History and Audit Logging (foundation)
- Kubernetes TTL Controller pattern
- Industry standards for audit log retention (SOC2, GDPR, HIPAA)