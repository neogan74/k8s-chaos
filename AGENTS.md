# Repository Guidelines

## Project Structure & Module Organization
- `api/v1alpha1/`: CRD types and validation; update these before regenerating manifests.
- `internal/controller/`: Reconciliation logic and experiment orchestration.
- `cmd/main.go` and `cmd/k8s-chaos-cli/`: Manager and CLI entrypoints.
- `config/` & `charts/`: Kustomize and Helm deployments; `dist/install.yaml` is built from these.
- `labs/` and `config/samples/`: Hands-on scenarios and demo manifests.
- `test/e2e/`: End-to-end tests (Kind-based); other packages use standard Go tests.

## Build, Test, and Development Commands
- `make build`: Compile controller binary to `bin/manager`.
- `make run`: Run controller locally against current kubeconfig (auto-generates manifests).
- `make test`: Unit/integration suite with envtest; writes `cover.out`.
- `make test-e2e`: Kind-backed e2e tests (`KIND_CLUSTER=k8s-chaos-test-e2e`).
- `make lint` / `make lint-fix`: golangci-lint check or autofix.
- `make build-cli` / `make install-cli`: Build/install the `k8s-chaos` CLI.
- `make install | deploy | undeploy`: Manage CRDs and controller in the active cluster.

## Coding Style & Naming Conventions
- Go 1.24+; run `gofmt` (tabs, std Go style) and `go vet` via `make test`.
- Lint with `golangci-lint` (config in repo); keep files free of warnings.
- Package names: lower-case, no underscores; APIs live under `chaos.gushchin.dev/v1alpha1`.
- Prefer small, focused controllers and reconcile helpers; keep kube client interactions in `internal/controller`.

## Testing Guidelines
- Unit tests follow `_test.go` naming; table-driven tests preferred.
- For controller changes: `go test ./internal/controller/... -v` locally before PR.
- For CRD or workflow changes: run `make test-e2e` (requires Kind and Docker).
- Capture coverage with `make test` (outputs `cover.out`); update flaky tests before merging.

## Commit & Pull Request Guidelines
- Commit style: Conventional Commit-like prefixes (`feat`, `fix`, `chore`, `docs`) as seen in history.
- Each PR should describe scope, risks, and linked issues; include CLI/log snippets or screenshots for UX changes.
- Run `make lint test` (and `make test-e2e` when touching controllers/CRDs) before opening a PR; paste key results.
- When APIs change, include regenerated artifacts (`make manifests generate`) and update docs in `docs/` or samples.

## Security & Configuration Tips
- Use dry-run and percentage limits in samples when testing against shared clusters.
- Keep dev/test clusters isolated (Kind contexts: `k8s-chaos-dev`, `k8s-chaos-test-e2e`); avoid deploying to production kubeconfig by default.
- Images: build with `make docker-build IMG=...` and avoid pushing unreviewed tags.***
