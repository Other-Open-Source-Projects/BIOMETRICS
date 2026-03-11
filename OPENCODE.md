# BIOMETRICS OpenCode Notes (V3)

The active OpenCode integration guide moved to:

- `docs/OPENCODE.md`
- Clone-to-run onboarding entrypoint: `./biometrics-onboard`

For V3 migration and operations, use:

- `docs/guides/MIGRATION_V3.md`
- `docs/guides/OPERATOR_RUNBOOK_V3.md`
- `docs/api/openapi-v3-controlplane.yaml`

Legacy plugin-heavy notes are no longer authoritative for BIOMETRICS V3 runtime.

Note: onboarding step telemetry (`onboard.step.*`) is local-only at `.biometrics/onboard/events.jsonl`.
Runtime run modes remain `autonomous` (default) and `supervised` (checkpointed via `run.supervision.checkpoint`).
Model routing remains additive and non-breaking (`model_preference`, `fallback_chain`, `model_id`, `context_budget` on `POST /api/v1/runs`).
Codex auth broker endpoints are available at `/api/v1/auth/codex/*`; provider inventory at `/api/v1/models`.
