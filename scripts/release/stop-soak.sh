#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SOAK_DIR="${ROOT_DIR}/logs/soak"

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

for p in "${profiles[@]}"; do
  pid_file="${SOAK_DIR}/active-${p}.pid"
  meta_file="${SOAK_DIR}/active-${p}.json"
  mode=""
  session=""
  meta_pid=""
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
      tmux kill-session -t "${session}" || true
      echo "Profile ${p}: stopped tmux session ${session}"
    else
      echo "Profile ${p}: tmux session ${session} not running (stale)"
    fi
    rm -f "${pid_file}" "${meta_file}"
    continue
  fi

  pid=""
  if [[ -f "${pid_file}" ]]; then
    pid="$(cat "${pid_file}")"
  elif [[ "${meta_pid}" =~ ^[0-9]+$ ]] && [[ "${meta_pid}" != "0" ]]; then
    pid="${meta_pid}"
  fi

  if [[ -z "${pid}" ]]; then
    echo "Profile ${p}: no active process metadata"
    rm -f "${pid_file}" "${meta_file}"
    continue
  fi

  if kill -0 "${pid}" 2>/dev/null; then
    kill "${pid}" || true
    sleep 1
    if kill -0 "${pid}" 2>/dev/null; then
      kill -9 "${pid}" || true
    fi
    echo "Profile ${p}: stopped pid ${pid}"
  else
    echo "Profile ${p}: pid ${pid} not running (stale)"
  fi

  rm -f "${pid_file}" "${meta_file}"
done
