# Changelog

All notable changes to this project will be documented in this file.

This project follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/) and adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## Project Information

- **Project Name**: BIOMETRICS
- **Repository**: [github.com/BIOMETRICS/BIOMETRICS](https://github.com/BIOMETRICS/BIOMETRICS)
- **License**: MIT
- **Primary Language**: TypeScript / Go
- **Framework**: OpenCode AI Agent System
- **Best Practices**: February 2026 Compliant

---

## [Unreleased]

### Added

- OpenCode extension surfaces (repo-first): `.opencode/commands/*`, `.opencode/plugins/biometrics.ts` (loader), `opencode-config/plugins/biometrics.ts` (implementation).
- Singleton OpenCode config gate: `scripts/check-opencode-singleton.sh` (enforced by `scripts/release/check-gates.sh`).
- Read-only OpenCode diagnostics tools: `biometrics.opencode.status`, `biometrics.controlplane.status`, `biometrics.controlplane.logs`.

### Fixed

- OpenCode CLI drift: all BIOMETRICS subprocess calls now use `opencode run` (compatible with OpenCode `1.2.24`), including health checks (`opencode --version`).
- Release gates: stop flagging `oh-my-opencode.json` mentions as legacy; enforce explicit repo-local config duplication bans instead.

### Changed

- OpenCode execution directory resolution: `BIOMETRICS_OPENCODE_DIR` → `BIOMETRICS_WORKSPACE` → process working directory.
- OpenCode plugin safety: all mutating tools require `confirm:true`.
- Archive legacy orchestrator code under `archive/legacy-v2/` and remove repo-local runtime OpenCode config copies.
- Repo hygiene: remove tracked Mach-O binaries from `biometrics-cli/` and treat binaries as build artifacts (`bin/`).

### Security

- Reduce shell-injection surface in the OpenCode plugin command runner and validate env keys.
- `biometrics.controlplane.logs` redacts common secret/token patterns.

---

## [1.0.0] - 2026-02-19

### Added

- Initial Release - Complete BIOMETRICS platform documentation by @Jeremy
- Go CLI Tool - Professional biometrics-onboard CLI with bubbletea TUI by @Jeremy
- Documentation Structure - 9 organized docs/ directories with 148+ files by @Jeremy
- Qwen 3.5 Integration - Primary AI brain with 5 skills by @Jeremy
- Best Practices - February 2026 compliant by @Jeremy

### Changed

- Project Structure - Complete reorganization to hierarchical docs/ by @Jeremy
- README Design - Modern layout with badges and tables by @Jeremy
- Documentation Standards - 500+ lines per file mandate by @Jeremy
- CLI Architecture - JavaScript to Go rewrite by @Jeremy

### Fixed

- Port Sovereignty - 22 port violations fixed by @Jeremy
- Timeout Issues - All timeout configurations removed by @Jeremy
- ESM Compatibility - Dependency updates by @Jeremy
- CI/CD Error Handling - Now blocking by @Jeremy

### Security

- Port Sovereignty - Unique ports prevent conflicts by @Jeremy
- NVIDIA Timeout - 120000ms for Qwen 3.5 latency by @Jeremy
- API Key Management - 90-day rotation schedule by @Jeremy

---

## [0.9.0] - 2026-02-17

### Added

- Initial documentation structure by @Delqhi-Platform
- Basic README.md setup by @Delqhi-Platform

---

**Last Updated**: 2026-02-21
**Maintainer**: Jeremy (@jeremy)
