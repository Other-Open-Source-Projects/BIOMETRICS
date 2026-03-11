# Blueprint Catalog (Authoritative)

This catalog defines the supported BIOMETRICS V3 blueprint profiles and modules.

## Authoritative Docs Set

The following documents are authoritative for blueprint behavior:

1. `docs/blueprints/CATALOG.md`
2. `docs/blueprints/SOURCE_MAP.md`
3. `templates/blueprints/catalog.json`
4. `templates/blueprints/core/BLUEPRINT.md`
5. `templates/blueprints/core/AGENTS.md`
6. `templates/blueprints/modules/*.md`

Any historical or narrative-only files are non-authoritative.

## Supported Profile(s)

### `universal-2026`

- Version: `2026.02.1`
- Source: curated from CODE-BLUEPRINTS concepts
- Purpose: baseline machine-readable project blueprint

Core artifacts generated/managed:

- `BLUEPRINT.md`
- `AGENTS.md`

Optional modules:

- `engine`
- `webapp`
- `website`
- `ecommerce`

## Runtime Integration Rules

1. Profile selection is optional on run creation.
2. Bootstrap is optional and explicit (`bootstrap=true`) unless caller policy sets a default.
3. Blueprint file updates are idempotent and marker-based.
4. Module content is managed under BIOMETRICS markers in `BLUEPRINT.md`.

## Event Contract for Blueprint Operations

The runtime emits the following event types:

- `blueprint.selected`
- `blueprint.bootstrap.started`
- `blueprint.module.applied`
- `blueprint.module.skipped`
- `blueprint.bootstrap.completed`

## Compatibility

- API v1 remains backward compatible.
- Existing runs without blueprint settings are unchanged.
