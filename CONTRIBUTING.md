# Contributing to niriksha-sdk-go

Thank you for helping improve the NirikshaAI Go SDK!  
Product: [niriksha.ai](https://niriksha.ai)

## Development Setup

```bash
# Clone and install tools
git clone https://github.com/san-data-systems/niriksha-sdk-go
cd niriksha-sdk-go

# Install golangci-lint
brew install golangci-lint        # macOS
# or: curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh

# Run tests
make test

# Run linter
make lint

# Check for vulnerabilities
make govulncheck
```

## Branch Naming & Strategy

See [RELEASE.md](RELEASE.md) for the full branching strategy and release process.

| Branch prefix | Purpose | Merges into |
|---------------|---------|-------------|
| `feature/` | New features | `main` |
| `fix/` | Bug fixes | `main` |
| `hotfix/` | Urgent production patches | `main` |
| `enhance/` | CI, docs, deps improvements | `main` |
| `release/x.y.z` | Release preparation | `main` |

**Rule:** Branch from `main`, PR back to `main`. Keep branches short-lived.

## Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add support for custom sampling callbacks
fix: retry on HTTP 503 from eval API
docs: add gRPC example to README
```

## Pull Request Process

1. Create a branch from `main`
2. Write tests first (TDD) — target 80%+ coverage
3. Run `make ci` to verify lint + test + build pass locally
4. Update `CHANGELOG.md` under `[Unreleased]`
5. Open a PR — all CI checks must pass before merge
6. Request review from a maintainer

## Reporting Issues

Security vulnerabilities: see [SECURITY.md](SECURITY.md)  
Bugs and features: [GitHub Issues](https://github.com/san-data-systems/niriksha-sdk-go/issues)
