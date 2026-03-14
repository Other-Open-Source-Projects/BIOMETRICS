#!/usr/bin/env bash
set -euo pipefail

API_BASE="${API_BASE:-http://127.0.0.1:59013}"
LIMIT="${LIMIT:-20}"
OUT="${OUT:-}"

usage() {
  cat <<USAGE
Usage: $(basename "$0") [--limit <n>] [--out <path>]

Creates an interim soak progress snapshot from /api/v1/runs.
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --limit)
      LIMIT="$2"
      shift 2
      ;;
    --out)
      OUT="$2"
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

if [[ -z "${OUT}" ]]; then
  ts="$(date -u +%Y%m%dT%H%M%SZ)"
  OUT="logs/release/soak-progress-${ts}.json"
fi

mkdir -p "$(dirname "${OUT}")"

runs_json="$(curl -fsS "${API_BASE}/api/v1/runs")"
python3 - <<PY "${runs_json}" "${LIMIT}" "${OUT}"
import json
import sys
from datetime import datetime, timezone
from pathlib import Path

runs = json.loads(sys.argv[1])
limit = int(sys.argv[2])
out_path = Path(sys.argv[3])

soak_runs = [r for r in runs if r.get("project_id") == "soak"][:limit]
counts = {
    "running": sum(1 for r in soak_runs if r.get("status") == "running"),
    "completed": sum(1 for r in soak_runs if r.get("status") == "completed"),
    "failed": sum(1 for r in soak_runs if r.get("status") == "failed"),
    "cancelled": sum(1 for r in soak_runs if r.get("status") == "cancelled"),
}

snapshot = {
    "captured_at": datetime.now(timezone.utc).isoformat(),
    "limit": limit,
    "counts": counts,
    "latest_soak_runs": [
        {
            "id": r.get("id"),
            "status": r.get("status"),
            "created_at": r.get("created_at"),
            "updated_at": r.get("updated_at"),
            "scheduler_mode": r.get("scheduler_mode"),
            "max_parallelism": r.get("max_parallelism"),
        }
        for r in soak_runs
    ],
}

out_path.write_text(json.dumps(snapshot, indent=2), encoding="utf-8")
print(json.dumps(snapshot, indent=2))
print(f"Wrote {out_path}")
PY
