package blueprints

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Applier struct {
	workspace string
	registry  *Registry
}

func NewApplier(workspace string, registry *Registry) *Applier {
	return &Applier{
		workspace: workspace,
		registry:  registry,
	}
}

func (a *Applier) Apply(opts ApplyOptions) (ApplyResult, error) {
	if a.registry == nil {
		return ApplyResult{}, fmt.Errorf("blueprint registry is not configured")
	}

	profile, err := a.resolveProfile(opts.ProfileID)
	if err != nil {
		return ApplyResult{}, err
	}

	projectPath, projectName, err := a.resolveProjectPath(opts.ProjectID)
	if err != nil {
		return ApplyResult{}, err
	}

	blueprintTemplate, err := a.registry.ResolveTemplatePath(profile.Core.BlueprintTemplate)
	if err != nil {
		return ApplyResult{}, err
	}
	agentsTemplate, err := a.registry.ResolveTemplatePath(profile.Core.AgentsTemplate)
	if err != nil {
		return ApplyResult{}, err
	}

	now := time.Now().UTC()
	renderVars := map[string]string{
		"PROJECT_NAME": projectName,
		"PROFILE_ID":   profile.ID,
		"GENERATED_AT": now.Format(time.RFC3339),
	}

	result := ApplyResult{
		ProjectPath:   projectPath,
		ProfileID:     profile.ID,
		BlueprintPath: filepath.Join(projectPath, "BLUEPRINT.md"),
		AgentsPath:    filepath.Join(projectPath, "AGENTS.md"),
		GeneratedAt:   now,
	}

	blueprintCoreRaw, err := os.ReadFile(blueprintTemplate)
	if err != nil {
		return ApplyResult{}, fmt.Errorf("read blueprint template: %w", err)
	}
	agentsCoreRaw, err := os.ReadFile(agentsTemplate)
	if err != nil {
		return ApplyResult{}, fmt.Errorf("read agents template: %w", err)
	}

	blueprintCore := renderTemplate(string(blueprintCoreRaw), renderVars)
	agentsCore := renderTemplate(string(agentsCoreRaw), renderVars)

	coreChanged, err := ensureFileWithContent(result.BlueprintPath, blueprintCore)
	if err != nil {
		return ApplyResult{}, err
	}
	if coreChanged {
		result.ChangedFiles = append(result.ChangedFiles, result.BlueprintPath)
	}

	agentsChanged, err := ensureFileWithContent(result.AgentsPath, agentsCore)
	if err != nil {
		return ApplyResult{}, err
	}
	if agentsChanged {
		result.ChangedFiles = append(result.ChangedFiles, result.AgentsPath)
	}

	metadata := fmt.Sprintf("profile_id: %s\nproject: %s\n", profile.ID, projectName)
	if changed, err := upsertMarkerBlock(result.BlueprintPath, "BOOTSTRAP:METADATA", metadata); err != nil {
		return ApplyResult{}, err
	} else if changed {
		result.ChangedFiles = append(result.ChangedFiles, result.BlueprintPath)
	}
	if changed, err := upsertMarkerBlock(result.AgentsPath, "BOOTSTRAP:METADATA", metadata); err != nil {
		return ApplyResult{}, err
	} else if changed {
		result.ChangedFiles = append(result.ChangedFiles, result.AgentsPath)
	}

	selectedModules, err := resolveModules(profile.Modules, opts.ModuleIDs)
	if err != nil {
		return ApplyResult{}, err
	}

	for _, module := range selectedModules {
		modulePath, err := a.registry.ResolveTemplatePath(module.Template)
		if err != nil {
			return ApplyResult{}, err
		}
		moduleRaw, err := os.ReadFile(modulePath)
		if err != nil {
			return ApplyResult{}, fmt.Errorf("read module template %s: %w", module.ID, err)
		}
		renderedModule := renderTemplate(string(moduleRaw), renderVars)
		moduleBlock := fmt.Sprintf("## Bootstrap Module: %s\n\n%s", module.Name, strings.TrimSpace(renderedModule))
		marker := "MODULE:" + module.ID
		changed, err := upsertMarkerBlock(result.BlueprintPath, marker, moduleBlock)
		if err != nil {
			return ApplyResult{}, err
		}
		if changed {
			result.AppliedModules = append(result.AppliedModules, module.ID)
			result.ChangedFiles = append(result.ChangedFiles, result.BlueprintPath)
		} else {
			result.SkippedModules = append(result.SkippedModules, module.ID)
		}
	}

	sort.Strings(result.AppliedModules)
	sort.Strings(result.SkippedModules)
	result.ChangedFiles = dedupeStrings(result.ChangedFiles)
	return result, nil
}

func (a *Applier) resolveProfile(profileID string) (ProfileSpec, error) {
	if strings.TrimSpace(profileID) != "" {
		return a.registry.GetProfile(profileID)
	}
	return a.registry.DefaultProfile()
}

func (a *Applier) resolveProjectPath(projectID string) (string, string, error) {
	if strings.TrimSpace(projectID) == "" {
		base := filepath.Clean(a.workspace)
		return base, filepath.Base(base), nil
	}

	cleanID := filepath.Clean(projectID)
	if strings.Contains(cleanID, "..") || filepath.IsAbs(cleanID) {
		return "", "", fmt.Errorf("invalid project id: %s", projectID)
	}

	candidate := filepath.Join(a.workspace, cleanID)
	workspaceClean := filepath.Clean(a.workspace)
	candidateClean := filepath.Clean(candidate)
	if !strings.HasPrefix(candidateClean, workspaceClean+string(filepath.Separator)) && candidateClean != workspaceClean {
		return "", "", fmt.Errorf("project path escapes workspace")
	}

	info, err := os.Stat(candidateClean)
	if err == nil && info.IsDir() {
		return candidateClean, cleanID, nil
	}

	return candidateClean, cleanID, nil
}

func resolveModules(all []ModuleSpec, selected []string) ([]ModuleSpec, error) {
	if len(selected) == 0 {
		return []ModuleSpec{}, nil
	}
	byID := make(map[string]ModuleSpec, len(all))
	for _, module := range all {
		byID[module.ID] = module
	}

	result := make([]ModuleSpec, 0, len(selected))
	seen := make(map[string]struct{})
	for _, id := range selected {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		module, ok := byID[id]
		if !ok {
			return nil, fmt.Errorf("unknown blueprint module: %s", id)
		}
		seen[id] = struct{}{}
		result = append(result, module)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})
	return result, nil
}

func ensureFileWithContent(path string, content string) (bool, error) {
	if strings.TrimSpace(content) == "" {
		return false, fmt.Errorf("cannot write empty content to %s", path)
	}
	existing, err := os.ReadFile(path)
	if err == nil {
		if string(existing) == content {
			return false, nil
		}
		return false, nil
	}
	if !os.IsNotExist(err) {
		return false, err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return false, err
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return false, err
	}
	return true, nil
}

func upsertMarkerBlock(path string, marker string, blockContent string) (bool, error) {
	start := fmt.Sprintf("<!-- BIOMETRICS:%s:START -->", marker)
	end := fmt.Sprintf("<!-- BIOMETRICS:%s:END -->", marker)
	section := fmt.Sprintf("%s\n%s\n%s", start, strings.TrimSpace(blockContent), end)

	existingRaw, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	existing := string(existingRaw)
	updated := existing

	startIdx := strings.Index(existing, start)
	endIdx := strings.Index(existing, end)
	if startIdx >= 0 && endIdx >= 0 && endIdx > startIdx {
		endIdx += len(end)
		updated = strings.TrimSpace(existing[:startIdx]) + "\n\n" + section + "\n\n" + strings.TrimSpace(existing[endIdx:])
	} else {
		updated = strings.TrimSpace(existing) + "\n\n" + section + "\n"
	}

	updated = strings.TrimSpace(updated) + "\n"
	if updated == existing {
		return false, nil
	}
	if err := os.WriteFile(path, []byte(updated), 0644); err != nil {
		return false, err
	}
	return true, nil
}

func renderTemplate(template string, vars map[string]string) string {
	out := template
	for key, value := range vars {
		out = strings.ReplaceAll(out, "{{"+key+"}}", value)
	}
	return out
}

func dedupeStrings(values []string) []string {
	if len(values) == 0 {
		return values
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, v := range values {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}
