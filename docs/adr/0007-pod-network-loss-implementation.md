# ADR 0007: Pod Network Loss Implementation

 **Status**: Implemented  
**Date**: 2026-03-02  
**Author**: k8s-chaos team

## Context

Network chaos is listed in the roadmap (Q2 2026) and is a top user request to validate resilience to packet loss. We need a safe, Kubernetes-native way to inject deterministic loss on targeted pods without requiring cluster-wide privileges or node agents. Constraints: reuse existing ChaosExperiment patterns, honor safety limits (dry-run, maxPercentage, exclusions, duration), and ensure deterministic cleanup.

## Decision

Implement `pod-network-loss` using Linux `tc netem` applied from an ephemeral container injected into target pods. Ephemeral containers share the pod network namespace, allowing us to add/remove qdisc rules without sidecars or host access.

### Key implementation details
- **Chaos action**: `pod-network-loss`.
- **Mechanism**: ephemeral container with `NET_ADMIN` capability running `iproute2` (`tc`) to add `netem loss` qdisc on `eth0` (or detected primary interface).
- **Lifecycle**:
  - On start: add `tc qdisc add dev <iface> root netem loss <pct>% [correlation <corr>%]`.
  - On completion/cleanup: `tc qdisc del dev <iface> root netem` (idempotent).
  - Store applied state per pod in status to ensure cleanup on retries.
- **Spec additions** (validated by CRD + webhook):
  - `lossPercentage` (float, 0 < p ≤ 40; default 5).
  - `correlation` (float, 0–100; default 0).
  - `direction` (enum: `egress` | `ingress` | `both`; default `both`).
  - `duration` required; uses existing duration handling.
  - `targets` reuse existing selector/count/safety filters.
- **Safety**: respect `maxPercentage`, exclusions, dry-run; refuse lossPercentage > 40 unless override flag (future-proofed).
- **Observability**: emit experiment status events and Prometheus metrics for injected pods and cleanup outcomes; include loss parameters in history/audit.

## Alternatives Considered

### Alternative 1: eBPF-based dropper
- **Pros**: Lower overhead, finer control (per-port/per-protocol).
- **Cons**: Requires BPF tooling/binaries, kernel support variance, higher complexity and debugging cost.
- **Why rejected**: More operational risk and portability concerns; tc netem is simpler and sufficient for first iteration.

### Alternative 2: Service mesh faults (Istio/Linkerd)
- **Pros**: Native mesh integration, no pod privileges.
- **Cons**: Mesh not guaranteed; configuration surface differs per mesh.
- **Why rejected**: Would only work in meshed environments; can be added later as mesh-specific chaos actions.

### Alternative 3: iptables DROP rules
- **Pros**: No netem dependency.
- **Cons**: Coarse-grained, hard to express correlation, trickier cleanup.
- **Why rejected**: Netem provides richer, reversible semantics.

## Consequences

### Positive
- Enables roadmap network chaos with minimal footprint and no node agents.
- Reuses existing ephemeral container/safety pipeline for consistent UX.
- Deterministic cleanup reduces risk of lingering network impairment.

### Negative
- Requires `NET_ADMIN` in ephemeral container; some clusters may block this.
- Netem precision can vary under heavy load; needs documentation of limitations.
- Adds dependency on `iproute2` image and interface detection logic.

### Neutral
- Introduces new validation paths in webhook and status reporting; increases test surface.

## Implementation Status

### Completed
- [x] CRD/schema fields and webhook validation
- [x] Controller logic for tc injection/cleanup
- [x] Metrics/events for network loss action
- [x] Sample YAML in `config/samples/chaos_v1alpha1_chaosexperiment_network_loss.yaml`

### Planned
- [ ] E2E scenario in `test/e2e` with Kind
- [ ] Docs: usage example in `docs/SCENARIOS.md`
- [ ] Safe override flag for >40% loss (if needed)

### Deferred
- [ ] Mesh-specific network loss actions (Istio/Linkerd)
- [ ] eBPF-based advanced filters

## References

- Roadmap: Network Chaos (Q2 2026)
- Existing ADRs: 0003/0005 stress patterns using ephemeral containers
- K8s ephemeral containers: https://kubernetes.io/docs/concepts/workloads/pods/ephemeral-containers/
