# Codex Upstream Mirror Governance

This directory is reserved for Codex upstream tracking metadata and mirror policy.

Current implementation:

- `upstream.lock.json` stores the tracked `openai/codex` release and lag policy.
- Lock policy tracks the stable tag stream with prefix `rust-v` (pre-releases excluded).
- CI workflow `.github/workflows/codex-upstream-watch.yml` validates lag daily.

Policy:

1. Do not place BIOMETRICS custom code in this directory.
2. Do not edit upstream snapshots directly; update via sync PRs only.
3. Keep tracked release lag at or below `max_release_lag_days` (default: 7).

Manual update command:

```bash
python3 scripts/upstream/check_codex_upstream.py --update-lock
```

After running the update command, open a PR with the updated lockfile and corresponding sync notes.
