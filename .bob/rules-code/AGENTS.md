# Code Mode Rules (Non-Obvious Only)

## Project-Specific Patterns
- Error wrapping: MUST use `github.com/pkg/errors` package (not standard library errors)
- Tools directory: All binaries install to `./bin/` (GOBIN override in Makefile)
- Container runtime detection: Makefile auto-detects podman/docker, don't hardcode
- Buildkit usage: Uses buildctl with containerized buildkitd (not direct docker build)

## Kubernetes Operator Specifics
- Finalizers: Two versions exist (finalizerV1 and finalizerV3) in controllers, prefer finalizerV3
- ETCD discovery: Automatic on OpenShift - copies certs from openshift-etcd namespace to instana-agent
- Namespace handling: Default namespace is `instana-agent`, not `default`
- CRD versions: InstanaAgent has both v1 and v1beta1, RemoteAgent only v1
- OpenShift detection: Uses `oc` command availability to detect OCP vs vanilla K8s

## Testing Requirements
- E2E tests: MUST have `e2e/.env` file (copy from `e2e/.env.example`)
- Test exclusions: `make test` excludes `mocks` and `e2e` directories automatically
- KUBEBUILDER_ASSETS: Set automatically by envtest for unit tests
- Secrets baseline: Run `detect-secrets scan --update .secrets.baseline` before committing

## Build System Quirks
- Multi-arch builds: Requires both BUILDPLATFORM and TARGETPLATFORM args
- Version injection: Uses VERSION, GIT_COMMIT, and DATE build args
- Buildkit container: Auto-managed, runs as privileged container named "buildkitd"
- Image loading: Uses `$(CONTAINER_CMD) load` after buildctl build