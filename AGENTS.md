# Repository Guidelines

## Project Structure & Modules
- `cmd/`: entrypoints for controller manager, monitoring gateway, agent proxy, and block manager; binaries land in `bin/` after builds.
- `pkg/`: core controllers, API types, and library code; prefer adding shared helpers here before reusing in `cmd/`.
- `config/`: Kubernetes manifests, CRDs (`config/crd`), kustomize overlays, and bundle output; charts live under `charts/`.
- `docs/` and `tools/docs/`: user/developer docs and CRD doc generator templates; update when behavior or APIs change.
- `test/`: Go unit/integration tests plus `test/e2e` for Ginkgo-based suites.
- `hack/`, `build/`, `config/manager`: scripts, Dockerfiles, and kustomize bases used by CI and releases.

## Build, Test, and Development Commands
- `make help`: list available targets.
- `make build`: generate code, format, vet, and build Go binaries into `bin/`.
- `make test`: run unit/integration tests (excluding e2e) with envtest; produces `cover.out`.
- `make test-e2e`: run end-to-end suite in `test/e2e` (requires a Kind-accessible cluster).
- `make lint` / `make lint-fix`: run or auto-fix `golangci-lint`.
- `make docker-build`: build container images for the four components using Docker or `CONTAINER_CLI`.
- Deployment helpers: `make install|uninstall` for CRDs, `make deploy|undeploy` for controller-manager to the current kubeconfig.

## Coding Style & Naming
- Go 1.25 module; rely on `gofmt` via `make fmt` and static checks via `make vet`/`make lint`.
- Package paths are lowercase; exported types/functions should include the domain concept (e.g., `TenantReconciler`, `GatewayOptions`).
- Tests live next to code with `_test.go` suffix; prefer table-driven cases and minimal global state.
- YAML manifests and Helm values use lowercase hyphenated keys; keep CRD fields consistent with existing API naming.

## Testing Guidelines
- Unit/integration: `make test` (runs with controller-runtime envtest assets). Aim to keep `cover.out` committed only when needed for reporting.
- End-to-end: `make test-e2e -ginkgo.focus=<regex>` when scoping; expect a running cluster and necessary CRDs installed.
- Add fixtures under `test/` and avoid hard-coded cluster credentials in tests; use env vars for endpoints/secrets.

## Commit & Pull Request Expectations
- Commits in this repo are short, imperative messages (e.g., `fix port shadowing`, `upgrade dependencies`); keep them scoped and consistent.
- Before opening a PR: run `make lint` and `make test`; include relevant logs or screenshots for UI/chart changes.
- PR descriptions should summarize behavior changes, list test commands executed, and link related issues. Update docs (`docs/` or chart README) and CRD manifests when APIs or defaults shift.
- For Kubernetes-facing changes, describe rollout/rollback considerations and any required configuration migrations.

## Security & Configuration Tips
- Do not commit kubeconfig, secrets, or cloud credentials; prefer env vars and sample manifests under `config/samples`.
- When modifying CRDs, regenerate manifests with `make manifests` and keep stripped-down CRDs in `charts/whizard-crds/` aligned via `make stripped-down-crds`.
- Container builds default to `docker`; set `CONTAINER_CLI=podman` or platform flags via `CONTAINER_BUILD_EXTRA_ARGS` if needed.
