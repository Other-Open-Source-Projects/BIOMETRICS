# Codex Upstream-First Strategy (BIOMETRICS)

Last updated: 2026-02-26

## Why this exists

BIOMETRICS should stop rebuilding what Codex already solves well and move to an upstream-first model:

- Keep Codex as the primary core.
- Add BIOMETRICS capabilities as additive modules.
- Preserve fast upstream updates with minimal merge pain.

## External facts (confirmed on 2026-02-26)

- `openai/codex` is open source under Apache-2.0 and has an active release train.
- Latest stable release in the tracked stream (`rust-v*`): `rust-v0.105.0`, published on 2026-02-25.
- Codex supports project-level guidance via `AGENTS.md`.
- GitHub fork best practices require explicit `upstream` remote and regular syncing.

## Decision

Adopt a strict "core + overlay" model:

1. Core (`Codex upstream`) stays as close to upstream as possible.
2. BIOMETRICS features live in additive namespaces, wrappers, and services.
3. No hard forks of core behavior unless a security or compliance blocker exists.

## Repository model

Target structure:

- `third_party/codex-upstream/` -> upstream mirror (read-only by policy)
- `third_party/codex-upstream/upstream.lock.json` -> tracked upstream release baseline
- `biometrics-cli/internal/controlplane/` -> BIOMETRICS runtime orchestration
- `biometrics-cli/internal/runtime/orchestrator/` -> 24/7 orchestration features
- `docs/guides/` -> governance + integration runbooks

## Upstream sync policy

Cadence:

- Daily `fetch` of upstream tags/commits.
- Weekly merge window (or same-day for critical security fixes).

Rules:

1. Never edit files under `third_party/codex-upstream/` directly.
2. All custom logic must be outside the upstream mirror.
3. If patching upstream is unavoidable, patch must be:
   - documented in a dedicated patch note,
   - isolated,
   - proposed upstream as a PR.

## Minimal sync playbook

```bash
git remote add codex-upstream https://github.com/openai/codex.git
git fetch codex-upstream --tags
git switch -c codex/upstream-sync-$(date +%Y%m%d)
git merge --no-ff codex-upstream/main
```

Then open an explicit upstream-sync PR into `main`. Never sync by direct push to `main`.

Automated watch gate:

- `.github/workflows/codex-upstream-watch.yml` checks `openai/codex` release lag daily against lockfile policy.

## KPIs (operational)

- Upstream lag: <= 7 days behind `openai/codex/main`.
- Critical security lag: <= 24h from upstream fix.
- Core drift: 0 direct local edits under upstream mirror paths.
- Compatibility: 100% pass on BIOMETRICS smoke + gate suite after each sync.

## Sources

- OpenAI Codex announcement and AGENTS.md guidance: <https://openai.com/index/introducing-codex/>
- Codex repository (license + releases): <https://github.com/openai/codex>
- GitHub docs: configure upstream remote: <https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/working-with-forks/configuring-a-remote-repository-for-a-fork>
- GitHub docs: syncing a fork: <https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/working-with-forks/syncing-a-fork>
