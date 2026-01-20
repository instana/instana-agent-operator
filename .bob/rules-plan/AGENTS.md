# Plan Mode Rules (Non-Obvious Only)

## Architecture Constraints
- Kubebuilder v3 framework: Controller-runtime based, follows operator pattern
- Two CRD versions: InstanaAgent (v1, v1beta1) and RemoteAgent (v1) - must maintain compatibility
- Finalizer versioning: Both finalizerV1 and finalizerV3 exist in codebase (migration pattern)
- Namespace isolation: Default `instana-agent` namespace, not `default`

## Hidden Dependencies
- ETCD discovery: OpenShift-specific - copies certs from openshift-etcd namespace automatically
- OpenShift detection: Based on `oc` command availability, not cluster API inspection
- Buildkit requirement: Cannot use direct docker/podman build, must use buildctl with containerized buildkitd
- Container runtime: Auto-detected (podman/docker), don't hardcode in plans

## Build System Architecture
- Tools isolation: All binaries in `./bin/` (GOBIN override), not system-wide
- Multi-arch support: Requires both BUILDPLATFORM and TARGETPLATFORM build args
- Version injection: VERSION, GIT_COMMIT, DATE passed as build args
- Image loading: Uses `$(CONTAINER_CMD) load` after buildctl (not direct push)

## Testing Architecture
- Unit tests: Automatically exclude `mocks` and `e2e` directories
- E2E tests: Require real cluster + credentials in `e2e/.env` file
- KUBEBUILDER_ASSETS: Auto-managed by envtest, don't plan manual setup
- Secrets scanning: Pre-commit hook enforced, must update baseline before commit

## Development Workflow Patterns
- Local dev: Two modes - `make run` (operator local) or `make dev-run-cluster` (full setup)
- Cleanup: `make purge` removes all resources including stuck finalizers
- Branch naming: No `/` allowed (breaks CI pipelines)
- CRD changes: Must update docs/ when modifying Custom Resource Definitions

## Non-Standard Patterns
- Error handling: Must use `github.com/pkg/errors` package (not stdlib errors)
- Container builds: Buildkit in privileged container, not native docker build
- OpenShift permissions: Auto-applied via `oc adm policy` when detected