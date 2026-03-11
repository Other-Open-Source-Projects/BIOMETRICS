# Skills Source Map

- Upstream project: [openai/skills](https://github.com/openai/skills)
- Integration model: curated parity for observable behavior (not proprietary reverse engineering)
- Mirrored system bundles:
  - `third_party/openai-skills/.system/skill-creator`
  - `third_party/openai-skills/.system/skill-installer`

## Pinned Inputs

- Codex CLI skills docs:
  - https://developers.openai.com/codex/cli/skills
  - https://developers.openai.com/codex/cli/agents-md
  - https://developers.openai.com/codex/cli/config
- System skill workflows mirrored from local Codex system skills into repository `third_party/openai-skills/.system/*`.

## Reused Concepts

- Skill discovery via `SKILL.md` frontmatter (`name`, `description`) and progressive disclosure.
- System skill operations (`skill-creator`, `skill-installer`) exposed through BIOMETRICS skill operations wrappers.
- Deterministic selection modes (`auto`, `explicit`, `off`) and strict non-breaking run-level integration.

## Exclusions

- No token/session import from external auth callbacks.
- No proprietary internal implementation details beyond published docs + mirrored open assets.
