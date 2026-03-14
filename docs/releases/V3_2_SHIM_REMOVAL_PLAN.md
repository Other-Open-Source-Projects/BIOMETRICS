# BIOMETRICS V3.2 Shim Removal Plan

Target removal date: 2026-04-22

## Objective
Remove `cmd/biometrics` compatibility shim after V3.1 GA stabilization window.

## Scope
- Remove shim entrypoint from active build targets.
- Keep V3 canonical runtime as `cmd/controlplane` only.
- Update docs and migration guides to remove shim instructions.

## Milestones
1. 2026-03-26 to 2026-04-05:
   - telemetry/usage check for shim invocation paths.
2. 2026-04-06 to 2026-04-12:
   - issue warnings in docs and release notes.
3. 2026-04-13 to 2026-04-19:
   - remove shim references from CI/build scripts.
4. 2026-04-22:
   - remove `cmd/biometrics` package and finalize V3.2 release notes.

## Exit criteria
1. No active scripts rely on `cmd/biometrics`.
2. CI no longer builds/tests shim path.
3. Migration docs provide exact replacement commands.
