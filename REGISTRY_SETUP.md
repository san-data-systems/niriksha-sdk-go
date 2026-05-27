# Go Module Distribution Setup

> NirikshaAI Go SDK: `github.com/san-data-systems/niriksha-sdk-go`  
> Product: [niriksha.ai](https://niriksha.ai) · Company: [San Data Systems](https://sandatasystem.ai)  
> Maintainer: vbhadauriya@sandatasystem.com

---

## Overview

**Go modules do not require a registry account.** The SDK is distributed via **GitHub + semantic version tags**. When you push a tag like `v0.2.0`, the Go module proxy (`proxy.golang.org`) automatically fetches from GitHub and indexes on `pkg.go.dev`.

**This guide covers:**
1. Ensuring GitHub is configured correctly
2. Understanding Go module resolution
3. Verifying pkg.go.dev auto-indexing

---

## Prerequisites

- Repository: `github.com/san-data-systems/niriksha-sdk-go` (must be **public**)
- Go version: 1.22+
- `go.mod` file with module path: `github.com/san-data-systems/niriksha-sdk-go`

---

## 1. Repository Configuration (One-time)

### Verify the repository is public

1. Go to [github.com/san-data-systems/niriksha-sdk-go/settings](https://github.com/san-data-systems/niriksha-sdk-go/settings)
2. Scroll to **"Danger Zone"** → **Repository Visibility**
3. Ensure **Public** is selected
4. If private, click **Change visibility** → select **Public** → confirm

### Verify `go.mod` module path

```bash
cd /Users/sandata/2026/niriksha-sdk-go
head -1 go.mod
# Output: module github.com/san-data-systems/niriksha-sdk-go
```

If incorrect, update it:
```go
module github.com/san-data-systems/niriksha-sdk-go
```

---

## 2. How Go Module Distribution Works

### User Installation

Users install your SDK with:

```bash
# Latest stable release
go get github.com/san-data-systems/niriksha-sdk-go

# Specific version
go get github.com/san-data-systems/niriksha-sdk-go@v0.2.0

# From develop branch (pseudo-version computed automatically)
go get github.com/san-data-systems/niriksha-sdk-go@develop

# Specific dev build
go get github.com/san-data-systems/niriksha-sdk-go@v0.2.0-dev.abc1234
```

### How it Works (No Registry Upload)

1. **User runs `go get`**
2. **Go fetches module list** from `proxy.golang.org` (Go module proxy)
3. **Proxy fetches from GitHub** automatically (your repo is public)
4. **Proxy caches the module** for fast future downloads
5. **pkg.go.dev indexes tags** automatically (within ~15 minutes)

**No manual upload step. No registry credentials needed.**

---

## 3. Release Workflow (Automated)

### Production Release

Every merge to `main` triggers auto-versioning. The `release.yml` workflow:

1. Reads commit history since last tag
2. Uses conventional commits to compute next version (e.g., `0.2.0`)
3. Creates and pushes Git tag: `v0.2.0`
4. Creates GitHub Release
5. Go proxy fetches the tag → users can `go get @v0.2.0`
6. pkg.go.dev indexes within ~15 minutes

**No secrets required** — only auto-provided `GITHUB_TOKEN`.

### Development Release

Every merge to `develop` triggers `dev-release.yml`:

1. Computes tag: `v0.1.0-dev.a1b2c3d` (short commit SHA)
2. Creates GitHub pre-release
3. Users can test with: `go get github.com/san-data-systems/niriksha-sdk-go@v0.1.0-dev.a1b2c3d`

---

## 4. Verifying Distribution

### After pushing a tag (e.g., `v0.2.0`)

#### Check GitHub Release

1. Go to [github.com/san-data-systems/niriksha-sdk-go/releases](https://github.com/san-data-systems/niriksha-sdk-go/releases)
2. Verify `v0.2.0` appears

#### Check Go Module Proxy

```bash
# This should return success (204 or 200):
curl -I https://proxy.golang.org/github.com/san-data-systems/niriksha-sdk-go/@v/v0.2.0.info
```

#### Test installation

```bash
# Create a test directory
mkdir -p /tmp/gotest && cd /tmp/gotest
go mod init test
go get github.com/san-data-systems/niriksha-sdk-go@v0.2.0

# Verify it's installed
go list -m all | grep niriksha
# Output: github.com/san-data-systems/niriksha-sdk-go v0.2.0
```

#### Force pkg.go.dev indexing (optional)

If the module doesn't appear on pkg.go.dev within 15 minutes, force indexing:

```bash
# This triggers pkg.go.dev to fetch and index the tag immediately:
go get -d github.com/san-data-systems/niriksha-sdk-go@v0.2.0
```

Then visit: https://pkg.go.dev/github.com/san-data-systems/niriksha-sdk-go@v0.2.0

---

## 5. Troubleshooting

### Module not found after push

**Symptom**: `go get` fails with `no matching versions for query "v0.2.0"`

**Cause**: The module proxy hasn't fetched the tag yet (can take ~5 minutes)

**Solution**:
1. Wait 5 minutes and retry
2. Verify the tag was pushed: `git ls-remote --tags origin | grep v0.2.0`
3. If tag missing locally, force-fetch: `git fetch origin tag v0.2.0`

### pkg.go.dev shows old version

**Symptom**: pkg.go.dev still displays `v0.1.0` after releasing `v0.2.0`

**Cause**: pkg.go.dev's index hasn't refreshed (normal lag)

**Solution**: Force reindex by visiting:
```
https://pkg.go.dev/github.com/san-data-systems/niriksha-sdk-go@v0.2.0?tab=doc
```

This will trigger pkg.go.dev to fetch and cache the new version.

### Release workflow didn't create a tag

**Symptom**: Commits merged to `main` but no new tag/release appears

**Cause**: Commits don't follow conventional commit format

**Solution**: Verify recent commits use proper format:
```bash
git log main --oneline -10
# Should show: feat: ..., fix: ..., chore: ...
# NOT: update, work in progress, merge pull request (auto-generated)
```

If commits are non-standard, manually tag:
```bash
git checkout main && git pull
git tag -a v0.2.0 -m "Release v0.2.0"
git push origin v0.2.0
```

---

## 6. Secrets & Credentials

**No external credentials needed.** The only secret used is:

| Secret | Source | Purpose |
|--------|--------|---------|
| `GITHUB_TOKEN` | Auto-provided by GitHub Actions | Create releases, push tags |

It's automatically injected into every workflow run. No configuration required.

---

## 7. Comparison: Go vs Other Languages

| Language | Registry | Distribution | Credentials |
|----------|----------|---------------|-----------  |
| **Python** | PyPI | Upload package file | PyPI token (OIDC) |
| **Node.js** | npm | Upload package file | npm token |
| **Java** | Maven Central | Upload package file | GPG key + Sonatype token |
| **Go** | None (VCS-based) | Git tag → module proxy | GitHub token (auto) |

Go's approach is simpler: **Tag your repo, the rest is automatic**.

---

## 8. FAQ

**Q: What if someone publishes a module with the same name?**  
A: Impossible. Go modules are tied to their repository URL. Only `san-data-systems` org can push to `github.com/san-data-systems/niriksha-sdk-go`.

**Q: Can I un-publish a version?**  
A: Not really. Once a tag is pushed, the module proxy caches it forever. You can delete the tag and create a new release, but the old one remains in caches. Best practice: use semantic versioning correctly and don't delete tags.

**Q: Do I need to register the module name somewhere?**  
A: No. Go uses the repo URL as the unique identifier. Public GitHub = public module.

**Q: How long until pkg.go.dev shows my release?**  
A: Usually 5–15 minutes. If delayed, force-index by visiting the URL (see Troubleshooting).

**Q: Can I use a private GitHub repo?**  
A: Yes, but users need credentials to `go get` it. For an open-source SDK, keep it public.

---

## Next Steps

1. Ensure repository is public ✅
2. Verify `go.mod` has correct module path ✅
3. Configure branch protection for `main` and `develop` ✅
4. Merge features to `develop`, releases to `main` ✅
5. Push a tag → auto-versioning handles the rest ✅

See [RELEASE.md](RELEASE.md) for the full release workflow.  
See [CONTRIBUTING.md](CONTRIBUTING.md) for development guidelines.
