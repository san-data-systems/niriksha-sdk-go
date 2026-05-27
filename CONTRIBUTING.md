# Contributing to niriksha-sdk-go

Thank you for helping improve the NirikshaAI Go SDK!  
Product: [niriksha.ai](https://niriksha.ai) · Company: [sandatasystem.ai](https://sandatasystem.ai)

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

## Branch Model & Strategy

This SDK follows a **two-branch release model**:

| Branch | Purpose | Protection | Auto Release |
|--------|---------|-----------|--------------|
| `develop` | Feature work, dev releases | Require PR + CI pass | Yes — `vX.Y.Z-dev.SHA` pre-release on every merge |
| `main` | Production releases only | Require PR from `develop` only + CI pass + `govulncheck` green | Yes — semver GitHub Release on merge |

### Workflow

1. **Create feature branches from `develop`**, not `main`:
   ```bash
   git checkout develop && git pull
   git checkout -b feature/your-feature
   ```

2. **Open PR against `develop`**:
   - All CI checks must pass (lint, test, vuln scan)
   - Code review required
   - After merge → dev release auto-triggers (`vX.Y.Z-dev.SHA`)

3. **When ready for production**, create a PR from `develop` → `main`:
   - Must come from `develop` (branch gate enforces this)
   - CI + `govulncheck` must be green (vulnerability gate)
   - After merge → auto-versioning computes next semver tag
   - Production GitHub Release created with `vX.Y.Z` tag

See [RELEASE.md](RELEASE.md) for the complete release process and versioning details.

## Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/) — they drive auto-versioning:

```
feat: add support for custom sampling callbacks     # → minor version bump
fix: retry on HTTP 503 from eval API               # → patch version bump
chore: upgrade Go to 1.25.10                        # → patch version bump
docs: add gRPC example to README                    # → no version bump (docs only)

# Breaking changes bump major version:
feat: remove deprecated Config.UseOldAPI

BREAKING CHANGE: UseOldAPI option removed in favor of NewAPI
```

Commit messages are processed by `mathieudutour/github-tag-action@v6.2` to compute the next semantic version.

## Pull Request Process

1. Create a branch from `develop` (not `main`)
2. Write tests first (TDD) — target 80%+ coverage
3. Run `make ci` to verify lint + test + build + vuln scan pass locally
4. Update `CHANGELOG.md` under `[Unreleased]`
5. Open a PR against `develop` — all CI checks must pass before merge
6. Request review from a maintainer
7. After merge to `develop` → GitHub pre-release auto-triggers
8. When ready for production, maintainer creates PR from `develop` → `main` (branch gate permits only this path)

## Reporting Issues

Security vulnerabilities: see [SECURITY.md](SECURITY.md)  
Bugs and features: [GitHub Issues](https://github.com/san-data-systems/niriksha-sdk-go/issues)
