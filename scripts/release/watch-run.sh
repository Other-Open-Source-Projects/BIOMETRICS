#!/usr/bin/env bash
set -euo pipefail

API_BASE="${API_BASE:-http://127.0.0.1:59013}"
RUN_ID=""
POLL_SECONDS="${POLL_SECONDS:-10}"
OUT_FILE=""

usage() {
  cat <<USAGE
Usage: $(basename "$0") --run-id <id> [--poll-seconds <n>] [--out <path>]

Watches a run until terminal status and emits progress snapshots as JSON lines.
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --run-id)
      RUN_ID="$2"
      shift 2
      ;;
    --poll-seconds)
      POLL_SECONDS="$2"
      shift 2
      ;;
    --out)
      OUT_FILE="$2"
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

if [[ -z "${RUN_ID}" ]]; then
  echo "--run-id is required" >&2
  exit 2
fi

if [[ -z "${OUT_FILE}" ]]; then
  ts="$(date -u +%Y%m%dT%H%M%SZ)"
  OUT_FILE="logs/release/run-watch-${RUN_ID}-${ts}.jsonl"
fi

mkdir -p "$(dirname "${OUT_FILE}")"
echo "Watching run ${RUN_ID}"
echo "Output: ${OUT_FILE}"

while true; do
  run_json="$(curl -fsS "${API_BASE}/api/v1/runs/${RUN_ID}")"
  tasks_json="$(curl -fsS "${API_BASE}/api/v1/runs/${RUN_ID}/tasks")"

  python3 - <<PY "${run_json}" "${tasks_json}" "${OUT_FILE}"
import json
import sys
from datetime import datetime, timezone
from pathlib import Path

run = json.loads(sys.argv[1])
tasks = json.loads(sys.argv[2])
out_path = Path(sys.argv[3])

def count_where(key, value):
    return sum(1 for t in tasks if t.get(key) == value)

snapshot = {
    "observed_at": datetime.now(timezone.utc).isoformat(),
    "run_id": run.get("id"),
    "status": run.get("status"),
    "run_updated_at": run.get("updated_at"),
    "tasks_total": len(tasks),
    "tasks_completed": count_where("status", "completed"),
    "tasks_running": count_where("status", "running"),
    "tasks_failed": count_where("status", "failed"),
    "tasks_cancelled": count_where("status", "cancelled"),
    "tasks_blocked": count_where("lifecycle_state", "blocked"),
    "tasks_retrying": count_where("lifecycle_state", "retrying"),
}
with out_path.open("a", encoding="utf-8") as fh:
    fh.write(json.dumps(snapshot) + "\n")
print(json.dumps(snapshot))
PY

  status="$(python3 - <<PY "${run_json}"
import json
import sys
print(json.loads(sys.argv[1]).get("status", ""))
PY
)"
  if [[ "${status}" == "completed" || "${status}" == "failed" || "${status}" == "cancelled" ]]; then
    echo "Run reached terminal status: ${status}"
    exit 0
  fi
  sleep "${POLL_SECONDS}"
done
