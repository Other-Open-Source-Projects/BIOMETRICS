#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
TAG_NAME="${TAG_NAME:-v3.1.0}"
DO_TAG=false

usage() {
  cat <<USAGE
Usage: $(basename "$0") [--tag]

Validates Gate B summary before GA cut. Pass --tag to create annotated git tag.
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --tag)
      DO_TAG=true
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

latest_gate_b="$(ls -1t "${ROOT_DIR}"/logs/release/gate-b-*.json | head -n1)"
if [[ -z "${latest_gate_b}" ]]; then
  echo "No gate-b summary found under logs/release" >&2
  exit 1
fi

overall="$(python3 - <<PY "${latest_gate_b}"
import json
import sys
from pathlib import Path
path = Path(sys.argv[1])
data = json.loads(path.read_text(encoding='utf-8'))
print(data.get('overall', 'fail'))
PY
)"

if [[ "${overall}" != "pass" ]]; then
  echo "Gate B summary indicates failure: ${latest_gate_b}" >&2
  exit 1
fi

echo "Gate B summary PASS: ${latest_gate_b}"
echo "GA cut prerequisites validated for ${TAG_NAME}"

if [[ "${DO_TAG}" == true ]]; then
  git -C "${ROOT_DIR}" tag -a "${TAG_NAME}" -m "BIOMETRICS ${TAG_NAME} GA release"
  echo "Created annotated tag ${TAG_NAME}"
else
  echo "Dry run only. Re-run with --tag to create ${TAG_NAME}."
fi
