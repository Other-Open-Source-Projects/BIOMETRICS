# SOURCE_MAP: CODE-BLUEPRINTS -> BIOMETRICS V3

This file records curated concept intake from CODE-BLUEPRINTS.

## Source Pin

- Repository: `https://github.com/Delqhi/CODE-BLUEPRINTS.git`
- Commit: `2d562d0f6e8c519574d7ca3b57a153ad0b446596`
- Source date: `2026-02-17`

## Intake Policy

1. No 1:1 file mirroring.
2. Concepts are rewritten in BIOMETRICS-native, machine-readable format.
3. Non-operational rhetoric is excluded.

## Concept Mapping

### UNIVERSAL-BLUEPRINT.md
- Reused:
  - strategy, architecture, API, security, CI/CD, testing, operations structure
- Adapted into:
  - `templates/blueprints/core/BLUEPRINT.md`

### MODULE-ENGINE.md / MODULE-WEBAPP.md / MODULE-WEBSITE.md / MODULE-ECOMMERCE.md
- Reused:
  - module composition idea and section scope
- Adapted into:
  - `templates/blueprints/modules/*.md`

### 01-ai-chat-orchestration-blueprint.md
- Reused:
  - event/streaming pattern semantics and guardrail ideas
- Adapted into:
  - runtime event types and API/UI event handling in BIOMETRICS

### 22-docker-governance.md
- Reused:
  - local-first backup/integrity/restore governance concept
- Adapted into:
  - BIOMETRICS operational recommendations (without strict absolute path mandates)

## Explicitly Excluded

- CEO/Singularity narrative language
- hard 500+ line mandates
- absolute host-specific filesystem path mandates
- environment assumptions not compatible with BIOMETRICS V3 scope
