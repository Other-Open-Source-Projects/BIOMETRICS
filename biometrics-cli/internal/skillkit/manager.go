package skillkit

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	toml "github.com/pelletier/go-toml/v2"
)

type Operations interface {
	List(ctx context.Context, experimental bool) (OperationResult, error)
	Install(ctx context.Context, req InstallRequest) (OperationResult, error)
	Create(ctx context.Context, req CreateRequest) (OperationResult, error)
	Validate(ctx context.Context, skillPath string) (OperationResult, error)
}

type ManagerOptions struct {
	Workspace           string
	CWD                 string
	CodexHome           string
	ProjectRootMarkers  []string
	ProjectDocFallbacks []string
	ProjectDocMaxBytes  int
}

type Manager struct {
	workspace          string
	cwd                string
	codexHome          string
	projectRootMarkers []string
	projectDocFallback []string
	projectDocMaxBytes int

	ops Operations

	mu      sync.RWMutex
	outcome LoadOutcome
}

type tomlSkillEntry struct {
	Path    string `toml:"path"`
	Enabled *bool  `toml:"enabled,omitempty"`
}

type tomlSkillConfig struct {
	Skills struct {
		Config []tomlSkillEntry `toml:"config"`
	} `toml:"skills"`
}

func NewManager(opts ManagerOptions) (*Manager, error) {
	workspace := strings.TrimSpace(opts.Workspace)
	cwd := strings.TrimSpace(opts.CWD)
	if workspace == "" && cwd == "" {
		return nil, fmt.Errorf("workspace or cwd is required")
	}
	if workspace == "" {
		workspace = cwd
	}
	if cwd == "" {
		cwd = workspace
	}

	m := &Manager{
		workspace:          workspace,
		cwd:                cwd,
		codexHome:          strings.TrimSpace(opts.CodexHome),
		projectRootMarkers: append([]string{}, opts.ProjectRootMarkers...),
		projectDocFallback: append([]string{}, opts.ProjectDocFallbacks...),
		projectDocMaxBytes: opts.ProjectDocMaxBytes,
		outcome: LoadOutcome{
			Skills:        []SkillMetadata{},
			Errors:        []SkillError{},
			DisabledPaths: map[string]struct{}{},
		},
	}
	if len(m.projectRootMarkers) == 0 {
		m.projectRootMarkers = []string{".git"}
	}
	if m.projectDocMaxBytes <= 0 {
		m.projectDocMaxBytes = defaultProjectDocMaxBytes
	}
	if m.codexHome == "" {
		m.codexHome = defaultCodexHome()
	}

	if _, err := m.Reload(); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *Manager) SetOperations(ops Operations) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ops = ops
}

func (m *Manager) Reload() (LoadOutcome, error) {
	disabled, configPath, cfgErr := m.readDisabledConfig()
	out := loadSkills(loadConfig{
		Workspace:          m.workspace,
		CWD:                m.cwd,
		CodexHome:          m.codexHome,
		ProjectRootMarkers: append([]string{}, m.projectRootMarkers...),
		DisabledPaths:      disabled,
	})
	out.ConfigFilePath = configPath
	sortSkillsStable(out.Skills)
	if cfgErr != nil {
		out.Errors = append(out.Errors, SkillError{Path: configPath, Message: cfgErr.Error()})
	}

	m.mu.Lock()
	m.outcome = out
	m.mu.Unlock()

	if cfgErr != nil {
		return out, cfgErr
	}
	return out, nil
}

func (m *Manager) Stats() (loaded int, errs int, ready bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.outcome.EnabledSkills()), len(m.outcome.Errors), len(m.outcome.Errors) == 0
}

func (m *Manager) List() ([]SkillMetadata, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := append([]SkillMetadata{}, m.outcome.Skills...)
	return out, nil
}

func (m *Manager) Get(name string) (SkillMetadata, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return SkillMetadata{}, fmt.Errorf("skill name is required")
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, skill := range m.outcome.Skills {
		if strings.EqualFold(skill.Name, name) {
			return skill, nil
		}
	}
	return SkillMetadata{}, fmt.Errorf("skill %q not found", name)
}

func (m *Manager) ResolveByNames(names []string) ([]SkillMetadata, error) {
	wanted := normalizeSkillNames(names)
	if len(wanted) == 0 {
		return []SkillMetadata{}, nil
	}

	m.mu.RLock()
	defer m.mu.RUnlock()
	byName := make(map[string]SkillMetadata, len(m.outcome.Skills))
	for _, skill := range m.outcome.Skills {
		byName[strings.ToLower(skill.Name)] = skill
	}

	resolved := make([]SkillMetadata, 0, len(wanted))
	for _, name := range wanted {
		skill, ok := byName[name]
		if !ok {
			continue
		}
		if !skill.Enabled {
			continue
		}
		resolved = append(resolved, skill)
	}
	return resolved, nil
}

func (m *Manager) Select(goal string, requested []string, mode SelectionMode) (SelectionResult, error) {
	m.mu.RLock()
	skills := append([]SkillMetadata{}, m.outcome.Skills...)
	m.mu.RUnlock()

	result := SelectionResult{Mode: mode, Selected: []SkillMetadata{}, Blocked: []BlockedSkill{}}
	if mode == SelectionModeOff {
		return result, nil
	}

	byName := make(map[string]SkillMetadata, len(skills))
	for _, skill := range skills {
		byName[strings.ToLower(skill.Name)] = skill
	}

	goalLower := strings.ToLower(goal)
	explicitSet := make(map[string]struct{})
	for _, name := range normalizeSkillNames(requested) {
		explicitSet[name] = struct{}{}
	}
	if mode != SelectionModeExplicit {
		for name := range byName {
			if strings.Contains(goalLower, "$"+name) || containsSkillToken(goalLower, name) {
				explicitSet[name] = struct{}{}
			}
		}
	}

	selectedByName := make(map[string]struct{})
	addSelected := func(skill SkillMetadata) {
		key := strings.ToLower(skill.Name)
		if _, exists := selectedByName[key]; exists {
			return
		}
		selectedByName[key] = struct{}{}
		result.Selected = append(result.Selected, skill)
	}
	addBlocked := func(name, reason string) {
		result.Blocked = append(result.Blocked, BlockedSkill{Name: name, Reason: reason})
	}

	for explicit := range explicitSet {
		skill, ok := byName[explicit]
		if !ok {
			addBlocked(explicit, "skill not found")
			continue
		}
		if !skill.Enabled {
			addBlocked(skill.Name, "skill disabled")
			continue
		}
		addSelected(skill)
	}

	if mode == SelectionModeExplicit {
		sortSkillsStable(result.Selected)
		sortBlocked(result.Blocked)
		return result, nil
	}

	for _, skill := range skills {
		if !skill.Enabled {
			continue
		}
		key := strings.ToLower(skill.Name)
		if _, explicit := explicitSet[key]; explicit {
			continue
		}

		allowImplicit := true
		if skill.Policy.AllowImplicitInvocation != nil {
			allowImplicit = *skill.Policy.AllowImplicitInvocation
		}
		if !allowImplicit {
			continue
		}

		if descriptionMatches(goalLower, strings.ToLower(skill.Description)) {
			addSelected(skill)
		}
	}

	sortSkillsStable(result.Selected)
	sortBlocked(result.Blocked)
	return result, nil
}

func (m *Manager) Render(skills []SkillMetadata) string {
	return RenderSkillsSection(skills)
}

func (m *Manager) ReadProjectDocs() (string, error) {
	opts := ProjectDocOptions{
		Workspace:          m.workspace,
		CWD:                m.cwd,
		FallbackFilenames:  append([]string{}, m.projectDocFallback...),
		ProjectRootMarkers: append([]string{}, m.projectRootMarkers...),
		MaxBytes:           m.projectDocMaxBytes,
	}
	if err := ValidateProjectDocOptions(opts); err != nil {
		return "", err
	}
	return ReadProjectDocs(opts)
}

func (m *Manager) ListInstallable(ctx context.Context, experimental bool) (OperationResult, error) {
	ops, err := m.operations()
	if err != nil {
		return OperationResult{}, err
	}
	return ops.List(ctx, experimental)
}

func (m *Manager) Install(ctx context.Context, req InstallRequest) (OperationResult, error) {
	ops, err := m.operations()
	if err != nil {
		return OperationResult{}, err
	}
	result, err := ops.Install(ctx, req)
	if err != nil {
		return result, err
	}
	_, _ = m.Reload()
	return result, nil
}

func (m *Manager) Create(ctx context.Context, req CreateRequest) (OperationResult, error) {
	ops, err := m.operations()
	if err != nil {
		return OperationResult{}, err
	}
	result, err := ops.Create(ctx, req)
	if err != nil {
		return result, err
	}
	_, _ = m.Reload()
	return result, nil
}

func (m *Manager) Enable(reference string) (OperationResult, error) {
	return m.setEnabled(reference, true)
}

func (m *Manager) Disable(reference string) (OperationResult, error) {
	return m.setEnabled(reference, false)
}

func (m *Manager) setEnabled(reference string, enabled bool) (OperationResult, error) {
	resolvedPath, displayName, err := m.resolveSkillReference(reference)
	if err != nil {
		return OperationResult{Status: "failed", Message: err.Error()}, err
	}

	cfgPath, cfg, err := m.readWriteConfig()
	if err != nil {
		return OperationResult{Status: "failed", Message: err.Error()}, err
	}

	updated := false
	for idx := range cfg.Skills.Config {
		entryPath, cerr := canonicalPath(cfg.Skills.Config[idx].Path)
		if cerr != nil {
			continue
		}
		if entryPath == resolvedPath {
			val := enabled
			cfg.Skills.Config[idx].Enabled = &val
			updated = true
			break
		}
	}
	if !updated {
		val := enabled
		cfg.Skills.Config = append(cfg.Skills.Config, tomlSkillEntry{Path: resolvedPath, Enabled: &val})
	}

	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o755); err != nil {
		return OperationResult{Status: "failed", Message: err.Error()}, err
	}
	raw, err := toml.Marshal(cfg)
	if err != nil {
		return OperationResult{Status: "failed", Message: err.Error()}, err
	}
	if err := os.WriteFile(cfgPath, raw, 0o644); err != nil {
		return OperationResult{Status: "failed", Message: err.Error()}, err
	}

	outcome, reloadErr := m.Reload()
	if reloadErr != nil {
		return OperationResult{Status: "failed", Message: reloadErr.Error()}, reloadErr
	}

	message := fmt.Sprintf("skill %s %s", displayName, ternary(enabled, "enabled", "disabled"))
	return OperationResult{
		Status:  "ok",
		Message: message,
		Output: []string{
			"config: " + cfgPath,
			fmt.Sprintf("skills_loaded=%d", len(outcome.EnabledSkills())),
			fmt.Sprintf("skills_errors=%d", len(outcome.Errors)),
		},
	}, nil
}

func (m *Manager) operations() (Operations, error) {
	m.mu.RLock()
	ops := m.ops
	m.mu.RUnlock()
	if ops == nil {
		return nil, fmt.Errorf("skill operations are not configured")
	}
	return ops, nil
}

func (m *Manager) resolveSkillReference(reference string) (path string, display string, err error) {
	ref := strings.TrimSpace(reference)
	if ref == "" {
		return "", "", fmt.Errorf("skill reference is required")
	}

	if strings.HasSuffix(strings.ToLower(ref), "/skill.md") || strings.HasSuffix(strings.ToLower(ref), "\\skill.md") {
		canon, cerr := canonicalPath(ref)
		if cerr != nil {
			return "", "", cerr
		}
		return canon, canon, nil
	}

	skill, err := m.Get(ref)
	if err != nil {
		return "", "", err
	}
	return skill.PathToSkillMD, skill.Name, nil
}

func (m *Manager) readDisabledConfig() (map[string]struct{}, string, error) {
	cfgPath, cfg, err := m.readConfigCandidates()
	if err != nil {
		return map[string]struct{}{}, cfgPath, err
	}

	disabled := make(map[string]struct{})
	for _, entry := range cfg.Skills.Config {
		enabled := true
		if entry.Enabled != nil {
			enabled = *entry.Enabled
		}
		if enabled {
			continue
		}
		canon, cerr := canonicalPath(entry.Path)
		if cerr != nil {
			continue
		}
		disabled[canon] = struct{}{}
	}
	return disabled, cfgPath, nil
}

func (m *Manager) readConfigCandidates() (string, tomlSkillConfig, error) {
	candidates := m.configCandidates()
	for _, candidate := range candidates {
		raw, err := os.ReadFile(candidate)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return candidate, tomlSkillConfig{}, fmt.Errorf("read %s: %w", candidate, err)
		}
		var cfg tomlSkillConfig
		if err := toml.Unmarshal(raw, &cfg); err != nil {
			return candidate, tomlSkillConfig{}, fmt.Errorf("parse %s: %w", candidate, err)
		}
		if cfg.Skills.Config == nil {
			cfg.Skills.Config = []tomlSkillEntry{}
		}
		return candidate, cfg, nil
	}
	return "", tomlSkillConfig{Skills: struct {
		Config []tomlSkillEntry `toml:"config"`
	}{Config: []tomlSkillEntry{}}}, nil
}

func (m *Manager) readWriteConfig() (string, tomlSkillConfig, error) {
	cfgPath := filepath.Join(m.workspace, ".codex", "config.toml")
	cfg := tomlSkillConfig{}
	raw, err := os.ReadFile(cfgPath)
	if err != nil {
		if os.IsNotExist(err) {
			cfg.Skills.Config = []tomlSkillEntry{}
			return cfgPath, cfg, nil
		}
		return cfgPath, cfg, fmt.Errorf("read %s: %w", cfgPath, err)
	}
	if err := toml.Unmarshal(raw, &cfg); err != nil {
		return cfgPath, cfg, fmt.Errorf("parse %s: %w", cfgPath, err)
	}
	if cfg.Skills.Config == nil {
		cfg.Skills.Config = []tomlSkillEntry{}
	}
	return cfgPath, cfg, nil
}

func (m *Manager) configCandidates() []string {
	candidates := []string{
		filepath.Join(m.workspace, ".codex", "config.toml"),
	}
	if m.codexHome != "" {
		candidates = append(candidates, filepath.Join(m.codexHome, "config.toml"))
	}
	candidates = append(candidates, filepath.Join(defaultCodexHome(), "config.toml"))
	return uniqueStrings(candidates)
}

func containsSkillToken(goalLower, name string) bool {
	if goalLower == "" || name == "" {
		return false
	}
	replacer := strings.NewReplacer("/", " ", "_", " ", "-", " ", ".", " ", ",", " ", ":", " ", ";", " ", "(", " ", ")", " ", "[", " ", "]", " ", "{", " ", "}", " ", "\n", " ")
	normalized := replacer.Replace(goalLower)
	tokens := strings.Fields(normalized)
	for _, token := range tokens {
		if token == name {
			return true
		}
	}
	return false
}

func descriptionMatches(goalLower, descriptionLower string) bool {
	if strings.TrimSpace(goalLower) == "" || strings.TrimSpace(descriptionLower) == "" {
		return false
	}
	goalTokens := tokenSet(goalLower)
	descTokens := tokenSet(descriptionLower)
	if len(goalTokens) == 0 || len(descTokens) == 0 {
		return false
	}
	matches := 0
	for token := range descTokens {
		if _, ok := goalTokens[token]; ok {
			matches++
			if matches >= 2 {
				return true
			}
		}
	}
	return false
}

func tokenSet(raw string) map[string]struct{} {
	replacer := strings.NewReplacer("/", " ", "_", " ", "-", " ", ".", " ", ",", " ", ":", " ", ";", " ", "(", " ", ")", " ", "[", " ", "]", " ", "{", " ", "}", " ", "\n", " ")
	normalized := replacer.Replace(strings.ToLower(raw))
	stop := map[string]struct{}{
		"the": {}, "and": {}, "with": {}, "from": {}, "this": {}, "that": {}, "your": {}, "into": {}, "about": {}, "for": {}, "use": {}, "when": {}, "needs": {}, "need": {},
	}
	out := map[string]struct{}{}
	for _, token := range strings.Fields(normalized) {
		if len(token) < 4 {
			continue
		}
		if _, isStop := stop[token]; isStop {
			continue
		}
		out[token] = struct{}{}
	}
	return out
}

func normalizeSkillNames(raw []string) []string {
	out := make([]string, 0, len(raw))
	seen := map[string]struct{}{}
	for _, entry := range raw {
		trimmed := strings.ToLower(strings.TrimSpace(entry))
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "$") {
			trimmed = strings.TrimPrefix(trimmed, "$")
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func sortBlocked(items []BlockedSkill) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Name != items[j].Name {
			return items[i].Name < items[j].Name
		}
		return items[i].Reason < items[j].Reason
	})
}

func uniqueStrings(items []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func ternary[T any](condition bool, a, b T) T {
	if condition {
		return a
	}
	return b
}
