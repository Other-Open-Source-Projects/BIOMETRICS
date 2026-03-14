## 2026-03-11

- Fixed OpenCode CLI drift: BIOMETRICS now executes non-interactive work via `opencode run` (OpenCode `1.2.24` compatible).
- Added deterministic OpenCode run directory resolution: `BIOMETRICS_OPENCODE_DIR` → `BIOMETRICS_WORKSPACE` → CWD.
- Marked legacy loop/orchestrator/codegen modules as deprecated (V3 runtime is `biometrics-cli/cmd/controlplane`).
- Updated docs: `README.md`, `docs/OPENCODE.md`, `CHANGELOG.md`.
- Added missing root `go.sum` (module metadata).
- Added 1-command OpenCode launcher: `scripts/opencode-biometrics.sh`.
- De-bloated repo tracking: `.sisyphus/` + `templates/**/node_modules/` removed from git tracking and ignored going forward.
- Sanitized OpenCode template config: removed hard-coded usernames/paths and stripped embedded API keys from templates.
- Removed default step caps/timeouts where safe:
  - Planner work-packages default to unlimited (override via `BIOMETRICS_MAX_WORK_PACKAGES`).
  - Durable journal step cap supports unlimited when configured (`maxSteps <= 0`).
  - Swarm watchdog disabled by default (`BIOMETRICS_WATCHDOG_TIMEOUT=0`).
