# BIOMETRICS Skills (V3.1)

BIOMETRICS implements Codex-compatible observable skill behavior for discovery, selection, install/create operations, and run-time injection.

## Skill Discovery

Discovery roots and precedence:

1. `<workspace>/.codex/skills`
2. Hierarchical `<project-root..cwd>/.agents/skills`
3. `$CODEX_HOME/skills` and `~/.codex/skills`
4. `$CODEX_HOME/skills/.system`
5. `/etc/codex/skills`

Conflict handling:

- Deduplicate by canonical `SKILL.md` path.
- Name precedence is deterministic by source priority:
  - `repo > user > system > admin`

Scan constraints:

- max depth: `6`
- required frontmatter fields in `SKILL.md`: `name`, `description`
- optional metadata: `metadata.short-description`
- optional `agents/openai.yaml`: `interface`, `dependencies`, `policy`

## Run Integration

`POST /api/v1/runs` supports:

- `skills: string[]` (optional)
- `skill_selection_mode: auto|explicit|off` (optional, default `auto`)

Selection behavior:

- `off`: no skills.
- `explicit`: only names in `skills[]`.
- `auto`: explicit mentions (`$skill-name` or name match) plus description-based matching.

Effective skill list is persisted in run snapshot fields (`run.skills`, `run.skill_selection_mode`).

Instruction order in prompt assembly:

1. base task instruction
2. discovered project docs (`AGENTS.override.md` -> `AGENTS.md` + configured fallbacks)
3. rendered skills section
4. context compiler selection

## API Surface

- `GET /api/v1/skills`
- `GET /api/v1/skills/{name}`
- `POST /api/v1/skills/reload`
- `POST /api/v1/skills/install`
- `POST /api/v1/skills/create`
- `POST /api/v1/skills/enable`
- `POST /api/v1/skills/disable`

Readiness additions (`/health/ready`):

- `skills_loaded`
- `skills_errors`
- `skills_system_ready`

Runtime event additions:

- `skills.loaded`
- `skill.selected`
- `skill.invocation.blocked`
- `skill.install.started|succeeded|failed`
- `skill.create.started|succeeded|failed`

## Onboarding

`./biometrics-onboard` includes a dedicated skills step:

- creates workspace skill roots (`.codex/skills`, `.agents/skills`)
- installs mirrored system bundles into `$CODEX_HOME/skills/.system`
- validates system skill presence in `--doctor`

`--doctor` remains non-mutating.

## Operations

System skill scripts are mirrored under:

- `third_party/openai-skills/.system/skill-creator`
- `third_party/openai-skills/.system/skill-installer`

They are invoked through internal wrappers in `internal/skillops`.
