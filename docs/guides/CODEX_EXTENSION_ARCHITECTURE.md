# Codex + BIOMETRICS Extension Architecture

Last updated: 2026-02-27

## Goal

Build BIOMETRICS as a production extension of Codex without breaking Codex updateability, while preserving Codex-native UX language and visual continuity.

## Architecture principle

Use layered ownership:

1. Layer 0: Upstream Codex Core
2. Layer 1: Compatibility Facade (adapters)
3. Layer 2: BIOMETRICS Runtime Features
4. Layer 3: BIOMETRICS Product Surface (branding, docs, ops UX)

Each layer may depend on lower layers, never the reverse.

## Layer definitions

### Layer 0: Upstream Codex Core (read-only policy)

- Command execution engine
- Native Codex configuration behavior
- Upstream CLI contracts

Constraint: no direct feature branching in core paths.

### Layer 1: Compatibility Facade

- API and event adapters between Codex outputs and BIOMETRICS control-plane contracts
- Stable schema translators (additive fields only)
- Feature flag bridge (`BIOMETRICS_*`)

Constraint: additive only, no breaking renames/removals.

### Layer 2: BIOMETRICS Runtime

- 24/7 orchestrator logic
- Apex gate optimizer
- Eval/validation pipelines
- Reliability controls (pause/resume/cancel, fallback chains, queue backpressure)

Constraint: all behavior must be callable independently of Codex core internals.

### Layer 3: Product Surface

- Public website and docs
- Operator UI extensions (Codex-native visual grammar, BIOMETRICS operations depth)
- Enterprise runbooks and compliance docs

Constraint: UX can change, interfaces to lower layers stay versioned and additive.

## Extension points to prioritize first

1. `AGENTS.md` policies for project behavior and team-specific instructions.
2. Config-driven behavior before code patches.
3. Tooling adapters (for MCP/server integrations) as separate modules.

## Non-breaking contract rules

1. Existing API routes and fields remain valid.
2. New behavior introduced behind feature flags.
3. New fields are additive and nullable-by-default.
4. Event streams get additive event types only.

## Hard anti-patterns

- Editing upstream files directly for branding.
- Embedding BIOMETRICS business logic into Codex core packages.
- Silent behavior changes without version or flag.
- Treating BIOMETRICS as a standalone Codex replacement app.

## Rollout model

1. Shadow mode first (observe, do not auto-apply).
2. Manual apply with full traceability.
3. Canary tenants/projects.
4. Default rollout after stability SLO is met.

## Sources

- OpenAI Codex announcement (AGENTS.md + local/remote workflow context): <https://openai.com/index/introducing-codex/>
- Codex repository (active codebase and release cadence): <https://github.com/openai/codex>
- Diataxis documentation system (separate tutorial/how-to/reference/explanation): <https://diataxis.fr/>
