# OPENCODE Integration (Codex-First V3)

This document defines the supported OpenCode integration for BIOMETRICS V3.

## Scope

Codex is the primary runtime core. OpenCode is used as an execution adapter by BIOMETRICS V3 overlay microagents (`coder`, `fixer`).
BIOMETRICS control-plane APIs are additive orchestration surfaces for policy and run supervision.
Codex/Gemini/NIM model routing is orchestrated by the BIOMETRICS LLM router (default chain: `codex -> gemini -> nim`).

- Runtime source of truth: `biometrics-cli/cmd/controlplane`
- Adapter implementation: `biometrics-cli/internal/executor/opencode`
- Agent orchestration: `biometrics-cli/internal/runtime/*`

## Supported Configuration Model

BIOMETRICS V3 supports local `.env` and OpenCode auth/config storage without committing secrets.

1. Keep project-local `.env` untracked.
2. Keep `.env.example` tracked and up to date.
3. Keep OpenCode config as local user config, not committed secrets.

Canonical config file (required; system-wide singleton):

- `~/.config/opencode/opencode.json`

Optional OMOC config (only if you use the `oh-my-opencode` plugin):

- `~/.config/opencode/oh-my-opencode.json`

BIOMETRICS does not require `oh-my-opencode` and does not read OMOC config at runtime.
Do not create project-local `opencode.json`, `.opencode/opencode.json`, or `.opencode/oh-my-opencode.json` duplicates.

## Minimal Setup

1. Run clone-to-run onboarding:

```bash
./biometrics-onboard
```

2. Verify OpenCode availability:

```bash
opencode --version
```

3. Start BIOMETRICS V3 control-plane:

```bash
./bin/biometrics-cli
```

4. Create a run via API or web-v3 UI.

5. Verify Codex broker auth status (official CLI flow):

```bash
curl -fsS http://127.0.0.1:59013/api/v1/auth/codex/status | jq
curl -fsS -X POST http://127.0.0.1:59013/api/v1/auth/codex/login | jq
```

## GA Runtime Policy

- If `opencode` is missing at runtime, BIOMETRICS attempts auto-install (`brew install opencode`) when `BIOMETRICS_OPENCODE_AUTO_INSTALL` is not explicitly disabled.
- Auto-install failures are hard-fail (no simulated-success path).
- BIOMETRICS executes OpenCode in non-interactive mode via `opencode run` (required for OpenCode CLI `>= 1.2.x`).
- Execution directory is resolved in this order:
  1. `BIOMETRICS_OPENCODE_DIR` (explicit override)
  2. `BIOMETRICS_WORKSPACE` (control-plane workspace)
  3. process working directory
- Codex primary routing requires broker-ready auth (`codex_auth_ready=true`) for Codex provider execution.
- Model routing request fields are additive on `POST /api/v1/runs`:
  - `model_preference`
  - `fallback_chain`
  - `model_id`
  - `context_budget`
- Installer lifecycle events are emitted into runtime event streams:
  - `opencode.install.started`
  - `opencode.install.succeeded`
  - `opencode.install.failed`
- Auth/model routing lifecycle events:
  - `auth.codex.login.started|succeeded|failed`
  - `model.selected`
  - `model.fallback.triggered`
  - `model.fallback.exhausted`
  - `context.compiled`
- Onboarding step events (`onboard.step.*`) are local onboarding telemetry only (`.biometrics/onboard/events.jsonl`) and are not part of `/api/v1/events` runtime SSE.

## Operational Notes

- BIOMETRICS V3 API is the canonical control surface (`/api/v1/*`).
- Event streaming is available via SSE (`/api/v1/events`) and WebSocket (`/api/v1/ws`).
- Model/provider inventory is available via `GET /api/v1/models`.
- Run lifecycle controls are `pause`, `resume`, and `cancel`.
- Run mode options for `POST /api/v1/runs`:
  - `autonomous` (default)
  - `supervised` (emits `run.supervision.checkpoint` and pauses for operator `resume`)

## Security Requirements

- Never commit `.env`.
- Never commit real API keys into docs, templates, or source files.
- Secret scanning in CI runs against tracked files and fails on high-confidence key patterns.

## Related Docs

- `README.md`
- `docs/guides/MIGRATION_V3.md`
- `docs/api/openapi-v3-controlplane.yaml`

## OpenCode Plugin (BIOMETRICS Full Controlplane)

BIOMETRICS ships a repo-first OpenCode plugin loader at:
- `.opencode/plugins/biometrics.ts` (loader)
- `opencode-config/plugins/biometrics.ts` (plugin implementation)

Repo-local plugin deps:

- `.opencode/node_modules/` is local-only and ignored; do not commit it.
- BIOMETRICS does not require a tracked `.opencode/package.json` for normal operation.

Binary Policy:

- All Go binaries (`biometrics-cli`, `biometrics-api`, `biometrics-tui`) are build artifacts.
- They are excluded via `.gitignore` and must be built via `make build` or CI.
- Do not commit compiled binaries to the repository.

Repo slash commands (inside OpenCode):

- `.opencode/commands/biometrics-plan.md` → `/biometrics-plan`
- `.opencode/commands/biometrics-work.md` → `/biometrics-work`

Confirmation gates:

- Mutating tools require `confirm:true` (for example `biometrics.bootstrap_all` and `biometrics.controlplane.start|stop`).

Tool surface (examples):
- `biometrics.bootstrap_all` (end-to-end: repo/env/onboard/build/start/gates; requires `confirm:true`)
- `biometrics.controlplane.start|stop` (requires `confirm:true`), `biometrics.check_gates` (requires `confirm:true`), `biometrics.health.ready` (read-only)

Quick launcher:

```bash
./scripts/opencode-biometrics.sh
./scripts/opencode-biometrics.sh --start
```
