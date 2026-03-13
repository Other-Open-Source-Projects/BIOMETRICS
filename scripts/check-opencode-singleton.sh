#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

log() {
  printf '[opencode-singleton] %s\n' "$*"
}

log "Checking for repo-local OpenCode config files (tracked, non-template)"
dups="$(cd "${ROOT_DIR}" && git ls-files | grep -E '(^|/)(opencode|oh-my-opencode)\.json$' | grep -v '^templates/' | grep -v '^.opencode/opencode.json.template$' || true)"
if [[ -n "${dups}" ]]; then
  echo "Repo-local OpenCode config files are prohibited (use ~/.config/opencode/* instead):" >&2
  printf '%s\n' "${dups}" >&2
  exit 1
fi

log "Checking for project-local .opencode/*.json runtime config files (tracked)"
tracked_opencode_json="$(cd "${ROOT_DIR}" && git ls-files | grep -E '^\.opencode/(opencode|oh-my-opencode)\.json$' | grep -v '\.opencode/opencode\.json\.template$' || true)"
if [[ -n "${tracked_opencode_json}" ]]; then
  echo "Project-local runtime config detected under .opencode/*.json (prohibited):" >&2
  printf '%s\n' "${tracked_opencode_json}" >&2
  exit 1
fi

log "OK"
