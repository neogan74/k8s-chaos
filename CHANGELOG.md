# Changelog

All notable changes to k8s-chaos will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **Event Recording**: Kubernetes events are now emitted for experiment lifecycle milestones
  - Events are created when experiments start, succeed, fail, and retry
  - Visible via `kubectl describe chaosexperiment <name>` for better observability
  - Event types: `ExperimentStarted` (Normal), `ExperimentSucceeded` (Normal), `ExperimentRetrying` (Warning), `ExperimentFailed` (Warning)
  - RBAC permissions for events automatically included in generated manifests
  - Helps with debugging and monitoring experiment execution in real-time

- **pod-restart action**: Gracefully restart containers by sending SIGTERM to the main process (PID 1)
  - Allows testing graceful shutdown and cleanup code paths
  - Supports `restartInterval` parameter to stagger restarts (e.g., "30s", "1m")
  - Integrates with all existing safety features (dry-run, maxPercentage, exclusions, production protection)
  - Tests planned maintenance scenarios vs crash scenarios (pod-failure uses SIGKILL)
  - Pods maintain same IP address after restart (unlike pod-kill which deletes the pod)
  - See ADR-0009 for full design rationale and implementation details
  - Sample configurations in `config/samples/chaos_v1alpha1_chaosexperiment_pod_restart.yaml`

- **CLI `validate` command**: validate ChaosExperiment manifests offline (`k8s-chaos validate -f FILE|DIR|-`)
  - Catches unknown fields, CRD schema violations and the webhook's cross-field rules without a cluster
  - New `ChaosExperiment.ValidateStatic()` in `api/v1alpha1` reuses the webhook's validators
  - A test now validates every manifest in `config/samples/`
- **Pause/resume**: documented the `paused` spec field (`docs/API.md`) and added a sample (`config/samples/chaos_v1alpha1_chaosexperiment_paused.yaml`)
- **CLI `check` command**: read-only cluster readiness check (`k8s-chaos check [-n NAMESPACE]`)
  - Verifies API connectivity, CRD installation, controller readiness, admission webhook registration and the current user's RBAC permissions
  - Exits non-zero only on failures; warnings (e.g. no webhook) do not fail the run

### Fixed
- Sample manifests that the CRD would reject: `node_stress` (missing `namespace`), `node_taint` (empty `selector`), `timewindows` (non-existent `delayMs` field)

### Changed

### Deprecated

### Removed

### Security

---

## Template for Future Releases

```markdown
## [X.Y.Z] - YYYY-MM-DD

### Added
- New features

### Changed
- Changes to existing functionality

### Deprecated
- Soon-to-be removed features

### Removed
- Removed features

### Fixed
- Bug fixes

### Security
- Security fixes
```