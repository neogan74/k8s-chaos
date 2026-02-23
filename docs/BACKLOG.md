# Backlog

Prioritized list of pending work. Items are grouped by theme and ordered by priority within each group.

**Last Updated:** February 23, 2026 (updated with code audit findings)

---

## 🔴 P0 – Critical / Blocks Core Functionality

### pod-disk-fill Implementation (ADR 0008)
Full implementation of the `pod-disk-fill` chaos action. Design is complete; execution is pending.

- [ ] CRD/schema: add `fillPercentage`, `targetPath`, `volumeName` fields with kubebuilder validation markers; update CRD enum to include `pod-disk-fill`
- [ ] Webhook: validate `fillPercentage` range (1–95%), require `duration`, cross-field checks; dry-run support
- [ ] Controller: inject ephemeral container that writes data to `targetPath` using `dd` or `fallocate`; clean up on completion/timeout
- [ ] Safety wiring: honor exclusion labels, `maxPercentage`, namespace protections, retry/backoff flow
- [ ] Observability: Prometheus metrics for inject/cleanup, record `fillPercentage` and path in status/history
- [ ] Samples: `config/samples/chaos_v1alpha1_chaosexperiment_disk_fill.yaml`
- [ ] Tests: unit tests for validation and reconcile; e2e scenario in `test/e2e` (Kind) verifying fill applied and cleaned up
- [ ] Docs: update `docs/API.md` and `docs/SCENARIOS.md`

### Edge Cases in Error Handling
- [ ] Handle pods already in `Terminating` state gracefully (currently may select terminating pods)
- [ ] Handle `permission denied` errors with clear status messages and no infinite retry

### Kubernetes Events
**Observation:** The controller struct has no `record.EventRecorder` field at all — events are completely absent from the codebase. Wiring one in is a few lines in `cmd/main.go` and `internal/controller/chaosexperiment_controller.go`.

- [ ] Add `record.EventRecorder` field to `ChaosExperimentReconciler` struct
- [ ] Wire recorder in `cmd/main.go` via `mgr.GetEventRecorderFor("chaosexperiment-controller")`
- [ ] Emit `Normal` event on experiment start (action, namespace, selector)
- [ ] Emit `Normal` event on experiment complete (pods affected, duration)
- [ ] Emit `Warning` event on experiment fail (error message, retry count)
- [ ] Emit `Normal` event on dry-run (resources that would be affected)
- [ ] Emit `Warning` event when safety block fires (production namespace, percentage limit, exclusion)
- [ ] Emit event on affected pods when chaos is injected (so `kubectl describe pod` shows it)

### `status.affectedPods` Completeness
**Observation:** `Status.AffectedPods` is populated only for ephemeral-container actions (`pod-cpu-stress`, `pod-memory-stress`, `pod-network-loss`) — used to track containers for cleanup. It is **not populated** for `pod-kill`, `pod-delay`, `pod-failure`, or `node-drain`.

- [ ] Populate `status.affectedPods` for all action types at experiment completion
- [ ] Keep it as a human-readable record (separate from the cleanup-tracking slice used internally)

---

## 🟡 P1 – High Priority

### Test Coverage
- [ ] Increase unit test coverage to 80% (current estimate: ~50–60%)
- [ ] Add integration tests for all chaos actions (pod-kill, pod-delay, cpu/mem stress, network-loss, disk-fill)
- [ ] E2E test scenarios using Kind:
  - [ ] Different pod selectors (label, field)
  - [ ] Multiple namespaces
  - [ ] Concurrent experiments
  - [ ] Experiment cancellation mid-run
  - [ ] Dry-run mode validation
- [ ] Add benchmark/performance tests for large pod counts
- [ ] Mock external dependencies better for cleaner unit test isolation

### New Chaos Actions – Network
- [ ] **pod-network-corruption** – Corrupt packets using `tc netem corrupt`; **trivial to add**: reuses the exact same ephemeral container and `injectNetworkLossContainer` scaffolding from `pod-network-loss`, only the `tc` command args change. Lowest-effort new action in the codebase.
- [ ] **pod-network-partition** – Block traffic between pod groups using iptables rules
- [ ] **dns-chaos** – Override DNS resolution inside pods (e.g., `/etc/hosts` manipulation or CoreDNS tampering)

### New Chaos Actions – Node
- [ ] **node-taint** – Add a taint to a node and auto-remove after duration
- [ ] **node-cpu-stress** – Stress node CPU (DaemonSet-based or host-PID ephemeral container)
- [ ] **node-disk-fill** – Fill node disk space via privileged ephemeral container

### New Chaos Actions – Application
- [ ] **pod-restart** – Graceful pod restart (delete + wait for re-schedule) instead of hard kill
- [ ] **http-chaos** – Manipulate HTTP responses (delay, 5xx injection) via sidecar or iptables redirect

### CLI – Advanced Commands
- [ ] **create** – Interactive wizard: guided prompts for action, namespace, selector, validation before apply, template selection
- [ ] **validate** – Validate experiment YAML offline: schema check, cross-field validation, namespace/selector verification
- [ ] **check** – Pre-flight health check: CRD installed, RBAC permissions, API connectivity
- [ ] **logs** – Show experiment execution history: filter by date/status, paginate, export to file
- [ ] **watch** – `--watch` flag on `list` for real-time updates
- [ ] **Shell completion** – Bash/Zsh/Fish tab completion (`kubectl` style)
- [ ] **Export formats** – `--output json` and `--output csv` for `stats` command
- [ ] **Config file** – Support `.k8s-chaos.yaml` per-project config file

### CONTRIBUTING.md
**Observation:** The file is referenced in `ROADMAP.md`, `labs/README.md`, and `RELEASE-NOTES.md` but **does not exist**. New contributors hit a dead end.

- [ ] Create `CONTRIBUTING.md` at repo root covering:
  - Prerequisites and dev environment setup (Kind, `make install`, `make run`)
  - How to add a new chaos action (step-by-step walkthrough of types → webhook → controller → metrics → tests → docs)
  - Testing requirements: unit tests for all new validation/reconcile logic, e2e scenario for Kind
  - PR template and code review expectations
  - Release process (tag format, changelog, Helm chart bump)

---

## 🟢 P2 – Medium Priority

### Scheduling Enhancements
- [ ] **Time Windows** – Define maintenance windows; auto-pause experiments outside them
- [ ] **Dependency Management** – Wait for another experiment to reach `Completed` before starting
- [ ] **Pause/Resume** – Allow pausing a running experiment and resuming it later

### Observability & Integrations
- [ ] **Prometheus AlertManager** – Example alert rules for experiment failures and safety violations
- [ ] **Slack/PagerDuty notifications** – Webhook call on experiment start/complete/fail (configurable in spec)
- [ ] **Grafana Dashboard improvements** – Add network-loss and disk-fill panels to existing dashboards
- [ ] **Service Mesh integration** – Istio/Linkerd chaos injection for L7 network faults

### RBAC & Security
- [ ] **Fine-grained permissions** – Separate ClusterRoles for read-only, limited chaos (no node-drain), full chaos
- [ ] **Namespace-scoped operator mode** – Run operator restricted to specific namespaces
- [ ] **Audit logging** – Record who triggered an experiment (via `kubectl.kubernetes.io/last-applied-configuration` or admission webhook attribution)
- [ ] **OPA Integration** – Policy-based experiment approval via Gatekeeper constraints

### CI/CD
**Observation:** All three workflows (`test.yml`, `lint.yml`, `test-e2e.yml`) are bare-minimum. Specific gaps found:

- [ ] **Coverage gate** – `test.yml` runs `make test` but never reports or enforces coverage; add `-covermode=atomic -coverprofile=coverage.out` and fail below 70% threshold
- [ ] **Pinned golangci-lint version** – `lint.yml` uses `version: latest` which can silently break on new lint releases; pin to a specific version (e.g., `v1.62.0`)
- [ ] **Release automation** – No release workflow exists; add GitHub Actions workflow triggered on `v*.*.*` tag push: build binaries, build+push Docker image, create GitHub Release with changelog
- [ ] **Container image scanning** – Trivy/Grype scan in CI on the built image, fail on HIGH/CRITICAL CVEs
- [ ] **Multi-arch builds** – ARM64 support (`linux/amd64,linux/arm64`) via `docker buildx` in CI
- [ ] **e2e workflow trigger** – `test-e2e.yml` exists but check it runs on PRs and not just manually

### Controller Refactor
**Observation:** `internal/controller/chaosexperiment_controller.go` is **1934 lines with 34 functions** — all action handlers, safety helpers, cleanup logic, and history recording live in one file. This makes it hard to navigate, review, and test individual actions.

- [ ] Split into per-action files: `actions/pod_kill.go`, `actions/pod_delay.go`, `actions/pod_network_loss.go`, etc.
- [ ] Extract safety helpers into `safety.go`
- [ ] Extract ephemeral container lifecycle (inject/cleanup) into `ephemeral.go` — shared by cpu-stress, memory-stress, network-loss, disk-fill
- [ ] Keep `chaosexperiment_controller.go` as the thin reconcile dispatcher only
- [ ] Standardize error handling: all action handlers should return `(ctrl.Result, error)` with the same wrapping pattern
- [ ] Remove/address inline `// TODO` comments in source code

### Performance & Production Readiness
- [ ] **Rate limiting** – Prevent overwhelming the API server with reconcile loops
- [ ] **Batch operations** – Efficient handling of experiments affecting many pods
- [ ] **Resource profiling** – Profile memory/CPU usage under load and optimize
- [ ] **Leader election verification** – Confirm controller handles leader election correctly under failover
- [ ] **Graceful shutdown** – Clean shutdown on SIGTERM (drain in-flight reconciles)

---

## 🔵 P3 – Low Priority / Future

### Experiment Orchestration
- [ ] **Scenario Support** – `ChaosScenario` CRD to chain multiple actions sequentially or in parallel
- [ ] **Conditional Chaos** – Trigger experiments based on Prometheus alert or metric threshold
- [ ] **Gradual Chaos** – Incrementally increase intensity (e.g., ramp `lossPercentage` from 5% to 30%)
- [ ] **Argo Workflows integration** – Use Argo Workflows steps to orchestrate chaos scenarios

### Web UI / Dashboard
- [ ] Visual experiment designer
- [ ] Real-time experiment monitoring (status, affected pods, metrics)
- [ ] Experiment catalog and templates
- [ ] Historical analysis and run comparison

### Multi-cluster & Multi-tenancy
- [ ] Coordinate chaos experiments across multiple clusters
- [ ] Per-team quota management and isolated namespaces
- [ ] Cross-cluster dependency testing

### Intelligent Chaos
- [ ] **Steady State Detection** – Automatically verify system health before and after experiments
- [ ] **Impact Analysis** – Map resource dependencies and estimate blast radius
- [ ] **Suggestion Engine** – Recommend experiments based on cluster topology and past runs

### Technical Debt
- [ ] OLM (Operator Lifecycle Manager) packaging and bundle creation
- [ ] Update Go dependencies to latest compatible versions
- [ ] Add `go:generate` targets for mocks (remove manual mock files)

---

## Notes

- Items marked with an ADR reference have architecture decisions already documented in `docs/adr/`.
- For P0/P1 items without an ADR, create one before implementation to document the approach.
- E2E tests require a Kind cluster; see `labs/infra/` for cluster configurations.
- When starting any item, check the related section in `ROADMAP.md` for quarterly context.

## Code Audit Findings (February 2026)

Quick summary of gaps found by reading the source directly:

| Finding | Location | Impact |
|---|---|---|
| No `EventRecorder` in controller | `internal/controller/chaosexperiment_controller.go` | `kubectl describe` shows no events |
| `status.affectedPods` only set for ephemeral-container actions | controller, lines 1909/1925 | pod-kill, pod-delay, pod-failure, node-drain don't record affected pods |
| `CONTRIBUTING.md` missing | repo root | referenced in 3 docs, breaks onboarding |
| `golangci-lint version: latest` unpinned | `.github/workflows/lint.yml` | lint can silently break on new releases |
| No coverage reporting or gate in CI | `.github/workflows/test.yml` | coverage regressions go undetected |
| No release workflow | `.github/workflows/` | releases are manual |
| Controller is 1934 lines / 34 functions | `internal/controller/chaosexperiment_controller.go` | hard to navigate and test |
| `pod-network-corruption` is trivial | — | reuses pod-network-loss scaffolding, just different `tc` args |
