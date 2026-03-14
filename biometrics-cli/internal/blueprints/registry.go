package blueprints

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const defaultCatalogPath = "templates/blueprints/catalog.json"

type Registry struct {
	workspace   string
	catalogPath string
	catalog     Catalog
}

func NewRegistry(workspace string, catalogPath string) (*Registry, error) {
	if strings.TrimSpace(workspace) == "" {
		return nil, fmt.Errorf("workspace is required")
	}
	if strings.TrimSpace(catalogPath) == "" {
		catalogPath = defaultCatalogPath
	}

	r := &Registry{workspace: workspace, catalogPath: catalogPath}
	if err := r.load(); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *Registry) CatalogPath() string {
	return r.catalogPath
}

func (r *Registry) Catalog() Catalog {
	return r.catalog
}

func (r *Registry) ListProfiles() []ProfileSummary {
	out := make([]ProfileSummary, 0, len(r.catalog.Profiles))
	for _, profile := range r.catalog.Profiles {
		modules := make([]ModuleSummary, 0, len(profile.Modules))
		for _, module := range profile.Modules {
			modules = append(modules, ModuleSummary{
				ID:          module.ID,
				Name:        module.Name,
				Description: module.Description,
			})
		}
		out = append(out, ProfileSummary{
			ID:          profile.ID,
			Name:        profile.Name,
			Version:     profile.Version,
			Description: profile.Description,
			Modules:     modules,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})
	return out
}

func (r *Registry) GetProfile(id string) (ProfileSpec, error) {
	for _, profile := range r.catalog.Profiles {
		if profile.ID == id {
			return profile, nil
		}
	}
	return ProfileSpec{}, fmt.Errorf("blueprint profile %q not found", id)
}

func (r *Registry) DefaultProfile() (ProfileSpec, error) {
	if len(r.catalog.Profiles) == 0 {
		return ProfileSpec{}, fmt.Errorf("catalog has no profiles")
	}
	profiles := make([]ProfileSpec, len(r.catalog.Profiles))
	copy(profiles, r.catalog.Profiles)
	sort.Slice(profiles, func(i, j int) bool {
		return profiles[i].ID < profiles[j].ID
	})
	return profiles[0], nil
}

func (r *Registry) ResolveTemplatePath(rel string) (string, error) {
	if strings.TrimSpace(rel) == "" {
		return "", fmt.Errorf("template path is empty")
	}
	cleaned := filepath.Clean(rel)
	abs := filepath.Join(r.workspace, cleaned)

	workspaceClean := filepath.Clean(r.workspace)
	absClean := filepath.Clean(abs)
	if !strings.HasPrefix(absClean, workspaceClean+string(filepath.Separator)) && absClean != workspaceClean {
		return "", fmt.Errorf("template path escapes workspace: %s", rel)
	}
	return abs, nil
}

func (r *Registry) load() error {
	catalogAbs, err := r.ResolveTemplatePath(r.catalogPath)
	if err != nil {
		return err
	}

	raw, err := os.ReadFile(catalogAbs)
	if err != nil {
		return fmt.Errorf("read blueprint catalog: %w", err)
	}

	var catalog Catalog
	if err := json.Unmarshal(raw, &catalog); err != nil {
		return fmt.Errorf("parse blueprint catalog: %w", err)
	}
	if err := r.validate(catalog); err != nil {
		return err
	}
	r.catalog = catalog
	return nil
}

func (r *Registry) validate(catalog Catalog) error {
	if strings.TrimSpace(catalog.Version) == "" {
		return fmt.Errorf("catalog version is required")
	}
	if len(catalog.Profiles) == 0 {
		return fmt.Errorf("catalog requires at least one profile")
	}

	profileIDs := make(map[string]struct{})
	for _, profile := range catalog.Profiles {
		if strings.TrimSpace(profile.ID) == "" {
			return fmt.Errorf("profile id is required")
		}
		if _, exists := profileIDs[profile.ID]; exists {
			return fmt.Errorf("duplicate profile id: %s", profile.ID)
		}
		profileIDs[profile.ID] = struct{}{}

		if strings.TrimSpace(profile.Core.BlueprintTemplate) == "" || strings.TrimSpace(profile.Core.AgentsTemplate) == "" {
			return fmt.Errorf("profile %s requires both core templates", profile.ID)
		}

		blueprintTemplate, err := r.ResolveTemplatePath(profile.Core.BlueprintTemplate)
		if err != nil {
			return fmt.Errorf("profile %s blueprint template: %w", profile.ID, err)
		}
		agentsTemplate, err := r.ResolveTemplatePath(profile.Core.AgentsTemplate)
		if err != nil {
			return fmt.Errorf("profile %s agents template: %w", profile.ID, err)
		}
		if err := validateRequiredSections(blueprintTemplate, []string{
			"## 1. Strategy",
			"## 2. Architecture",
			"## 3. API and Integrations",
			"## 4. Security and Compliance",
			"## 5. CI and Deployment",
			"## 6. Testing Strategy",
			"## 7. Operations",
			"## 8. Maintenance",
		}); err != nil {
			return fmt.Errorf("profile %s blueprint template validation: %w", profile.ID, err)
		}
		if err := validateRequiredSections(agentsTemplate, []string{
			"## Scope",
			"## Runtime Rules",
			"## Planning Rules",
			"## Execution Rules",
			"## Quality Gates",
			"## Security Rules",
			"## Documentation Rules",
		}); err != nil {
			return fmt.Errorf("profile %s agents template validation: %w", profile.ID, err)
		}

		moduleIDs := make(map[string]struct{})
		for _, module := range profile.Modules {
			if strings.TrimSpace(module.ID) == "" {
				return fmt.Errorf("profile %s has module with empty id", profile.ID)
			}
			if _, exists := moduleIDs[module.ID]; exists {
				return fmt.Errorf("profile %s has duplicate module id %s", profile.ID, module.ID)
			}
			moduleIDs[module.ID] = struct{}{}

			if strings.TrimSpace(module.Template) == "" {
				return fmt.Errorf("profile %s module %s missing template", profile.ID, module.ID)
			}
			moduleTemplate, err := r.ResolveTemplatePath(module.Template)
			if err != nil {
				return fmt.Errorf("profile %s module %s template: %w", profile.ID, module.ID, err)
			}
			if _, err := os.Stat(moduleTemplate); err != nil {
				return fmt.Errorf("profile %s module %s template not found: %w", profile.ID, module.ID, err)
			}
		}
	}

	return nil
}

func validateRequiredSections(path string, required []string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := string(raw)
	for _, heading := range required {
		if !strings.Contains(content, heading) {
			return fmt.Errorf("missing required section %q in %s", heading, path)
		}
	}
	return nil
}
