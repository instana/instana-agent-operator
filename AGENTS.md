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

### Fetching Logs
- Fetch logs: `ibmcloud dev tekton-logs <PIPELINE_ID> --run-id <RUN_ID>`
- Filter by task: Add `--task-name <task>` to narrow logs to specific task
- Available tasks (from `.pipeline-config.yaml`): `pr-code-checks`, `pr-code-checks-2`, `pr-code-checks-3`, `code-pr-finish`
- Structured output: Add `--output json` for JSON format
- Debug mode: Add `--trace` for extra debug details
- URL format: `https://cloud.ibm.com/devops/pipelines/tekton/{PIPELINE_ID}/runs/{RUN_ID}/{task-name}/{step-name}?env_id={env}&view=logs`
- Extract PIPELINE_ID and RUN_ID from URL to use with CLI commands
- **GitHub PR status checks**: SPS pipeline URLs are available in PR status check "Details" links - if a check is failing, click Details to get the Tekton URL, then extract PIPELINE_ID/RUN_ID to fetch logs and pinpoint the issue

### Log Parsing for AI Agents (CRITICAL for Token Efficiency)

**Problem**: Raw Tekton logs are extremely verbose (7,000+ lines) with only 50-100 lines of actual failure information, causing high token costs (~145K tokens vs ~3K tokens parsed).

**Solution**: ALWAYS use the log parser before analyzing failures:

```bash
# Recommended workflow - parse logs directly from CLI
ibmcloud dev tekton-logs <PIPELINE_ID> --run-id <RUN_ID> | ./ci/sps-scripts/parse-tekton-logs.sh > failure-summary.txt

# Then analyze the parsed output (98% token reduction)
# failure-summary.txt contains only: test failures, panic traces, error messages, exit codes
```

**What the parser extracts**:
- Test failures with 10 lines of context
- Panic stack traces with full call chains
- Significant error messages (filtered)
- Exit codes and make errors
- Test execution summary

**What gets filtered out** (noise):
- Docker/containerd initialization (thousands of lines)
- Package installation output (yum, apt, go get)
- Debug messages and trace logs
- Known warnings (AUFS, GPG keys)
- Successful test output

**Best practices**:
1. Always parse before analyzing - saves 95-98% of tokens
2. Use task-specific parsing when possible: `--task-name pr-code-checks-2` (for Go unit tests)
3. Archive parsed summaries instead of full logs
4. For specific issues: `./ci/sps-scripts/parse-tekton-logs.sh log.txt | grep -A 5 "TestName"`

**Token savings**: ~145K tokens → ~3K tokens (98% reduction, ~$0.15-0.30 → ~$0.003-0.006 per analysis)