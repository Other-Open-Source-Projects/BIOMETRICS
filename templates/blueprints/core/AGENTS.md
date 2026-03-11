# AGENTS

Project: {{PROJECT_NAME}}
Profile: {{PROFILE_ID}}
Generated At: {{GENERATED_AT}}

## Scope
- This file defines local project agent behavior.
- Repository-wide global rules remain authoritative when stricter.

## Runtime Rules
1. Prefer deterministic, typed contracts over implicit behavior.
2. Keep changes minimal and idempotent.
3. Validate path safety before file reads/writes.
4. Emit structured events for user-visible progress.

## Planning Rules
1. Convert goals into explicit tasks with completion criteria.
2. Surface assumptions in output and tests.
3. Record blocking errors with actionable remediation.

## Execution Rules
1. Avoid destructive operations unless explicitly requested.
2. Preserve unrelated workspace changes.
3. Use project-specific templates and contracts as source of truth.

## Quality Gates
1. Build and test must pass before completion.
2. Contract and API changes require schema updates.
3. Runtime behavior changes require event and state verification.

## Security Rules
1. Never commit secrets or local `.env` values.
2. Restrict file operations to the workspace boundary.
3. Redact sensitive values in logs and event payloads.

## Documentation Rules
1. Keep docs machine-readable and concise.
2. Keep migration notes for operator-impacting changes.
3. Mark historical docs as non-authoritative.

<!-- BIOMETRICS:POLICY:START -->
policy_version: 1
autonomous_default: true
manual_controls: [pause, resume, cancel]
<!-- BIOMETRICS:POLICY:END -->
