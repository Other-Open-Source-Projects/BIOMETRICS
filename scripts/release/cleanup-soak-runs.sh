#!/usr/bin/env bash
set -euo pipefail

API_BASE="${API_BASE:-http://127.0.0.1:59013}"
OLDER_THAN_MINUTES="${OLDER_THAN_MINUTES:-30}"
DRY_RUN=false
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
DB_PATH="${DB_PATH:-${ROOT_DIR}/.biometrics/v3.db}"

usage() {
  cat <<USAGE
Usage: $(basename "$0") [--older-than-minutes <n>] [--dry-run]

Cancels stale running soak runs (project_id=soak) older than N minutes.
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --older-than-minutes)
      OLDER_THAN_MINUTES="$2"
      shift 2
      ;;
    --dry-run)
      DRY_RUN=true
      shift
      ;;
    --db-path)
      DB_PATH="$2"
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

if ! curl -fsS "${API_BASE}/health" >/dev/null; then
  echo "Control-plane health endpoint not reachable at ${API_BASE}/health" >&2
  exit 1
fi

reconcile_db_run() {
  local run_id="$1"
  local note="$2"
  if ! command -v sqlite3 >/dev/null 2>&1; then
    return 1
  fi
  if [[ ! -f "${DB_PATH}" ]]; then
    return 1
  fi

  sqlite3 "${DB_PATH}" <<SQL >/dev/null 2>&1
UPDATE runs
SET status='cancelled',
    error='${note}',
    updated_at=strftime('%Y-%m-%dT%H:%M:%fZ','now')
WHERE id='${run_id}' AND status='running';
UPDATE tasks
SET status='cancelled',
    lifecycle_state='cancelled',
    updated_at=strftime('%Y-%m-%dT%H:%M:%fZ','now'),
    finished_at=COALESCE(finished_at, strftime('%Y-%m-%dT%H:%M:%fZ','now'))
WHERE run_id='${run_id}' AND status='running';
SQL
}

now_epoch="$(date +%s)"
cutoff="$(( now_epoch - (OLDER_THAN_MINUTES * 60) ))"

runs_json="$(curl -fsS "${API_BASE}/api/v1/runs")"
targets="$(python3 - <<PY "${runs_json}" "${cutoff}"
import datetime
import json
import sys

runs = json.loads(sys.argv[1])
cutoff = int(sys.argv[2])

def to_epoch(value: str) -> int:
    value = value.strip()
    if value.endswith("Z"):
        value = value[:-1] + "+00:00"
    dt = datetime.datetime.fromisoformat(value)
    return int(dt.timestamp())

for run in runs:
    if run.get("project_id") != "soak":
        continue
    if run.get("status") != "running":
        continue
    created_at = run.get("created_at", "")
    if not created_at:
        continue
    try:
        created_epoch = to_epoch(created_at)
    except Exception:
        continue
    if created_epoch < cutoff:
        print(f"{run.get('id','')}|{created_at}")
PY
)"

if [[ -z "${targets}" ]]; then
  echo "No stale soak runs older than ${OLDER_THAN_MINUTES}m found."
  exit 0
fi

echo "Stale soak runs to cancel (older than ${OLDER_THAN_MINUTES}m):"
while IFS= read -r line; do
  [[ -z "${line}" ]] && continue
  run_id="${line%%|*}"
  created_at="${line#*|}"
  echo "  ${run_id} (${created_at})"
  if [[ "${DRY_RUN}" == "true" ]]; then
    continue
  fi
  body_file="$(mktemp)"
  code="$(curl -sS -o "${body_file}" -w "%{http_code}" -X POST "${API_BASE}/api/v1/runs/${run_id}/cancel" || true)"
  if [[ "${code}" =~ ^2 ]]; then
    echo "    cancelled"
  else
    reason="$(tr '\n' ' ' < "${body_file}" | sed 's/[[:space:]]\+/ /g' | cut -c1-180)"
    if [[ "${reason}" == *"not active"* ]]; then
      if reconcile_db_run "${run_id}" "orphaned stale run reconciled by cleanup-soak-runs"; then
        echo "    reconciled in sqlite (http ${code}, not active in runtime)"
      else
        echo "    skip (http ${code}): ${reason}"
      fi
    else
      echo "    skip (http ${code}): ${reason}"
    fi
  fi
  rm -f "${body_file}"
done <<< "${targets}"
