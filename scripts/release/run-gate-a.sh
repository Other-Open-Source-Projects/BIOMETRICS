#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
CLI_DIR="${ROOT_DIR}/biometrics-cli"
WEB_DIR="${CLI_DIR}/web-v3"
LOG_DIR="${ROOT_DIR}/logs/release"
mkdir -p "${LOG_DIR}"

timestamp="$(date -u +%Y%m%dT%H%M%SZ)"
log_file="${LOG_DIR}/gate-a-${timestamp}.log"
summary_file="${LOG_DIR}/gate-a-${timestamp}.json"
AUTO_EXECUTION_LOG="${AUTO_EXECUTION_LOG:-true}"

write_report=false
while [[ $# -gt 0 ]]; do
  case "$1" in
    --write-report)
      write_report=true
      shift
      ;;
    *)
      echo "Unknown argument: $1" >&2
      exit 2
      ;;
  esac
done

exec > >(tee "${log_file}") 2>&1

append_execution_log() {
  if [[ "${AUTO_EXECUTION_LOG}" != "true" ]]; then
    return
  fi
  if [[ ! -x "${ROOT_DIR}/scripts/release/append-execution-log.sh" ]]; then
    return
  fi
  "${ROOT_DIR}/scripts/release/append-execution-log.sh" \
    "Gate A execution ${timestamp}: overall=${overall}." \
    "Gate A evidence summary: ${summary_file}" \
    "Gate A execution log: ${log_file}"
}

echo "[gate-a] started at ${timestamp}"
branch="$(git -C "${ROOT_DIR}" rev-parse --abbrev-ref HEAD)"

status_check_gates="fail"
status_high_card_go="fail"
status_scenarios_go="fail"
status_web_e2e="fail"

if "${ROOT_DIR}/scripts/release/check-gates.sh"; then
  status_check_gates="pass"
fi

if (
  cd "${CLI_DIR}"
  go test -count=1 -timeout=6m -v ./internal/planning ./internal/runtime/scheduler -run 'TestBuildPlanCapsAtFiftyWorkPackages|TestBuildTaskNodeDefsHighCardinality|TestTopologicalNodeOrderDetectsCycle|TestHighCardinalityRunCompletesWithoutQueueDepthFallback|TestMetricsSnapshotIncludesQueueAndEventBusFields'
); then
  status_high_card_go="pass"
fi

if (
  cd "${CLI_DIR}"
  go test -count=1 -timeout=6m -v ./internal/api/http -run 'TestRunGraphAndAttemptsEndpoints|TestEventOrderingStartedBeforeCompleted|TestHealthReadyAndMetricsEndpoints|TestDAGParallelHighCardinalityRunCompletesWithoutFallback|TestSSEEndpointReplaysLatestLimitWithEventIDs|TestSSEEndpointStreamsLiveTypedAndMessageFrames|TestAPIContractDriftRunTaskEventSchemas'
); then
  status_scenarios_go="pass"
fi

if (
  cd "${WEB_DIR}"
  if [[ ! -d node_modules ]]; then
    npm ci
  fi
  npx playwright install chromium
  npm run test:e2e
); then
  status_web_e2e="pass"
fi

overall="pass"
for status in "${status_check_gates}" "${status_high_card_go}" "${status_scenarios_go}" "${status_web_e2e}"; do
  if [[ "${status}" != "pass" ]]; then
    overall="fail"
  fi
done

python3 - <<PY > "${summary_file}"
import json
summary = {
  "timestamp": "${timestamp}",
  "branch": "${branch}",
  "log_file": "${log_file}",
  "steps": {
    "check_gates": "${status_check_gates}",
    "high_cardinality_go": "${status_high_card_go}",
    "scenario_go": "${status_scenarios_go}",
    "web_e2e": "${status_web_e2e}"
  },
  "overall": "${overall}"
}
print(json.dumps(summary, indent=2))
PY

if [[ "${write_report}" == true ]]; then
  python3 - <<'PY' "${ROOT_DIR}/docs/releases/GATE_A_REPORT.md" "${summary_file}"
import json
import sys
from pathlib import Path

report = Path(sys.argv[1])
summary_file = Path(sys.argv[2])
summary = json.loads(summary_file.read_text(encoding="utf-8"))
start = "<!-- BEGIN AUTO_EXEC -->"
end = "<!-- END AUTO_EXEC -->"
block = f"""{start}
## Latest Local Execution Evidence
- Timestamp (UTC): {summary['timestamp']}
- Branch: `{summary['branch']}`
- Summary: `{summary_file}`
- Log: `{summary['log_file']}`
- check-gates: {summary['steps']['check_gates']}
- high-cardinality-go: {summary['steps']['high_cardinality_go']}
- scenario-go: {summary['steps']['scenario_go']}
- web-e2e: {summary['steps']['web_e2e']}
- overall: **{summary['overall'].upper()}**
{end}"""
content = report.read_text(encoding="utf-8")
if start in content and end in content:
    prefix = content.split(start, 1)[0]
    suffix = content.split(end, 1)[1]
    content = prefix + block + suffix
else:
    content = content.rstrip() + "\n\n" + block + "\n"
report.write_text(content, encoding="utf-8")
print(f"Updated {report} with local evidence")
PY
fi

echo "[gate-a] summary: ${summary_file}"
if [[ "${overall}" != "pass" ]]; then
  append_execution_log
  echo "[gate-a] FAILED" >&2
  exit 1
fi

append_execution_log
echo "[gate-a] PASSED"
