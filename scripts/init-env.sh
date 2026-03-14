#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_TEMPLATE="${ROOT_DIR}/.env.example"
ENV_FILE="${ROOT_DIR}/.env"

if [[ ! -f "${ENV_TEMPLATE}" ]]; then
  echo "ERROR: ${ENV_TEMPLATE} not found."
  exit 1
fi

if [[ ! -f "${ENV_FILE}" ]]; then
  cp "${ENV_TEMPLATE}" "${ENV_FILE}"
  echo "Created ${ENV_FILE} from ${ENV_TEMPLATE}."
else
  echo "Using existing ${ENV_FILE}."
fi

echo "Required-key checklist (from .env.example):"
missing=0
while IFS= read -r line || [[ -n "${line}" ]]; do
  trimmed="${line%$'\r'}"
  [[ -z "${trimmed}" ]] && continue
  [[ "${trimmed}" =~ ^[[:space:]]*# ]] && continue

  key="${trimmed%%=*}"
  key="${key//[[:space:]]/}"
  [[ -z "${key}" ]] && continue

  current="$(grep -E "^${key}=" "${ENV_FILE}" | tail -n 1 | cut -d '=' -f2- || true)"
  current="$(printf '%s' "${current}" | sed 's/^[[:space:]]*//; s/[[:space:]]*$//')"

  if [[ -z "${current}" ]] || [[ "${current}" == "<set-me>" ]] || [[ "${current}" == "set-me" ]] || [[ "${current}" == "CHANGEME" ]] || [[ "${current}" == "YOUR_VALUE_HERE" ]] || [[ "${current}" =~ ^(your-|YOUR_|example|todo|TODO) ]]; then
    echo " - ${key}"
    missing=1
  fi
done < "${ENV_TEMPLATE}"

if [[ "${missing}" -eq 0 ]]; then
  echo "All required template keys are set."
else
  echo "Fill the missing keys in ${ENV_FILE} before running production workflows."
fi
