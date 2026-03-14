#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SOAK_DIR="${ROOT_DIR}/logs/soak"
ACTIVE_WINDOW_MINUTES="${ACTIVE_WINDOW_MINUTES:-30}"
GA_CLOSURE_STATE_FILE="${ROOT_DIR}/logs/release/ga-closure-state.json"

PROFILE="${PROFILE:-all}"

usage() {
  cat <<USAGE
Usage: $(basename "$0") [--profile <rehearsal-6h|rehearsal-24h|ga-72h|all>]
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --profile)
      PROFILE="$2"
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

ga_closure_status="$(
python3 - <<'PY' "${GA_CLOSURE_STATE_FILE}" "${ACTIVE_WINDOW_MINUTES}"
import json
import re
import shutil
import subprocess
import sys
import time
from pathlib import Path

state_path = Path(sys.argv[1])
active_window_minutes = int(sys.argv[2])
result = {
    "active": "0",
    "profile": "",
    "step": "",
    "session": "",
    "log_file": "",
    "last_event_at": "",
}

if state_path.exists():
    try:
        state = json.loads(state_path.read_text(encoding="utf-8"))
    except Exception:
        state = {}

    events = state.get("events")
    if not isinstance(events, list):
        events = []

    open_steps = {}
    for event in events:
        if not isinstance(event, dict):
            continue
        step = str(event.get("step", "")).strip()
        status = str(event.get("status", "")).strip()
        at = str(event.get("at", "")).strip()
        if not step:
            continue
        if status in {"started", "heartbeat"}:
            open_steps[step] = at
        elif status in {"completed", "failed"}:
            open_steps.pop(step, None)

    step = ""
    for candidate in ("rehearsal-program", "soak-72h"):
        if candidate in open_steps:
            step = candidate
            break
    result["step"] = step

    last_event = state.get("last_event")
    if isinstance(last_event, dict):
        result["last_event_at"] = str(last_event.get("at", "")).strip()
    elif step in open_steps:
        result["last_event_at"] = str(open_steps.get(step, "")).strip()

    log_file = str(state.get("log_file", "")).strip()
    result["log_file"] = log_file

    state_session = ""
    if log_file:
        match = re.search(r"(ga-closure-[0-9TZ]+)\.log$", Path(log_file).name)
        if match:
            state_session = match.group(1)

    def tmux_has_session(session_name: str) -> bool:
        if not session_name:
            return False
        try:
            subprocess.run(
                ["tmux", "has-session", "-t", session_name],
                stdout=subprocess.DEVNULL,
                stderr=subprocess.DEVNULL,
                check=True,
            )
            return True
        except Exception:
            return False

    active_session = ""
    if shutil.which("tmux"):
        if state_session and tmux_has_session(state_session):
            active_session = state_session
        else:
            try:
                tmux_ls = subprocess.run(
                    ["tmux", "ls"],
                    capture_output=True,
                    text=True,
                    check=False,
                )
                for raw_line in tmux_ls.stdout.splitlines():
                    session_name = raw_line.split(":", 1)[0].strip()
                    if session_name.startswith("ga-closure-"):
                        active_session = session_name
                        break
            except Exception:
                pass

    process_active = False
    try:
        pgrep = subprocess.run(
            ["pgrep", "-f", "run-ga-closure-program.sh"],
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
            check=False,
        )
        process_active = pgrep.returncode == 0
    except Exception:
        process_active = False

    log_recent = False
    if log_file:
        try:
            log_age_seconds = max(0.0, time.time() - Path(log_file).stat().st_mtime)
            log_recent = log_age_seconds <= (active_window_minutes * 60)
        except Exception:
            log_recent = False

    runtime_active = process_active or bool(active_session and log_recent)
    profile = ""

    if step == "soak-72h":
        profile = "ga-72h"
    elif step == "rehearsal-program":
        text = ""
        if log_file:
            try:
                text = Path(log_file).read_text(encoding="utf-8", errors="replace")
            except Exception:
                text = ""

        def started(label: str) -> bool:
            return f"[rehearsal-program] starting {label}" in text

        def done(label: str) -> bool:
            return (
                f"[rehearsal-program] {label} PASSED" in text
                or f"[rehearsal-program] {label} FAILED" in text
            )

        if started("rehearsal-24h") and not done("rehearsal-24h"):
            profile = "rehearsal-24h"
        elif started("rehearsal-6h") and not done("rehearsal-6h"):
            profile = "rehearsal-6h"
        else:
            labels = re.findall(
                r"Profile label:\s*(rehearsal-6h|rehearsal-24h|ga-72h)",
                text,
            )
            if labels:
                profile = labels[-1]

    if runtime_active and profile:
        result["active"] = "1"
        result["profile"] = profile
        result["session"] = active_session or state_session

def clean(value: str) -> str:
    return str(value).replace("\t", " ").replace("\n", " ").strip()

fields = [
    result.get("active", "0"),
    result.get("profile", ""),
    result.get("step", ""),
    result.get("session", ""),
    result.get("log_file", ""),
    result.get("last_event_at", ""),
]
print("\t".join(clean(v) for v in fields))
PY
)"

IFS=$'\t' read -r ga_closure_active ga_closure_profile ga_closure_step ga_closure_session ga_closure_log_file ga_closure_last_event_at <<< "${ga_closure_status}"

profiles=()
case "${PROFILE}" in
  all) profiles=("rehearsal-6h" "rehearsal-24h" "ga-72h") ;;
  rehearsal-6h|rehearsal-24h|ga-72h) profiles=("${PROFILE}") ;;
  *)
    echo "Invalid profile: ${PROFILE}" >&2
    usage
    exit 2
    ;;
esac

echo "Soak status ($(date -u +%Y-%m-%dT%H:%M:%SZ))"
echo

for p in "${profiles[@]}"; do
  pid_file="${SOAK_DIR}/active-${p}.pid"
  meta_file="${SOAK_DIR}/active-${p}.json"
  echo "Profile: ${p}"
  source_note=""
  mode=""
  session=""
  meta_pid="0"
  if [[ -f "${meta_file}" ]]; then
    mode="$(python3 - <<'PY' "${meta_file}"
import json
import sys
from pathlib import Path
try:
    data = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
except Exception:
    data = {}
print(data.get("detach_mode", ""))
PY
)"
    session="$(python3 - <<'PY' "${meta_file}"
import json
import sys
from pathlib import Path
try:
    data = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
except Exception:
    data = {}
print(data.get("tmux_session", ""))
PY
)"
    meta_pid="$(python3 - <<'PY' "${meta_file}"
import json
import sys
from pathlib import Path
try:
    data = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
except Exception:
    data = {}
pid = data.get("pid", 0)
print(pid if isinstance(pid, int) else 0)
PY
)"
  fi

  if [[ "${mode}" == "tmux" && -n "${session}" ]]; then
    if command -v tmux >/dev/null 2>&1 && tmux has-session -t "${session}" 2>/dev/null; then
      echo "  Process: RUNNING (tmux session=${session})"
    else
      echo "  Process: STOPPED (stale tmux session=${session})"
    fi
  elif [[ -f "${pid_file}" ]]; then
    pid="$(cat "${pid_file}")"
    if kill -0 "${pid}" 2>/dev/null; then
      echo "  Process: RUNNING (pid=${pid})"
    else
      echo "  Process: STOPPED (stale pid=${pid})"
    fi
  elif [[ "${meta_pid}" =~ ^[0-9]+$ ]] && [[ "${meta_pid}" != "0" ]]; then
    if kill -0 "${meta_pid}" 2>/dev/null; then
      echo "  Process: RUNNING (pid=${meta_pid})"
    else
      echo "  Process: STOPPED (stale pid=${meta_pid})"
    fi
  elif [[ "${ga_closure_active}" == "1" && "${ga_closure_profile}" == "${p}" ]]; then
    if [[ -n "${ga_closure_session}" ]]; then
      echo "  Process: RUNNING (ga-closure step=${ga_closure_step}, session=${ga_closure_session})"
    elif [[ -n "${ga_closure_step}" ]]; then
      echo "  Process: RUNNING (ga-closure step=${ga_closure_step})"
    else
      echo "  Process: RUNNING (ga-closure)"
    fi
    source_note="ga-closure-state"
  else
    echo "  Process: NOT STARTED"
  fi

  if [[ -f "${meta_file}" ]]; then
    if [[ -n "${mode}" ]]; then
      echo "  Mode: ${mode}"
    fi
    if [[ -n "${session}" ]]; then
      echo "  Session: ${session}"
    fi
    python3 - <<PY "${meta_file}"
import json
import sys
from pathlib import Path
m = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
print(f"  Started: {m.get('started_at_utc', '')}")
print(f"  Log: {m.get('log_file', '')}")
PY
  fi

  if [[ "${source_note}" == "ga-closure-state" ]]; then
    if [[ -n "${ga_closure_log_file}" ]]; then
      echo "  Log: ${ga_closure_log_file}"
    fi
    if [[ -n "${ga_closure_last_event_at}" ]]; then
      echo "  Last GA event: ${ga_closure_last_event_at}"
    fi
    echo "  Source: ga-closure-state"
  fi

  latest_summary="$(python3 - <<PY "${ROOT_DIR}" "${p}"
import json
import sys
from pathlib import Path

root = Path(sys.argv[1])
profile = sys.argv[2]
best = None
for path in sorted((root / "logs" / "soak").glob("soak-summary-*.json"), reverse=True):
    try:
        data = json.loads(path.read_text(encoding="utf-8"))
    except Exception:
        continue
    if data.get("profile_label") == profile:
        best = str(path)
        break
print(best or "")
PY
)"
  if [[ -n "${latest_summary}" ]]; then
    python3 - <<PY "${latest_summary}"
import json
import sys
from pathlib import Path
s = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
print(f"  Latest summary: {sys.argv[1]}")
print(f"  Runs started: {s.get('runs_started', 0)}")
print(f"  Success rate: {float(s.get('run_success_rate', 0.0)):.4f}")
print(f"  Timeouts: {s.get('runs_timed_out', 0)}")
print(f"  Dispatch p95 ms: {float(s.get('dispatch_latency_p95_estimate_ms', 0.0)):.2f}")
print(f"  Backpressure/run: {float(s.get('run_backpressure_per_run', 0.0)):.2f}")
PY
  else
    echo "  Latest summary: none"
  fi
  echo
done

if curl -fsS "http://127.0.0.1:59013/health" >/dev/null 2>&1; then
  latest_run="$(curl -fsS http://127.0.0.1:59013/api/v1/runs | jq -r '.[0] | "\(.id) \(.status) \(.created_at) \(.updated_at)"' 2>/dev/null || true)"
  if [[ -n "${latest_run}" ]]; then
    echo "Control-plane latest run: ${latest_run}"
  fi

  active_soak_json="$(curl -fsS http://127.0.0.1:59013/api/v1/runs 2>/dev/null || true)"
  python3 - <<PY "${active_soak_json}" "${ACTIVE_WINDOW_MINUTES}"
import datetime
import json
import sys

runs = json.loads(sys.argv[1])
window_minutes = int(sys.argv[2])
now = datetime.datetime.now(datetime.timezone.utc)
recent = []
stale_count = 0

for run in runs:
    if run.get("project_id") != "soak" or run.get("status") != "running":
        continue
    created_at = str(run.get("created_at", ""))
    if not created_at:
        continue
    try:
        dt = datetime.datetime.fromisoformat(created_at.replace("Z", "+00:00"))
    except Exception:
        continue
    age_minutes = (now - dt).total_seconds() / 60.0
    if age_minutes <= window_minutes:
        recent.append((run.get("id", ""), created_at, age_minutes))
    else:
        stale_count += 1

if recent:
    print(f"Active soak runs via API (<= {window_minutes}m):")
    for run_id, created_at, age in recent:
        print(f"  {run_id} running {created_at} age={age:.1f}m")
if stale_count:
    print(f"Stale running soak rows (> {window_minutes}m): {stale_count} (likely orphan metadata)")
PY
else
  echo "Control-plane health endpoint not reachable."
fi
