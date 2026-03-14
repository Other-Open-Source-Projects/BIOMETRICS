#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
HOURS="${HOURS:-6}"
API_BASE="${API_BASE:-http://127.0.0.1:59013}"
RUN_INTERVAL_SECONDS="${RUN_INTERVAL_SECONDS:-15}"
POLL_INTERVAL_SECONDS="${POLL_INTERVAL_SECONDS:-2}"
RUN_TIMEOUT_SECONDS="${RUN_TIMEOUT_SECONDS:-1800}"
GOAL_PARTS="${GOAL_PARTS:-50}"
MAX_PARALLELISM="${MAX_PARALLELISM:-8}"
OUTPUT_DIR="${OUTPUT_DIR:-${ROOT_DIR}/logs/soak}"
DURATION_SECONDS="${DURATION_SECONDS:-}"
FAIL_ON_GATES="${FAIL_ON_GATES:-true}"
PROFILE_LABEL="${PROFILE_LABEL:-}"

usage() {
  cat <<USAGE
Usage: $(basename "$0") [--hours <6|24|N>] [--duration-seconds <n>] [--profile-label <rehearsal-6h|rehearsal-24h|ga-72h>] [--api-base <url>] [--output-dir <path>] [--fail-on-gates <true|false>]
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --hours)
      HOURS="$2"
      shift 2
      ;;
    --api-base)
      API_BASE="$2"
      shift 2
      ;;
    --output-dir)
      OUTPUT_DIR="$2"
      shift 2
      ;;
    --duration-seconds)
      DURATION_SECONDS="$2"
      shift 2
      ;;
    --profile-label)
      PROFILE_LABEL="$2"
      shift 2
      ;;
    --fail-on-gates)
      FAIL_ON_GATES="$2"
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

mkdir -p "${OUTPUT_DIR}"
previous_latest_summary="$(ls -1t "${OUTPUT_DIR}"/soak-summary-*.json 2>/dev/null | head -n1 || true)"

if [[ -z "${PROFILE_LABEL}" ]]; then
  echo "--profile-label is required (rehearsal-6h|rehearsal-24h|ga-72h)" >&2
  exit 2
fi
case "${PROFILE_LABEL}" in
  rehearsal-6h|rehearsal-24h|ga-72h)
    ;;
  *)
    echo "invalid --profile-label '${PROFILE_LABEL}' (allowed: rehearsal-6h|rehearsal-24h|ga-72h)" >&2
    exit 2
    ;;
esac

if [[ -n "${DURATION_SECONDS}" ]]; then
  export DURATION_SECONDS
fi

set +e
DURATION_HOURS="${HOURS}" \
API_BASE="${API_BASE}" \
RUN_INTERVAL_SECONDS="${RUN_INTERVAL_SECONDS}" \
POLL_INTERVAL_SECONDS="${POLL_INTERVAL_SECONDS}" \
RUN_TIMEOUT_SECONDS="${RUN_TIMEOUT_SECONDS}" \
GOAL_PARTS="${GOAL_PARTS}" \
SCHEDULER_MODE="dag_parallel_v1" \
MAX_PARALLELISM="${MAX_PARALLELISM}" \
PROFILE_LABEL="${PROFILE_LABEL}" \
FAIL_ON_GATES="${FAIL_ON_GATES}" \
OUTPUT_DIR="${OUTPUT_DIR}" \
"${ROOT_DIR}/scripts/run-soak.sh"
soak_exit=$?
set -e

latest_summary="$(ls -1t "${OUTPUT_DIR}"/soak-summary-*.json 2>/dev/null | head -n1 || true)"
if [[ -z "${latest_summary}" ]]; then
  echo "No soak summary was produced in ${OUTPUT_DIR}" >&2
  exit 1
fi
if [[ -n "${previous_latest_summary}" && "${latest_summary}" == "${previous_latest_summary}" ]]; then
  echo "No new soak summary produced; latest remains ${latest_summary}" >&2
  exit 1
fi
"${ROOT_DIR}/scripts/release/update-soak-report.py" --summary "${latest_summary}"

echo "Rehearsal completed with summary: ${latest_summary}"
exit "${soak_exit}"
