#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
ALLOWLIST="${ROOT_DIR}/scripts/release/rc-allowlist.txt"
REQUIRED_BRANCH_PREFIX="codex/v3.1-ga-closure"
SKIP_BRANCH_CHECK=false

usage() {
  cat <<USAGE
Usage: $(basename "$0") [--allowlist <file>] [--skip-branch-check]

Validates that all changed files in the working tree are inside the GA closure scope.
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --allowlist)
      [[ $# -lt 2 ]] && { echo "--allowlist requires a value" >&2; exit 2; }
      ALLOWLIST="$2"
      shift 2
      ;;
    --skip-branch-check)
      SKIP_BRANCH_CHECK=true
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

if [[ ! -f "${ALLOWLIST}" ]]; then
  echo "Allowlist not found: ${ALLOWLIST}" >&2
  exit 2
fi

if [[ "${SKIP_BRANCH_CHECK}" != true ]]; then
  current_branch="$(git -C "${ROOT_DIR}" rev-parse --abbrev-ref HEAD)"
  if [[ "${current_branch}" != "${REQUIRED_BRANCH_PREFIX}" ]]; then
    echo "Expected branch '${REQUIRED_BRANCH_PREFIX}', got '${current_branch}'" >&2
    exit 1
  fi
fi

patterns=()
while IFS= read -r pattern; do
  [[ -z "${pattern}" ]] && continue
  if [[ "${pattern}" =~ ^[[:space:]]*# ]]; then
    continue
  fi
  patterns+=("${pattern}")
done < "${ALLOWLIST}"
if [[ "${#patterns[@]}" -eq 0 ]]; then
  echo "Allowlist has no patterns: ${ALLOWLIST}" >&2
  exit 2
fi

unscoped=()
while IFS= read -r line; do
  [[ -z "${line}" ]] && continue
  path="${line:3}"

  # Rename rows are represented as "old -> new" in porcelain output.
  if [[ "${path}" == *" -> "* ]]; then
    path="${path##* -> }"
  fi

  in_scope=false
  for pattern in "${patterns[@]}"; do
    if [[ "${path}" == ${pattern} ]]; then
      in_scope=true
      break
    fi
  done

  if [[ "${in_scope}" == false ]]; then
    unscoped+=("${path}")
  fi
done < <(git -C "${ROOT_DIR}" status --porcelain)

if [[ "${#unscoped[@]}" -gt 0 ]]; then
  echo "Found files outside RC scope:" >&2
  for path in "${unscoped[@]}"; do
    echo "  - ${path}" >&2
  done
  exit 1
fi

echo "RC scope check passed for branch $(git -C "${ROOT_DIR}" rev-parse --abbrev-ref HEAD)"
