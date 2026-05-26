# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.0.1] - 2026-05-27

### Added
- `dev-release.yml` workflow — auto-creates `v{version}-dev.{sha}` GitHub pre-release on every merge to `main`
- `RELEASE.md` — comprehensive versioning, branching, and release process guide
- Structured logging via `log/slog` replacing `log.Printf`
- `WithLogger` option for custom logger injection
- Unit test suite with ≥80% coverage target
- `golangci-lint` configuration with security linters (`gosec`)
- `govulncheck` dependency vulnerability scanning
- GitHub Actions CI pipeline (lint, test, build, vuln scan)
- GitHub Actions release workflow (auto-publishes on `v*` tags)
- CodeQL security analysis workflow
- CONTRIBUTING.md, SECURITY.md, CODE_OF_CONDUCT.md

### Fixed
- ReDoS vulnerability in credit-card PII regex (replaced backtracking pattern)
- TLS insecure mode now emits a warning log in non-test contexts

## [0.1.0] - 2025-05-01

### Added
- Initial release
- OpenTelemetry traces, metrics, and logs via OTLP/gRPC
- LLM span helpers: conversation, RAG chunks, tool calls
- PII redaction (email, phone, SSN, credit card)
- W3C Baggage context propagation helpers
- HTTP middleware for trace context extraction
- Serverless/Lambda flush wrapper
- Eval submission (single and batch) with retry logic
- Prompt vault client with 5-minute in-process cache
- Configurable sampling rate
- TLS options: system roots, custom CA, skip-verify, plaintext

[Unreleased]: https://github.com/san-data-systems/niriksha-sdk-go/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/san-data-systems/niriksha-sdk-go/releases/tag/v0.1.0
