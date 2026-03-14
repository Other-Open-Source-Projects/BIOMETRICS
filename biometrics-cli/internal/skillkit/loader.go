package skillkit

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	skillFilename            = "SKILL.md"
	openAIMetadataPath       = "agents/openai.yaml"
	defaultMaxScanDepth      = 6
	maxSkillNameChars        = 64
	maxDescriptionChars      = 1024
	maxShortDescriptionChars = 1024
)

var skillNamePattern = regexp.MustCompile(`^[a-z0-9-]+$`)

type loadConfig struct {
	Workspace          string
	CWD                string
	CodexHome          string
	ProjectRootMarkers []string
	DisabledPaths      map[string]struct{}
}

type rootSpec struct {
	Path  string
	Scope Scope
}

type frontmatterDoc struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Metadata    struct {
		ShortDescription string `yaml:"short-description"`
	} `yaml:"metadata"`
}

type metadataDoc struct {
	Interface *struct {
		DisplayName      string `yaml:"display_name"`
		ShortDescription string `yaml:"short_description"`
		IconSmall        string `yaml:"icon_small"`
		IconLarge        string `yaml:"icon_large"`
		BrandColor       string `yaml:"brand_color"`
		DefaultPrompt    string `yaml:"default_prompt"`
	} `yaml:"interface"`
	Dependencies *struct {
		Tools []struct {
			Type        string `yaml:"type"`
			Value       string `yaml:"value"`
			Description string `yaml:"description"`
			Transport   string `yaml:"transport"`
			Command     string `yaml:"command"`
			URL         string `yaml:"url"`
		} `yaml:"tools"`
	} `yaml:"dependencies"`
	Policy *struct {
		AllowImplicitInvocation *bool `yaml:"allow_implicit_invocation"`
	} `yaml:"policy"`
}

func loadSkills(cfg loadConfig) LoadOutcome {
	roots := discoverRoots(cfg)
	out := LoadOutcome{
		Skills:        make([]SkillMetadata, 0, 64),
		Errors:        make([]SkillError, 0, 16),
		DisabledPaths: cfg.DisabledPaths,
		LoadedAt:      time.Now().UTC(),
	}

	seenByPath := map[string]struct{}{}
	seenByName := map[string]struct{}{}

	for _, root := range roots {
		skills, errs := scanRoot(root)
		if len(errs) > 0 {
			out.Errors = append(out.Errors, errs...)
		}
		for _, skill := range skills {
			if _, ok := seenByPath[skill.PathToSkillMD]; ok {
				continue
			}
			if _, ok := seenByName[skill.Name]; ok {
				continue
			}
			if _, disabled := cfg.DisabledPaths[skill.PathToSkillMD]; disabled {
				skill.Enabled = false
			} else {
				skill.Enabled = true
			}
			seenByPath[skill.PathToSkillMD] = struct{}{}
			seenByName[skill.Name] = struct{}{}
			out.Skills = append(out.Skills, skill)
		}
	}

	return out
}

func discoverRoots(cfg loadConfig) []rootSpec {
	workspace := strings.TrimSpace(cfg.Workspace)
	cwd := strings.TrimSpace(cfg.CWD)
	if cwd == "" {
		cwd = workspace
	}
	if workspace == "" {
		workspace = cwd
	}

	markers := cfg.ProjectRootMarkers
	if len(markers) == 0 {
		markers = []string{".git"}
	}

	projectRoot := findProjectRoot(cwd, markers)
	if projectRoot == "" {
		projectRoot = cwd
	}

	roots := make([]rootSpec, 0, 16)
	if workspace != "" {
		roots = append(roots, rootSpec{Path: filepath.Join(workspace, ".codex", "skills"), Scope: ScopeRepo})
	}

	for _, dir := range dirsBetween(projectRoot, cwd) {
		roots = append(roots, rootSpec{Path: filepath.Join(dir, ".agents", "skills"), Scope: ScopeRepo})
	}

	codexHome := strings.TrimSpace(cfg.CodexHome)
	if codexHome == "" {
		codexHome = defaultCodexHome()
	}
	homeCodex := defaultCodexHome()

	roots = append(roots, rootSpec{Path: filepath.Join(codexHome, "skills"), Scope: ScopeUser})
	if homeCodex != codexHome {
		roots = append(roots, rootSpec{Path: filepath.Join(homeCodex, "skills"), Scope: ScopeUser})
	}
	roots = append(roots, rootSpec{Path: filepath.Join(codexHome, "skills", ".system"), Scope: ScopeSystem})
	roots = append(roots, rootSpec{Path: filepath.Join("/etc", "codex", "skills"), Scope: ScopeAdmin})

	// de-duplicate canonical paths while preserving order.
	seen := map[string]struct{}{}
	filtered := make([]rootSpec, 0, len(roots))
	for _, root := range roots {
		canonical, err := canonicalPath(root.Path)
		if err != nil {
			canonical = filepath.Clean(root.Path)
		}
		if _, ok := seen[canonical]; ok {
			continue
		}
		seen[canonical] = struct{}{}
		filtered = append(filtered, rootSpec{Path: canonical, Scope: root.Scope})
	}

	return filtered
}

func scanRoot(root rootSpec) ([]SkillMetadata, []SkillError) {
	info, err := os.Stat(root.Path)
	if err != nil || !info.IsDir() {
		return nil, nil
	}
	rootCanonical, rerr := canonicalPath(root.Path)
	if rerr != nil {
		rootCanonical = filepath.Clean(root.Path)
	}

	type queueNode struct {
		path  string
		depth int
	}

	queue := []queueNode{{path: rootCanonical, depth: 0}}
	visited := map[string]struct{}{rootCanonical: {}}
	skills := make([]SkillMetadata, 0, 16)
	errorsList := make([]SkillError, 0, 8)
	followSymlinkDirs := root.Scope != ScopeSystem

	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]

		entries, readErr := os.ReadDir(node.path)
		if readErr != nil {
			errorsList = append(errorsList, SkillError{Path: node.path, Message: readErr.Error()})
			continue
		}

		for _, entry := range entries {
			name := entry.Name()
			if strings.HasPrefix(name, ".") {
				continue
			}
			fullPath := filepath.Join(node.path, name)

			if entry.IsDir() {
				if node.depth+1 > defaultMaxScanDepth {
					continue
				}
				canonical, cerr := canonicalPath(fullPath)
				if cerr != nil {
					continue
				}
				if !pathWithinRoot(rootCanonical, canonical) {
					errorsList = append(errorsList, SkillError{
						Path:    fullPath,
						Message: "path escapes skill root",
					})
					continue
				}
				if _, ok := visited[canonical]; ok {
					continue
				}
				visited[canonical] = struct{}{}
				queue = append(queue, queueNode{path: canonical, depth: node.depth + 1})
				continue
			}

			if entry.Type()&fs.ModeSymlink != 0 {
				if !followSymlinkDirs || node.depth+1 > defaultMaxScanDepth {
					continue
				}
				resolved, serr := filepath.EvalSymlinks(fullPath)
				if serr != nil {
					continue
				}
				resolvedInfo, serr := os.Stat(resolved)
				if serr != nil || !resolvedInfo.IsDir() {
					continue
				}
				canonical, cerr := canonicalPath(resolved)
				if cerr != nil {
					continue
				}
				if !pathWithinRoot(rootCanonical, canonical) {
					errorsList = append(errorsList, SkillError{
						Path:    fullPath,
						Message: "symlink path escapes skill root",
					})
					continue
				}
				if _, ok := visited[canonical]; ok {
					continue
				}
				visited[canonical] = struct{}{}
				queue = append(queue, queueNode{path: canonical, depth: node.depth + 1})
				continue
			}

			if name != skillFilename {
				continue
			}
			skill, parseErr := parseSkillFile(fullPath, root.Scope)
			if parseErr != nil {
				errorsList = append(errorsList, SkillError{Path: fullPath, Message: parseErr.Error()})
				continue
			}
			if !pathWithinRoot(rootCanonical, skill.PathToSkillMD) {
				errorsList = append(errorsList, SkillError{
					Path:    fullPath,
					Message: "skill file escapes skill root",
				})
				continue
			}
			skills = append(skills, skill)
		}
	}

	return skills, errorsList
}

func parseSkillFile(path string, scope Scope) (SkillMetadata, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return SkillMetadata{}, err
	}

	frontmatterText, err := extractFrontmatter(string(contents))
	if err != nil {
		return SkillMetadata{}, err
	}

	var parsed frontmatterDoc
	if err := yaml.Unmarshal([]byte(frontmatterText), &parsed); err != nil {
		return SkillMetadata{}, fmt.Errorf("invalid yaml frontmatter: %w", err)
	}

	name := sanitizeSingleLine(parsed.Name)
	description := sanitizeSingleLine(parsed.Description)
	shortDescription := sanitizeSingleLine(parsed.Metadata.ShortDescription)

	if err := validateSkillName(name); err != nil {
		return SkillMetadata{}, err
	}
	if description == "" {
		return SkillMetadata{}, fmt.Errorf("missing description")
	}
	if len(description) > maxDescriptionChars {
		return SkillMetadata{}, fmt.Errorf("description exceeds %d chars", maxDescriptionChars)
	}
	if strings.Contains(description, "<") || strings.Contains(description, ">") {
		return SkillMetadata{}, fmt.Errorf("description contains disallowed angle bracket")
	}
	if len(shortDescription) > maxShortDescriptionChars {
		shortDescription = shortDescription[:maxShortDescriptionChars]
	}

	canonical, cerr := canonicalPath(path)
	if cerr != nil {
		canonical = filepath.Clean(path)
	}

	skill := SkillMetadata{
		Name:             name,
		Description:      description,
		ShortDescription: shortDescription,
		PathToSkillMD:    canonical,
		Scope:            scope,
		Enabled:          true,
	}

	loadOptionalMetadata(&skill)
	return skill, nil
}

func loadOptionalMetadata(skill *SkillMetadata) {
	skillDir := filepath.Dir(skill.PathToSkillMD)
	metaPath := filepath.Join(skillDir, openAIMetadataPath)
	raw, err := os.ReadFile(metaPath)
	if err != nil {
		return
	}

	var doc metadataDoc
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return
	}

	if doc.Interface != nil {
		skill.Interface = &SkillInterface{
			DisplayName:      sanitizeSingleLine(doc.Interface.DisplayName),
			ShortDescription: sanitizeSingleLine(doc.Interface.ShortDescription),
			IconSmall:        normalizeAssetPath(skillDir, doc.Interface.IconSmall),
			IconLarge:        normalizeAssetPath(skillDir, doc.Interface.IconLarge),
			BrandColor:       sanitizeSingleLine(doc.Interface.BrandColor),
			DefaultPrompt:    sanitizeSingleLine(doc.Interface.DefaultPrompt),
		}
	}

	if doc.Dependencies != nil && len(doc.Dependencies.Tools) > 0 {
		deps := make([]SkillToolDependency, 0, len(doc.Dependencies.Tools))
		for _, tool := range doc.Dependencies.Tools {
			typeVal := sanitizeSingleLine(tool.Type)
			valueVal := sanitizeSingleLine(tool.Value)
			if typeVal == "" || valueVal == "" {
				continue
			}
			deps = append(deps, SkillToolDependency{
				Type:        typeVal,
				Value:       valueVal,
				Description: sanitizeSingleLine(tool.Description),
				Transport:   sanitizeSingleLine(tool.Transport),
				Command:     sanitizeSingleLine(tool.Command),
				URL:         sanitizeSingleLine(tool.URL),
			})
		}
		if len(deps) > 0 {
			skill.Dependencies = &SkillDependencies{Tools: deps}
		}
	}

	if doc.Policy != nil {
		skill.Policy.AllowImplicitInvocation = doc.Policy.AllowImplicitInvocation
	}
}

func normalizeAssetPath(skillDir, raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	if filepath.IsAbs(trimmed) {
		return ""
	}
	normalized := filepath.Clean(trimmed)
	if normalized == "." || strings.HasPrefix(normalized, "..") {
		return ""
	}
	parts := strings.Split(filepath.ToSlash(normalized), "/")
	if len(parts) == 0 || parts[0] != "assets" {
		return ""
	}
	return filepath.Join(skillDir, normalized)
}

func extractFrontmatter(contents string) (string, error) {
	lines := strings.Split(contents, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return "", fmt.Errorf("missing YAML frontmatter delimited by ---")
	}
	front := make([]string, 0, 32)
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			if len(front) == 0 {
				return "", fmt.Errorf("missing YAML frontmatter delimited by ---")
			}
			return strings.Join(front, "\n"), nil
		}
		front = append(front, lines[i])
	}
	return "", fmt.Errorf("missing YAML frontmatter closing delimiter")
}

func validateSkillName(name string) error {
	if name == "" {
		return fmt.Errorf("missing name")
	}
	if len(name) > maxSkillNameChars {
		return fmt.Errorf("name exceeds %d chars", maxSkillNameChars)
	}
	if !skillNamePattern.MatchString(name) {
		return fmt.Errorf("name must match [a-z0-9-]+")
	}
	if strings.HasPrefix(name, "-") || strings.HasSuffix(name, "-") || strings.Contains(name, "--") {
		return fmt.Errorf("name cannot start/end with '-' or contain '--'")
	}
	return nil
}

func sanitizeSingleLine(raw string) string {
	return strings.Join(strings.Fields(raw), " ")
}

func defaultCodexHome() string {
	if env := strings.TrimSpace(os.Getenv("CODEX_HOME")); env != "" {
		return env
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".codex"
	}
	return filepath.Join(home, ".codex")
}

func findProjectRoot(cwd string, markers []string) string {
	if cwd == "" {
		return ""
	}
	clean := filepath.Clean(cwd)
	if len(markers) == 0 {
		return clean
	}

	cursor := clean
	for {
		for _, marker := range markers {
			if strings.TrimSpace(marker) == "" {
				continue
			}
			candidate := filepath.Join(cursor, marker)
			if _, err := os.Stat(candidate); err == nil {
				return cursor
			}
		}

		next := filepath.Dir(cursor)
		if next == cursor {
			break
		}
		cursor = next
	}

	return clean
}

func dirsBetween(projectRoot, cwd string) []string {
	projectRoot = filepath.Clean(projectRoot)
	cwd = filepath.Clean(cwd)
	if projectRoot == "" || cwd == "" {
		return nil
	}

	dirs := []string{}
	cursor := cwd
	for {
		dirs = append(dirs, cursor)
		if cursor == projectRoot {
			break
		}
		next := filepath.Dir(cursor)
		if next == cursor {
			break
		}
		cursor = next
	}
	for i, j := 0, len(dirs)-1; i < j; i, j = i+1, j-1 {
		dirs[i], dirs[j] = dirs[j], dirs[i]
	}
	return dirs
}

func canonicalPath(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", errors.New("empty path")
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		if os.IsNotExist(err) {
			return filepath.Clean(path), nil
		}
		return "", err
	}
	abs, err := filepath.Abs(resolved)
	if err != nil {
		return "", err
	}
	return abs, nil
}

func pathWithinRoot(root, candidate string) bool {
	root = filepath.Clean(root)
	candidate = filepath.Clean(candidate)
	rel, err := filepath.Rel(root, candidate)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	if rel == ".." {
		return false
	}
	return !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func scopeRank(scope Scope) int {
	switch scope {
	case ScopeRepo:
		return 0
	case ScopeUser:
		return 1
	case ScopeSystem:
		return 2
	default:
		return 3
	}
}

func sortSkillsStable(skills []SkillMetadata) {
	sort.SliceStable(skills, func(i, j int) bool {
		ri := scopeRank(skills[i].Scope)
		rj := scopeRank(skills[j].Scope)
		if ri != rj {
			return ri < rj
		}
		if skills[i].Name != skills[j].Name {
			return skills[i].Name < skills[j].Name
		}
		return skills[i].PathToSkillMD < skills[j].PathToSkillMD
	})
}
