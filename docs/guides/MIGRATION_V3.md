# Migration Guide: BIOMETRICS V3

This guide covers migration from pre-V3 behavior to the hardened V3 runtime.

## What Is Canonical in V3

1. Runtime: `biometrics-cli/cmd/controlplane`
2. API surface: `/api/v1/*`
3. Event streams: `/api/v1/events` (SSE) and `/api/v1/ws` (WebSocket)
   - SSE emits typed event frames and `message` compatibility frames with identical event IDs.
4. UI: `biometrics-cli/web-v3`
5. Contracts: `docs/specs/contracts/*.json` (`run`, `task`, `event`, `attempt`, `graph`, `error`, `model`)

## Temporary Compatibility

`biometrics-cli/cmd/biometrics` remains available as a temporary shim.
It prints a deprecation notice and forwards to control-plane runtime behavior.
Removal target for this shim is V3.2 on **April 22, 2026**.

## Migration Steps

1. Run clone-to-run onboarding:

```bash
./biometrics-onboard
```

2. Build and start V3:

```bash
make build
./bin/biometrics-cli
```

Resume/diagnostic modes:

```bash
./biometrics-onboard --resume
./biometrics-onboard --doctor
```

3. Verify health and API:

```bash
curl http://127.0.0.1:59013/health
curl http://127.0.0.1:59013/health/ready
curl http://127.0.0.1:59013/metrics
curl http://127.0.0.1:59013/api/v1/projects
curl http://127.0.0.1:59013/api/v1/models
curl http://127.0.0.1:59013/api/v1/auth/codex/status
```

`/health/ready` includes additive runtime fields:
- `opencode_available` (bool)
- `codex_auth_ready` (bool)
- `provider_status` (object)
- `onboard_last_status` (optional string)

4. Start a run:

```bash
curl -X POST http://127.0.0.1:59013/api/v1/runs \
  -H 'Content-Type: application/json' \
  -d '{"project_id":"default","goal":"implement feature","mode":"autonomous","scheduler_mode":"dag_parallel_v1","max_parallelism":8,"model_preference":"codex","fallback_chain":["gemini","nim"],"model_id":"gpt-5-codex","context_budget":24000}'
```

Supervised mode remains compatible and adds checkpoint pauses:

```bash
curl -X POST http://127.0.0.1:59013/api/v1/runs \
  -H 'Content-Type: application/json' \
  -d '{"project_id":"default","goal":"checkpointed review","mode":"supervised","scheduler_mode":"dag_parallel_v1","max_parallelism":8}'
```

5. Inspect DAG and attempts:

```bash
curl http://127.0.0.1:59013/api/v1/runs/<run_id>/graph
curl http://127.0.0.1:59013/api/v1/runs/<run_id>/attempts
```

6. Optional soak verification:

```bash
DURATION_SECONDS=1800 RUN_INTERVAL_SECONDS=10 ./scripts/run-soak.sh
./scripts/analyze-soak.py --summary logs/soak/soak-summary-<timestamp>.json
```

## Runtime vs Onboarding Event Contract

- Runtime event streams (`/api/v1/events`, `/api/v1/ws`) carry runtime bus events only.
- Runtime emits typed SSE frames and `message` compatibility frames with identical IDs.
- Local onboarding telemetry events (`onboard.step.*`) are written to `.biometrics/onboard/events.jsonl` and are not runtime SSE events.

## Legacy Content Policy

- Active docs must not prescribe removed runtime paths or deprecated endpoint families.

## Related References

- `docs/api/openapi-v3-controlplane.yaml`
- `docs/guides/OPERATOR_RUNBOOK_V3.md`
- `docs/OPENCODE.md`
