# Release Notes: V3 Cutover Complete

Date: 2026-02-25

## Highlights

1. BIOMETRICS runtime is now fully centered on V3 control-plane architecture.
2. `cmd/biometrics` was converted to a temporary compatibility shim that forwards to V3 behavior.
3. Active docs were swept to remove stale V2 operational guidance.
4. Hybrid `.env` policy is enforced (`.env.example` tracked, `.env` local/ignored).
5. V3.1 hardening surface added: DAG graph/attempts API, readiness/metrics endpoints, scheduler controls, fallback/backpressure events.
6. CI hardening completed with migration gate, tracked-file secret scan, API smoke test, soak-subset, and nightly soak loop.
7. SSE transport now emits typed + `message` compatibility frames with stable event IDs.
8. Runtime metrics extended with event-bus and scheduler queue-depth diagnostics.
9. Model-routing core is active with default provider chain `codex -> gemini -> nim`, exposed as additive run fields (`model_preference`, `fallback_chain`, `model_id`, `context_budget`).
10. Codex auth broker endpoints are available under `/api/v1/auth/codex/*`, with provider inventory at `/api/v1/models`.
11. Runtime/onboarding event contract is explicit: runtime streams include runtime bus events only, while `onboard.step.*` remains local telemetry at `.biometrics/onboard/events.jsonl`.

## Canonical Interfaces

- API: `/api/v1/*` (including `/api/v1/models` and `/api/v1/auth/codex/*`)
- Event streams: SSE `/api/v1/events`, WS `/api/v1/ws`
- OpenAPI: `docs/api/openapi-v3-controlplane.yaml`
- Contracts: `docs/specs/contracts/*.json` (including `model.schema.json`)

## Legacy Policy

- V2 implementation and historical reports remain under `archive/legacy-v2`.
- Legacy paths are unsupported in active runtime/docs.

## Operator Reference

Use `docs/guides/OPERATOR_RUNBOOK_V3.md` for startup, controls, and troubleshooting.

## Forward Deprecation Plan

- `cmd/biometrics` stays as compatibility shim in V3.1.
- Planned removal in V3.2 on **April 22, 2026**.
