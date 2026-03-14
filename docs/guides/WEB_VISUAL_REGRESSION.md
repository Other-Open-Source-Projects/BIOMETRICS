# WEB V3 Visual Regression Guard

## Objective
Keep BIOMETRICS web-v3 visually stable at release quality while allowing intentional design improvements.

## Current Guard Model
1. Baseline screenshots are curated on macOS (`darwin`) under:
   - `biometrics-cli/web-v3/tests/e2e/visual.spec.ts-snapshots/shell-baseline-darwin.png`
   - `biometrics-cli/web-v3/tests/e2e/visual.spec.ts-snapshots/graph-fallback-baseline-darwin.png`
2. Cross-platform style safety is enforced by token assertions in `visual.spec.ts` (`accent`, `text`, heading metrics).
3. CI always runs the visual suite; screenshot assertions are platform-gated, token guard always runs.

## Commands

Run all E2E checks:

```bash
cd biometrics-cli/web-v3
npm run test:e2e
```

Run visual guard only:

```bash
cd biometrics-cli/web-v3
npm run test:visual
```

Update macOS visual baselines after approved UI changes:

```bash
cd biometrics-cli/web-v3
npm run test:visual:update
```

## CI Integration
`/.github/workflows/ci.yml` includes:
1. `web-v3-e2e` for full interaction coverage.
2. `web-v3-visual-guard` for dedicated visual checks.
3. Artifact uploads (`test-results`, `playwright-report`) for audit and failure debugging.

## Baseline Governance
1. Never refresh snapshots automatically in CI.
2. Snapshot updates require explicit UI-review sign-off.
3. Every baseline change must be logged in `docs/releases/EXECUTION_LOG.md` with rationale.

## Forward Plan (Linux Baseline Track)
1. Add Linux baseline images (`*-linux.png`) after rendering parity approval.
2. Remove platform skip once Linux and macOS baselines are both trusted.
3. Keep token guards as a second line of defense even after full dual-platform snapshots.
