# OH-MY-OPENCODE Status in BIOMETRICS V3

`oh-my-opencode` is not part of the supported BIOMETRICS V3 runtime surface.

## Current Policy

- BIOMETRICS V3 does not require `oh-my-opencode`.
- Active integration guidance is documented in `docs/OPENCODE.md`.
- V3 control and contracts are defined by:
  - `docs/api/openapi-v3-controlplane.yaml`
  - `docs/specs/contracts/*.json`
  - `docs/guides/MIGRATION_V3.md`

## Migration Guidance

If an environment still has legacy plugin settings:

1. Keep plugin config local only.
2. Remove project-level dependency on plugin-specific files.
3. Ensure runs are started and controlled through `/api/v1/*`.
4. Validate behavior with V3 smoke tests.

## Historical Note

Older plugin-heavy setup notes were intentionally removed from active docs during V3 release hardening.
