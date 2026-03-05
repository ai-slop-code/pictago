# Contributing to Pictago

Thanks for your interest in contributing. This document explains how to get set up and what we expect from patches.

## Prerequisites

- **Go 1.21+** (see `go.mod` for the exact version; the project uses a `toolchain` directive for reproducibility).
- Optional: [golangci-lint](https://golangci-lint.run/) and [govulncheck](https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck) for local checks.

## Getting started

1. Fork and clone the repo.
2. From the project root:
   ```bash
   go mod download
   make build
   make test
   ```
3. Run the app: `make run` (or `./pictago`). Default: http://localhost:8080.

## Code style and quality

- **Format**: Run `make fmt` (runs `gofmt -s -w .`). Commit only formatted code.
- **Vet**: Run `make vet` before pushing.
- **Lint**: Run `make lint` if you have golangci-lint installed (`go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest`). CI will run it on push/PR.
- **Tests**: Add or update tests for new or changed behaviour. Run `make test`. Coverage: `make cover` (generates `coverage.out`; view with `go tool cover -html=coverage.out`).
- **Vulnerabilities**: Run `make vulncheck` to check dependencies with govulncheck (`go install golang.org/x/vuln/cmd/govulncheck@latest`). CI runs this as well.

## Before submitting

1. `make fmt`
2. `make test`
3. `make vet`
4. (Optional) `make lint` and `make vulncheck`

CI will run tests, lint, vulncheck, and build on your branch. Fix any failures before requesting review.

## Pull requests

- Keep changes focused; prefer several small PRs over one large one.
- Update the **CHANGELOG** under `[Unreleased]` for user-facing changes.
- New features or config: update **README** (and env table if applicable).

## Security

Do not open public issues for security vulnerabilities. See [SECURITY.md](SECURITY.md) for how to report them.

## License

By contributing, you agree that your contributions will be licensed under the project’s MIT License.
