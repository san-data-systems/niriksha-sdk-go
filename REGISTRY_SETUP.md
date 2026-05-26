# Registry Account Setup Guide

> This guide covers one-time account creation and configuration for all NirikshaAI SDK registries.  
> Product: [niriksha.ai](https://niriksha.ai) · Company: [San Data Systems](https://sandatasystem.ai)  
> Maintainer: vbhadauriya@redcloudcomputing.com

---

## Table of Contents

1. [PyPI — Python SDK](#1-pypi--python-sdk)
2. [npm — Node.js SDK](#2-npm--nodejs-sdk)
3. [Maven Central — Java SDK](#3-maven-central--java-sdk)
4. [GitHub Secrets — All SDKs](#4-github-secrets--all-sdks)
5. [GitHub Environments — Python](#5-github-environments--python)

---

## 1. PyPI — Python SDK

### What it is
[PyPI](https://pypi.org) (Python Package Index) is where Python packages are published. Users install with `pip install niriksha-sdk`.

### 1.1 Create a PyPI Account

1. Go to [pypi.org/account/register](https://pypi.org/account/register/)
2. Fill in:
   - **Username:** `nirikshaai` _(use the product name, not a personal username)_
   - **Email:** use the niriksha.ai product email
   - **Password:** strong password, store in a password manager
3. Verify your email address
4. Enable **Two-Factor Authentication (2FA)**:
   - Go to [pypi.org/manage/account/two-factor](https://pypi.org/manage/account/two-factor/)
   - Use an authenticator app (Google Authenticator, 1Password, etc.)
   - Save recovery codes securely

### 1.2 Create the `niriksha-sdk` Project on PyPI

The project is created automatically the first time you publish. However, you can reserve the name:

1. Log in at [pypi.org](https://pypi.org)
2. The project `niriksha-sdk` will appear at [pypi.org/project/niriksha-sdk](https://pypi.org/project/niriksha-sdk/) after first publish

### 1.3 Configure Trusted Publisher (OIDC — no token needed)

Trusted Publisher lets GitHub Actions publish directly without an API token, using OIDC identity.

1. Log in to PyPI as the `nirikshaai` account
2. Go to [pypi.org/manage/account/publishing](https://pypi.org/manage/account/publishing/)
3. Under **"Add a new pending publisher"**, add **two entries**:

**Entry 1 — Stable releases:**

| Field | Value |
|-------|-------|
| PyPI Project Name | `niriksha-sdk` |
| Owner (GitHub) | `san-data-systems` |
| Repository | `niriksha-sdk-python` |
| Workflow filename | `release.yml` |
| Environment name | `pypi` |

**Entry 2 — Dev builds:**

| Field | Value |
|-------|-------|
| PyPI Project Name | `niriksha-sdk` |
| Owner (GitHub) | `san-data-systems` |
| Repository | `niriksha-sdk-python` |
| Workflow filename | `dev-release.yml` |
| Environment name | `pypi-dev` |

4. Click **Add** for each entry — they appear as "Pending" until first publish

### 1.4 Verify Setup

After merging and tagging `v0.2.0`:
```bash
pip install niriksha-sdk
python -c "import niriksha_sdk; print(niriksha_sdk.__version__)"
```

---

## 2. npm — Node.js SDK

### What it is
[npm](https://npmjs.com) is the Node.js package registry. Users install with `npm install @nirikshaai/sdk`.

### 2.1 Create an npm Account

1. Go to [npmjs.com/signup](https://www.npmjs.com/signup)
2. Fill in:
   - **Username:** `nirikshaai` _(product account, not personal)_
   - **Email:** use the niriksha.ai product email
   - **Password:** strong password
3. Verify your email address
4. Enable **Two-Factor Authentication**:
   - Go to [npmjs.com/settings/nirikshaai/profile](https://www.npmjs.com/settings/nirikshaai/profile)
   - Security → Enable 2FA → choose **"Auth and writes"** mode
   - Save recovery codes

### 2.2 Create the `@nirikshaai` Organization

1. Log in as `nirikshaai`
2. Go to [npmjs.com/org/create](https://www.npmjs.com/org/create)
3. Fill in:
   - **Organization name:** `nirikshaai`
   - **Plan:** Free (for open source packages)
4. The scope `@nirikshaai` is now reserved

### 2.3 Generate an Automation Token

> Use **Automation** token type — it bypasses 2FA enforcement in CI (Classic and Granular tokens fail when 2FA is enabled).

1. Log in as `nirikshaai` at [npmjs.com](https://www.npmjs.com)
2. Go to [npmjs.com/settings/nirikshaai/tokens](https://www.npmjs.com/settings/nirikshaai/tokens)
   _(or: avatar → Access Tokens)_
3. Click **Generate New Token** → **Classic Token**
4. Select type: **Automation**
5. Copy the token immediately — it is shown only once
6. Store it in a password manager

### 2.4 Add Token to GitHub

1. Go to [github.com/san-data-systems/niriksha-sdk-node/settings/secrets/actions](https://github.com/san-data-systems/niriksha-sdk-node/settings/secrets/actions)
2. Click **New repository secret**
3. Name: `NPM_TOKEN`
4. Value: paste the Automation token
5. Click **Add secret**

### 2.5 Verify Setup

After merging and tagging `v0.2.0`:
```bash
npm install @nirikshaai/sdk
node -e "const sdk = require('@nirikshaai/sdk'); console.log('ok')"
```

---

## 3. Maven Central — Java SDK

### What it is
[Maven Central](https://central.sonatype.com) is the standard Java package registry. Users add `ai.niriksha:niriksha-sdk-java` to their `pom.xml` or `build.gradle`.

### 3.1 Create a Sonatype Central Account

1. Go to [central.sonatype.com/sign-up](https://central.sonatype.com/sign-up)
2. Fill in:
   - **Username / email:** use the niriksha.ai product email
   - **Password:** strong password
3. Verify your email
4. Enable **Two-Factor Authentication** in account settings

### 3.2 Verify the `ai.niriksha` Namespace

You must prove you own the `niriksha.ai` domain (which gives you the `ai.niriksha` groupId by Java's reversed-domain convention).

1. Log in at [central.sonatype.com](https://central.sonatype.com)
2. Go to **Publishing → Namespaces** → [central.sonatype.com/publishing/namespaces](https://central.sonatype.com/publishing/namespaces)
3. Click **Add Namespace**
4. Enter: `ai.niriksha`
5. Sonatype will show a **verification key** (a random string)
6. Add a DNS TXT record on `niriksha.ai`:
   ```
   Type:  TXT
   Name:  @ (or niriksha.ai)
   Value: <verification key from Sonatype>
   TTL:   300
   ```
7. Click **Verify Namespace** — DNS propagation takes 5–30 minutes
8. Status changes to **Verified** ✅

> **Alternative:** If you control the GitHub org `san-data-systems`, you can verify via GitHub instead by setting `io.github.san-data-systems` as the namespace. However, using `ai.niriksha` is preferred for brand consistency.

### 3.3 Generate a Deployment Token

1. Log in to [central.sonatype.com](https://central.sonatype.com)
2. Click your avatar → **View Account**
3. Scroll to **Generate User Token**
4. Click **Generate User Token**
5. Copy both values:
   - **Token username** → this is `OSSRH_USERNAME`
   - **Token password** → this is `OSSRH_PASSWORD`
6. Store both in a password manager — the password is shown only once

### 3.4 Generate a GPG Signing Key

Maven Central requires all artifacts to be GPG-signed.

```bash
# 1. Generate the key — use releases@niriksha.ai as the identity
gpg --gen-key
#    Real name:    NirikshaAI Releases
#    Email:        releases@niriksha.ai
#    Passphrase:   <choose a strong passphrase — save it>

# 2. List keys to get your KEY_ID (the long hex after "sec rsa...")
gpg --list-secret-keys --keyid-format LONG
# Example output:
# sec   rsa4096/AABBCCDD11223344 2025-01-01
#                ^^^^^^^^^^^^^^^^ this is your KEY_ID

# 3. Upload public key to the Ubuntu keyserver
gpg --keyserver keyserver.ubuntu.com --send-keys AABBCCDD11223344

# Also upload to keys.openpgp.org (Maven Central checks multiple servers)
gpg --keyserver keys.openpgp.org --send-keys AABBCCDD11223344

# 4. Export the armored private key (for GitHub secret)
gpg --armor --export-secret-keys AABBCCDD11223344 > niriksha-releases.gpg.asc
cat niriksha-releases.gpg.asc
# Copy the entire output including -----BEGIN PGP PRIVATE KEY BLOCK----- lines
```

### 3.5 Add All Secrets to GitHub

1. Go to [github.com/san-data-systems/niriksha-sdk-java/settings/secrets/actions](https://github.com/san-data-systems/niriksha-sdk-java/settings/secrets/actions)
2. Add each secret:

| Secret name | Value |
|-------------|-------|
| `OSSRH_USERNAME` | Token username from step 3.3 |
| `OSSRH_PASSWORD` | Token password from step 3.3 |
| `GPG_PRIVATE_KEY` | Full output of `gpg --armor --export-secret-keys` (including header/footer lines) |
| `GPG_PASSPHRASE` | The passphrase you chose in step 3.4 |

### 3.6 Verify Setup

After merging and tagging `v0.1.0`:
```xml
<!-- Wait ~15 minutes for Maven Central sync, then test: -->
<dependency>
  <groupId>ai.niriksha</groupId>
  <artifactId>niriksha-sdk-java</artifactId>
  <version>0.1.0</version>
</dependency>
```

Or search: [central.sonatype.com/search?q=ai.niriksha](https://central.sonatype.com/search?q=ai.niriksha)

---

## 4. GitHub Secrets — All SDKs

Summary of every secret needed across all four repositories:

| Repository | Secret | Source | Required for |
|-----------|--------|--------|-------------|
| `niriksha-sdk-python` | `GITHUB_TOKEN` | Auto-provided | Releases, tags |
| `niriksha-sdk-go` | `GITHUB_TOKEN` | Auto-provided | Releases, tags |
| `niriksha-sdk-node` | `GITHUB_TOKEN` | Auto-provided | Releases, tags |
| `niriksha-sdk-node` | `NPM_TOKEN` | npm → Automation token | `npm publish` |
| `niriksha-sdk-java` | `GITHUB_TOKEN` | Auto-provided | Releases, tags |
| `niriksha-sdk-java` | `OSSRH_USERNAME` | Sonatype → Generate User Token | `mvn deploy` |
| `niriksha-sdk-java` | `OSSRH_PASSWORD` | Sonatype → Generate User Token | `mvn deploy` |
| `niriksha-sdk-java` | `GPG_PRIVATE_KEY` | Local GPG key export | Artifact signing |
| `niriksha-sdk-java` | `GPG_PASSPHRASE` | Your GPG key passphrase | Artifact signing |

> `GITHUB_TOKEN` is injected automatically by GitHub Actions into every workflow run — nothing to configure.

**Add a secret to a repo:**
1. Go to the repo on GitHub
2. Settings → Secrets and variables → Actions
3. New repository secret → enter name and value → Add secret

---

## 5. GitHub Environments — Python

The Python SDK workflows use GitHub Environments to gate PyPI deployments. Create them before the first release.

### Create `pypi` environment (stable releases)

1. Go to [github.com/san-data-systems/niriksha-sdk-python/settings/environments](https://github.com/san-data-systems/niriksha-sdk-python/settings/environments)
2. Click **New environment**
3. Name: `pypi`
4. Click **Configure environment**
5. Optionally add **Required reviewers** (e.g. `vbhadauriya`) for manual approval before stable release
6. Save protection rules

### Create `pypi-dev` environment (dev builds)

1. Same page → **New environment**
2. Name: `pypi-dev`
3. No required reviewers needed (dev builds are automatic)
4. Save

---

## Quick Reference

| Registry | URL | Account email | Username |
|----------|-----|--------------|---------|
| PyPI | [pypi.org](https://pypi.org) | niriksha.ai product email | `nirikshaai` |
| npm | [npmjs.com](https://npmjs.com) | niriksha.ai product email | `nirikshaai` |
| Maven Central | [central.sonatype.com](https://central.sonatype.com) | niriksha.ai product email | niriksha.ai account |
| GitHub | [github.com/san-data-systems](https://github.com/san-data-systems) | vbhadauriya@redcloudcomputing.com | `V-Bhadauriya` |

---

> For release workflow and versioning details, see [RELEASE.md](RELEASE.md).  
> For contributing guidelines, see [CONTRIBUTING.md](CONTRIBUTING.md).
