#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
CLI_DIR="${ROOT_DIR}/biometrics-cli"
WEB_DIR="${CLI_DIR}/web-v3"
WEBSITE_DIR="${ROOT_DIR}/website"

log() {
  printf '[gate-check] %s\n' "$*"
}

require_file() {
  local path="$1"
  if [[ ! -f "${path}" ]]; then
    echo "Missing required file: ${path}" >&2
    exit 1
  fi
}

require_tool() {
  local tool="$1"
  if ! command -v "${tool}" >/dev/null 2>&1; then
    echo "Missing required tool: ${tool}" >&2
    exit 1
  fi
}

GO_VERIFY_MODE="${BIOMETRICS_GO_VERIFY_MODE:-run}"

run_go_hard_suite() {
  local pkgs=(
    ./cmd/controlplane
    ./cmd/biometrics
    ./cmd/onboard
    ./cmd/skills
    ./internal/api/http
    ./internal/blueprints
    ./internal/controlplane
    ./internal/onboarding
    ./internal/planning
    ./internal/policy
    ./internal/runtime/...
    ./internal/store/sqlite
    ./internal/skillkit
    ./internal/skillops
  )

  case "${GO_VERIFY_MODE}" in
    run)
      go test -timeout=5m "${pkgs[@]}"
      ;;
    compile)
      local tmp_dir
      tmp_dir="$(mktemp -d "${TMPDIR:-/tmp}/biometrics-go-compile.XXXXXX")"
      echo "Go test execution disabled (BIOMETRICS_GO_VERIFY_MODE=compile), compiling test binaries into ${tmp_dir}" >&2
      local pkg
      for pkg in "${pkgs[@]}"; do
        if [[ "${pkg}" == *"..." ]]; then
          while IFS= read -r expanded; do
            [[ -z "${expanded}" ]] && continue
            go test -c "${expanded}" -o "${tmp_dir}/$(echo "${expanded}" | tr '/.' '__').test"
          done < <(go list "${pkg}")
          continue
        fi
        go test -c "${pkg}" -o "${tmp_dir}/$(echo "${pkg}" | tr '/.' '__').test"
      done
      ;;
    *)
      echo "Invalid BIOMETRICS_GO_VERIFY_MODE=${GO_VERIFY_MODE} (expected run|compile)" >&2
      exit 1
      ;;
  esac
}

run_go_onboarding_smoke_suite() {
  case "${GO_VERIFY_MODE}" in
    run)
      go test -timeout=2m ./internal/onboarding -run 'TestDoctorRunStepDoesNotWriteArtifacts|TestStepExposeCommandPathMissingCreatesWarning|TestResumeInitStateRetainsCompletedStep|TestPersistReportIncludesWarnings'
      ;;
    compile)
      local tmp_dir
      tmp_dir="$(mktemp -d "${TMPDIR:-/tmp}/biometrics-go-compile-onboarding.XXXXXX")"
      go test -c ./internal/onboarding -o "${tmp_dir}/internal_onboarding.test"
      ;;
    *)
      echo "Invalid BIOMETRICS_GO_VERIFY_MODE=${GO_VERIFY_MODE} (expected run|compile)" >&2
      exit 1
      ;;
  esac
}

log "Validating required files"
require_file "${ROOT_DIR}/docs/releases/GATE_A_REPORT.md"
require_file "${ROOT_DIR}/docs/releases/GATE_B_REPORT.md"
require_file "${ROOT_DIR}/docs/releases/SOAK_72H_REPORT.md"
require_file "${ROOT_DIR}/docs/releases/EXECUTION_LOG.md"
require_file "${ROOT_DIR}/docs/guides/RELEASE_FREEZE_V3_1.md"
require_file "${ROOT_DIR}/docs/guides/WEB_VISUAL_REGRESSION.md"
require_file "${ROOT_DIR}/docs/api/openapi-v3-controlplane.yaml"
require_file "${ROOT_DIR}/docs/specs/contracts/run.schema.json"
require_file "${ROOT_DIR}/docs/specs/contracts/task.schema.json"
require_file "${ROOT_DIR}/docs/specs/contracts/event.schema.json"
require_file "${ROOT_DIR}/docs/specs/contracts/skill.schema.json"
require_file "${ROOT_DIR}/docs/specs/contracts/attempt.schema.json"
require_file "${ROOT_DIR}/docs/specs/contracts/graph.schema.json"
require_file "${ROOT_DIR}/docs/specs/contracts/error.schema.json"
require_file "${ROOT_DIR}/docs/specs/contracts/model.schema.json"
require_file "${ROOT_DIR}/docs/specs/index.json"
require_file "${ROOT_DIR}/scripts/run-soak.sh"
require_file "${ROOT_DIR}/scripts/release/run-soak-rehearsal.sh"
require_file "${ROOT_DIR}/scripts/release/run-soak-72h.sh"
require_file "${ROOT_DIR}/scripts/release/run-rehearsal-program.sh"
require_file "${ROOT_DIR}/scripts/release/start-soak.sh"
require_file "${ROOT_DIR}/scripts/release/soak-status.sh"
require_file "${ROOT_DIR}/scripts/release/stop-soak.sh"
require_file "${ROOT_DIR}/scripts/release/cleanup-soak-runs.sh"
require_file "${ROOT_DIR}/scripts/release/snapshot-soak-progress.sh"
require_file "${ROOT_DIR}/scripts/release/watch-soak-progress.sh"
require_file "${ROOT_DIR}/scripts/release/update-soak-report.py"
require_file "${ROOT_DIR}/scripts/release/run-ga-closure-program.sh"
require_file "${ROOT_DIR}/scripts/release/append-execution-log.sh"
require_file "${ROOT_DIR}/biometrics-onboard"
require_file "${ROOT_DIR}/biometrics-cli/web-v3/playwright.config.ts"
require_file "${ROOT_DIR}/biometrics-cli/web-v3/pnpm-lock.yaml"
require_file "${ROOT_DIR}/biometrics-cli/web-v3/tests/e2e/visual.spec.ts"
require_file "${ROOT_DIR}/biometrics-cli/web-v3/tests/e2e/visual.spec.ts-snapshots/shell-baseline-darwin.png"
require_file "${ROOT_DIR}/biometrics-cli/web-v3/tests/e2e/visual.spec.ts-snapshots/graph-fallback-baseline-darwin.png"
require_file "${ROOT_DIR}/website/package.json"
require_file "${ROOT_DIR}/website/pnpm-lock.yaml"
require_file "${ROOT_DIR}/website/wrangler.toml"
require_file "${ROOT_DIR}/website/next.config.mjs"
require_file "${ROOT_DIR}/website/theme.config.tsx"
require_file "${ROOT_DIR}/website/pages/index.mdx"
require_file "${ROOT_DIR}/website/pages/quickstart.mdx"
require_file "${ROOT_DIR}/website/pages/install/index.mdx"
require_file "${ROOT_DIR}/website/pages/docs/index.mdx"
require_file "${ROOT_DIR}/website/pages/de/index.mdx"
require_file "${ROOT_DIR}/website/public/sitemap.xml"
require_file "${ROOT_DIR}/website/public/robots.txt"
require_file "${ROOT_DIR}/website/public/_headers"
require_file "${ROOT_DIR}/website/playwright.config.ts"

log "Enforcing pnpm-only lockfile policy for web surfaces"
if [[ -f "${ROOT_DIR}/biometrics-cli/web-v3/package-lock.json" ]]; then
  echo "Legacy npm lockfile detected: biometrics-cli/web-v3/package-lock.json" >&2
  exit 1
fi
if [[ -f "${ROOT_DIR}/website/package-lock.json" ]]; then
  echo "Legacy npm lockfile detected: website/package-lock.json" >&2
  exit 1
fi

log "Checking that soak report references raw evidence"
if ! grep -q 'logs/soak/' "${ROOT_DIR}/docs/releases/SOAK_72H_REPORT.md"; then
  echo "SOAK_72H_REPORT.md must reference logs/soak evidence paths" >&2
  exit 1
fi

log "Validating contract index inventory"
python3 - <<'PY' "${ROOT_DIR}/docs/specs/index.json"
import json
import sys
from pathlib import Path

index_path = Path(sys.argv[1])
expected = {
    "docs/specs/contracts/run.schema.json",
    "docs/specs/contracts/task.schema.json",
    "docs/specs/contracts/event.schema.json",
    "docs/specs/contracts/skill.schema.json",
    "docs/specs/contracts/attempt.schema.json",
    "docs/specs/contracts/graph.schema.json",
    "docs/specs/contracts/error.schema.json",
    "docs/specs/contracts/model.schema.json",
}
index = json.loads(index_path.read_text(encoding="utf-8"))
contracts = set(index.get("contracts", []))
missing = sorted(expected - contracts)
if missing:
    print(f"Missing contracts in index.json: {missing}", file=sys.stderr)
    sys.exit(1)
PY

log "Validating release script syntax"
bash -n \
  "${ROOT_DIR}/scripts/run-soak.sh" \
  "${ROOT_DIR}/scripts/release/run-soak-rehearsal.sh" \
  "${ROOT_DIR}/scripts/release/run-soak-72h.sh" \
  "${ROOT_DIR}/scripts/release/run-rehearsal-program.sh" \
  "${ROOT_DIR}/scripts/release/start-soak.sh" \
  "${ROOT_DIR}/scripts/release/soak-status.sh" \
  "${ROOT_DIR}/scripts/release/stop-soak.sh" \
  "${ROOT_DIR}/scripts/release/cleanup-soak-runs.sh" \
  "${ROOT_DIR}/scripts/release/snapshot-soak-progress.sh" \
  "${ROOT_DIR}/scripts/release/watch-soak-progress.sh" \
  "${ROOT_DIR}/scripts/release/run-ga-closure-program.sh" \
  "${ROOT_DIR}/scripts/release/run-gate-a.sh" \
  "${ROOT_DIR}/scripts/release/run-gate-b.sh" \
  "${ROOT_DIR}/scripts/release/check-gates.sh"

log "Running Go tests with hard timeout"
(
  cd "${CLI_DIR}"
  run_go_hard_suite
)

log "Running onboarding non-mutation doctor smoke tests"
(
  cd "${CLI_DIR}"
  run_go_onboarding_smoke_suite
)

log "Running web build (zero-warning core gate)"
require_tool pnpm
(
  cd "${WEB_DIR}"
  if command -v corepack >/dev/null 2>&1; then
    corepack enable
  else
    echo "corepack not found; continuing because pnpm is available (local-only deviation)" >&2
  fi
  pnpm install --frozen-lockfile
  pnpm run build 2>&1 | tee /tmp/biometrics-web-build.log
)
if grep -q '^warn - ' /tmp/biometrics-web-build.log; then
  echo "Core web build warnings are not allowed" >&2
  exit 1
fi

log "Running public website build and content checks"
(
  cd "${WEBSITE_DIR}"
  if command -v corepack >/dev/null 2>&1; then
    corepack enable
  else
    echo "corepack not found; continuing because pnpm is available (local-only deviation)" >&2
  fi
  pnpm install --frozen-lockfile
  pnpm run build
  pnpm run test:content
)

log "Enforcing OpenCode singleton config policy"
"${ROOT_DIR}/scripts/check-opencode-singleton.sh"

log "Checking for active legacy V2 references"
pattern='cmd/api-server|cmd/orchestrator|cmd/tui|cmd/agent-loop|biometrics-cli/web-ui|/api/v0|/api/v2'
found=0
while IFS= read -r -d '' file; do
  [[ "${file}" == archive/* ]] && continue
  [[ ! -f "${ROOT_DIR}/${file}" ]] && continue
  if grep -nIE "${pattern}" "${ROOT_DIR}/${file}" >/dev/null; then
    echo "Legacy reference in ${file}" >&2
    found=1
  fi
done < <(cd "${ROOT_DIR}" && git ls-files -z README.md OPENCODE.md docs rules templates/blueprints Makefile biometrics-onboard biometrics-cli/cmd/controlplane biometrics-cli/cmd/biometrics biometrics-cli/cmd/onboard biometrics-cli/internal biometrics-cli/web-v3/src website)

if [[ "${found}" -ne 0 ]]; then
  echo "Legacy V2 references found in active files" >&2
  exit 1
fi

log "Running tracked-file secret scan"
pattern='AIza[0-9A-Za-z_-]{30,}|nvapi-[0-9A-Za-z_-]{20,}|glpat-[0-9A-Za-z_-]{20,}|ghp_[0-9A-Za-z]{20,}|github_pat_[0-9A-Za-z_]{20,}|sk-(live|proj)-[0-9A-Za-z_-]{20,}|eyJ[A-Za-z0-9_-]{10,}\\.[A-Za-z0-9_-]{10,}\\.[A-Za-z0-9_-]{10,}'
found=0
while IFS= read -r -d '' file; do
  [[ "${file}" == archive/* ]] && continue
  [[ "${file}" == .sisyphus/archive/* ]] && continue
  [[ ! -f "${ROOT_DIR}/${file}" ]] && continue
  raw_matches="$(grep -nHIE "${pattern}" "${ROOT_DIR}/${file}" || true)"
  [[ -z "${raw_matches}" ]] && continue
  filtered_matches="$(printf '%s\n' "${raw_matches}" | grep -viE 'YOUR_|_KEY_HERE|xxxxxxxx|<REDACTED|example|placeholder|changeme|set-me' || true)"
  if [[ -n "${filtered_matches}" ]]; then
    printf '%s\n' "${filtered_matches}" >&2
    found=1
  fi
done < <(cd "${ROOT_DIR}" && git ls-files -z README.md OPENCODE.md docs rules scripts .github .env.example Makefile biometrics-onboard biometrics-cli/cmd/controlplane biometrics-cli/cmd/biometrics biometrics-cli/cmd/onboard biometrics-cli/internal biometrics-cli/web-v3/src website)

if [[ "${found}" -ne 0 ]]; then
  echo "Potential secret detected in tracked files" >&2
  exit 1
fi

log "Gate checks passed"
