# Backlog

## ADR 0007 – Pod Network Loss
- [ ] CRD/schema: add `lossPercentage`, `correlation`, `direction` with validation defaults; require `duration`.
- [ ] Webhook: enforce loss caps (default ≤40%), disallow zero duration, validate directions, dry-run support.
- [ ] Controller logic: inject ephemeral container with `iproute2` + `NET_ADMIN`, detect interface, apply/remove `tc netem loss`; persist applied state for cleanup/retries.
- [ ] Image/config: choose lightweight `tc` image, wire into controller image/build (Makefile/Kustomize/Helm values).
- [ ] Safety wiring: honor exclusion labels, `maxPercentage`, namespace protections, retry/backoff flow.
- [ ] Observability: emit Events and Prometheus metrics for inject/cleanup, record parameters in history/status.
- [ ] Samples/docs: add example manifest under `config/samples/` and scenario in `docs/SCENARIOS.md`; mention limits/privilege requirements.
- [ ] Tests: unit tests for validation and reconcile; e2e scenario in `test/e2e` (Kind) verifying loss applied and cleaned up.
