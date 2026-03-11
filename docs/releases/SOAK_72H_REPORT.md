# BIOMETRICS V3.1 72h Soak Report

Window: 2026-03-19 to 2026-03-22  
Release train: V3.1.0

## Configuration
- Scheduler mode: `dag_parallel_v1`
- Max parallelism: `8`
- Goal parts: `50` (high-cardinality, 200+ tasks/run)
- Run interval: `15s`
- Poll interval: `2s`
- Timeout per run: `1800s`

## Commands
```bash
PROFILE_LABEL=ga-72h DURATION_HOURS=72 RUN_INTERVAL_SECONDS=15 GOAL_PARTS=50 ./scripts/run-soak.sh
./scripts/analyze-soak.py --summary logs/soak/soak-summary-<timestamp>.json \
  --min-success-rate 0.98 \
  --max-timeouts 0 \
  --max-dispatch-p95-ms 250 \
  --max-fallback-rate 0.05 \
  --max-backpressure-per-run 20
```

Shortcut scripts:

```bash
./scripts/release/run-soak-rehearsal.sh --hours 6 --profile-label rehearsal-6h
./scripts/release/run-soak-rehearsal.sh --hours 24 --profile-label rehearsal-24h
./scripts/release/run-soak-72h.sh
./scripts/release/update-soak-report.py --summary logs/soak/soak-summary-<timestamp>.json
```

## Results Summary
- Runs started:
- Runs completed:
- Runs failed:
- Runs timed out:
- Configured duration seconds:
- Actual harness duration seconds:
- Records window seconds:
- Success rate:
- Dispatch p95 (ms):
- Fallback rate:
- Backpressure per run:
- Queue depth max:

## Incidents
- Deadlocks/hangs:
- Data corruption symptoms:
- Event ordering violations:

## Evidence Paths
- Soak records JSONL: `logs/soak/soak-runs-<timestamp>.jsonl`
- Baseline metrics snapshot: `logs/soak/soak-metrics-baseline-<timestamp>.prom`
- Metrics snapshot: `logs/soak/soak-metrics-<timestamp>.prom`
- Soak summary: `logs/soak/soak-summary-<timestamp>.json`

## Decision
- [ ] PASS (all thresholds met)
- [ ] FAIL (requires P0/P1 fix before Gate B)
- Notes:

<!-- BEGIN AUTO_RESULTS -->
## Auto-Generated Results (soak-summary-20260226T054141Z.json)
- Window: 2026-03-19 to 2026-03-22
- Class: Rehearsal / non-GA
- Profile label: rehearsal-6h
- Configured duration seconds: 21600
- Actual harness duration seconds: 21630
- Records window seconds: 21628
- Runs started: 59
- Runs completed: 1
- Runs failed: 58
- Runs timed out: 0
- Success rate: 0.0169
- Dispatch p95 (ms): 25.00
- Fallback rate: 0.0000
- Backpressure per run: 0.00
- Queue depth max: 50.00
- Records file: `/Users/jeremy/dev/BIOMETRICS/logs/soak/soak-runs-20260226T054141Z.jsonl`
- Baseline metrics file: `/Users/jeremy/dev/BIOMETRICS/logs/soak/soak-metrics-baseline-20260226T054141Z.prom`
- Metrics file: `/Users/jeremy/dev/BIOMETRICS/logs/soak/soak-metrics-20260226T054141Z.prom`
- Summary file: `/Users/jeremy/dev/BIOMETRICS/logs/soak/soak-summary-20260226T054141Z.json`

### Metrics Delta (Soak Window)
- Dispatch count: 3625
- Backpressure signals: 0.0
- Fallbacks triggered: 0.0

### Auto Decision
- Thresholds: FAIL
- GA qualification: FAIL
- Overall: FAIL
<!-- END AUTO_RESULTS -->
