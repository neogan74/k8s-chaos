# Backlog

This file tracks remaining work items not yet completed. For the full historical list, see `TODO.md`.

## P0 (blocking/critical)
- Add history TTL cleanup (in addition to retention limit).
- Add E2E coverage for `pod-network-loss`.
- Add unit tests for `node-taint` and `node-cpu-stress` handlers.

## P1 (important)
- `node-disk-fill` (follow ADR 0008 pattern).
- `network-partition` — simulate network splits between pod groups.
- Pause/resume experiments via `paused: true` spec field.
- Expand E2E scenarios (selectors, namespaces, concurrency, cancellation).
- `create` interactive wizard (CLI).
- `validate` command for YAML (CLI).
- `check` command for cluster readiness (CLI).
- `logs` command for history browsing/export (CLI).
- Fine-grained RBAC permissions and namespace isolation.
- Resource quotas and rate limiting.
- Leader election verification and horizontal scaling tests.
- Graceful shutdown improvements (SIGTERM handling).
- Prometheus AlertManager integration and Slack/PagerDuty notifications.
- Increase unit test coverage to 80%.

## P2 (nice-to-have)
- `dns-chaos` — DNS resolution failures.
- `http-chaos` — HTTP response manipulation.
- `pod-io-stress` — Filesystem I/O stress inside pods.
- `pod-jvm-stress` — JVM heap/GC pressure for Java workloads.
- Scenario/workflow support (chained actions, conditional chaos, gradual chaos).
- OPA policy integration for experiment approval.
- Add coverage threshold enforcement in CI.
- Performance/benchmark testing.
- Watch mode and export formats (JSON/CSV) for CLI.
- Shell completion and `.k8s-chaos.yaml` config file support.
- Operator Lifecycle Manager (OLM) support.
- Multi-tenancy support.
- Grafana dashboard updates for new chaos actions (node-taint, node-cpu-stress).
- Service mesh integrations (Istio/Linkerd).
- Impact analysis, steady-state checks, automated reports.
- Multi-cluster support.
