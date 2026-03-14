# Codex-Based Release and Security Governance

Last updated: 2026-02-26

## Objective

Ship BIOMETRICS features on top of Codex fast, but with enterprise-grade safety and predictable upgrades.

## Implemented baseline (2026-02-26)

- Daily upstream lag enforcement: `.github/workflows/codex-upstream-watch.yml`
- Tracked upstream release lock: `third_party/codex-upstream/upstream.lock.json`
- Supply-chain pipeline (Scorecard + CycloneDX SBOM + provenance attestation): `.github/workflows/supply-chain.yml`

## Mandatory governance artifacts

For this repository, maintain and enforce:

1. `CODEOWNERS` for critical paths (runtime, API contracts, CI, security).
2. Branch protection/rulesets for `main`:
   - required PR reviews,
   - required status checks,
   - no force-push,
   - no direct pushes.
3. `SECURITY.md` with vuln reporting SLA and escalation path.
4. Release notes per upstream-sync and per BIOMETRICS release.

## Supply-chain baseline (free and strong)

1. OpenSSF Scorecard check in CI.
2. SLSA provenance generation for release artifacts.
3. SBOM generation and storage per release.
4. Dependency update policy:
   - lockfile required,
   - signed/tagged release inputs only for critical paths,
   - staged rollout for runtime-impacting updates.

## Release gates (Codex + BIOMETRICS)

Every upstream or BIOMETRICS release candidate must pass:

1. Upstream sync integrity gate:
   - clean merge history,
   - no undocumented core patch.
2. API compatibility gate:
   - contract tests pass,
   - additive-only schema changes.
3. Runtime reliability gate:
   - run/pause/resume/cancel unchanged,
   - regression gate pass.
4. Security gate:
   - secret scan pass,
   - Scorecard threshold pass,
   - provenance generated.

## Operating cadence

1. Daily: upstream monitoring + security advisories.
2. Weekly: planned upstream sync PR.
3. Monthly: hardening release and governance audit.
4. Immediate: out-of-band hotfix sync for critical CVEs.

## Incident and rollback policy

1. If an upstream sync breaks core flows, rollback by reverting only the sync PR.
2. Keep BIOMETRICS overlays isolated so rollback does not require broad cherry-picking.
3. Publish postmortem within 48h for production-impact incidents.

## Definition of done for "Codex-core migration complete"

1. BIOMETRICS can track Codex upstream within <= 7 days.
2. All BIOMETRICS differentiators run as additive layers.
3. No direct patch debt in upstream mirror paths.
4. CI proves compatibility and security on every release branch.

## Sources

- GitHub docs: configuring rulesets and branch protections: <https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/about-rulesets>
- GitHub docs: about CODEOWNERS: <https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/customizing-your-repository/about-code-owners>
- OpenSSF Scorecard: <https://github.com/ossf/scorecard>
- SLSA framework and provenance model: <https://slsa.dev/>
