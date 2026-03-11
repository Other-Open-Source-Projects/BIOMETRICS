#!/usr/bin/env bash
set -euo pipefail

API_BASE="${API_BASE:-http://127.0.0.1:59013}"

endpoints=(
  "/health"
  "/health/ready"
  "/api/v1/models"
  "/api/v1/auth/codex/status"
  "/api/v1/skills"
  "/api/v1/orchestrator/capabilities"
  "/api/v1/evals/leaderboard"
)

echo "[runtime-smoke] API base: ${API_BASE}"

for endpoint in "${endpoints[@]}"; do
  code="$(curl -sS -o /tmp/runtime-surface-smoke-body.$$ -w "%{http_code}" "${API_BASE}${endpoint}" || true)"
  if [[ "${code}" != "200" ]]; then
    echo "[runtime-smoke] FAIL ${endpoint} status=${code}" >&2
    if [[ -f /tmp/runtime-surface-smoke-body.$$ ]]; then
      echo "[runtime-smoke] body: $(tr '\n' ' ' < /tmp/runtime-surface-smoke-body.$$ | sed 's/[[:space:]]\+/ /g' | cut -c1-260)" >&2
    fi
    rm -f /tmp/runtime-surface-smoke-body.$$
    exit 1
  fi
  echo "[runtime-smoke] PASS ${endpoint}"
done

rm -f /tmp/runtime-surface-smoke-body.$$
echo "[runtime-smoke] all required endpoints responded 200"
