# AGENTS.md

This file provides guidance to agents when working with code in this repository.

## Build & Test Commands
- Build: `make build`
- Lint: `make lint`
- Format specific file: `./bin/golangci-lint fmt ./path/to/file`
- Run tests: `make test` (excludes mocks and e2e directories)
- Run single test: `go test -v ./path/to/package -run TestName`
- Run e2e tests: `make e2e` (requires e2e/.env file with credentials)
- Install golangci-lint if missing: `make golangci-lint`
- Validate renovate.json: `npx --yes --package renovate -- renovate-config-validator`
- Detect secrets update: `detect-secrets scan --update .secrets.baseline`
- Detect secrets audit: `detect-secrets audit .secrets.baseline`

## Code Style
- Go version: Check `go.mod` toolchain directive (currently go 1.25.0)
- Line length: 120 characters max (enforced by lll linter)
- Formatters: gofmt, goimports, golines (via golangci-lint)
- Linters: govet, ineffassign, unused, misspell, exhaustive, errcheck, lll
- Imports: Group standard library, external, and internal imports
- Error handling: Always check errors, use `pkg/errors` for wrapping
- Commit messages: Include DCO sign-off with `git commit -s`. Use conventional commits syntax for subject, concise message with brief summary in body
- Types: Use strong typing, avoid interface{} when possible
- Tests: Write unit tests for all new functionality
- Naming: Follow Go conventions (CamelCase for exported, camelCase for private)

## Pull Requests
- Fill out pull request template in .github/pull_request_template.md and use as PR body

## Project-Specific Rules
- Branch naming: Avoid `/` in branch names (breaks CI pipelines)
- CRD changes: Always update docs when changing Custom Resource Definitions
- Tools location: All tools install to `./bin/` directory (GOBIN is set to this)
- Container runtime: Makefile auto-detects podman or docker
- E2E tests: Require `e2e/.env` file (copy from `e2e/.env.example`)
- Buildkit: Uses buildctl with containerized buildkitd (auto-managed by Makefile)

## Kubernetes Operator Specifics
- Framework: Kubebuilder v3 with controller-runtime
- CRDs: InstanaAgent (v1, v1beta1) and RemoteAgent (v1)
- Namespace: Default is `instana-agent`
- ETCD discovery: Automatic on OpenShift (copies certs from openshift-etcd namespace)
- Local dev: Use `make dev-run-cluster` for full setup or `make run` to run operator locally
- Cleanup: Use `make purge` to remove all cluster resources including finalizers