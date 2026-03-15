#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
LOG_DIR="${ROOT_DIR}/logs/release"
mkdir -p "${LOG_DIR}"

STATE_FILE="${LOG_DIR}/ga-closure-state.json"
LOCK_DIR="${LOG_DIR}/.ga-closure-lock"
P0_COUNT="${P0_COUNT:-0}"
P1_COUNT="${P1_COUNT:-0}"
SOAK_CLEANUP_OLDER_THAN_MINUTES="${SOAK_CLEANUP_OLDER_THAN_MINUTES:-0}"
GA_CLOSURE_HEARTBEAT_SECONDS="${GA_CLOSURE_HEARTBEAT_SECONDS:-120}"
GA_CLOSURE_ENABLE_SOAK_WATCHER="${GA_CLOSURE_ENABLE_SOAK_WATCHER:-true}"
GA_CLOSURE_WATCH_INTERVAL_SECONDS="${GA_CLOSURE_WATCH_INTERVAL_SECONDS:-120}"
GA_CLOSURE_WATCH_PROFILE="${GA_CLOSURE_WATCH_PROFILE:-all}"
BIOMETRICS_AGENT_TIMEOUT_DEFAULT_SECONDS="${BIOMETRICS_AGENT_TIMEOUT_DEFAULT_SECONDS:-120}"
BIOMETRICS_AGENT_TIMEOUT_CODER_SECONDS="${BIOMETRICS_AGENT_TIMEOUT_CODER_SECONDS:-600}"
BIOMETRICS_AGENT_TIMEOUT_FIXER_SECONDS="${BIOMETRICS_AGENT_TIMEOUT_FIXER_SECONDS:-600}"
RESUME=true
FROM_STEP=""
CREATE_TAG=false
SOAK_WATCHER_PID=""
SOAK_WATCHER_LOG=""

timestamp="$(date -u +%Y%m%dT%H%M%SZ)"
log_file="${LOG_DIR}/ga-closure-${timestamp}.log"
exec > >(tee -a "${log_file}") 2>&1

if [[ "${VISUAL_TRUTH:-0}" == "1" && -z "${VISUAL_TRUTH_SESSION:-}" ]]; then
  export VISUAL_TRUTH_SESSION="ga-closure-${timestamp}"
fi

if [[ "${VISUAL_TRUTH:-0}" == "1" && -x "${ROOT_DIR}/scripts/visual_truth/vt" ]]; then
  echo "[ga-closure] visual truth enabled: running vt doctor"
  "${ROOT_DIR}/scripts/visual_truth/vt" doctor
fi

usage() {
  cat <<USAGE
Usage: $(basename "$0") [options]

Options:
  --p0-count <n>      P0 backlog count for Gate B (default: 0)
  --p1-count <n>      P1 backlog count for Gate B (default: 0)
  --from-step <name>  Start execution at step name
  --no-resume         Ignore previous completed steps in state file
  --tag               Execute GA tag step (run-ga-cut.sh --tag)

Step names:
  rc-scope-lock
  gate-check-baseline
  gate-a
  runtime-surface-smoke
  soak-cleanup
  rehearsal-program
  soak-72h
  gate-b
  ga-cut-dry-run
  ga-cut-tag
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --p0-count)
      P0_COUNT="$2"
      shift 2
      ;;
    --p1-count)
      P1_COUNT="$2"
      shift 2
      ;;
    --from-step)
      FROM_STEP="$2"
      shift 2
      ;;
    --no-resume)
      RESUME=false
      shift
      ;;
    --tag)
      CREATE_TAG=true
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2
      usage
      exit 2
      ;;
  esac
done

export BIOMETRICS_AGENT_TIMEOUT_DEFAULT_SECONDS
export BIOMETRICS_AGENT_TIMEOUT_CODER_SECONDS
export BIOMETRICS_AGENT_TIMEOUT_FIXER_SECONDS

cleanup() {
  if [[ -n "${SOAK_WATCHER_PID}" ]] && kill -0 "${SOAK_WATCHER_PID}" 2>/dev/null; then
    kill "${SOAK_WATCHER_PID}" 2>/dev/null || true
    wait "${SOAK_WATCHER_PID}" 2>/dev/null || true
  fi
  rmdir "${LOCK_DIR}" 2>/dev/null || true
}

if mkdir "${LOCK_DIR}" 2>/dev/null; then
  trap cleanup EXIT INT TERM
else
  echo "Another GA closure program appears active (lock: ${LOCK_DIR})" >&2
  exit 1
fi

init_state() {
  python3 - <<PY "${STATE_FILE}" "${timestamp}" "${log_file}" "${RESUME}"
import json
import sys
from pathlib import Path

path = Path(sys.argv[1])
now = sys.argv[2]
log_file = sys.argv[3]
resume = sys.argv[4].lower() == "true"
if path.exists():
    try:
        state = json.loads(path.read_text(encoding="utf-8"))
    except Exception:
        state = {}
else:
    state = {}
if not resume:
    state = {}
state.setdefault("created_at", now)
state["last_started_at"] = now
state["log_file"] = log_file
state.setdefault("events", [])
state.setdefault("completed_steps", [])
state.setdefault("failed_steps", [])
path.write_text(json.dumps(state, indent=2), encoding="utf-8")
PY
}

append_state_event() {
  local step="$1"
  local status="$2"
  local message="$3"
  python3 - <<PY "${STATE_FILE}" "${step}" "${status}" "${message}"
import json
import sys
from datetime import datetime, timezone
from pathlib import Path

path = Path(sys.argv[1])
step = sys.argv[2]
status = sys.argv[3]
message = sys.argv[4]
state = {}
if path.exists():
    try:
        state = json.loads(path.read_text(encoding="utf-8"))
    except Exception:
        state = {}
state.setdefault("events", [])
state.setdefault("completed_steps", [])
state.setdefault("failed_steps", [])
event = {
    "at": datetime.now(timezone.utc).isoformat(),
    "step": step,
    "status": status,
    "message": message,
}
state["events"].append(event)
if status == "completed" and step not in state["completed_steps"]:
    state["completed_steps"].append(step)
if status == "failed" and step not in state["failed_steps"]:
    state["failed_steps"].append(step)
state["last_event"] = event
path.write_text(json.dumps(state, indent=2), encoding="utf-8")
PY
}

start_soak_watcher() {
  if [[ "${GA_CLOSURE_ENABLE_SOAK_WATCHER}" != "true" ]]; then
    return
  fi

  if ! [[ "${GA_CLOSURE_WATCH_INTERVAL_SECONDS}" =~ ^[0-9]+$ ]] || (( GA_CLOSURE_WATCH_INTERVAL_SECONDS <= 0 )); then
    echo "[ga-closure] soak watcher disabled: invalid GA_CLOSURE_WATCH_INTERVAL_SECONDS='${GA_CLOSURE_WATCH_INTERVAL_SECONDS}'" >&2
    return
  fi

  if [[ ! -x "${ROOT_DIR}/scripts/release/watch-soak-progress.sh" ]]; then
    echo "[ga-closure] soak watcher disabled: missing executable scripts/release/watch-soak-progress.sh" >&2
    return
  fi

  SOAK_WATCHER_LOG="${LOG_DIR}/ga-closure-soak-watch-${timestamp}.log"
  "${ROOT_DIR}/scripts/release/watch-soak-progress.sh" \
    --interval-seconds "${GA_CLOSURE_WATCH_INTERVAL_SECONDS}" \
    --profile "${GA_CLOSURE_WATCH_PROFILE}" \
    --out-dir "${LOG_DIR}" >"${SOAK_WATCHER_LOG}" 2>&1 &
  SOAK_WATCHER_PID="$!"

  append_state_event "program" "watcher.started" "pid=${SOAK_WATCHER_PID} profile=${GA_CLOSURE_WATCH_PROFILE} interval=${GA_CLOSURE_WATCH_INTERVAL_SECONDS}s log=${SOAK_WATCHER_LOG}"
  echo "[ga-closure] started soak watcher pid=${SOAK_WATCHER_PID} profile=${GA_CLOSURE_WATCH_PROFILE} interval=${GA_CLOSURE_WATCH_INTERVAL_SECONDS}s"
}

stop_soak_watcher() {
  if [[ -z "${SOAK_WATCHER_PID}" ]]; then
    return
  fi
  if kill -0 "${SOAK_WATCHER_PID}" 2>/dev/null; then
    kill "${SOAK_WATCHER_PID}" 2>/dev/null || true
    wait "${SOAK_WATCHER_PID}" 2>/dev/null || true
  fi
  append_state_event "program" "watcher.stopped" "pid=${SOAK_WATCHER_PID} log=${SOAK_WATCHER_LOG}"
  echo "[ga-closure] stopped soak watcher pid=${SOAK_WATCHER_PID}"
  SOAK_WATCHER_PID=""
}

is_step_completed() {
  local step="$1"
  python3 - <<PY "${STATE_FILE}" "${step}"
import json
import sys
from pathlib import Path

path = Path(sys.argv[1])
step = sys.argv[2]
if not path.exists():
    raise SystemExit(1)
state = json.loads(path.read_text(encoding="utf-8"))
completed = set(state.get("completed_steps", []))
raise SystemExit(0 if step in completed else 1)
PY
}

latest_ga72h_summary() {
  python3 - <<PY "${ROOT_DIR}"
import json
import sys
from pathlib import Path

root = Path(sys.argv[1])
best = None
for path in sorted((root / "logs" / "soak").glob("soak-summary-*.json"), reverse=True):
    try:
        summary = json.loads(path.read_text(encoding="utf-8"))
    except Exception:
        continue
    if summary.get("profile_label") != "ga-72h":
        continue
    try:
        duration = int(summary.get("duration_seconds", 0))
    except Exception:
        duration = 0
    if duration < 72 * 3600:
        continue
    best = path
    break

if best:
    print(str(best))
PY
}

run_step() {
  local step="$1"
  shift

  if [[ -n "${FROM_STEP}" && "${FROM_STEP}" != "${step}" ]]; then
    if [[ "${_from_reached}" != "true" ]]; then
      echo "[ga-closure] skipping ${step} (before --from-step ${FROM_STEP})"
      return 0
    fi
  fi
  if [[ "${step}" == "${FROM_STEP}" ]]; then
    _from_reached=true
  fi

  if [[ "${RESUME}" == "true" ]] && is_step_completed "${step}" >/dev/null 2>&1; then
    echo "[ga-closure] skipping ${step} (already completed in state)"
    return 0
  fi

  echo "[ga-closure] step=${step} started"
  local cmd_pid=""
  local cmd_rc=0
  local vt_bin="${ROOT_DIR}/scripts/visual_truth/vt"
  if [[ "${VISUAL_TRUTH:-0}" == "1" && -x "${vt_bin}" ]]; then
    "${vt_bin}" step --name "ga-closure:${step}" -- "$@" &
  else
    "$@" &
  fi
  cmd_pid="$!"
  append_state_event "${step}" "started" "started pid=${cmd_pid}"

  if [[ "${GA_CLOSURE_HEARTBEAT_SECONDS}" =~ ^[0-9]+$ ]] && (( GA_CLOSURE_HEARTBEAT_SECONDS > 0 )); then
    while kill -0 "${cmd_pid}" 2>/dev/null; do
      sleep "${GA_CLOSURE_HEARTBEAT_SECONDS}"
      if kill -0 "${cmd_pid}" 2>/dev/null; then
        append_state_event "${step}" "heartbeat" "running"
      fi
    done
  fi

  if wait "${cmd_pid}"; then
    cmd_rc=0
  else
    cmd_rc=$?
  fi

  if [[ "${cmd_rc}" -eq 0 ]]; then
    append_state_event "${step}" "completed" "completed"
    echo "[ga-closure] step=${step} completed"
    return 0
  fi

  append_state_event "${step}" "failed" "failed"
  echo "[ga-closure] step=${step} failed (exit=${cmd_rc})" >&2
  return "${cmd_rc}"
}

_from_reached=false
if [[ -z "${FROM_STEP}" ]]; then
  _from_reached=true
fi

init_state
append_state_event "program" "started" "ga closure program started"
echo "[ga-closure] agent timeouts: default=${BIOMETRICS_AGENT_TIMEOUT_DEFAULT_SECONDS}s coder=${BIOMETRICS_AGENT_TIMEOUT_CODER_SECONDS}s fixer=${BIOMETRICS_AGENT_TIMEOUT_FIXER_SECONDS}s"
start_soak_watcher

run_step "rc-scope-lock" "${ROOT_DIR}/scripts/release/lock-rc-scope.sh"
run_step "gate-check-baseline" "${ROOT_DIR}/scripts/release/check-gates.sh"
run_step "gate-a" "${ROOT_DIR}/scripts/release/run-gate-a.sh" --write-report
run_step "runtime-surface-smoke" "${ROOT_DIR}/scripts/release/runtime-surface-smoke.sh"
run_step "soak-cleanup" "${ROOT_DIR}/scripts/release/cleanup-soak-runs.sh" --older-than-minutes "${SOAK_CLEANUP_OLDER_THAN_MINUTES}"
run_step "rehearsal-program" "${ROOT_DIR}/scripts/release/run-rehearsal-program.sh"
run_step "soak-72h" "${ROOT_DIR}/scripts/release/run-soak-72h.sh"

ga_summary="$(latest_ga72h_summary)"
if [[ -z "${ga_summary}" ]]; then
  append_state_event "gate-b" "failed" "no ga-72h soak summary found"
  echo "[ga-closure] no ga-72h soak summary found for gate-b step" >&2
  exit 1
fi
run_step "gate-b" "${ROOT_DIR}/scripts/release/run-gate-b.sh" --soak-summary "${ga_summary}" --p0-count "${P0_COUNT}" --p1-count "${P1_COUNT}" --write-report
run_step "ga-cut-dry-run" "${ROOT_DIR}/scripts/release/run-ga-cut.sh"

if [[ "${CREATE_TAG}" == "true" ]]; then
  run_step "ga-cut-tag" "${ROOT_DIR}/scripts/release/run-ga-cut.sh" --tag
fi

stop_soak_watcher
append_state_event "program" "completed" "ga closure program completed"
echo "[ga-closure] completed successfully"
