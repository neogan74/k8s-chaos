# Backlog

This file tracks remaining work items, grouped by area and priority. Source of truth is `backlog.md`.

## P0 (blocking/critical)
- ~~Handle terminating pods gracefully during selection/execution.~~ **[COMPLETED 2025-12-26]** - Pods with DeletionTimestamp are now filtered in `getEligiblePods()`, metrics track via `chaos_safety_excluded_resources_total{reason="terminating"}`.
- ~~Add E2E coverage for `pod-network-loss`.~~ **[COMPLETED 2025-12-26]** - Comprehensive E2E tests added in `test/e2e/e2e_test.go` covering: basic injection, dry-run mode, lossCorrelation parameter, no eligible pods handling, and maxPercentage safety limit.
- Emit Kubernetes Events for experiments and affected pods.
- Add history TTL cleanup (in addition to retention limit).

## P1 (important)
- Improve permission-denied error handling and user-facing messages.
- `pod-disk-fill` (ADR 0008).
- `pod-restart` (restart pods without delete).
- `network-partition`.
- Maintenance time windows.
- Dependency management between experiments.
- Pause/resume experiments.
- Expand E2E scenarios (selectors, namespaces, concurrency, cancellation).
- `create` wizard.
- `validate` command for YAML.
- `check` command for cluster readiness.
- `logs` command for history browsing/export.
- Fine-grained permissions and namespace isolation.
- Resource quotas and rate limiting.
- Leader election verification and horizontal scaling tests.
- Graceful shutdown improvements.
- Prometheus alerts and notification integrations (Slack/PagerDuty).

## P2 (nice-to-have)
- `pod-network-corruption`.
- `node-taint`.
- `node-cpu-stress`.
- `node-disk-fill`.
- `dns-chaos`.
- `http-chaos`.
- Scenario/workflow support (chained actions, conditional chaos, gradual chaos).
- Increase test coverage and add edge-case tests.
- Add coverage threshold enforcement.
- Performance/benchmark testing.
- Watch mode and export formats (JSON/CSV).
- Shell completion and config file support.
- Operator Lifecycle Manager (OLM) support.
- Multi-tenancy support.
- Grafana dashboards updates.
- Service mesh integrations (Istio/Linkerd).
- Impact analysis, steady-state checks, automated reports.
