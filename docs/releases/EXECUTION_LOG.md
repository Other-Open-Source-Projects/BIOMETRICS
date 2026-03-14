# BIOMETRICS V3.1 Execution Log

This log records every operational/script change made during GA closure.

## 2026-02-25T18:30:30Z
- Updated `scripts/run-soak.sh` to persist measured runtime evidence:
  - Added `harness_started_epoch`, `harness_finished_epoch`, `harness_elapsed_seconds`.
  - Added `records_window_seconds` from run records.
  - Added gate flag `actual_duration_at_least_72h`.
- Updated `scripts/release/run-gate-b.sh`:
  - Added hard gate `soak_actual_duration_gate` requiring `harness_elapsed_seconds >= 259200`.
  - Added `soak_actual_duration_seconds` into Gate-B summary output.
  - Added auto-report rendering for actual duration gate/evidence.
- Updated `scripts/release/update-soak-report.py`:
  - Auto-report now prints configured duration, actual harness duration, and records window duration.
- Updated `docs/releases/GATE_B_REPORT.md` checklist:
  - Explicitly requires both configured and measured 72h duration gates.
- Updated `docs/releases/SOAK_72H_REPORT.md` summary template:
  - Added fields for configured duration, actual harness duration, and records window duration.
- Runtime state observed while changes were applied:
  - Active rehearsal loop in `ga-closure-20260225T174014Z`.
  - Repeated 203-task runs in `dag_parallel_v1` with no timeout/fail in observed window.

## 2026-02-25T18:31:09Z
- Added scripts/release/append-execution-log.sh for mandatory timestamped change documentation.
- Updated scripts/release/check-gates.sh to require EXECUTION_LOG.md and append-execution-log.sh in gate prerequisites.

## 2026-02-25T18:31:45Z
- Validated modified release scripts with syntax check (run-soak.sh, run-gate-b.sh, update-soak-report.py, append-execution-log.sh).
- Executed Gate-B negative test against rehearsal evidence: hard reject now includes actual-duration gate.
- Re-synced SOAK_72H_REPORT.md auto block using update-soak-report.py.
- Captured soak progress snapshot logs/release/soak-progress-20260225T183139Z.json while rehearsal program continued live.

## 2026-02-25T18:32:48Z
- Updated docs/guides/RELEASE_FREEZE_V3_1.md to make EXECUTION_LOG.md updates mandatory for freeze-approved changes.
- Updated docs/releases/V3_1_EXECUTION_CALENDAR.md with explicit append-execution-log.sh step and final log verification.

## 2026-02-25T18:34:09Z
- Ran scripts/release/check-gates.sh after documentation-policy updates: PASS.
- Ran scripts/release/run-gate-a.sh --write-report after hardening updates: PASS (summary logs/release/gate-a-20260225T183321Z.json).
- Gate-A run included API/runtime tests and Playwright E2E (3/3 passed).

## 2026-02-25T18:34:22Z
- Captured live soak snapshot logs/release/soak-progress-20260225T183416Z.json: 1 running + 19 completed in latest 20 soak runs, no failures/cancellations.
- Observed ongoing rehearsal run a030748c-654c-4810-8013-0c042f85d91c (dag_parallel_v1, max_parallelism=8).

## 2026-02-25T18:35:27Z
- Added scripts/release/watch-soak-progress.sh for periodic soak evidence capture (progress JSON + soak-status text).
- Started live watcher session soak-watch-183514 with 120s interval for continuous operational evidence during active rehearsal.
- Updated check-gates.sh and V3_1_EXECUTION_CALENDAR.md to include watch-soak-progress workflow.

## 2026-02-25T18:35:43Z
- Re-ran scripts/release/check-gates.sh after adding soak watcher support: PASS.

## 2026-02-25T18:36:46Z
- Gate B execution 20260225T183632Z: overall=fail.
- Gate B evidence summary: /Users/jeremy/dev/BIOMETRICS/logs/release/gate-b-20260225T183632Z.json
- Gate B soak source: /Users/jeremy/dev/BIOMETRICS/logs/soak/soak-summary-20260225T161656Z.json
- Gate B execution log: /Users/jeremy/dev/BIOMETRICS/logs/release/gate-b-20260225T183632Z.log

## 2026-02-25T18:37:26Z
- Gate A execution 20260225T183651Z: overall=pass.
- Gate A evidence summary: /Users/jeremy/dev/BIOMETRICS/logs/release/gate-a-20260225T183651Z.json
- Gate A execution log: /Users/jeremy/dev/BIOMETRICS/logs/release/gate-a-20260225T183651Z.log

## 2026-02-25T18:37:39Z
- Verified continuous watcher output: soak-progress-20260225T183716Z.json and soak-status-20260225T183716Z.txt created on 120s cadence.
- Confirmed tmux session soak-watch-183514 is actively capturing periodic soak evidence.

## 2026-02-25T18:47:17Z
- Redesigned web-v3 UI in biometrics-cli/web-v3/src/App.tsx with Codex-inspired three-pane operator layout, improved hierarchy, and preserved API/test contracts.
- Rebuilt biometrics-cli/web-v3/src/styles.css with professional design system (typography, gradient atmosphere, motion, status pills, responsive desktop/mobile behavior).
- Validated frontend integrity: npm run build PASS and Playwright E2E PASS (3/3).

## 2026-02-25T18:47:33Z
- Post-redesign gate validation: scripts/release/check-gates.sh PASS with zero-warning web build.

## 2026-02-25T18:55:32Z
- Enhanced web-v3 visual system in App.tsx: added run hero block, task snapshot KPIs, quick-command rail, and stronger operator information hierarchy.
- Extended styles.css with advanced panel staggering, status motion sweep for running cards, refined composer layout, and improved responsive ergonomics.
- Validated redesign hard: web-v3 npm run build PASS, Playwright E2E PASS (3/3), release gate check PASS.

## 2026-02-25T19:03:26Z
- Rebuilt web-v3 App.tsx with stabilized architecture plus icon system (SVG icons for headings, tabs, controls, quick commands) while preserving all existing data-testid/API flows.
- Added refined operator UX: event tone styling, richer run hero metadata, and command execution rail with iconized slash shortcuts.
- Validated quality gates after rewrite: web-v3 npm run build PASS, Playwright E2E PASS (3/3), scripts/release/check-gates.sh PASS.

## 2026-02-25T19:12:43Z
- Implemented Next-Step Program: rebuilt web-v3 App.tsx with professional iconized information architecture while preserving existing data-testid and API behavior.
- Added visual regression suite tests/e2e/visual.spec.ts (shell baseline, graph fallback baseline, style token guard) and deterministic mock timestamp in tests/e2e/mockControlPlane.ts.
- Generated baseline snapshots under tests/e2e/visual.spec.ts-snapshots/*.png and made screenshot checks platform-aware (darwin baselines + cross-platform token guard).
- Validation complete: npm run test:e2e PASS (6/6), scripts/release/check-gates.sh PASS.

## 2026-02-25T19:24:10Z
- Finalized CI hardening for visual quality: added web-v3-visual-guard job and artifact uploads (test-results + playwright-report) in .github/workflows/ci.yml.
- Hardened release prerequisites in scripts/release/check-gates.sh to require visual guard assets and WEB_VISUAL_REGRESSION.md.
- Added visual operations guide docs/guides/WEB_VISUAL_REGRESSION.md and linked it from README.md (plus npm run test:visual command).
- Stabilized Playwright ops: playwright.config.ts now emits HTML report + retains failure traces/videos/screenshots; package.json includes test:visual and test:visual:update scripts.
- Validation complete after interruption recovery: npm run test:e2e PASS (6/6) and scripts/release/check-gates.sh PASS.

## 2026-02-25T19:48:31Z
- Closed soak-status operational gap for ga-closure/tmux execution path.
- Updated scripts/release/soak-status.sh to detect active ga-closure profile/session from state+log and report correct running profile.
- Updated scripts/release/run-rehearsal-program.sh to write and clear active profile metadata (active-<profile>.pid/json) for deterministic status tracking in future rehearsal runs.
- Validated scripts with bash -n and live check: scripts/release/soak-status.sh now reports rehearsal-6h RUNNING from ga-closure session ga-closure-20260225T174014Z.

## 2026-02-25T19:50:06Z
- Gate A execution 20260225T194854Z: overall=pass.
- Gate A evidence summary: /Users/jeremy/dev/BIOMETRICS/logs/release/gate-a-20260225T194854Z.json
- Gate A execution log: /Users/jeremy/dev/BIOMETRICS/logs/release/gate-a-20260225T194854Z.log

## 2026-02-25T19:50:22Z
- Gate B execution 20260225T195011Z: overall=fail.
- Gate B evidence summary: /Users/jeremy/dev/BIOMETRICS/logs/release/gate-b-20260225T195011Z.json
- Gate B soak source: /Users/jeremy/dev/BIOMETRICS/logs/soak/soak-summary-20260225T161656Z.json
- Gate B execution log: /Users/jeremy/dev/BIOMETRICS/logs/release/gate-b-20260225T195011Z.log

## 2026-02-25T19:50:46Z
- Updated OPERATOR_RUNBOOK_V3 soak procedure with dual-path status model (start-soak metadata + ga-closure-state detection).
- Documented expected soak-status marker Source: ga-closure-state for orchestrated rehearsal/72h runs.

## 2026-02-25T19:51:49Z
- Hardened soak-status ga-closure detection against stale tmux sessions by requiring recent ga-closure log activity (ACTIVE_WINDOW_MINUTES).
- Revalidated run-rehearsal-program.sh and soak-status.sh syntax plus live status output after patch.

## 2026-02-25T19:52:58Z
- Executed RC hygiene cycle after soak-status hardening: lock-rc-scope PASS, check-gates PASS, gate-a PASS (logs/release/gate-a-20260225T194854Z.json).
- Executed gate-b re-evaluation with current non-GA soak evidence: expected FAIL at logs/release/gate-b-20260225T195011Z.json (profile/duration/actual-duration gates).
- Confirmed live rehearsal continues in ga-closure tmux with completed iterations and no timeout/failure in observed window.

## 2026-02-25T20:59:08Z
- Implemented Go-native clone-to-run onboarding command via biometrics-cli/cmd/onboard and internal/onboarding runner with idempotent state/report persistence.
- Added root bootstrap entrypoint ./biometrics-onboard plus Makefile targets make onboard and make onboard-doctor.
- Enforced GA opencode policy in internal/executor/opencode/adapter.go: no simulated-success path, auto-install attempt via Homebrew, hard-fail remediation, and install lifecycle events (started/succeeded/failed).
- Wired opencode install events into controlplane bus publication (source=executor.opencode) and extended readiness payload with opencode_available + onboard_last_status.
- Extended onboarding runner event journal (.biometrics/onboard/events.jsonl) with onboard.step.started/completed/failed emission.
- Updated contracts/openapi/docs for onboarding/opencode operational events and clone-to-run workflow (README, docs/OPENCODE.md, MIGRATION_V3.md, OPERATOR_RUNBOOK_V3.md, OPENCODE.md).
- Updated CI and gate checks to include cmd/onboard/internal/onboarding coverage and biometrics-onboard tracked surface.
- Validation completed: targeted go test package set PASS, make build PASS, biometrics-onboard --doctor PASS, readiness smoke check PASS, scripts/release/check-gates.sh PASS.

## 2026-02-25T21:00:13Z
- Added opencode adapter unit tests (internal/executor/opencode/adapter_test.go) to enforce hard-fail behavior when binary is missing and no empty-prompt execution.
- Revalidated release gates after onboarding/opencode hardening patch set: scripts/release/check-gates.sh PASS including Go package set, web zero-warning build, legacy-reference scan, and secret scan.

## 2026-02-25T21:01:02Z
- Added onboarding step event journal support at .biometrics/onboard/events.jsonl and documented artifact path in README/runbook.
- Post-update verification rerun: scripts/release/check-gates.sh PASS.

## 2026-02-25T21:01:18Z
- Validated onboarding event journal by running biometrics-onboard --doctor; confirmed onboard.step.started/completed entries are written to .biometrics/onboard/events.jsonl.

## 2026-02-25T21:25:45Z
- Implemented clone-to-run bootstrap hardening in ./biometrics-onboard: when bin/biometrics-onboard is missing, wrapper now validates/installs Homebrew and Go with prompt-aware policy (--yes/--non-interactive), builds cmd/onboard, and then executes the binary with hard-fail remediation messages.
- Refactored internal/onboarding/runner.go for strict doctor non-mutation: NewRunner no longer creates onboard artifact dirs in doctor mode, persistState/persistReport/emitStepEvent are no-op in doctor mode, and completion logs now explicitly state no artifacts were written.
- Replaced preflight write-probe with permission-safe stat/perm check and introduced explicit non-darwin override gate BIOMETRICS_ONBOARD_ALLOW_NON_DARWIN (default false).
- Implemented PATH hardening as non-blocking warning: stepExposeCommand now records warnings instead of hard-failing when ~/.local/bin is missing from PATH, and report.json now carries additive warnings[].
- Adjusted command-step doctor behavior to check-only (no symlink writes), while normal mode still creates/updates ~/.local/bin/biometrics-onboard symlink.
- Aligned event/docs contract: removed onboard.step.* from runtime event schema/OpenAPI SSE runtime list, documented onboard.step.* as local onboarding telemetry in .biometrics/onboard/events.jsonl, and updated README/OPENCODE docs accordingly.
- Deprecated legacy setup paths by replacing scripts/setup.sh and bootstrap.sh with thin delegating wrappers to ./biometrics-onboard.
- Extended test coverage: added onboarding test suite (doctor non-mutation, PATH warning behavior, resume determinism, report warnings) and added opencode adapter auto-install failure test with installer hook validation.
- Extended CI/gates: added explicit onboarding non-mutation doctor smoke test runs in ci.yml and scripts/release/check-gates.sh.
- Validation complete: targeted go test sets PASS, onboarding bootstrap path exercised by rebuilding missing binary PASS, and scripts/release/check-gates.sh PASS after all changes.

## 2026-02-25T21:50:07Z
- Implemented supervised run-mode validation in scheduler with normalized allowed values: autonomous|supervised.
- Added supervision checkpoints to runtime execution flow with new runtime event run.supervision.checkpoint and automatic pause/resume gating at after-planner, after-work-package-block, and before-integrator.
- Updated run.created event payload to include mode for downstream UI/API observability.
- Extended contracts: added RunMode constants/helpers in internal/contracts/types.go.
- Updated web-v3 UI to support run mode selection, supervised payload submission, and supervision checkpoint banner (data-testid=supervision-banner).
- Updated web-v3 SSE subscription list and event renderer for run.supervision.checkpoint.
- Updated Playwright E2E suite: validated autonomous payload mode, added supervised mode end-to-end test, and refreshed visual baselines after UI contract changes.
- Updated mock control-plane to preserve posted run mode in run creation responses.
- Updated schemas/docs: run.schema mode enum, event.schema event enum, OpenAPI mode enums and runtime SSE event notes, README/docs/OPENCODE.md/operator runbook supervised flow notes.
- Added backend tests for invalid run mode rejection and supervised checkpoint pause/resume completion path; added scheduler-level supervised checkpoint test coverage.
- Validation executed: gofmt on changed Go files; go test -timeout=10m ./internal/runtime/scheduler ./internal/api/http ./internal/contracts (from biometrics-cli) PASS; npm run test:e2e PASS; ./scripts/release/check-gates.sh PASS.

## 2026-02-25T21:50:40Z
- Synced root OPENCODE.md with runtime run-mode contract: autonomous|supervised and supervision checkpoint event note.

## 2026-02-25T22:38:14Z
- Implemented Codex-parity model routing core in V3.1: added LLM gateway/router with provider chain defaults codex->gemini->nim under biometrics-cli/internal/llm/* and wired coder/fixer to gateway execution.
- Added official Codex auth broker integration (biometrics-cli/internal/auth/codexbroker) and new API surfaces /api/v1/models and /api/v1/auth/codex/{status,login,logout} with runtime events auth.codex.login.* and model.*.
- Extended run/runtime persistence and contracts additively for model routing (model_preference, fallback_chain, model_id, context_budget), context compilation telemetry (context.compiled), and provider-attempt metadata in TaskAttempt.
- Integrated deterministic context build/compile pipeline (internal/context/indexer + internal/prompt/compiler) into scheduler dispatch and persisted provider trail/latency/token stats per attempt.
- Hardened secret hygiene: centralized EventBus payload redaction hook, expanded policy redaction patterns for id_token/access_token/authorization/cookie/bearer/JWT, and expanded CI/check-gates secret-scan regexes for JWT-like leaks.
- Updated docs/specs/openapi/readme/opencode alignment including new model.schema.json, index inventory, readiness fields codex_auth_ready/provider_status, and additive run payload fields.
- Validation complete: go test targeted package matrix PASS with GOCACHE override, npm run build PASS, scripts/release/check-gates.sh PASS.

## 2026-02-25T22:41:34Z
- Synchronized release/operator migration docs with implemented Codex auth and model-routing runtime surfaces.
- Updated docs/guides/MIGRATION_V3.md with /api/v1/models, /api/v1/auth/codex/status, additive run model fields, readiness field mapping, and runtime-vs-onboarding event contract.
- Updated docs/guides/OPERATOR_RUNBOOK_V3.md with codex_auth_ready/provider_status readiness checks, model/auth operational commands, Codex-primary troubleshooting path, and explicit SSE vs onboard telemetry expectations.
- Updated docs/releases/V3_CUTOVER_COMPLETE.md highlights and canonical interface section to include Codex auth broker endpoints, model routing fields, and model.schema.json contract inventory.
- Revalidated release integrity after docs sync: scripts/release/check-gates.sh PASS.

## 2026-02-26T00:18:09Z
- GA-closure launched in background from step soak-cleanup (local Gate-A rerun was blocked by Playwright visual hang).
- Launch PID 45370, log /Users/jeremy/dev/BIOMETRICS/logs/release/ga-closure-launch-20260226T001806Z.log

## 2026-02-26T00:18:22Z
- Gate A execution 20260226T000816Z: overall=fail.
- Gate A evidence summary: /Users/jeremy/dev/BIOMETRICS/logs/release/gate-a-20260226T000816Z.json
- Gate A execution log: /Users/jeremy/dev/BIOMETRICS/logs/release/gate-a-20260226T000816Z.log

## 2026-02-26T00:22:07Z
- Fixed syntax break in scripts/release/run-ga-closure-program.sh (Python heredoc/function boundary) to restore executable GA orchestration.
- Launched detached GA closure session ga-closure-live-20260226T002154Z from step soak-cleanup with watcher interval 300s.

## 2026-02-26T00:26:32Z
- GA-closure active session ga-closure-live-20260226T002154Z progressed to rehearsal-program; soak-cleanup completed.
- Rehearsal-6h soak started with summary target logs/soak/soak-summary-20260226T002400Z.json.

## 2026-02-26T01:16:49Z
- Apex orchestrator kernel implemented (plans/runs/resume/scorecard + explainable events)
- Control plane extended with /api/v1/orchestrator/* and /api/v1/evals/* plus contracts/openapi sync and targeted test pass

## 2026-02-26T01:34:26Z
- Apex validation hardening: deterministic event replay ordering and terminal run control idempotence fixes
- api/http + orchestrator + evals + sqlite test bundles PASS after Apex endpoint integration

## 2026-02-26T01:58:54Z
- Apex stream expanded: risk-gated orchestrator execution, arena workspace isolation, workflow presets, memory TTL/provenance, and Orchestrator Control UI
- check-gates.sh PASS after Apex hardening and contract/openapi sync

## 2026-02-26T02:24:53Z
- rehearsal-6h aborted: 6 failures in first 19 runs, >=98% no longer reachable
- starting timeout/runtime recovery before fresh GA-closure run

## 2026-02-26T02:58:56Z
- implemented agent-specific scheduler timeouts + runtime surface smoke gate before rehearsal
- implemented dataset-driven eval evidence pipeline with apex scoreboard reporting

## 2026-02-26T03:00:14Z
- ga-closure no-resume blocked at rc-scope-lock on branch codex/apex-orchestrator
- restarting ga-closure from soak-cleanup with watcher to keep closure stream progressing

## 2026-02-26T03:02:30Z
- ga-closure soak-cleanup stalled during sqlite reconcile of stale soak run
- restarting ga-closure from rehearsal-program to bypass cleanup deadlock risk

## 2026-02-26T03:06:31Z
- runtime rebinded on 59013 with new surface checks passing (models/skills/orchestrator/evals)
- apex eval evidence generated under logs/evals and APEX_SCOREBOARD auto-updated

## 2026-02-26T03:06:31Z
- ga-closure restarted from rehearsal-program with watcher after branch/cleanup blockers
- rehearsal-6h active in session ga-closure-apex-rehearsal-20260226T030243Z

## 2026-02-26T03:17:29Z
- added deterministic CI regression gate test for adaptive vs deterministic quality delta <=3%
- eval gate tests pass in internal/evals

## 2026-02-26T04:13:04Z
- Gate A execution 20260226T034545Z: overall=fail.
- Gate A evidence summary: /Users/jeremy/dev/BIOMETRICS/logs/release/gate-a-20260226T034545Z.json
- Gate A execution log: /Users/jeremy/dev/BIOMETRICS/logs/release/gate-a-20260226T034545Z.log

## 2026-02-26T05:26:45Z
- Gate A execution 20260226T052303Z: overall=pass.
- Gate A evidence summary: /Users/jeremy/dev/BIOMETRICS/logs/release/gate-a-20260226T052303Z.json
- Gate A execution log: /Users/jeremy/dev/BIOMETRICS/logs/release/gate-a-20260226T052303Z.log

## 2026-02-26T05:27:02Z
- GA-closure start baseline complete
- Gate-A baseline written

## 2026-02-26T05:27:19Z
- GA-closure relaunched with no-resume after Gate-A PASS
- Detached session log logs/release/ga-closure-live-20260226T052713Z.log

## 2026-02-26T05:31:41Z
- Apex eval evidence refreshed (dataset apex-suite-v1, seed 20260226)
- APEX_SCOREBOARD updated from eval-run-61400088-a313-4423-bfce-a092a72de876

## 2026-02-26T05:35:40Z
- Gate A execution 20260226T053134Z: overall=pass.
- Gate A evidence summary: /Users/jeremy/dev/BIOMETRICS/logs/release/gate-a-20260226T053134Z.json
- Gate A execution log: /Users/jeremy/dev/BIOMETRICS/logs/release/gate-a-20260226T053134Z.log

## 2026-02-26T05:42:02Z
- GA-closure reached rehearsal-program with Gate-A PASS
- rehearsal-6h started under ga-closure state 20260226T052729Z

## 2026-02-26T05:51:59Z
- rehearsal-6h active: task completion progressing without early timeout
- run 6ffa5168-09ae-4a0d-952c-556ad02bf6c2 advanced to 88 completed / 8 running / 107 pending

## 2026-02-26T12:41:03Z
- scheduler defaults updated: coder/fixer timeout 600s + release timeout env defaults
- blocked: local macOS loader hangs new binaries in _dyld_start; controlplane cannot be restarted until host recovered

## 2026-02-26T12:44:05Z
- host exec-policy anomaly confirmed: newly launched local binaries block in dyld_start (sampled)
- runtime smoke remains fail (59013 unreachable) until host-level binary execution recovers
