#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

printf '[deprecated] bootstrap.sh is deprecated. Use ./biometrics-onboard instead.\n' >&2
exec "${ROOT_DIR}/biometrics-onboard" "$@"
