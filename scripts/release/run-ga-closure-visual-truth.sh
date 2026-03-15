#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

export VISUAL_TRUTH=1
export VISUAL_TRUTH_DIR="${VISUAL_TRUTH_DIR:-/tmp/automation_logs}"
export VISUAL_TRUTH_VALIDATE=1
export VISUAL_TRUTH_VALIDATE_STRICT=1
export VISUAL_TRUTH_ALLOW_UPLOAD="${VISUAL_TRUTH_ALLOW_UPLOAD:-0}"

if [[ -z "${VISUAL_TRUTH_SESSION:-}" ]]; then
  export VISUAL_TRUTH_SESSION="ga-closure-vt-$(date -u +%Y%m%dT%H%M%SZ)"
fi

vt_bin="${ROOT_DIR}/scripts/visual_truth/vt"
if [[ ! -x "${vt_bin}" ]]; then
  echo "missing visual truth runner: ${vt_bin}" >&2
  exit 2
fi

if [[ -z "${NIM_API_KEY:-}" && -z "${NVIDIA_API_KEY:-}" ]]; then
  echo "missing NIM_API_KEY (or NVIDIA_API_KEY)" >&2
  exit 2
fi

if [[ "${VISUAL_TRUTH_ALLOW_UPLOAD}" != "1" ]]; then
  echo "VISUAL_TRUTH_ALLOW_UPLOAD must be 1" >&2
  exit 2
fi

if [[ -z "${VISUAL_TRUTH_UPLOAD_CMD:-}" && "${VISUAL_TRUTH_ALLOW_BASE64:-0}" != "1" ]]; then
  echo "set VISUAL_TRUTH_UPLOAD_CMD or VISUAL_TRUTH_ALLOW_BASE64=1" >&2
  exit 2
fi

"${vt_bin}" doctor
exec "${ROOT_DIR}/scripts/release/run-ga-closure-program.sh" "$@"
