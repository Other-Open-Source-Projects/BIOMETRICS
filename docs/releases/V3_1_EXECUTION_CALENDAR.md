# BIOMETRICS V3.1 Execution Calendar

## Program window
- Start: 2026-02-25
- Gate A target: 2026-03-11
- Gate B target: 2026-03-23
- GA target: 2026-03-25

## One-command execution
1. `./scripts/release/run-ga-closure-program.sh`
2. Optional final tag in the same flow: `./scripts/release/run-ga-closure-program.sh --tag`

## Workstream 0 (2026-02-25 to 2026-02-27)
1. `git switch -c codex/v3.1-ga-closure`
2. `./scripts/release/lock-rc-scope.sh`
3. `./scripts/release/check-gates.sh`
4. `./scripts/release/run-gate-a.sh --write-report`
5. Document changes:
   - `./scripts/release/append-execution-log.sh "<change-line-1>" "<change-line-2>"`

## Workstream 1 (2026-02-28 to 2026-03-11)
1. Repeat 203-node and event ordering proof via `run-gate-a.sh`.
2. Keep `docs/releases/GATE_A_REPORT.md` updated with concrete evidence links.
3. Resolve all P0/P1 findings before Gate A sign-off.

## Workstream 2 (2026-03-12 to 2026-03-18)
1. Rehearsal program (recommended, sequential 6h + 24h with incident logging):
   - `./scripts/release/run-rehearsal-program.sh`
2. Start/monitor controls (optional, long-run operation):
   - `./scripts/release/cleanup-soak-runs.sh --older-than-minutes 30 --dry-run`
   - `./scripts/release/cleanup-soak-runs.sh --older-than-minutes 30`
   - `./scripts/release/start-soak.sh --profile rehearsal-6h`
   - `./scripts/release/start-soak.sh --profile rehearsal-24h`
   - `./scripts/release/soak-status.sh --profile all`
   - `./scripts/release/watch-soak-progress.sh --interval-seconds 300 --profile all`
   - `./scripts/release/stop-soak.sh --profile rehearsal-6h`
3. Manual fallback:
   - `./scripts/release/run-soak-rehearsal.sh --hours 6 --profile-label rehearsal-6h --fail-on-gates true`
   - `./scripts/release/run-soak-rehearsal.sh --hours 24 --profile-label rehearsal-24h --fail-on-gates true`
4. Update report:
   - `./scripts/release/update-soak-report.py --summary logs/soak/soak-summary-<timestamp>.json`
5. If a stage fails, incident must be recorded under:
   - `logs/release/incident-*.md`

## Workstream 3 (2026-03-19 to 2026-03-22)
1. 72h mandatory soak:
   - `./scripts/release/run-soak-72h.sh`
2. Validate thresholds:
   - `./scripts/analyze-soak.py --summary logs/soak/soak-summary-<timestamp>.json --min-success-rate 0.98 --max-timeouts 0 --max-dispatch-p95-ms 250 --max-fallback-rate 0.05 --max-backpressure-per-run 20`
3. Persist evidence into `docs/releases/SOAK_72H_REPORT.md`.

## Workstream 4 (2026-03-23 to 2026-03-25)
1. Gate B execution:
   - `./scripts/release/run-gate-b.sh --soak-summary logs/soak/soak-summary-<timestamp>.json --p0-count 0 --p1-count 0 --write-report`
2. Freeze policy enforcement per `docs/guides/RELEASE_FREEZE_V3_1.md`.
3. Final GA dry run:
   - `./scripts/release/run-ga-cut.sh`
4. Final tag after sign-off:
   - `./scripts/release/run-ga-cut.sh --tag`
5. Ensure the final operational deltas are logged in:
   - `docs/releases/EXECUTION_LOG.md`

## Required sign-offs
- Engineering lead
- Release manager
- Operations owner
