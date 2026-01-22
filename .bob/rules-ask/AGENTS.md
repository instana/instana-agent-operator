# Ask Mode Rules (Non-Obvious Only)

## Documentation Context
- README.md: Contains local dev setup with Minikube and two deployment options
- docs/ directory: Contains ADRs and specific feature documentation (ETCD, secrets, env vars)
- config/samples/: Contains example CRDs - use these as canonical references
- Operator uses Kubebuilder v3 framework (see PROJECT file for structure)

## Non-Standard Directory Organization
- `e2e/`: End-to-end tests require `.env` file (not checked in, copy from `.env.example`)
- `bin/`: All tools install here (GOBIN override), not in GOPATH
- `ci/`: Contains both pipeline configs and helper scripts in ci/scripts/
- `version/`: Contains FEDRAMP_VERSION file for FedRAMP-specific versioning
- `hack/`: Contains bundle patching scripts for OLM

## Hidden Requirements
- OpenShift detection: Uses `oc` command presence to determine cluster type
- ETCD metrics: Auto-configured on OpenShift by copying certs from openshift-etcd namespace
- Buildkit: Requires containerized buildkitd (not direct docker/podman build)
- Multi-arch: Dockerfile requires both BUILDPLATFORM and TARGETPLATFORM args
- Secrets: Must run detect-secrets before committing (pre-commit hook configured)

## Testing Context
- Unit tests: Exclude `mocks` and `e2e` directories automatically
- E2E tests: Run against real cluster, need credentials in `e2e/.env`
- KUBEBUILDER_ASSETS: Auto-set by envtest, don't manually configure
- Test timeout: Default 30s may be too short for e2e tests (extend in IDE settings)

## Operator-Specific Knowledge
- Two CRD versions: InstanaAgent (v1, v1beta1) and RemoteAgent (v1 only)
- Two finalizer versions: finalizerV1 and finalizerV3 (both in use)
- Default namespace: `instana-agent` (not `default`)
- Cleanup: Use `make purge` to remove all resources including stuck finalizers