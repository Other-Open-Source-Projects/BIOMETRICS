# erotic-models SSOT

This file is the single source of truth for continuing work on this repo/thread if context is lost.

## Current Context (Resume Here)
- Repo root: /Users/jeremy/dev/BIOMETRICS
- Branch: main (dirty working tree)
- Last merged PR: https://github.com/Delqhi/BIOMETRICS/pull/16 (merge commit: 8a4cc81)
- Goal: make GA-closure soak/rehearsal runs reliable locally (controlplane + opencode execution + soak harness).

## Current Issue
- Release rehearsal/soak runs previously had very low success rate (example: `logs/soak/soak-summary-20260226T054141Z.json` shows 1/59 completed).
- Recent local runs showed:
  - codex provider is unavailable when not logged in (should fall back to gemini/nim)
  - actor timeouts did not cancel underlying agent execution cleanly
  - opencode adapter enforced a hard 180s timeout even when scheduler allows 600s for coder/fixer

## Changes In Working Tree (Not Committed)
- `biometrics-cli/internal/contracts/types.go`
  - add `AgentEnvelope.Ctx context.Context` (json ignored) to carry a per-task context
- `biometrics-cli/internal/runtime/actor/system.go`
  - create per-send `taskCtx` with timeout; set `env.Ctx`; pass it into handler; remove `time.After`
- `biometrics-cli/internal/executor/opencode/adapter.go`
  - honor existing context deadline; only apply 10m default when caller provides none
- `scripts/run-soak.sh`
  - default `GOAL_PREFIX` is now a no-op instruction: `soak noop (reply ok only / no file edits / no commands / no internet)`

- `scripts/visual_truth/vt`
  - Visual Truth step runner: records desktop via `ffmpeg -f avfoundation` into `/tmp/automation_logs/<session>/<step>/...`
  - Enforces "no recording, no command": refuses to execute step unless mp4 file is created and growing
  - `vt doctor` prints device list and runs a short probe capture; currently fails until macOS Screen Recording permission is granted

- `scripts/visual_truth/vt_validate.py`
  - Optional NVIDIA NIM video validation (OpenAI-compatible `POST /v1/chat/completions` with `video_url`)
  - Hard-gated: requires `VISUAL_TRUTH_ALLOW_UPLOAD=1` plus either `VISUAL_TRUTH_UPLOAD_CMD` or `VISUAL_TRUTH_ALLOW_BASE64=1`
  - Accepts `VALIDATED` or `STATUS: VALIDATED` (and ERROR variants)

- `scripts/visual_truth/visual_truth.py`
  - Python context-manager template: `with VideoRecorder(task="..."):` for micro scripts

- `scripts/release/run-ga-closure-program.sh`
  - When `VISUAL_TRUTH=1`, each `run_step` is executed via `scripts/visual_truth/vt step --name ga-closure:<step>`
  - Auto-sets `VISUAL_TRUTH_SESSION=ga-closure-<timestamp>` if not provided
  - Runs `scripts/visual_truth/vt doctor` preflight when `VISUAL_TRUTH=1`

- `scripts/release/run-ga-closure-visual-truth.sh`
  - Strict wrapper that enforces Visual Truth + NIM validation (fails fast if permissions/keys/upload mode missing)

## Local Verification (2026-03-14)
- `cd biometrics-cli && go test ./...` PASS
- `./scripts/release/check-gates.sh` PASS
- controlplane boot + API smoke: `./scripts/release/runtime-surface-smoke.sh` PASS (needs controlplane running)
- API smoke runs:
  - model_preference=gemini run completed; coder output `ok` (run_id: d0880332-7fe1-4862-9ebe-3874d3fafa76)
  - default routing with 3 work packages (noop goal prefix) completed (run_id: c30b313a-dab4-44b2-bb18-bfb1b374a166)

## Running Controlplane (Local)
- build: `cd biometrics-cli && go build -o ../bin/controlplane ./cmd/controlplane`
- start: `PORT=59013 BIOMETRICS_BIND_ADDR=127.0.0.1 ./bin/controlplane`
- pid file: `logs/release/controlplane.pid`
- stop: `kill $(cat logs/release/controlplane.pid)`

## Next Minimal Actions
1) Commit these 4 files on a new branch and open a PR to main.
2) After merge, rerun a short soak/rehearsal to confirm success rate is stable.
