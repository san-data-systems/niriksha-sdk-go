# Release Guide — niriksha-sdk-go

> Product: [niriksha.ai](https://niriksha.ai) · Company: [San Data Systems](https://sandatasystem.ai)  
> Module path: `github.com/san-data-systems/niriksha-sdk-go`  
> Minimum Go version: 1.22

---

## Overview

This SDK uses a **fully automated two-branch release model**:

- **`develop` branch**: Feature work merges here → CI passes → Dev build auto-triggers (`vX.Y.Z-dev.SHA` pre-release)
- **`main` branch**: Only `develop` merges here → CI + `govulncheck` pass → Production release auto-triggers (semver GitHub Release)

**No manual version bumping.** Semantic versioning is computed from conventional commit messages:
- `feat:` → minor bump (`0.1.0` → `0.2.0`)
- `fix:` / `chore:` → patch bump (`0.1.0` → `0.1.1`)
- `BREAKING CHANGE` footer → major bump (`0.1.0` → `1.0.0`)

The workflow is driven by **GitHub Actions** + **`mathieudutour/github-tag-action@v6.2`**.

---

## Branching Strategy

```
┌─────────────────────────────────────────────────────────────────────┐
│                           GitHub Repository                          │
└─────────────────────────────────────────────────────────────────────┘
                                    │
                                    ├── develop (default branch)
                                    │   ├── feature/foo (PR → develop)
                                    │   ├── feature/bar (PR → develop)
                                    │   └── fix/bug (PR → develop)
                                    │
                                    └── main (production branch)
                                        └── PR from develop only
```

### Branch Protections (configure in GitHub → Settings → Branches)

| Branch | Protection Rules |
|--------|-----------------|
| `develop` | ✅ Require pull request ✅ Require status checks to pass (CI) |
| `main` | ✅ Require pull request ✅ Require status checks to pass (CI + govulncheck) ✅ Restrict who can push (admin only) ✅ Require branches to be up to date |

### Branch Gate (CI Job)

A GitHub Actions job in `ci.yml` enforces: **Only PRs from `develop` can merge to `main`**.

```yaml
# Example job in your ci.yml:
- name: Check PR source branch
  if: github.event_name == 'pull_request' && github.base_ref == 'main'
  run: |
    if [ "${{ github.head_ref }}" != "develop" ]; then
      echo "Only 'develop' can merge to 'main'"
      exit 1
    fi
```

---

## Release Workflows

### 1. Development Release (Automatic)

**When**: Every merge to `develop`  
**Trigger**: `dev-release.yml` workflow  
**Output**: GitHub pre-release tagged `vX.Y.Z-dev.<short-sha>` (e.g., `v0.1.0-dev.a1b2c3d`)

**What happens automatically:**
1. New commit merged to `develop`
2. `dev-release.yml` runs
3. Reads the current version from `go.mod` or a version constant
4. Computes tag: `v{version}-dev.{short-sha}`
5. Creates GitHub pre-release (not a stable release)
6. Users can test with: `go get github.com/san-data-systems/niriksha-sdk-go@v0.1.0-dev.a1b2c3d`

**No secrets required** — uses auto-provided `GITHUB_TOKEN`.

---

### 2. Production Release (Automated)

**When**: PR from `develop` → `main` is merged  
**Trigger**: `release.yml` workflow (runs on tags matching `v*`)  
**Output**: GitHub Release with semver tag + auto-indexed on pkg.go.dev

**Workflow:**

1. **Create release PR** (maintainer only):
   ```bash
   git checkout develop && git pull
   git checkout -b release/prepare
   # Nothing to edit — just prepare for the PR
   git push -u origin release/prepare
   gh pr create --base main --title "chore: release v0.2.0" \
     --body "Release v0.2.0 with new features"
   ```

2. **CI validates**:
   - `lint` ✅
   - `test` ✅
   - `govulncheck` ✅ (blocks merge if any CVE detected)
   - Branch gate ✅ (only `develop` can merge to `main`)

3. **Merge to `main`**:
   ```bash
   # After PR is approved and CI is green, merge via GitHub UI or:
   gh pr merge <PR-number> --merge
   ```

4. **Auto-versioning runs** (on merge commit):
   - `mathieudutour/github-tag-action@v6.2` analyzes commits since last tag
   - Computes new version (e.g., `0.2.0` based on `feat:` commits)
   - Creates and pushes tag: `v0.2.0`

5. **Release workflow runs** (triggered by `v0.2.0` tag):
   - Creates GitHub Release for `v0.2.0`
   - Publishes release notes
   - Go module proxy auto-indexes within ~15 minutes on pkg.go.dev

**No secrets required** — uses auto-provided `GITHUB_TOKEN`.

---

## How Go Modules Are Distributed

Go doesn't use a central registry like npm or PyPI. Instead:

1. **VCS-based**: The module is fetched directly from GitHub
2. **Tag-based**: `go get` resolves versions from Git tags (`v0.1.0`, `v0.2.0`, etc.)
3. **Auto-indexed**: `pkg.go.dev` indexes new tags automatically within ~15 minutes

**Users install with:**
```bash
# Stable release
go get github.com/san-data-systems/niriksha-sdk-go@v0.2.0

# Latest from develop (go computes a pseudo-version)
go get github.com/san-data-systems/niriksha-sdk-go@develop

# Specific dev build
go get github.com/san-data-systems/niriksha-sdk-go@v0.2.0-dev.abc1234
```

---

## Commit Message Format (Important!)

Conventional commits drive auto-versioning. Always follow the format:

```
<type>: <description>

<optional body>

<optional footer>
```

### Examples

**Patch bump** (e.g., `0.1.0` → `0.1.1`):
```
fix: retry on HTTP 503 from eval API

When the eval API returns 503, retry with exponential backoff
instead of failing immediately.
```

**Minor bump** (e.g., `0.1.0` → `0.2.0`):
```
feat: add support for custom OpenTelemetry attributes

Users can now pass custom attributes to Init() which are applied
to all spans, metrics, and logs.
```

**Major bump** (e.g., `0.1.0` → `1.0.0`):
```
feat: redesign Config structure

BREAKING CHANGE: The Options struct has been refactored.
See MIGRATION.md for upgrade instructions.
```

### Commit Types

| Type | Effect | When to use |
|------|--------|-----------|
| `feat` | Minor version bump | New feature |
| `fix` | Patch version bump | Bug fix |
| `chore` | Patch version bump | Dependency updates, config, tooling |
| `docs` | No version bump | Documentation only |
| `refactor` | No version bump | Code restructure (no behavior change) |
| `perf` | No version bump | Performance optimization |
| `test` | No version bump | Test additions/fixes |
| `ci` | No version bump | CI/CD workflow changes |

---

## Vulnerability Gating

`govulncheck` must pass on every merge to `main`. This job:

1. Scans all dependencies for known CVEs
2. Blocks the merge if any vulnerability is detected
3. Requires vulnerabilities to be remediated before release

```bash
# Run locally before pushing:
make govulncheck
# or manually:
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
```

---

## Troubleshooting

### Merge to `main` blocked by branch gate

**Error**: "Only 'develop' can merge to 'main'"

**Solution**: The PR must originate from the `develop` branch. If you accidentally branched from `main`, you must:
1. Rebase your branch onto `develop`
2. Create a new PR against `main` with `develop` as the source

```bash
git checkout your-branch
git rebase develop
git push --force-with-lease
# Close old PR, create new one against main from develop
```

### Merge blocked by `govulncheck`

**Error**: "Failed due to vulnerable dependency"

**Solution**:
1. Identify the vulnerable package: `go run golang.org/x/vuln/cmd/govulncheck@latest ./...`
2. Upgrade or patch the dependency:
   ```bash
   go get -u path/to/vulnerable/package
   ```
3. Run `go mod tidy` and commit
4. Push — CI will re-run `govulncheck`

### Version tag not created

**Symptom**: Merge to `main` completed but no GitHub Release appeared.

**Debugging**:
1. Check GitHub Actions → `release.yml` workflow logs
2. Verify commits since last tag use conventional commit format
3. If commits are non-standard, manually tag:
   ```bash
   git checkout main && git pull
   git tag -a v0.2.0 -m "Release v0.2.0"
   git push origin v0.2.0
   ```

---

## Release Checklist

Before merging a release PR to `main`:

- [ ] All commits to `develop` are merged (no pending features)
- [ ] `CHANGELOG.md` is updated — `[Unreleased]` section reviewed
- [ ] All CI checks on the release PR are passing:
  - [ ] Lint (`golangci-lint`)
  - [ ] Tests (with `-race` detector)
  - [ ] Build
  - [ ] `govulncheck` (no CVEs)
- [ ] Branch gate check passes (PR originates from `develop`)
- [ ] No other PRs to `main` are in-flight
- [ ] Merge strategy is set to "Create a merge commit" (not squash)

---

## FAQ

**Q: How do I fix a typo in a commit message after pushing?**  
A: Use `git commit --amend` and `git push --force-with-lease` on your branch (before merge to `develop`). After merge, use `git revert` instead.

**Q: What if I need to release a hotfix?**  
A: Create the fix on `develop` first, test it, then merge to `main` via the normal PR process. The branch gate ensures all production code goes through `develop`.

**Q: Can I push directly to `main` or `develop`?**  
A: No. Both branches are protected and require PRs. Use feature branches and open PRs.

**Q: How often do I update the version constant?**  
A: Never manually. The auto-versioning action reads commit history and computes the next version automatically. Just use conventional commits.

**Q: Where does pkg.go.dev get the docs?**  
A: From `doc.go` comments and JSDoc in your public functions. After a tag is pushed, pkg.go.dev indexes it within ~15 minutes.

---

## Rollback

If a production release has a critical bug:

```bash
# 1. Create a fix on develop
git checkout develop && git pull
git checkout -b fix/critical-issue
# Fix code...
git commit -m "fix: critical bug in span export"
git push -u origin fix/critical-issue
gh pr create --base develop

# 2. After merge to develop, create PR to main
# 3. After merge to main, the bad release tag will still exist
# 4. Users on the bad version can upgrade to the new version

# Optional: delete the bad tag locally (do NOT force-push to main)
git tag -d v0.1.1
# This doesn't remove it from GitHub, but prevents accidental use
```

New users will install the fixed version via `go get -u`, and pkg.go.dev will prioritize the newer stable release.

---

For contributing guidelines, see [CONTRIBUTING.md](CONTRIBUTING.md).  
For security issues, see [SECURITY.md](SECURITY.md).
