# Archived Tasks

## P0 (blocking/critical)
- ~~Handle terminating pods gracefully during selection/execution.~~ **[COMPLETED 2025-12-26]** - Pods with DeletionTimestamp are now filtered in `getEligiblePods()`, metrics track via `chaos_safety_excluded_resources_total{reason="terminating"}`.
- ~~Add E2E coverage for `pod-network-loss`.~~ **[COMPLETED 2025-12-26]** - Comprehensive E2E tests added in `test/e2e/e2e_test.go` covering: basic injection, dry-run mode, lossCorrelation parameter, no eligible pods handling, and maxPercentage safety limit.
- ~~Emit Kubernetes Events for experiments and affected pods.~~ **[COMPLETED 2025-12-27]** - EventRecorder now emits events at key lifecycle points: ExperimentStarted, ExperimentSucceeded, ExperimentRetrying, and ExperimentFailed. Events visible via `kubectl describe chaosexperiment`.
- ~~Add history TTL cleanup (in addition to retention limit).~~ **[COMPLETED 2026-01-02]** - History records now support automatic TTL-based cleanup in addition to retention limits.

## P1 (important)
- ~~Improve permission-denied error handling and user-facing messages.~~ **[COMPLETED 2026-01-02]** - Implemented structured error handling with typed errors, detailed permission messages, 1-retry limit for RBAC errors, error_type metrics label, and comprehensive documentation. See ADR 0010.
- ~~`pod-disk-fill` (ADR 0008).~~ **[COMPLETED 2026-01-02]** - Pod disk fill action implemented using ephemeral containers with dd to fill disk space.
- ~~`pod-restart` (restart pods without delete).~~ **[COMPLETED 2025-12-27]** - Pod restart action implemented with graceful SIGTERM and configurable restart intervals.
- ~~Maintenance time windows.~~ **[COMPLETED 2026-01-25]** - Implemented `maintenanceWindows` in ChaosExperiment CRD and Controller to block experiments during specified time ranges.
- ~~Expand E2E scenarios (selectors, namespaces, concurrency, cancellation).~~ **[COMPLETED 2026-01-25]** - Added advanced E2E tests covering label selectors, multi-namespace experiments, concurrent execution, and cancellation flows.
