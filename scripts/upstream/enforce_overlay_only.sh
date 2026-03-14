#!/usr/bin/env bash
set -euo pipefail

BASE_RANGE=""
if [[ -n "${GITHUB_BASE_REF:-}" ]]; then
  BASE_RANGE="origin/${GITHUB_BASE_REF}...HEAD"
else
  if git rev-parse --verify HEAD~1 >/dev/null 2>&1; then
    BASE_RANGE="HEAD~1...HEAD"
  fi
fi

changed_files=""
if [[ -n "${BASE_RANGE}" ]]; then
  changed_files="$(git diff --name-only "${BASE_RANGE}" || true)"
else
  changed_files="$(git show --name-only --pretty='' HEAD || true)"
fi

if [[ -z "${changed_files}" ]]; then
  echo "overlay-only check: no changed files"
  exit 0
fi

violations="$(echo "${changed_files}" | grep '^third_party/codex-upstream/' | grep -v '^third_party/codex-upstream/upstream.lock.json$' || true)"

if [[ -n "${violations}" ]]; then
  echo "overlay-only check failed: direct edits under third_party/codex-upstream are blocked"
  echo "allowed exception: third_party/codex-upstream/upstream.lock.json"
  echo "violating files:"
  echo "${violations}"
  exit 1
fi

echo "overlay-only check passed"
