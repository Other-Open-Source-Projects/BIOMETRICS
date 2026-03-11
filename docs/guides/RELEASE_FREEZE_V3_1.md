# BIOMETRICS V3.1 Release Freeze Policy

Effective date: 2026-03-23  
GA target: 2026-03-25

## Policy
After Gate B starts, the branch is in freeze mode. Only P0/P1 fixes are allowed.

## Allowed During Freeze
- P0/P1 correctness and security fixes
- Release-blocking regressions introduced after Gate B
- Documentation corrections required for safe operation

## Not Allowed During Freeze
- New feature work
- Refactors without direct release risk reduction
- Contract or API behavior changes that are not strictly bug fixes

## Required Workflow
1. File issue with severity (`P0` or `P1`) and reproducible evidence.
2. Link fix PR to issue and label `release-freeze`.
3. Attach test evidence showing regression and fix.
4. Record each approved change in `docs/releases/EXECUTION_LOG.md`.
   - Shortcut: `./scripts/release/append-execution-log.sh "<line-1>" "<line-2>"`
5. Require explicit release-owner approval before merge.
6. Re-run `scripts/release/check-gates.sh` after merge.

## Sign-off Roles
- Engineering owner
- Release owner
- Operator sign-off for runbook-impacting changes

## Exit Criteria
- All freeze-approved fixes merged and validated
- Gate checklist still green
- Release notes and runbook updated
