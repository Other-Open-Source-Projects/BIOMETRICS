#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
LOG_DIR="${ROOT_DIR}/logs/release"
mkdir -p "${LOG_DIR}"

timestamp="$(date -u +%Y%m%dT%H%M%SZ)"
log_file="${LOG_DIR}/gate-b-${timestamp}.log"
summary_file="${LOG_DIR}/gate-b-${timestamp}.json"
AUTO_EXECUTION_LOG="${AUTO_EXECUTION_LOG:-true}"

SOAK_SUMMARY=""
P0_COUNT="${P0_COUNT:-0}"
P1_COUNT="${P1_COUNT:-0}"
write_report=false

usage() {
  cat <<USAGE
Usage: $(basename "$0") [--soak-summary <path>] [--p0-count <n>] [--p1-count <n>] [--write-report]
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --soak-summary)
      SOAK_SUMMARY="$2"
      shift 2
      ;;
    --p0-count)
      P0_COUNT="$2"
      shift 2
      ;;
    --p1-count)
      P1_COUNT="$2"
      shift 2
      ;;
    --write-report)
      write_report=true
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

if [[ -z "${SOAK_SUMMARY}" ]]; then
  SOAK_SUMMARY="$(ls -1t "${ROOT_DIR}"/logs/soak/soak-summary-*.json | head -n1)"
fi
if [[ -z "${SOAK_SUMMARY}" || ! -f "${SOAK_SUMMARY}" ]]; then
  echo "Missing soak summary file: ${SOAK_SUMMARY}" >&2
  exit 2
fi

exec > >(tee "${log_file}") 2>&1

append_execution_log() {
  if [[ "${AUTO_EXECUTION_LOG}" != "true" ]]; then
    return
  fi
  if [[ ! -x "${ROOT_DIR}/scripts/release/append-execution-log.sh" ]]; then
    return
  fi
  "${ROOT_DIR}/scripts/release/append-execution-log.sh" \
    "Gate B execution ${timestamp}: overall=${overall}." \
    "Gate B evidence summary: ${summary_file}" \
    "Gate B soak source: ${SOAK_SUMMARY}" \
    "Gate B execution log: ${log_file}"
}

echo "[gate-b] started at ${timestamp}"
branch="$(git -C "${ROOT_DIR}" rev-parse --abbrev-ref HEAD)"

status_soak="fail"
status_core="fail"
status_backlog="fail"
status_profile="fail"
status_duration="fail"
status_actual_duration="fail"
status_evidence="fail"

summary_profile_label="$(python3 - <<'PY' "${SOAK_SUMMARY}"
import json
import sys
from pathlib import Path
data = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
print(data.get("profile_label", ""))
PY
)"
summary_duration_seconds="$(python3 - <<'PY' "${SOAK_SUMMARY}"
import json
import sys
from pathlib import Path
data = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
value = data.get("duration_seconds", 0)
try:
    print(int(value))
except Exception:
    print(0)
PY
)"
summary_actual_duration_seconds="$(python3 - <<'PY' "${SOAK_SUMMARY}"
import json
import sys
from pathlib import Path
data = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
value = data.get("harness_elapsed_seconds", 0)
try:
    print(int(value))
except Exception:
    print(0)
PY
)"
summary_has_delta_evidence="$(python3 - <<'PY' "${SOAK_SUMMARY}"
import json
import sys
from pathlib import Path
data = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
baseline = data.get("baseline_metrics_file")
metrics_delta = data.get("metrics_delta")
if isinstance(baseline, str) and baseline.strip() and isinstance(metrics_delta, dict):
    print("true")
else:
    print("false")
PY
)"

if [[ "${summary_profile_label}" == "ga-72h" ]]; then
  status_profile="pass"
else
  echo "Gate B hard reject: profile_label must be ga-72h, got '${summary_profile_label}'" >&2
fi

if (( summary_duration_seconds >= 259200 )); then
  status_duration="pass"
else
  echo "Gate B hard reject: duration_seconds must be >= 259200, got '${summary_duration_seconds}'" >&2
fi

if (( summary_actual_duration_seconds >= 259200 )); then
  status_actual_duration="pass"
else
  echo "Gate B hard reject: harness_elapsed_seconds must be >= 259200, got '${summary_actual_duration_seconds}'" >&2
fi

if [[ "${summary_has_delta_evidence}" == "true" ]]; then
  status_evidence="pass"
else
  echo "Gate B hard reject: soak summary must include baseline_metrics_file and metrics_delta evidence" >&2
fi

if "${ROOT_DIR}/scripts/analyze-soak.py" \
  --summary "${SOAK_SUMMARY}" \
  --min-success-rate 0.98 \
  --max-timeouts 0 \
  --max-dispatch-p95-ms 250 \
  --max-fallback-rate 0.05 \
  --max-backpressure-per-run 20; then
  status_soak="pass"
fi

if "${ROOT_DIR}/scripts/release/check-gates.sh"; then
  status_core="pass"
fi

if [[ "${P0_COUNT}" == "0" && "${P1_COUNT}" == "0" ]]; then
  status_backlog="pass"
else
  echo "P0/P1 backlog not clean: P0=${P0_COUNT}, P1=${P1_COUNT}" >&2
fi

overall="pass"
for status in "${status_profile}" "${status_duration}" "${status_actual_duration}" "${status_evidence}" "${status_soak}" "${status_core}" "${status_backlog}"; do
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
  "soak_summary": "${SOAK_SUMMARY}",
  "soak_profile_label": "${summary_profile_label}",
  "soak_duration_seconds": int("${summary_duration_seconds}"),
  "soak_actual_duration_seconds": int("${summary_actual_duration_seconds}"),
  "soak_has_delta_evidence": "${summary_has_delta_evidence}",
  "p0_count": int("${P0_COUNT}"),
  "p1_count": int("${P1_COUNT}"),
  "steps": {
    "soak_profile_gate": "${status_profile}",
    "soak_duration_gate": "${status_duration}",
    "soak_actual_duration_gate": "${status_actual_duration}",
    "soak_delta_evidence_gate": "${status_evidence}",
    "soak_thresholds": "${status_soak}",
    "core_gate_checks": "${status_core}",
    "p0_p1_backlog": "${status_backlog}"
  },
  "overall": "${overall}"
}
print(json.dumps(summary, indent=2))
PY

if [[ "${write_report}" == true ]]; then
  python3 - <<'PY' "${ROOT_DIR}/docs/releases/GATE_B_REPORT.md" "${summary_file}"
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
- Soak summary: `{summary['soak_summary']}`
- Soak profile label: `{summary.get('soak_profile_label', '')}`
- Soak duration seconds: {summary.get('soak_duration_seconds', 0)}
- Soak actual duration seconds: {summary.get('soak_actual_duration_seconds', 0)}
- Soak has delta evidence: `{summary.get('soak_has_delta_evidence', '')}`
- P0 count: {summary['p0_count']}
- P1 count: {summary['p1_count']}
- profile-gate: {summary['steps']['soak_profile_gate']}
- duration-gate: {summary['steps']['soak_duration_gate']}
- actual-duration-gate: {summary['steps']['soak_actual_duration_gate']}
- delta-evidence-gate: {summary['steps']['soak_delta_evidence_gate']}
- soak-thresholds: {summary['steps']['soak_thresholds']}
- core-gates: {summary['steps']['core_gate_checks']}
- backlog-check: {summary['steps']['p0_p1_backlog']}
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

echo "[gate-b] summary: ${summary_file}"
if [[ "${overall}" != "pass" ]]; then
  append_execution_log
  echo "[gate-b] FAILED" >&2
  exit 1
fi

append_execution_log
echo "[gate-b] PASSED"
