## Build & Test Commands
- Build: `make build`
- Lint: `make lint`
- Format: `make fmt`
- Run tests: `make test`
- Run single test: `go test -v ./path/to/package -run TestName`
- Run e2e tests: `make e2e`
- Generate mocks: `make gen-mocks`

## Code Style Guidelines
- Go version: 1.24
- Line length: 120 characters max
- Formatting: Use `gofmt`, `goimports`, and `golines`
- Linters: `govet`, `ineffassign`, `unused`, `misspell`, `exhaustive`, `errcheck`, `lll`
- Linter errors: `./bin/golangci-lint fmt ./path/to/file`
- If golangeci-lint is not found, install it via make: `make golangci-lint`
- Imports: Group standard library, external, and internal imports
- Error handling: Always check errors, use `pkg/errors` for wrapping
- Commit messages: Include DCO sign-off with `git commit -s`
- Types: Use strong typing, avoid interface{} when possible
- Tests: Write unit tests for all new functionality
- Documentation: Update docs when changing public APIs, especially on Custom Resource Definition changes
- Naming: Follow Go conventions (CamelCase for exported, camelCase for private)

## Pull Requests
- If you create a pull request (PR) fill out the pull request template in .github/pull_request_template.md and use it as the PR body.
- If you create new branches, avoid using `/` in the name, this will break CI pipelines.
