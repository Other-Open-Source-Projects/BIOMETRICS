#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
OUT_DIR="${OUT_DIR:-${ROOT_DIR}/logs/release}"
INTERVAL_SECONDS="${INTERVAL_SECONDS:-300}"
DURATION_SECONDS="${DURATION_SECONDS:-0}"
PROFILE="${PROFILE:-all}"

usage() {
  cat <<USAGE
Usage: $(basename "$0") [--interval-seconds <n>] [--duration-seconds <n>] [--profile <rehearsal-6h|rehearsal-24h|ga-72h|all>] [--out-dir <path>]

Continuously snapshots soak progress and soak-status output for audit evidence.
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --interval-seconds)
      INTERVAL_SECONDS="$2"
      shift 2
      ;;
    --duration-seconds)
      DURATION_SECONDS="$2"
      shift 2
      ;;
    --profile)
      PROFILE="$2"
      shift 2
      ;;
    --out-dir)
      OUT_DIR="$2"
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

if ! [[ "${INTERVAL_SECONDS}" =~ ^[0-9]+$ ]] || (( INTERVAL_SECONDS <= 0 )); then
  echo "--interval-seconds must be a positive integer" >&2
  exit 2
fi
if ! [[ "${DURATION_SECONDS}" =~ ^[0-9]+$ ]] || (( DURATION_SECONDS < 0 )); then
  echo "--duration-seconds must be an integer >= 0" >&2
  exit 2
fi

mkdir -p "${OUT_DIR}"
started_epoch="$(date +%s)"

echo "[watch-soak-progress] started at $(date -u +%Y-%m-%dT%H:%M:%SZ)"
echo "[watch-soak-progress] interval=${INTERVAL_SECONDS}s duration=${DURATION_SECONDS}s profile=${PROFILE}"
echo "[watch-soak-progress] out_dir=${OUT_DIR}"

while true; do
  now_epoch="$(date +%s)"
  elapsed=$(( now_epoch - started_epoch ))
  if (( DURATION_SECONDS > 0 && elapsed > DURATION_SECONDS )); then
    break
  fi

  ts="$(date -u +%Y%m%dT%H%M%SZ)"
  progress_out="${OUT_DIR}/soak-progress-${ts}.json"
  status_out="${OUT_DIR}/soak-status-${ts}.txt"

  "${ROOT_DIR}/scripts/release/snapshot-soak-progress.sh" --out "${progress_out}" >/dev/null
  "${ROOT_DIR}/scripts/release/soak-status.sh" --profile "${PROFILE}" > "${status_out}" 2>&1 || true

  echo "[watch-soak-progress] captured ${progress_out} and ${status_out}"

  if (( DURATION_SECONDS > 0 && elapsed >= DURATION_SECONDS )); then
    break
  fi
  sleep "${INTERVAL_SECONDS}"
done

echo "[watch-soak-progress] finished at $(date -u +%Y-%m-%dT%H:%M:%SZ)"
