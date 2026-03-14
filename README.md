# BIOMETRICS V3 (OpenCode Extension, Codex-First)

BIOMETRICS V3 is an OpenCode CLI extension layer (Codex-first) for autonomous 24/7 orchestration, policy enforcement, and operator workflows. Codex core stays upstream; BIOMETRICS adds additive runtime modules and keeps a Codex-native visual style.

## Codex-First Positioning

- Codex upstream is the primary coding engine and baseline behavior.
- BIOMETRICS provides overlay capabilities (orchestration, skills, policy, supervision, governance).
- We do not position BIOMETRICS as a replacement coding app fork.

## Canonical Runtime

- Primary core baseline (read-only by policy): `third_party/codex-upstream/`
- BIOMETRICS overlay runtime entrypoint: `biometrics-cli/cmd/controlplane`
- BIOMETRICS operator tooling entrypoints: `./biometrics-onboard`, `./bin/biometrics-skills`
- Temporary compatibility shim: `biometrics-cli/cmd/biometrics`
  - The shim prints a deprecation notice and forwards to V3 runtime behavior.

## Clone-to-Run (Official)

```bash
git clone <your-biometrics-repo-url> BIOMETRICS
cd BIOMETRICS
./biometrics-onboard
```

After first run, onboarding exposes `biometrics-onboard` in `~/.local/bin` (if your PATH includes it), installs missing system/project dependencies, builds artifacts, and runs smoke checks.
Legacy bootstrap entrypoints (`./bootstrap.sh`, `./scripts/setup.sh`) are deprecated wrappers that delegate to `./biometrics-onboard`.

Non-interactive/ops modes:

```bash
./biometrics-onboard --doctor
./biometrics-onboard --resume
./biometrics-onboard --non-interactive --yes
```

State and report artifacts:
- `.biometrics/onboard/state.json`
- `.biometrics/onboard/report.json`
- `.biometrics/onboard/events.jsonl`
`report.json` may include `warnings[]` for non-blocking remediation items (for example PATH export hints).

## Quick Start (Manual Path)

```bash
make env
make build
./bin/biometrics-cli
```

BIOMETRICS overlay API default endpoint: `http://127.0.0.1:59013`  
Override bind address explicitly with `BIOMETRICS_BIND_ADDR` when remote binding is required.

## OpenCode (Extension Packaging)

BIOMETRICS ships OpenCode extension assets in-repo (for example `.opencode/commands/*` and `.opencode/plugins/biometrics.ts`). You should not need a separate “plugin installer” step for normal operation.

Operational notes:
- Non-interactive execution uses `opencode run` (OpenCode `>= 1.2.x`).
- Execution directory resolution is `BIOMETRICS_OPENCODE_DIR` → `BIOMETRICS_WORKSPACE` → process working directory.
- Integration details: `docs/OPENCODE.md`.
- BIOMETRICS also ships a full OpenCode plugin under `.opencode/plugins/biometrics.ts` (tools `biometrics.*`).

One-command launcher (prints the plugin flow, optionally starts opencode):

```bash
./scripts/opencode-biometrics.sh
./scripts/opencode-biometrics.sh --start
```

Optional web-v3 dev mode:

```bash
cd biometrics-cli/web-v3
pnpm install --frozen-lockfile
pnpm run dev
```

Public website dev mode (Next.js + Nextra):

```bash
cd website
pnpm install --frozen-lockfile
pnpm run dev
```

Build web-v3 bundle served by Go runtime:

```bash
cd biometrics-cli/web-v3
pnpm install --frozen-lockfile
pnpm run build
```

## API v1 (Canonical)

- `POST /api/v1/runs`
  - `mode` values: `autonomous` (default) or `supervised`
  - Optional run payload fields: `scheduler_mode`, `max_parallelism`, `model_preference`, `fallback_chain`, `model_id`, `context_budget`, `blueprint_profile`, `blueprint_modules`, `bootstrap`
- `GET /api/v1/runs`
- `GET /api/v1/runs/{run_id}`
- `GET /api/v1/runs/{run_id}/tasks`
- `GET /api/v1/runs/{run_id}/graph`
- `GET /api/v1/runs/{run_id}/attempts`
- `POST /api/v1/runs/{run_id}/pause`
- `POST /api/v1/runs/{run_id}/resume`
- `POST /api/v1/runs/{run_id}/cancel`
- `GET /api/v1/blueprints`
- `GET /api/v1/blueprints/{profile}`
- `GET /api/v1/models`
- `GET /api/v1/agents/background`
- `POST /api/v1/agents/background`
- `GET /api/v1/agents/background/{job_id}`
- `POST /api/v1/agents/background/{job_id}/cancel`
- `GET /api/v1/auth/codex/status`
- `POST /api/v1/auth/codex/login`
- `POST /api/v1/auth/codex/logout`
- `GET /api/v1/projects`
- `POST /api/v1/projects/{project_id}/bootstrap`
- `GET /api/v1/fs/tree?path=`
- `GET /api/v1/fs/file?path=`
- `GET /api/v1/events` (SSE)
- `GET /api/v1/ws` (WebSocket)
- `GET /health/ready`
- `GET /metrics`

OpenAPI: `docs/api/openapi-v3-controlplane.yaml`

Readiness payload fields include:
- `opencode_available`
- `codex_auth_ready`
- `provider_status`
- `onboard_last_status` (optional)

## Contracts

- `docs/specs/contracts/run.schema.json`
- `docs/specs/contracts/task.schema.json`
- `docs/specs/contracts/event.schema.json`
- `docs/specs/contracts/attempt.schema.json`
- `docs/specs/contracts/graph.schema.json`
- `docs/specs/contracts/error.schema.json`
- `docs/specs/contracts/model.schema.json`
- `docs/specs/index.json`

## Documentation

- Migration guide: `docs/guides/MIGRATION_V3.md`
- Operator runbook: `docs/guides/OPERATOR_RUNBOOK_V3.md`
- Visual regression guard: `docs/guides/WEB_VISUAL_REGRESSION.md`
- Codex upstream-first strategy: `docs/guides/CODEX_UPSTREAM_FIRST_STRATEGY.md`
- Codex extension architecture: `docs/guides/CODEX_EXTENSION_ARCHITECTURE.md`
- Codex release/security governance: `docs/guides/CODEX_RELEASE_SECURITY_GOVERNANCE.md`
- Cloudflare enterprise web blueprint: `docs/guides/CLOUDFLARE_ENTERPRISE_WEB_BLUEPRINT.md`
- Codex upstream watch lock: `third_party/codex-upstream/upstream.lock.json`
- Release notes: `docs/releases/V3_CUTOVER_COMPLETE.md`
- OpenCode integration: `docs/OPENCODE.md`
- Blueprint catalog: `docs/blueprints/CATALOG.md`
- Blueprint source mapping: `docs/blueprints/SOURCE_MAP.md`

## Environment Policy (Hybrid)

- `.env.example` is the tracked canonical template.
- Local `.env` usage is supported.
- `.env` is ignored and must not be committed with real secrets.
- Bootstrap from template:

```bash
./scripts/init-env.sh
```

## Build and Test

```bash
make test
```

CI enforces Go build/test, web-v3 build, link checks, migration gate checks, and tracked-file secret scanning.

Release gate check (local):

```bash
./scripts/release/check-gates.sh
```

Release closure automation:

```bash
git switch -c codex/v3.1-ga-closure
./scripts/release/run-ga-closure-program.sh
# optional final tag in same orchestrated flow:
./scripts/release/run-ga-closure-program.sh --tag
```

Manual step-by-step (equivalent):

```bash
git switch -c codex/v3.1-ga-closure
./scripts/release/lock-rc-scope.sh
./scripts/release/run-gate-a.sh --write-report
./scripts/release/cleanup-soak-runs.sh --older-than-minutes 30
./scripts/release/run-rehearsal-program.sh
# or explicit control:
./scripts/release/start-soak.sh --profile rehearsal-6h
./scripts/release/soak-status.sh --profile rehearsal-6h
./scripts/release/stop-soak.sh --profile rehearsal-6h
./scripts/release/run-soak-72h.sh
./scripts/release/run-gate-b.sh --p0-count 0 --p1-count 0 --write-report
./scripts/release/run-ga-cut.sh
```

Web V3 E2E:

```bash
cd biometrics-cli/web-v3
pnpm run test:e2e
```

Web V3 visual guard:

```bash
cd biometrics-cli/web-v3
pnpm run test:visual
```

Public website quality checks:

```bash
cd website
pnpm run test:content
pnpm run test:e2e
pnpm run test:lighthouse
```

Public website deploy (Cloudflare Pages):

```bash
cd website
pnpm run cf:project:create    # first-time only
pnpm run deploy:cloudflare
```

## Soak Validation (V3.1)

Run a local soak profile and evaluate gates:

```bash
PROFILE_LABEL=rehearsal-6h DURATION_SECONDS=1800 RUN_INTERVAL_SECONDS=10 GOAL_PARTS=50 ./scripts/run-soak.sh
./scripts/analyze-soak.py --summary logs/soak/soak-summary-<timestamp>.json
./scripts/release/update-soak-report.py --summary logs/soak/soak-summary-<timestamp>.json
```

Default release thresholds:
- run success rate `>= 0.98`
- timed-out runs `== 0`
- dispatch latency p95 estimate `<= 250ms`
- fallback rate per run `<= 0.05`
- backpressure signals per run `<= 20`

SSE compatibility note:
- `/api/v1/events` emits both typed SSE events and `message` compatibility frames with identical event IDs.
- Runtime operations emit opencode installer events:
  - `opencode.install.started|succeeded|failed`
- Runtime auth/model routing emits:
  - `auth.codex.login.started|succeeded|failed`
  - `model.selected`
  - `model.fallback.triggered|exhausted`
  - `context.compiled`
- Supervised runs emit runtime checkpoint events:
  - `run.supervision.checkpoint`
- Onboarding step telemetry is local-only in `.biometrics/onboard/events.jsonl`.

## Shim Deprecation

`cmd/biometrics` remains a temporary compatibility shim in V3.1 and is scheduled for removal in V3.2 on **April 22, 2026**.
