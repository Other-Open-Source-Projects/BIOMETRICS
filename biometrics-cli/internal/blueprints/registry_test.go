package blueprints

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewRegistryRejectsInvalidTemplate(t *testing.T) {
	workspace := t.TempDir()
	mustWrite(t, filepath.Join(workspace, "templates/blueprints/core/BLUEPRINT.md"), "# BLUEPRINT\n")
	mustWrite(t, filepath.Join(workspace, "templates/blueprints/core/AGENTS.md"), "# AGENTS\n## Scope\n## Runtime Rules\n## Planning Rules\n## Execution Rules\n## Quality Gates\n## Security Rules\n## Documentation Rules\n")
	mustWrite(t, filepath.Join(workspace, "templates/blueprints/modules/engine.md"), "## Module: Engine Backend\n")
	mustWrite(t, filepath.Join(workspace, "templates/blueprints/catalog.json"), `{
  "version": "1.0.0",
  "source": {"repo":"x","commit":"y"},
  "profiles": [
    {
      "id": "broken",
      "name": "Broken",
      "version": "1",
      "description": "broken",
      "core": {
        "blueprint_template": "templates/blueprints/core/BLUEPRINT.md",
        "agents_template": "templates/blueprints/core/AGENTS.md"
      },
      "modules": [
        {"id":"engine","name":"Engine","description":"x","template":"templates/blueprints/modules/engine.md"}
      ]
    }
  ]
}`)

	if _, err := NewRegistry(workspace, ""); err == nil {
		t.Fatal("expected validation error for missing required blueprint sections")
	}
}

func TestRegistryListProfiles(t *testing.T) {
	workspace := t.TempDir()
	writeValidCatalogFixture(t, workspace)

	registry, err := NewRegistry(workspace, "")
	if err != nil {
		t.Fatalf("new registry: %v", err)
	}

	profiles := registry.ListProfiles()
	if len(profiles) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(profiles))
	}
	if profiles[0].ID != "universal-2026" {
		t.Fatalf("unexpected profile id: %s", profiles[0].ID)
	}
	if len(profiles[0].Modules) != 4 {
		t.Fatalf("expected 4 modules, got %d", len(profiles[0].Modules))
	}
}

func writeValidCatalogFixture(t *testing.T, workspace string) {
	t.Helper()
	mustWrite(t, filepath.Join(workspace, "templates/blueprints/core/BLUEPRINT.md"), `# BLUEPRINT
## 1. Strategy
## 2. Architecture
## 3. API and Integrations
## 4. Security and Compliance
## 5. CI and Deployment
## 6. Testing Strategy
## 7. Operations
## 8. Maintenance
`)
	mustWrite(t, filepath.Join(workspace, "templates/blueprints/core/AGENTS.md"), `# AGENTS
## Scope
## Runtime Rules
## Planning Rules
## Execution Rules
## Quality Gates
## Security Rules
## Documentation Rules
`)
	mustWrite(t, filepath.Join(workspace, "templates/blueprints/modules/engine.md"), "## Module: Engine Backend\n")
	mustWrite(t, filepath.Join(workspace, "templates/blueprints/modules/webapp.md"), "## Module: Web App SaaS\n")
	mustWrite(t, filepath.Join(workspace, "templates/blueprints/modules/website.md"), "## Module: Website\n")
	mustWrite(t, filepath.Join(workspace, "templates/blueprints/modules/ecommerce.md"), "## Module: Ecommerce\n")
	mustWrite(t, filepath.Join(workspace, "templates/blueprints/catalog.json"), `{
  "version": "1.0.0",
  "source": {"repo":"x","commit":"y"},
  "profiles": [
    {
      "id": "universal-2026",
      "name": "Universal",
      "version": "2026.02.1",
      "description": "curated",
      "core": {
        "blueprint_template": "templates/blueprints/core/BLUEPRINT.md",
        "agents_template": "templates/blueprints/core/AGENTS.md"
      },
      "modules": [
        {"id":"engine","name":"Engine","description":"x","template":"templates/blueprints/modules/engine.md"},
        {"id":"webapp","name":"WebApp","description":"x","template":"templates/blueprints/modules/webapp.md"},
        {"id":"website","name":"Website","description":"x","template":"templates/blueprints/modules/website.md"},
        {"id":"ecommerce","name":"Ecommerce","description":"x","template":"templates/blueprints/modules/ecommerce.md"}
      ]
    }
  ]
}`)
}

func mustWrite(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
