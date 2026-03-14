#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

API_BASE="${API_BASE:-http://127.0.0.1:59013}"
PROJECT_ID="${PROJECT_ID:-soak}"
GOAL_PREFIX="${GOAL_PREFIX:-soak run}"
GOAL_PARTS="${GOAL_PARTS:-50}"
SCHEDULER_MODE="${SCHEDULER_MODE:-dag_parallel_v1}"
MAX_PARALLELISM="${MAX_PARALLELISM:-8}"
PROFILE_LABEL="${PROFILE_LABEL:-unspecified}"
RUN_INTERVAL_SECONDS="${RUN_INTERVAL_SECONDS:-15}"
POLL_INTERVAL_SECONDS="${POLL_INTERVAL_SECONDS:-2}"
RUN_TIMEOUT_SECONDS="${RUN_TIMEOUT_SECONDS:-1800}"
DURATION_HOURS="${DURATION_HOURS:-72}"
DURATION_SECONDS="${DURATION_SECONDS:-$((DURATION_HOURS * 3600))}"
OUTPUT_DIR="${OUTPUT_DIR:-${ROOT_DIR}/logs/soak}"
FAIL_ON_GATES="${FAIL_ON_GATES:-false}"
ACTIVE_RUN_WINDOW_MINUTES="${ACTIVE_RUN_WINDOW_MINUTES:-30}"

mkdir -p "${OUTPUT_DIR}"
timestamp="$(date -u +%Y%m%dT%H%M%SZ)"
run_log="${OUTPUT_DIR}/soak-runs-${timestamp}.jsonl"
baseline_metrics_file="${OUTPUT_DIR}/soak-metrics-baseline-${timestamp}.prom"
metrics_file="${OUTPUT_DIR}/soak-metrics-${timestamp}.prom"
summary_file="${OUTPUT_DIR}/soak-summary-${timestamp}.json"
lock_dir="${OUTPUT_DIR}/.soak-global-lock"
lock_pid_file="${lock_dir}/pid"

cleanup_lock() {
  if [[ -d "${lock_dir}" ]]; then
    rm -f "${lock_pid_file}" || true
    rmdir "${lock_dir}" || true
  fi
}

if mkdir "${lock_dir}" 2>/dev/null; then
  echo "$$" > "${lock_pid_file}"
  trap cleanup_lock EXIT INT TERM
else
  existing_pid=""
  if [[ -f "${lock_pid_file}" ]]; then
    existing_pid="$(cat "${lock_pid_file}" 2>/dev/null || true)"
  fi

  if [[ -n "${existing_pid}" ]] && kill -0 "${existing_pid}" 2>/dev/null; then
    echo "ERROR: another soak harness appears to be active (lock: ${lock_dir}, pid: ${existing_pid})" >&2
    exit 1
  fi

  echo "WARN: stale soak lock detected at ${lock_dir}; recovering lock ownership" >&2
  rm -f "${lock_pid_file}" 2>/dev/null || true
  rmdir "${lock_dir}" 2>/dev/null || rm -rf "${lock_dir}"

  if ! mkdir "${lock_dir}" 2>/dev/null; then
    echo "ERROR: unable to recover stale soak lock at ${lock_dir}" >&2
    exit 1
  fi
  echo "$$" > "${lock_pid_file}"
  trap cleanup_lock EXIT INT TERM
fi

echo "Starting soak run"
echo "  API base: ${API_BASE}"
echo "  Duration seconds: ${DURATION_SECONDS}"
echo "  Interval seconds: ${RUN_INTERVAL_SECONDS}"
echo "  Profile label: ${PROFILE_LABEL}"
echo "  Output: ${summary_file}"

if ! curl -fsS "${API_BASE}/health" >/dev/null; then
  echo "ERROR: control-plane health endpoint is not reachable at ${API_BASE}/health"
  exit 1
fi

curl -fsS "${API_BASE}/metrics" > "${baseline_metrics_file}" || true

harness_started_epoch="$(date +%s)"
deadline=$(( $(date +%s) + DURATION_SECONDS ))
iteration=0

while [ "$(date +%s)" -lt "${deadline}" ]; do
  if ! conflict_reason="$(python3 - "${API_BASE}" "${PROJECT_ID}" "${ACTIVE_RUN_WINDOW_MINUTES}" <<'PY'
import datetime
import json
import sys
import urllib.request

api_base = sys.argv[1].rstrip("/")
project_id = sys.argv[2]
window_minutes = int(sys.argv[3])

try:
    with urllib.request.urlopen(f"{api_base}/api/v1/runs", timeout=10) as resp:
        runs = json.loads(resp.read().decode("utf-8"))
except Exception as exc:
    print(f"unable to fetch runs for contamination guard: {exc}")
    raise SystemExit(2)

now = datetime.datetime.now(datetime.timezone.utc)
recent_same = 0
recent_other = 0
for run in runs:
    if run.get("status") != "running":
        continue
    created_at = str(run.get("created_at", ""))
    if not created_at:
        continue
    try:
        dt = datetime.datetime.fromisoformat(created_at.replace("Z", "+00:00"))
    except Exception:
        continue
    age_minutes = (now - dt).total_seconds() / 60.0
    if age_minutes > window_minutes:
        continue
    if run.get("project_id") == project_id:
        recent_same += 1
    else:
        recent_other += 1

if recent_same > 0:
    print(f"recent running soak run already exists for project_id={project_id} within {window_minutes}m window")
    raise SystemExit(1)
if recent_other > 0:
    print(f"recent running non-soak runs detected ({recent_other}) within {window_minutes}m window")
    raise SystemExit(1)

print("")
PY
  )"; then
    if [[ "${conflict_reason}" == recent\ running\ soak\ run* ]]; then
      echo "WARN: contamination guard deferred this iteration: ${conflict_reason}" >&2
      "${ROOT_DIR}/scripts/release/cleanup-soak-runs.sh" --older-than-minutes 0 >/dev/null 2>&1 || true
      sleep "${POLL_INTERVAL_SECONDS}"
      continue
    fi
    echo "ERROR: soak contamination guard blocked start: ${conflict_reason}" >&2
    exit 1
  fi

  iteration=$((iteration + 1))
  started_epoch="$(date +%s)"
  goal="$(python3 - "${GOAL_PREFIX}" "${GOAL_PARTS}" "${iteration}" "$(date -u +%Y-%m-%dT%H:%M:%SZ)" <<'PY'
import sys

prefix = sys.argv[1].strip() or "soak run"
try:
    parts = max(1, int(sys.argv[2]))
except Exception:
    parts = 50
iteration = sys.argv[3]
timestamp = sys.argv[4]

segments = [f"{prefix} package {i+1:02d} iteration {iteration} at {timestamp}" for i in range(parts)]
print(", ".join(segments))
PY
)"

  payload="$(python3 - "${PROJECT_ID}" "${goal}" "${SCHEDULER_MODE}" "${MAX_PARALLELISM}" <<'PY'
import json
import sys

project_id, goal, scheduler_mode, max_parallelism = sys.argv[1:5]
payload = {
    "project_id": project_id,
    "goal": goal,
    "mode": "autonomous",
    "scheduler_mode": scheduler_mode,
    "max_parallelism": int(max_parallelism),
}
print(json.dumps(payload))
PY
)"

  run_json="$(curl -fsS -X POST "${API_BASE}/api/v1/runs" -H "Content-Type: application/json" -d "${payload}" || true)"
  run_id="$(python3 - "${run_json}" <<'PY'
import json
import sys

raw = sys.argv[1]
if not raw:
    print("")
    raise SystemExit(0)
try:
    data = json.loads(raw)
except Exception:
    print("")
    raise SystemExit(0)
print(data.get("id", ""))
PY
)"

  status="submission_failed"
  error=""
  timed_out=0

  if [ -z "${run_id}" ]; then
    error="run creation failed"
  else
    status=""
    poll_deadline=$(( started_epoch + RUN_TIMEOUT_SECONDS ))
    while [ "$(date +%s)" -lt "${poll_deadline}" ]; do
      run_state_json="$(curl -fsS "${API_BASE}/api/v1/runs/${run_id}" || true)"
      status="$(python3 - "${run_state_json}" <<'PY'
import json
import sys

raw = sys.argv[1]
if not raw:
    print("")
    raise SystemExit(0)
try:
    data = json.loads(raw)
except Exception:
    print("")
    raise SystemExit(0)
print(data.get("status", ""))
PY
)"

      if [ "${status}" = "completed" ] || [ "${status}" = "failed" ] || [ "${status}" = "cancelled" ]; then
        break
      fi
      sleep "${POLL_INTERVAL_SECONDS}"
    done

    if [ "${status}" != "completed" ] && [ "${status}" != "failed" ] && [ "${status}" != "cancelled" ]; then
      status="timeout"
      timed_out=1
      error="run timeout"
    fi
  fi

  finished_epoch="$(date +%s)"
  duration_seconds=$(( finished_epoch - started_epoch ))

  python3 - "${run_log}" "${run_id}" "${status}" "${duration_seconds}" "${timed_out}" "${error}" "${started_epoch}" "${finished_epoch}" <<'PY'
import json
import sys

run_log, run_id, status, duration_seconds, timed_out, error, started_epoch, finished_epoch = sys.argv[1:9]
entry = {
    "run_id": run_id,
    "status": status,
    "duration_seconds": int(duration_seconds),
    "timed_out": bool(int(timed_out)),
    "error": error,
    "started_epoch": int(started_epoch),
    "finished_epoch": int(finished_epoch),
}
with open(run_log, "a", encoding="utf-8") as f:
    f.write(json.dumps(entry) + "\n")
PY

  if [ "${status}" = "timeout" ] && [ -n "${run_id}" ]; then
    body_file="$(mktemp)"
    code="$(curl -sS -o "${body_file}" -w "%{http_code}" -X POST "${API_BASE}/api/v1/runs/${run_id}/cancel" || true)"
    if [[ "${code}" =~ ^2 ]]; then
      echo "timeout cleanup: cancelled run_id=${run_id}"
    else
      reason="$(tr '\n' ' ' < "${body_file}" | sed 's/[[:space:]]\+/ /g' | cut -c1-180)"
      echo "timeout cleanup: cancel returned http=${code} (${reason})"
    fi
    rm -f "${body_file}"
    "${ROOT_DIR}/scripts/release/cleanup-soak-runs.sh" --older-than-minutes 0 >/dev/null 2>&1 || true
  fi

  if [[ -n "${run_id}" ]]; then
    echo "iteration=${iteration} run_id=${run_id} status=${status} duration_s=${duration_seconds} timed_out=${timed_out}"
  else
    echo "iteration=${iteration} run_id=<none> status=${status} duration_s=${duration_seconds} timed_out=${timed_out} error=${error}"
  fi

  if [ "$(date +%s)" -lt "${deadline}" ]; then
    sleep "${RUN_INTERVAL_SECONDS}"
  fi
done

curl -fsS "${API_BASE}/metrics" > "${metrics_file}" || true
harness_finished_epoch="$(date +%s)"

python3 - "${run_log}" "${baseline_metrics_file}" "${metrics_file}" "${summary_file}" "${DURATION_SECONDS}" "${RUN_TIMEOUT_SECONDS}" "${GOAL_PARTS}" "${SCHEDULER_MODE}" "${MAX_PARALLELISM}" "${PROFILE_LABEL}" "${harness_started_epoch}" "${harness_finished_epoch}" <<'PY'
import json
import math
import sys
from pathlib import Path

run_log_path = Path(sys.argv[1])
baseline_metrics_path = Path(sys.argv[2])
metrics_path = Path(sys.argv[3])
summary_path = Path(sys.argv[4])
duration_seconds_cfg = int(sys.argv[5])
run_timeout_seconds_cfg = int(sys.argv[6])
goal_parts_cfg = int(sys.argv[7])
scheduler_mode_cfg = sys.argv[8]
max_parallelism_cfg = int(sys.argv[9])
profile_label_cfg = sys.argv[10]
harness_started_epoch = int(sys.argv[11])
harness_finished_epoch = int(sys.argv[12])

records = []
if run_log_path.exists():
    for line in run_log_path.read_text(encoding="utf-8").splitlines():
        line = line.strip()
        if not line:
            continue
        records.append(json.loads(line))

def parse_prom_metrics(path: Path):
    out = {}
    if not path.exists():
        return out
    for line in path.read_text(encoding="utf-8").splitlines():
        line = line.strip()
        if not line or line.startswith("#"):
            continue
        parts = line.split()
        if len(parts) != 2:
            continue
        name, value = parts
        try:
            out[name] = float(value)
        except ValueError:
            pass
    return out

baseline_metrics = parse_prom_metrics(baseline_metrics_path)
end_metrics = parse_prom_metrics(metrics_path)

def counter_delta(name: str) -> float:
    return max(0.0, end_metrics.get(name, 0.0) - baseline_metrics.get(name, 0.0))

def gauge_end(name: str) -> float:
    return end_metrics.get(name, 0.0)

started = [r for r in records if r["status"] != "submission_failed"]
completed = [r for r in started if r["status"] == "completed"]
failed = [r for r in started if r["status"] == "failed"]
cancelled = [r for r in started if r["status"] == "cancelled"]
timeouts = [r for r in started if r["status"] == "timeout"]
submission_failed = [r for r in records if r["status"] == "submission_failed"]

durations = [r["duration_seconds"] for r in started if isinstance(r.get("duration_seconds"), int)]
duration_p95_seconds = 0.0
if durations:
    durations_sorted = sorted(durations)
    idx = max(0, math.ceil(len(durations_sorted) * 0.95) - 1)
    duration_p95_seconds = float(durations_sorted[idx])

success_rate = (len(completed) / len(started)) if started else 0.0

records_started_epochs = [
    int(r["started_epoch"])
    for r in started
    if isinstance(r.get("started_epoch"), int)
]
records_finished_epochs = [
    int(r["finished_epoch"])
    for r in started
    if isinstance(r.get("finished_epoch"), int)
]
records_window_seconds = 0
if records_started_epochs and records_finished_epochs:
    records_window_seconds = max(0, max(records_finished_epochs) - min(records_started_epochs))
harness_elapsed_seconds = max(0, harness_finished_epoch - harness_started_epoch)

dispatch_count = int(counter_delta("biometrics_task_dispatch_latency_count"))
dispatch_le25 = int(counter_delta("biometrics_task_dispatch_latency_bucket_le_25_ms"))
dispatch_le50 = int(counter_delta("biometrics_task_dispatch_latency_bucket_le_50_ms"))
dispatch_le100 = int(counter_delta("biometrics_task_dispatch_latency_bucket_le_100_ms"))
dispatch_le250 = int(counter_delta("biometrics_task_dispatch_latency_bucket_le_250_ms"))
dispatch_le500 = int(counter_delta("biometrics_task_dispatch_latency_bucket_le_500_ms"))
dispatch_le1000 = int(counter_delta("biometrics_task_dispatch_latency_bucket_le_1000_ms"))
dispatch_le2000 = int(counter_delta("biometrics_task_dispatch_latency_bucket_le_2000_ms"))
dispatch_gt2000 = int(counter_delta("biometrics_task_dispatch_latency_bucket_gt_2000_ms"))
dispatch_max_end = int(gauge_end("biometrics_task_dispatch_latency_max_ms"))

dispatch_p95_ms = 0.0
if dispatch_count > 0:
    threshold = max(1, math.ceil(dispatch_count * 0.95))
    cumulative = 0
    buckets = [
        (25, dispatch_le25),
        (50, dispatch_le50),
        (100, dispatch_le100),
        (250, dispatch_le250),
        (500, dispatch_le500),
        (1000, dispatch_le1000),
        (2000, dispatch_le2000),
    ]
    for limit, bucket_count in buckets:
        cumulative += bucket_count
        if cumulative >= threshold:
            dispatch_p95_ms = float(limit)
            break
    if dispatch_p95_ms == 0.0:
        dispatch_p95_ms = float(dispatch_max_end if dispatch_max_end > 0 else 2000)

fallbacks_total = counter_delta("biometrics_fallbacks_triggered")
backpressure_total = counter_delta("biometrics_backpressure_signals")
ready_queue_depth_max = gauge_end("biometrics_scheduler_ready_queue_depth_max")
fallback_rate = (fallbacks_total / len(started)) if started else 0.0
backpressure_per_run = (backpressure_total / len(started)) if started else 0.0

summary = {
    "records_file": str(run_log_path),
    "baseline_metrics_file": str(baseline_metrics_path),
    "metrics_file": str(metrics_path),
    "duration_seconds": duration_seconds_cfg,
    "run_timeout_seconds": run_timeout_seconds_cfg,
    "goal_parts": goal_parts_cfg,
    "scheduler_mode": scheduler_mode_cfg,
    "max_parallelism": max_parallelism_cfg,
    "profile_label": profile_label_cfg,
    "harness_started_epoch": harness_started_epoch,
    "harness_finished_epoch": harness_finished_epoch,
    "harness_elapsed_seconds": harness_elapsed_seconds,
    "records_window_seconds": records_window_seconds,
    "total_records": len(records),
    "runs_started": len(started),
    "runs_completed": len(completed),
    "runs_failed": len(failed),
    "runs_cancelled": len(cancelled),
    "runs_timed_out": len(timeouts),
    "run_submissions_failed": len(submission_failed),
    "run_success_rate": success_rate,
    "run_duration_p95_seconds": duration_p95_seconds,
    "dispatch_latency_p95_estimate_ms": dispatch_p95_ms,
    "run_fallback_rate": fallback_rate,
    "run_backpressure_per_run": backpressure_per_run,
    "fallbacks_total": fallbacks_total,
    "backpressure_signals_total": backpressure_total,
    "ready_queue_depth_max": ready_queue_depth_max,
    "gates": {
        "success_rate_gte_98": success_rate >= 0.98,
        "no_timeouts": len(timeouts) == 0,
        "dispatch_p95_lte_250": dispatch_p95_ms <= 250.0,
        "fallback_rate_lte_0_05": fallback_rate <= 0.05,
        "backpressure_per_run_lte_20": backpressure_per_run <= 20.0,
        "profile_is_ga_72h": profile_label_cfg == "ga-72h",
        "duration_at_least_72h": duration_seconds_cfg >= 72 * 3600,
        "actual_duration_at_least_72h": harness_elapsed_seconds >= 72 * 3600,
    },
    "metrics_end": end_metrics,
    "metrics_delta": {
        "biometrics_fallbacks_triggered": fallbacks_total,
        "biometrics_backpressure_signals": backpressure_total,
        "biometrics_task_dispatch_latency_count": dispatch_count,
        "biometrics_task_dispatch_latency_bucket_le_25_ms": dispatch_le25,
        "biometrics_task_dispatch_latency_bucket_le_50_ms": dispatch_le50,
        "biometrics_task_dispatch_latency_bucket_le_100_ms": dispatch_le100,
        "biometrics_task_dispatch_latency_bucket_le_250_ms": dispatch_le250,
        "biometrics_task_dispatch_latency_bucket_le_500_ms": dispatch_le500,
        "biometrics_task_dispatch_latency_bucket_le_1000_ms": dispatch_le1000,
        "biometrics_task_dispatch_latency_bucket_le_2000_ms": dispatch_le2000,
        "biometrics_task_dispatch_latency_bucket_gt_2000_ms": dispatch_gt2000,
    },
}

summary_path.write_text(json.dumps(summary, indent=2), encoding="utf-8")
print(json.dumps(summary, indent=2))
PY

if [ "${FAIL_ON_GATES}" = "true" ]; then
  python3 "${ROOT_DIR}/scripts/analyze-soak.py" \
    --summary "${summary_file}" \
    --min-success-rate 0.98 \
    --max-timeouts 0 \
    --max-dispatch-p95-ms 250 \
    --max-fallback-rate 0.05 \
    --max-backpressure-per-run 20
fi

echo "Soak run complete"
echo "  Summary: ${summary_file}"
echo "  Runs:    ${run_log}"
