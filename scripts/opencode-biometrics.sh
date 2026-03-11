#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

usage() {
  cat <<'EOF'
Usage:
  ./scripts/opencode-biometrics.sh [--start]

What it does:
  - Verifies opencode is installed
  - Verifies the BIOMETRICS OpenCode plugin loader exists in this repo
  - Prints the recommended BIOMETRICS plugin tool flow
  - Optionally starts `opencode` in this repo (interactive) with --start

Notes:
  - Mutating plugin tools require `confirm:true` (e.g. biometrics.bootstrap_all).
EOF
}

START=false
for arg in "$@"; do
  case "${arg}" in
    --start) START=true ;;
    -h|--help) usage; exit 0 ;;
    *) echo "Unknown arg: ${arg}" >&2; usage; exit 2 ;;
  esac
done

if ! command -v opencode >/dev/null 2>&1; then
  echo "Missing: opencode (install via brew: brew install opencode)" >&2
  exit 1
fi

LOADER_PATH="${ROOT_DIR}/.opencode/plugins/biometrics.ts"
IMPL_PATH="${ROOT_DIR}/opencode-config/plugins/biometrics.ts"

if [[ ! -f "${LOADER_PATH}" ]]; then
  echo "Missing plugin loader: ${LOADER_PATH}" >&2
  echo "Expected: this repo ships the BIOMETRICS plugin loader under .opencode/plugins/." >&2
  exit 1
fi

if [[ ! -f "${IMPL_PATH}" ]]; then
  echo "Missing plugin implementation: ${IMPL_PATH}" >&2
  exit 1
fi

echo "OK: opencode=$(opencode --version | head -n 1)"
echo "OK: loader=${LOADER_PATH}"
echo "OK: plugin=${IMPL_PATH}"
echo ""
echo "Recommended flow (inside opencode):"
cat <<'EOF'
1) (Optional) Create a plan queue file:
   - biometrics.bootstrap_plans

2) Full end-to-end bootstrap (repo/env/onboard/build/start/gates):
   - biometrics.bootstrap_all
     {
       "repo_url": "https://github.com/Delqhi/BIOMETRICS.git",
       "repo_dir": "~/BIOMETRICS",
       "ref": "main",
       "onboard_args": [],
       "base_url": "http://127.0.0.1:59013",
       "run_gates": true,
       "confirm": true
     }

3) Controlplane lifecycle:
   - biometrics.controlplane.start { "repo_dir": "~/BIOMETRICS", "confirm": true }
   - biometrics.health.ready { "base_url": "http://127.0.0.1:59013" }
   - biometrics.controlplane.stop { "repo_dir": "~/BIOMETRICS", "force": false, "confirm": true }
EOF
echo ""

if [[ "${START}" == "true" ]]; then
  echo "Starting opencode in ${ROOT_DIR} ..."
  cd "${ROOT_DIR}"
  exec opencode
fi

echo "Tip: run with --start to launch opencode here."

