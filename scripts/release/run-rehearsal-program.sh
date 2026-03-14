#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
LOG_DIR="${ROOT_DIR}/logs/release"
SOAK_DIR="${ROOT_DIR}/logs/soak"
mkdir -p "${LOG_DIR}" "${SOAK_DIR}"

RUN_6H=true
RUN_24H=true
CLEANUP_STALE="${CLEANUP_STALE:-true}"
STALE_MINUTES="${STALE_MINUTES:-30}"
ACTIVE_PROFILE=""
BIOMETRICS_AGENT_TIMEOUT_DEFAULT_SECONDS="${BIOMETRICS_AGENT_TIMEOUT_DEFAULT_SECONDS:-120}"
BIOMETRICS_AGENT_TIMEOUT_CODER_SECONDS="${BIOMETRICS_AGENT_TIMEOUT_CODER_SECONDS:-600}"
BIOMETRICS_AGENT_TIMEOUT_FIXER_SECONDS="${BIOMETRICS_AGENT_TIMEOUT_FIXER_SECONDS:-600}"

usage() {
  cat <<USAGE
Usage: $(basename "$0") [--only-6h] [--only-24h]

Runs the mandatory rehearsal program with hard-gate evaluation:
  1) 6h rehearsal  (profile: rehearsal-6h)
  2) 24h rehearsal (profile: rehearsal-24h)
USAGE
}

current_tmux_session() {
  if [[ -z "${TMUX:-}" ]]; then
    echo ""
    return
  fi
  if ! command -v tmux >/dev/null 2>&1; then
    echo ""
    return
  fi
  tmux display-message -p '#S' 2>/dev/null || true
}

register_active_profile() {
  local profile="$1"
  local stage_name="$2"
  local stage_log="$3"
  local pid_file="${SOAK_DIR}/active-${profile}.pid"
  local meta_file="${SOAK_DIR}/active-${profile}.json"
  local tmux_session
  tmux_session="$(current_tmux_session)"

  echo "$$" > "${pid_file}"
  python3 - <<'PY' "${meta_file}" "$$" "${profile}" "${stage_name}" "${stage_log}" "${tmux_session}"
import json
import sys
from datetime import datetime, timezone
from pathlib import Path

meta_path = Path(sys.argv[1])
pid = int(sys.argv[2])
meta = {
    "pid": pid,
    "profile": sys.argv[3],
    "stage": sys.argv[4],
    "owner": "run-rehearsal-program",
    "detach_mode": "inline",
    "tmux_session": sys.argv[6],
    "log_file": sys.argv[5],
    "started_at_utc": datetime.now(timezone.utc).isoformat(),
}
meta_path.write_text(json.dumps(meta, indent=2), encoding="utf-8")
PY
}

clear_active_profile() {
  local profile="$1"
  local pid_file="${SOAK_DIR}/active-${profile}.pid"
  local meta_file="${SOAK_DIR}/active-${profile}.json"
  local should_clear="false"

  if [[ -f "${meta_file}" ]]; then
    should_clear="$(python3 - <<'PY' "${meta_file}" "$$"
import json
import sys
from pathlib import Path

path = Path(sys.argv[1])
pid = int(sys.argv[2])
try:
    data = json.loads(path.read_text(encoding="utf-8"))
except Exception:
    print("false")
    raise SystemExit(0)
owner = data.get("owner")
meta_pid = data.get("pid")
if owner == "run-rehearsal-program" and meta_pid == pid:
    print("true")
else:
    print("false")
PY
)"
  fi

  if [[ "${should_clear}" != "true" && -f "${pid_file}" ]]; then
    local pid_from_file
    pid_from_file="$(cat "${pid_file}" 2>/dev/null || true)"
    if [[ "${pid_from_file}" == "$$" ]]; then
      should_clear="true"
    fi
  fi

  if [[ "${should_clear}" == "true" ]]; then
    rm -f "${pid_file}" "${meta_file}"
  fi
}

cleanup_active_profile_on_exit() {
  if [[ -n "${ACTIVE_PROFILE}" ]]; then
    clear_active_profile "${ACTIVE_PROFILE}"
  fi
}

trap cleanup_active_profile_on_exit EXIT INT TERM

export BIOMETRICS_AGENT_TIMEOUT_DEFAULT_SECONDS
export BIOMETRICS_AGENT_TIMEOUT_CODER_SECONDS
export BIOMETRICS_AGENT_TIMEOUT_FIXER_SECONDS

while [[ $# -gt 0 ]]; do
  case "$1" in
    --only-6h)
      RUN_6H=true
      RUN_24H=false
      shift
      ;;
    --only-24h)
      RUN_6H=false
      RUN_24H=true
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

write_incident() {
  local stage="$1"
  local stage_log="$2"
  local summary_path="$3"
  local ts
  ts="$(date -u +%Y%m%dT%H%M%SZ)"
  local incident_file="${LOG_DIR}/incident-${stage}-${ts}.md"
  cat >"${incident_file}" <<EOF
# BIOMETRICS Soak Incident

- Timestamp (UTC): ${ts}
- Stage: ${stage}
- Severity: P1 (release-blocking until resolved)
- Owner: <assign-owner>
- Status: OPEN

## Failure Context
- Stage log: \`${stage_log}\`
- Soak summary: \`${summary_path}\`

## Required Actions
1. Root cause analysis
2. Fix owner + ETA
3. Re-run failing stage and attach new evidence
EOF
  echo "${incident_file}"
}

run_stage() {
  local hours="$1"
  local label="$2"
  local stage_name="$3"
  local ts
  ts="$(date -u +%Y%m%dT%H%M%SZ)"
  local stage_log="${LOG_DIR}/rehearsal-${label}-${ts}.log"
  ACTIVE_PROFILE="${label}"
  register_active_profile "${label}" "${stage_name}" "${stage_log}"

  echo "[rehearsal-program] starting ${stage_name} (${hours}h, ${label})"
  set +e
  "${ROOT_DIR}/scripts/release/run-soak-rehearsal.sh" \
    --hours "${hours}" \
    --profile-label "${label}" \
    --fail-on-gates true | tee "${stage_log}"
  local stage_status=${PIPESTATUS[0]}
  set -e
  clear_active_profile "${label}"
  ACTIVE_PROFILE=""

  local summary_path
  summary_path="$(grep -Eo '/Users/[^ ]*soak-summary-[0-9TZ]+\.json' "${stage_log}" | tail -n1 || true)"
  if [[ -z "${summary_path}" ]]; then
    summary_path="(not-detected)"
  fi

  if [[ ${stage_status} -ne 0 ]]; then
    local incident_file
    incident_file="$(write_incident "${stage_name}" "${stage_log}" "${summary_path}")"
    echo "[rehearsal-program] ${stage_name} FAILED"
    echo "[rehearsal-program] incident logged: ${incident_file}"
    return ${stage_status}
  fi

  echo "[rehearsal-program] ${stage_name} PASSED"
  echo "[rehearsal-program] summary: ${summary_path}"
}

if [[ "${CLEANUP_STALE}" == "true" ]]; then
  "${ROOT_DIR}/scripts/release/cleanup-soak-runs.sh" --older-than-minutes "${STALE_MINUTES}" || true
fi

echo "[rehearsal-program] agent timeouts: default=${BIOMETRICS_AGENT_TIMEOUT_DEFAULT_SECONDS}s coder=${BIOMETRICS_AGENT_TIMEOUT_CODER_SECONDS}s fixer=${BIOMETRICS_AGENT_TIMEOUT_FIXER_SECONDS}s"

if [[ "${RUN_6H}" == "true" ]]; then
  run_stage 6 rehearsal-6h rehearsal-6h
fi

if [[ "${RUN_24H}" == "true" ]]; then
  run_stage 24 rehearsal-24h rehearsal-24h
fi

echo "[rehearsal-program] complete"
