#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SOAK_DIR="${ROOT_DIR}/logs/soak"
RELEASE_DIR="${ROOT_DIR}/logs/release"
mkdir -p "${SOAK_DIR}" "${RELEASE_DIR}"

PROFILE="${PROFILE:-rehearsal-6h}"
HOURS=""
DURATION_SECONDS=""
FAIL_ON_GATES="${FAIL_ON_GATES:-true}"
DETACH_MODE="${DETACH_MODE:-nohup}"
FORCE_RESTART=false
CLEANUP_STALE=true
STALE_MINUTES="${STALE_MINUTES:-30}"

usage() {
  cat <<USAGE
Usage: $(basename "$0") --profile <rehearsal-6h|rehearsal-24h|ga-72h> [options]

Options:
  --hours <n>              Override profile default duration in hours.
  --duration-seconds <n>   Override exact duration for test runs.
  --fail-on-gates <bool>   true|false (default: true).
  --detach-mode <mode>     nohup|tmux (default: nohup).
  --force-restart          Stop existing process for profile before starting.
  --cleanup-stale <bool>   Cancel stale running soak runs before start (default: true).
  --stale-minutes <n>      Age threshold for stale runs (default: 30).
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --profile)
      PROFILE="$2"
      shift 2
      ;;
    --hours)
      HOURS="$2"
      shift 2
      ;;
    --duration-seconds)
      DURATION_SECONDS="$2"
      shift 2
      ;;
    --fail-on-gates)
      FAIL_ON_GATES="$2"
      shift 2
      ;;
    --detach-mode)
      DETACH_MODE="$2"
      shift 2
      ;;
    --force-restart)
      FORCE_RESTART=true
      shift
      ;;
    --cleanup-stale)
      CLEANUP_STALE="$2"
      shift 2
      ;;
    --stale-minutes)
      STALE_MINUTES="$2"
      shift 2
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

case "${DETACH_MODE}" in
  nohup|tmux) : ;;
  *)
    echo "Invalid detach mode: ${DETACH_MODE} (allowed: nohup|tmux)" >&2
    usage
    exit 2
    ;;
esac

case "${PROFILE}" in
  rehearsal-6h) : ;;
  rehearsal-24h) : ;;
  ga-72h) : ;;
  *)
    echo "Invalid profile: ${PROFILE}" >&2
    usage
    exit 2
    ;;
esac

if [[ -z "${HOURS}" ]]; then
  case "${PROFILE}" in
    rehearsal-6h) HOURS="6" ;;
    rehearsal-24h) HOURS="24" ;;
    ga-72h) HOURS="72" ;;
  esac
fi

pid_file="${SOAK_DIR}/active-${PROFILE}.pid"
meta_file="${SOAK_DIR}/active-${PROFILE}.json"
timestamp="$(date -u +%Y%m%dT%H%M%SZ)"
log_file="${RELEASE_DIR}/soak-${PROFILE}-${timestamp}.log"
tmux_session=""
pid=0

if [[ -f "${pid_file}" ]]; then
  old_pid="$(cat "${pid_file}" 2>/dev/null || true)"
  if [[ "${old_pid}" =~ ^[0-9]+$ ]] && kill -0 "${old_pid}" 2>/dev/null; then
    if [[ "${FORCE_RESTART}" != "true" ]]; then
      echo "Profile ${PROFILE} already running with PID ${old_pid}. Use --force-restart to replace." >&2
      exit 1
    fi
    kill "${old_pid}" || true
    sleep 1
  fi
fi

if [[ -f "${meta_file}" ]]; then
  old_mode="$(python3 - <<'PY' "${meta_file}"
import json
import sys
from pathlib import Path
path = Path(sys.argv[1])
try:
    data = json.loads(path.read_text(encoding="utf-8"))
except Exception:
    print("")
    raise SystemExit(0)
print(data.get("detach_mode", ""))
PY
)"
  old_session="$(python3 - <<'PY' "${meta_file}"
import json
import sys
from pathlib import Path
path = Path(sys.argv[1])
try:
    data = json.loads(path.read_text(encoding="utf-8"))
except Exception:
    print("")
    raise SystemExit(0)
print(data.get("tmux_session", ""))
PY
)"
  if [[ "${old_mode}" == "tmux" && -n "${old_session}" ]]; then
    if tmux has-session -t "${old_session}" 2>/dev/null; then
      if [[ "${FORCE_RESTART}" != "true" ]]; then
        echo "Profile ${PROFILE} already running in tmux session ${old_session}. Use --force-restart to replace." >&2
        exit 1
      fi
      tmux kill-session -t "${old_session}" || true
      sleep 1
    fi
  fi
fi

if [[ "${CLEANUP_STALE}" == "true" ]]; then
  "${ROOT_DIR}/scripts/release/cleanup-soak-runs.sh" --older-than-minutes "${STALE_MINUTES}"
fi

cmd=(
  "./scripts/release/run-soak-rehearsal.sh"
  "--hours" "${HOURS}"
  "--profile-label" "${PROFILE}"
  "--fail-on-gates" "${FAIL_ON_GATES}"
)
if [[ -n "${DURATION_SECONDS}" ]]; then
  cmd+=("--duration-seconds" "${DURATION_SECONDS}")
fi

quoted_cmd="$(printf "%q " "${cmd[@]}")"
if [[ "${DETACH_MODE}" == "tmux" ]]; then
  if ! command -v tmux >/dev/null 2>&1; then
    echo "tmux is required for --detach-mode tmux" >&2
    exit 1
  fi
  tmux_session="soak-${PROFILE}-${timestamp}"
  tmux new-session -d -s "${tmux_session}" "cd ${ROOT_DIR} && ${quoted_cmd} | tee ${log_file}"
  pane_pid="$(tmux list-panes -t "${tmux_session}" -F '#{pane_pid}' 2>/dev/null | head -n1 || true)"
  if [[ "${pane_pid}" =~ ^[0-9]+$ ]]; then
    pid="${pane_pid}"
    echo "${pid}" > "${pid_file}"
  else
    rm -f "${pid_file}"
  fi
else
  nohup bash -lc "cd ${ROOT_DIR} && ${quoted_cmd}" >"${log_file}" 2>&1 &
  pid="$!"
  echo "${pid}" >"${pid_file}"
fi

python3 - <<PY "${meta_file}" "${pid}" "${PROFILE}" "${HOURS}" "${FAIL_ON_GATES}" "${DURATION_SECONDS}" "${log_file}" "${timestamp}" "${quoted_cmd}" "${DETACH_MODE}" "${tmux_session}"
import json
import sys
from pathlib import Path

meta_path = Path(sys.argv[1])
meta = {
    "pid": int(sys.argv[2]) if str(sys.argv[2]).isdigit() else 0,
    "profile": sys.argv[3],
    "hours": int(sys.argv[4]),
    "fail_on_gates": sys.argv[5],
    "duration_seconds_override": sys.argv[6],
    "log_file": sys.argv[7],
    "started_at_utc": sys.argv[8],
    "command": sys.argv[9].strip(),
    "detach_mode": sys.argv[10],
    "tmux_session": sys.argv[11],
}
meta_path.write_text(json.dumps(meta, indent=2), encoding="utf-8")
PY

sleep 2
if [[ "${DETACH_MODE}" == "tmux" ]]; then
  if ! tmux has-session -t "${tmux_session}" 2>/dev/null; then
    echo "Failed to start soak profile ${PROFILE}: tmux session exited immediately (${tmux_session})" >&2
    echo "Log tail:" >&2
    tail -n 40 "${log_file}" >&2 || true
    rm -f "${pid_file}" "${meta_file}"
    exit 1
  fi
else
  if ! kill -0 "${pid}" 2>/dev/null; then
    echo "Failed to start soak profile ${PROFILE}: process exited immediately (pid=${pid})" >&2
    echo "Log tail:" >&2
    tail -n 40 "${log_file}" >&2 || true
    rm -f "${pid_file}" "${meta_file}"
    exit 1
  fi
fi

echo "Started soak profile ${PROFILE}"
if [[ "${DETACH_MODE}" == "tmux" ]]; then
  echo "  Mode: tmux"
  echo "  Session: ${tmux_session}"
  if [[ "${pid}" -gt 0 ]]; then
    echo "  Pane PID: ${pid}"
  fi
else
  echo "  Mode: nohup"
  echo "  PID: ${pid}"
fi
echo "  Log: ${log_file}"
if [[ -f "${pid_file}" ]]; then
  echo "  PID file: ${pid_file}"
fi
echo "  Meta file: ${meta_file}"
