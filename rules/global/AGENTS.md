# BIOMETRICS Global Agent Rules (V3)

Version: 3.0.0
Status: Active
Last Updated: 2026-02-25

## 1. Scope

This rulebook applies to all BIOMETRICS automation agents and developer operators.
It defines the active V3 behavior and replaces legacy V2 conventions.

## 2. Canonical Runtime and API

1. Runtime entrypoint is `biometrics-cli/cmd/controlplane`.
2. `biometrics-cli/cmd/biometrics` is a temporary compatibility shim only.
3. API v1 is the only supported API surface.
4. Contract source of truth:
   - `docs/specs/contracts/run.schema.json`
   - `docs/specs/contracts/task.schema.json`
   - `docs/specs/contracts/event.schema.json`
   - `docs/specs/index.json`
5. OpenAPI source of truth:
   - `docs/api/openapi-v3-controlplane.yaml`

## 3. Runtime Model

1. V3 is actor-based with typed message boundaries.
2. Microagents are isolated workers with deterministic I/O contracts.
3. Control-plane supervision restarts crashed actors with backoff.
4. State and events are persisted in SQLite (WAL mode).

## 4. Agent Responsibilities

1. `planner`: build task DAG from run goal.
2. `scoper`: determine file/context scope.
3. `coder`: generate and update code.
4. `tester`: run checks/tests.
5. `reviewer`: validate diffs and policy compliance.
6. `fixer`: repair failed checks/reviews.
7. `integrator`: finalize run outputs and completion status.
8. `reporter`: emit and stream run/task events.

## 5. Run Lifecycle Rules

1. Valid run states: `queued`, `running`, `paused`, `completed`, `failed`, `cancelled`.
2. Pause/resume/cancel operations must be idempotent.
3. Event ordering requirement: `task.started` must precede terminal task events.
4. Failed tester/reviewer tasks should route through fixer and retry policy.

## 6. UI and Operator Controls

1. Web UI source is `biometrics-cli/web-v3/src`.
2. Build output `biometrics-cli/web-v3/dist` is artifact-only.
3. Supported slash commands:
   - `/run`
   - `/pause`
   - `/resume`
   - `/cancel`
   - `/retry-failed`
4. Human control is emergency-only: pause, stop/cancel, resume, rollback workflow at operator level.

## 7. Security and Config Policy

1. `.env.example` is the tracked template.
2. `.env` is local-only and ignored by git.
3. Real secrets must never be committed.
4. CI secret scanner runs on tracked files and must fail on high-confidence key patterns.
5. File-system APIs must block traversal and out-of-workspace access.

## 7A. OpenCode / OMOC Singleton Config Rule

1. OpenCode runtime config is a system-wide singleton at `~/.config/opencode/opencode.json`.
2. If `oh-my-opencode` is used, its config must live at `~/.config/opencode/oh-my-opencode.json` (optional).
3. Project-local duplicates are prohibited: `opencode.json`, `.opencode/opencode.json`, `.opencode/oh-my-opencode.json`, `~/.opencode/opencode.json`, and any extra `oh-my-opencode.json`.
4. Any template or repo guidance must reference the canonical global paths only and must not ship a second live copy.

## 8. Documentation Rules

1. Active docs must describe V3 behavior only.
2. Historical V2 documents must not be retained in-repo.
3. Migration guidance lives in `docs/guides/MIGRATION_V3.md`.
4. Operator procedures live in `docs/guides/OPERATOR_RUNBOOK_V3.md`.

## 9. CI and Release Gates

1. Go build/test for controlplane/runtime/API/store must pass.
2. web-v3 build must pass.
3. Link validation must pass.
4. Migration gate must fail if active files reference removed V2 paths.
5. Smoke test must verify:
   - `/health`
   - `/api/v1/projects`
   - run creation
   - task listing

## 10. Legacy Prohibitions

The following are archived-only and must not be used by active runtime/docs:

1. Removed command trees previously used in V2 runtime.
2. Removed V2 endpoint families.
3. Removed legacy web UI pathing.

Any new feature work must target V3 controlplane contracts and API v1 only.
