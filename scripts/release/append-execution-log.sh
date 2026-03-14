#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
LOG_FILE="${ROOT_DIR}/docs/releases/EXECUTION_LOG.md"

usage() {
  cat <<USAGE
Usage: $(basename "$0") <entry-line-1> [entry-line-2 ...]

Appends a timestamped block to docs/releases/EXECUTION_LOG.md.
Each argument becomes one bullet line.
USAGE
}

if [[ $# -lt 1 ]]; then
  usage
  exit 2
fi

if [[ ! -f "${LOG_FILE}" ]]; then
  echo "Execution log not found: ${LOG_FILE}" >&2
  exit 1
fi

timestamp="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
{
  echo
  echo "## ${timestamp}"
  for line in "$@"; do
    echo "- ${line}"
  done
} >> "${LOG_FILE}"

echo "Appended execution log entry at ${timestamp}"
