# BIOMETRICS Apex Scoreboard

Window: 2026-02-26 onward  
Objective: evidence-based comparison across BIOMETRICS strategies and competitor baselines.

## Scope
- Candidate strategy metrics from `/api/v1/evals/run`
- Baseline and competitor comparison from generated `logs/evals/eval-results-*.json`
- Evidence-first reporting for release decisions

## Evidence Inputs
- Eval manifest: `logs/evals/eval-manifest-<run_id>-<timestamp>.json`
- Eval results: `logs/evals/eval-results-<run_id>-<timestamp>.json`

## Decision Gates
- quality_score >= 0.90
- median_time_to_green improvement >= 25% vs baseline
- cost_per_success improvement >= 20% vs baseline
- regression_detected = false

<!-- BEGIN AUTO_APEX -->
## Latest Apex Evidence
- Eval run id: `eval-run-61400088-a313-4423-bfce-a092a72de876`
- Results file: `/Users/jeremy/dev/BIOMETRICS/logs/evals/eval-results-eval-run-61400088-a313-4423-bfce-a092a72de876-20260226T053116Z.json`
- Candidate strategy: `adaptive`
- Baseline strategy: `deterministic`
- Candidate quality_score: 0.8120
- Candidate median_time_to_green_seconds: 593.60
- Candidate cost_per_success: 0.004887
- Baseline quality_score: 0.7600
- Baseline median_time_to_green_seconds: 712.49
- Baseline cost_per_success: 0.005703
- vs baseline time improvement: 16.69%
- vs baseline cost improvement: 14.31%
- regression_detected: `false`

### Comparison Table
- claude_code: quality_delta=-0.0340, time_improvement=-2.26%, cost_improvement=6.43%, composite_delta=-0.0216
- codex: quality_delta=-0.0220, time_improvement=-2.09%, cost_improvement=5.93%, composite_delta=-0.0143
- copilot_agent: quality_delta=0.0020, time_improvement=4.70%, cost_improvement=1.39%, composite_delta=0.0039
- cursor: quality_delta=0.0060, time_improvement=-1.91%, cost_improvement=4.08%, composite_delta=0.0026
- deterministic: quality_delta=0.0520, time_improvement=16.69%, cost_improvement=14.31%, composite_delta=0.0417
- windsurf: quality_delta=0.0520, time_improvement=-1.30%, cost_improvement=8.35%, composite_delta=0.0305

### Gate Status
- quality>=0.90: FAIL
- time_improvement>=25%: FAIL
- cost_improvement>=20%: FAIL
- regression_detected=false: PASS
- overall: **FAIL**
<!-- END AUTO_APEX -->
