# BIOMETRICS V3 Operator Runbook

## Startup

1. Run onboarding (clone-to-run path):

```bash
./biometrics-onboard
```

2. Build control-plane binary (optional if onboarding already built artifacts):

```bash
make build
```

3. Start control-plane:

```bash
./bin/biometrics-cli
```

4. Optional: run via compatibility shim (temporary):

```bash
cd biometrics-cli
go run ./cmd/biometrics
```

Onboarding state artifacts:
- `.biometrics/onboard/state.json`
- `.biometrics/onboard/report.json`
- `.biometrics/onboard/events.jsonl`

## Health and API Checks

```bash
curl -sS http://127.0.0.1:59013/health
curl -sS http://127.0.0.1:59013/health/ready
curl -sS http://127.0.0.1:59013/metrics
curl -sS http://127.0.0.1:59013/api/v1/projects
```

`/health/ready` includes onboarding/runtime fields:
- `opencode_available` (boolean)
- `codex_auth_ready` (boolean)
- `provider_status` (object summary)
- `onboard_last_status` (optional string)

Model/auth quick checks:

```bash
curl -sS http://127.0.0.1:59013/api/v1/models | jq
curl -sS http://127.0.0.1:59013/api/v1/auth/codex/status | jq
curl -sS -X POST http://127.0.0.1:59013/api/v1/auth/codex/login | jq
```

Create run:

```bash
curl -sS -X POST http://127.0.0.1:59013/api/v1/runs \
  -H 'Content-Type: application/json' \
  -d '{"project_id":"default","goal":"operator smoke run","mode":"autonomous","scheduler_mode":"dag_parallel_v1","max_parallelism":8,"model_preference":"codex","fallback_chain":["gemini","nim"],"model_id":"gpt-5-codex","context_budget":24000}'
```

Supervised checkpoint run (operator-gated resume points):

```bash
curl -sS -X POST http://127.0.0.1:59013/api/v1/runs \
  -H 'Content-Type: application/json' \
  -d '{"project_id":"default","goal":"supervised operator review","mode":"supervised","scheduler_mode":"dag_parallel_v1","max_parallelism":8}'
```

## Emergency Controls

Pause run:

```bash
curl -sS -X POST http://127.0.0.1:59013/api/v1/runs/<run_id>/pause
```

Resume run:

```bash
curl -sS -X POST http://127.0.0.1:59013/api/v1/runs/<run_id>/resume
```

Cancel run:

```bash
curl -sS -X POST http://127.0.0.1:59013/api/v1/runs/<run_id>/cancel
```

## Troubleshooting

1. Service does not start:
   - Verify port availability (`PORT`, default `59013`).
   - Check logs for SQLite open/migration errors.

2. UI is unavailable:
   - Build web bundle: `cd biometrics-cli/web-v3 && npm install && npm run build`.
   - Retry loading control-plane root URL.

3. File API returns path errors:
   - Confirm requested path is under workspace root.

4. Runs stall or fail repeatedly:
   - Inspect `/api/v1/runs/{run_id}`, `/api/v1/runs/{run_id}/tasks`, and `/api/v1/runs/{run_id}/graph`.
   - Stream events with `/api/v1/events?run_id=<run_id>`.
   - If Codex-primary routing is configured, verify `codex_auth_ready=true` from `/health/ready` and retry login via `/api/v1/auth/codex/login`.

5. Remote access is unintentionally exposed:
   - V3.1 binds to `127.0.0.1` by default.
   - Remote bind requires explicit `BIOMETRICS_BIND_ADDR`.

6. Event expectations mismatch:
   - Runtime SSE/WS only includes runtime bus events (for example `auth.codex.login.*`, `model.*`, `context.compiled`, `run.supervision.checkpoint`).
   - Onboarding telemetry (`onboard.step.*`) is local-only in `.biometrics/onboard/events.jsonl`.

## Release Checklist

One-command full closure orchestration:

```bash
./scripts/release/run-ga-closure-program.sh
# with final tag:
./scripts/release/run-ga-closure-program.sh --tag
```
Runtime state and events are written to `logs/release/ga-closure-state.json`.

1. CI green for build/test/web/link/secret/migration/smoke/soak-subset jobs.
2. Contracts unchanged or versioned.
3. Active docs aligned with V3 runtime/API.
4. Legacy content remains archive-only.
5. Gate script green: `./scripts/release/check-gates.sh`.
6. RC scope locked: `./scripts/release/lock-rc-scope.sh`.
7. Gate A evidence generated: `./scripts/release/run-gate-a.sh --write-report`.
8. Gate B evidence generated: `./scripts/release/run-gate-b.sh --p0-count 0 --p1-count 0 --write-report`.

## Soak Procedure (V3.1)

1. Start control-plane and verify readiness.
2. Cleanup stale soak metadata before new soak window:

```bash
./scripts/release/cleanup-soak-runs.sh --older-than-minutes 30 --dry-run
./scripts/release/cleanup-soak-runs.sh --older-than-minutes 30
```
The cleanup script also reconciles orphaned `running` rows in SQLite when runtime cancel returns `not active`.

3. Start soak profile:

```bash
./scripts/release/start-soak.sh --profile rehearsal-6h
./scripts/release/start-soak.sh --profile rehearsal-24h
./scripts/release/start-soak.sh --profile ga-72h
# detached via tmux (recommended for long local runs):
./scripts/release/start-soak.sh --profile ga-72h --detach-mode tmux
```

4. Monitor soak status:

```bash
./scripts/release/soak-status.sh --profile rehearsal-6h
./scripts/release/soak-status.sh --profile rehearsal-24h
./scripts/release/soak-status.sh --profile ga-72h
./scripts/release/snapshot-soak-progress.sh --limit 20
# optional live pane tail when using tmux mode:
tmux capture-pane -pt soak-ga-72h-<timestamp> | tail -n 50
```

`soak-status` now resolves two execution paths:
1. direct/start-soak managed profiles via `active-<profile>.pid/json`.
2. GA orchestrator path (`run-ga-closure-program.sh`) via `logs/release/ga-closure-state.json` + tmux session detection.

For GA orchestrator runs, status output includes `Source: ga-closure-state`.

5. Manual direct harness (fallback):

```bash
PROFILE_LABEL=ga-72h DURATION_HOURS=72 RUN_INTERVAL_SECONDS=15 GOAL_PARTS=50 ./scripts/run-soak.sh
```

6. Evaluate release gates:

```bash
./scripts/release/run-rehearsal-program.sh
# or manual:
./scripts/release/run-soak-rehearsal.sh --hours 6 --profile-label rehearsal-6h --fail-on-gates true
./scripts/release/run-soak-rehearsal.sh --hours 24 --profile-label rehearsal-24h --fail-on-gates true
./scripts/release/run-soak-72h.sh
./scripts/analyze-soak.py --summary logs/soak/soak-summary-<timestamp>.json \
  --min-success-rate 0.98 \
  --max-timeouts 0 \
  --max-dispatch-p95-ms 250 \
  --max-fallback-rate 0.05 \
  --max-backpressure-per-run 20
```

7. Gate B can only pass when all soak checks pass and P0/P1 issues are zero.
8. Any failed rehearsal stage must produce an incident record in `logs/release/incident-*.md`.
9. `soak-status` reports stale running rows (>30m) separately; these must be reviewed/cleaned before Gate B evidence collection.
10. Stop a running soak profile if needed:

```bash
./scripts/release/stop-soak.sh --profile rehearsal-6h
./scripts/release/stop-soak.sh --profile rehearsal-24h
./scripts/release/stop-soak.sh --profile ga-72h
# if started in tmux, stop-soak also kills the tracked tmux session.
```

## GA Cut Command

Dry run (recommended first):

```bash
./scripts/release/run-ga-cut.sh
```

Create GA tag after final sign-off:

```bash
./scripts/release/run-ga-cut.sh --tag
```

## Shim Removal Timeline

`cmd/biometrics` is the final compatibility shim in V3.1 and is targeted for removal in V3.2 on **April 22, 2026**.
