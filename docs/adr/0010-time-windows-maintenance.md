# ADR 0010: Experiment Time Windows (Maintenance Scheduling)

**Status**: Accepted
**Date**: 2026-01-03
**Authors**: k8s-chaos team

## Context

Chaos experiments are often safe only within defined maintenance windows. Teams need to:
- Avoid running chaos during peak traffic or release freeze periods
- Align with on-call coverage and incident response readiness
- Respect regional time zones and business hours
- Permit both recurring windows (e.g., weekdays 22:00-02:00) and one-off windows

Today, experiments run whenever a controller reconciles them. Operators must manually enable/disable experiments or rely on external schedulers. This leads to human error and inconsistent enforcement.

## Decision

Introduce **Time Windows** in `ChaosExperimentSpec` to restrict when experiments can run. If current time is outside all windows, the controller will skip execution and requeue for the next valid window boundary.

### Proposed API

```go
type TimeWindowType string

const (
	TimeWindowRecurring TimeWindowType = "Recurring"
	TimeWindowAbsolute  TimeWindowType = "Absolute"
)

type TimeWindow struct {
	// Type selects recurring or absolute window semantics.
	// +kubebuilder:validation:Enum=Recurring;Absolute
	Type TimeWindowType `json:"type"`

	// Start and End use HH:MM for Recurring, RFC3339 for Absolute.
	// +optional
	Start string `json:"start,omitempty"`
	// +optional
	End string `json:"end,omitempty"`

	// Timezone applies to Recurring windows (IANA TZ, e.g., "Europe/Berlin").
	// Defaults to UTC when omitted.
	// +optional
	Timezone string `json:"timezone,omitempty"`

	// DaysOfWeek applies to Recurring windows. Empty means every day.
	// Values: Mon, Tue, Wed, Thu, Fri, Sat, Sun
	// +optional
	DaysOfWeek []string `json:"daysOfWeek,omitempty"`
}

// TimeWindows limits when the experiment may execute.
// If empty or omitted, experiment is allowed at any time.
// +optional
TimeWindows []TimeWindow `json:"timeWindows,omitempty"`
```

### Behavior

- **Allowed** if current time matches any window.
- **Blocked** if no windows match; controller sets a status condition and requeues at the next boundary.
- **Recurring windows** support time zones and wrap-around (e.g., 22:00-02:00).
- **Absolute windows** are one-off RFC3339 intervals.

### Example

```yaml
spec:
  action: pod-cpu-stress
  timeWindows:
    - type: Recurring
      daysOfWeek: ["Mon", "Tue", "Wed", "Thu", "Fri"]
      start: "22:00"
      end: "02:00"
      timezone: "Europe/Berlin"
    - type: Absolute
      start: "2026-01-10T01:00:00Z"
      end: "2026-01-10T03:00:00Z"
```

## Alternatives Considered

### Alternative 1: External scheduling (CronJob/automation)
- Description: Users enable/disable experiments via external cron or CI.
- Pros: No CRD changes, simple controller logic.
- Cons: Easy to misconfigure, no unified visibility, harder to audit.
- Why rejected: Does not enforce safety within the operator itself.

### Alternative 2: Global maintenance window in controller flags
- Description: Operator configured with a single cluster-wide window.
- Pros: Simple configuration, centralized control.
- Cons: Not flexible per experiment, not suitable for multi-tenant use.
- Why rejected: Experiments often have different maintenance requirements.

### Alternative 3: Cron-style schedules in spec
- Description: Support cron expressions for experiment execution.
- Pros: Very flexible scheduling.
- Cons: Complex validation, higher foot-gun risk, poor readability.
- Why rejected: Time windows are safer and easier to reason about.

## Consequences

### Positive
- Enforces maintenance windows consistently and automatically.
- Reduces operational risk by preventing chaos during busy periods.
- Improves auditability and reduces manual toggling.

### Negative
- Adds spec complexity and time zone handling in the controller.
- Requires careful validation of window formats and overlaps.

### Neutral
- Experiments with no windows behave exactly as today.
- Some schedules may still require external automation (e.g., holiday rules).

## Implementation Status

### Completed
- [x] Add `TimeWindow` types to `api/v1alpha1` with validation tags and defaults
- [x] Regenerate CRDs and manifests (`make manifests generate`)
- [x] Implement time window parsing (HH:MM for recurring, RFC3339 for absolute) with timezone handling
- [x] Add a matcher that supports wrap-around windows and empty `DaysOfWeek`
- [x] Compute next boundary and requeue when blocked; avoid tight loops
- [x] Emit a status condition (`BlockedByTimeWindow`) with next eligible time
- [x] Add controller tests for window matching and boundary calculations
- [x] Add sample manifests for recurring and absolute windows

### Planned
- [ ] None

### Deferred
- [ ] Calendar integrations and holiday-aware scheduling.

## References

- [Kubernetes Timezone Database](https://en.wikipedia.org/wiki/List_of_tz_database_time_zones)
- [RFC 3339](https://www.rfc-editor.org/rfc/rfc3339)

## Notes

Open question: Should time windows apply to each reconcile tick or only to the initial selection of targets?
