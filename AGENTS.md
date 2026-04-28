# AGENTS.md

This file provides guidance to agents when working with code in this repository.

## Build & Test Commands
- Run single test: `go test -v ./path/to/package -run TestName`
- Format specific file: `./bin/golangci-lint fmt ./path/to/file`
- E2E tests: `make e2e` (requires `e2e/.env` file - copy from `e2e/.env.example`)
- Secrets baseline: `detect-secrets scan --update .secrets.baseline` before committing

## Project-Specific Rules (Non-Obvious Only)
- Error wrapping: MUST use `github.com/pkg/errors` package (not standard library errors)
- Tools directory: All binaries install to `./bin/` (GOBIN override in Makefile)
- Branch naming: Avoid `/` in branch names (breaks CI pipelines)
- Container builds: Uses buildctl with containerized buildkitd (not direct docker build)
- Commit messages: MUST include DCO sign-off with `git commit -s`
- PR template: Fill out `.github/pull_request_template.md` and use as PR body

## Kubernetes Operator Specifics
- Finalizers: Two versions exist (finalizerV1 and finalizerV3) in controllers, prefer finalizerV3
- ETCD discovery: Automatic on OpenShift - copies certs from openshift-etcd namespace to instana-agent
- Namespace handling: Default namespace is `instana-agent`, not `default`
- CRD versions: InstanaAgent has both v1 and v1beta1, RemoteAgent only v1
- OpenShift detection: Uses `oc` command availability to detect OCP vs vanilla K8s
- Local dev: `make dev-run-cluster` for full setup, `make run` to run operator locally
- Cleanup: `make purge` removes all cluster resources including finalizers

## Build System Quirks
- Multi-arch builds: Requires both BUILDPLATFORM and TARGETPLATFORM args
- Version injection: Uses VERSION, GIT_COMMIT, and DATE build args
- Buildkit container: Auto-managed, runs as privileged container named "buildkitd"
- Image loading: Uses `$(CONTAINER_CMD) load` after buildctl build

## CI/CD Pipeline Logs (SPS/Tekton)
- Fetch logs: `ibmcloud dev tekton-logs <PIPELINE_ID> --run-id <RUN_ID>`
- Filter by task: Add `--task-name <task>` to narrow logs to specific task
- Structured output: Add `--output json` for JSON format
- Debug mode: Add `--trace` for extra debug details
- URL format: `https://cloud.ibm.com/devops/pipelines/tekton/{PIPELINE_ID}/runs/{RUN_ID}/{task-name}/{step-name}?env_id={env}&view=logs`
- Extract PIPELINE_ID and RUN_ID from URL to use with CLI commands
- **GitHub PR status checks**: SPS pipeline URLs are available in PR status check "Details" links - if a check is failing, click Details to get the Tekton URL, then extract PIPELINE_ID/RUN_ID to fetch logs and pinpoint the issue