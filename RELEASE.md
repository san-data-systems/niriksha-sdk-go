# Release Guide — niriksha-sdk-go

> Product: [niriksha.ai](https://niriksha.ai) · Company: [sandatasystem.ai](https://sandatasystem.ai)  
> Maintainer: vbhadauriya@redcloudcomputing.com

---

## Versioning Scheme

This SDK follows [Semantic Versioning 2.0.0](https://semver.org):

```
v MAJOR . MINOR . PATCH
  │       │       └── Bug fixes, security patches (backwards compatible)
  │       └────────── New features (backwards compatible)
  └────────────────── Breaking API changes
```

### Version Lifecycle

| Version Pattern | Meaning | Published to |
|----------------|---------|-------------|
| `v0.1.0-dev.abc1234` | Auto dev build (every merge to `main`) | GitHub pre-release + pkg.go.dev |
| `v0.1.0-alpha.1` | Alpha — early feature preview | GitHub pre-release + pkg.go.dev |
| `v0.1.0-beta.1` | Beta — feature complete, needs testing | GitHub pre-release + pkg.go.dev |
| `v0.1.0-rc.1` | Release candidate — final testing | GitHub pre-release + pkg.go.dev |
| `v0.1.0` | Stable release | GitHub Release + pkg.go.dev |
| `v1.0.0` | First stable API contract | GitHub Release + pkg.go.dev |

> **Why not v0.0.0?** We start at `v0.1.0`. `v0.0.0` is a placeholder meaning "not yet versioned". Even our pre-1.0 releases have real version numbers. `v0.x.y` means the public API may still evolve; `v1.0.0` signals a stable, committed public API.

### Install a specific version

```bash
# Stable
go get github.com/san-data-systems/niriksha-sdk-go@v0.1.0

# Latest from main (Go computes a pseudo-version automatically)
go get github.com/san-data-systems/niriksha-sdk-go@main

# Specific dev build
go get github.com/san-data-systems/niriksha-sdk-go@v0.1.0-dev.abc1234

# Pre-release
go get github.com/san-data-systems/niriksha-sdk-go@v0.1.1-beta.1
```

> **Go module note:** Go modules automatically publish when a tag is pushed. No registry upload step is needed — the module proxy (proxy.golang.org) fetches from GitHub automatically.

---

## Branching Strategy

```
main                  ← Protected. Every merge auto-creates a dev pre-release.
│
├── feature/xxx       ← New features. PR → main.
├── fix/xxx           ← Bug fixes. PR → main.
├── hotfix/xxx        ← Urgent production patches. PR → main.
├── enhance/xxx       ← Improvements (docs, CI, deps). PR → main.
└── release/x.y.z     ← Release preparation. PR → main, then tag.
```

### Branch rules (configure in GitHub → Settings → Branches)

| Branch | Protection |
|--------|-----------|
| `main` | Require PR, require CI to pass, no force-push |

---

## Release Types

### 1. Patch Release (v0.1.0 → v0.1.1)
**When:** Bug fix, security patch, dependency bump. No new public API.

```bash
# 1. Branch from main
git checkout main && git pull
git checkout -b release/0.1.1

# 2. Bump version constant in nirikshaai.go
#    const Version = "0.1.0"  →  const Version = "0.1.1"
vim nirikshaai.go

# 3. Update CHANGELOG.md
#    Move [Unreleased] entries to [0.1.1] with today's date

# 4. Commit and open PR
git add nirikshaai.go CHANGELOG.md
git commit -m "chore: release 0.1.1"
git push -u origin release/0.1.1
gh pr create --base main --title "chore: release 0.1.1"

# 5. After PR merged to main, tag it
git checkout main && git pull
git tag -a v0.1.1 -m "Release v0.1.1"
git push origin v0.1.1
# → release.yml creates GitHub Release and triggers pkg.go.dev indexing
```

### 2. Minor Release (v0.1.0 → v0.2.0)
**When:** New backwards-compatible features.

Same steps, bump minor: `const Version = "0.2.0"`, tag `v0.2.0`.

### 3. Major Release (v0.x.y → v1.0.0)
**When:** Breaking public API changes.

```bash
git checkout -b release/1.0.0
# const Version = "1.0.0"
# Update go.mod if needed
# Add migration guide to README
```

> **Go major version note:** For v2+, the module path must change:  
> `module github.com/san-data-systems/niriksha-sdk-go/v2`  
> See [Go module major versions](https://go.dev/blog/v2-go-modules).

### 4. Pre-release (alpha / beta / RC)

```bash
# Alpha
git tag -a v0.2.0-alpha.1 -m "Alpha 1 for v0.2.0"
git push origin v0.2.0-alpha.1

# Beta
git tag -a v0.2.0-beta.1 -m "Beta 1 for v0.2.0"
git push origin v0.2.0-beta.1

# Release Candidate
git tag -a v0.2.0-rc.1 -m "RC 1 for v0.2.0"
git push origin v0.2.0-rc.1
```

All pre-release tags trigger `release.yml`, published as GitHub pre-releases.

### 5. Dev Build (automatic)
**When:** Every merge to `main` — no manual action required.

The `dev-release.yml` workflow automatically:
1. Computes tag `v{version}-dev.{short-sha}`
2. Pushes the tag to GitHub
3. Creates a GitHub pre-release
4. Triggers pkg.go.dev indexing

Users install with:
```bash
go get github.com/san-data-systems/niriksha-sdk-go@main
# or a specific dev tag:
go get github.com/san-data-systems/niriksha-sdk-go@v0.1.0-dev.abc1234
```

---

## Required Secrets & Setup (One-time)

> **Important:** All credentials below must be created or managed under the **niriksha.ai product account / `san-data-systems` GitHub org**, not a personal developer account. This keeps niriksha tokens separate from other San Data Systems products.

| Secret | Purpose | How to get |
|--------|---------|-----------|
| `GITHUB_TOKEN` | Create releases, push tags | Auto-provided by GitHub Actions — no setup needed |

> Go modules require no registry credentials — publishing is just pushing a tag. pkg.go.dev indexes from GitHub automatically.

| Step | Action | URL |
|------|--------|-----|

---

## Release Checklist

- [ ] All CI checks green on `main`
- [ ] `make test` passes (race detector on)
- [ ] `make lint` clean (golangci-lint)
- [ ] `make govulncheck` clean
- [ ] CHANGELOG.md updated — `[Unreleased]` moved to `[x.y.z]` with date
- [ ] `Version` constant bumped in `nirikshaai.go`
- [ ] PR merged to `main`
- [ ] Tag pushed: `git tag -a vX.Y.Z -m "Release vX.Y.Z" && git push origin vX.Y.Z`
- [ ] GitHub Release created (auto by `release.yml`)
- [ ] pkg.go.dev updated (auto-indexed within ~15 min)

---

## Hotfix Process

```bash
git checkout main && git pull
git checkout -b hotfix/fix-description

# Fix + test + bump patch version
make test

git commit -m "fix: critical bug"
git push -u origin hotfix/fix-description
gh pr create --base main --title "hotfix: critical bug"

# After merge
git checkout main && git pull
git tag -a v0.1.1 -m "Hotfix: critical bug"
git push origin v0.1.1
```

---

## CHANGELOG Management

Format: [Keep a Changelog](https://keepachangelog.com/en/1.1.0/)

```markdown
## [Unreleased]          ← new changes go here first
## [0.1.1] - 2025-06-01 ← moved here when releasing
## [0.1.0] - 2025-05-01
```

Every PR must include a CHANGELOG entry under `[Unreleased]`.
