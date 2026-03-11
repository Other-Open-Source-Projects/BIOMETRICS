package skillkit

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultProjectDocFilename  = "AGENTS.md"
	defaultProjectDocOverride  = "AGENTS.override.md"
	defaultProjectDocMaxBytes  = 32768
	projectDocSectionSeparator = "\n\n--- project-doc ---\n\n"
)

type ProjectDocOptions struct {
	Workspace          string
	CWD                string
	FallbackFilenames  []string
	ProjectRootMarkers []string
	MaxBytes           int
}

func DiscoverProjectDocPaths(opts ProjectDocOptions) ([]string, error) {
	cwd := strings.TrimSpace(opts.CWD)
	if cwd == "" {
		cwd = strings.TrimSpace(opts.Workspace)
	}
	if cwd == "" {
		return nil, nil
	}
	markers := opts.ProjectRootMarkers
	if len(markers) == 0 {
		markers = []string{".git"}
	}
	projectRoot := findProjectRoot(cwd, markers)
	if projectRoot == "" {
		projectRoot = cwd
	}

	searchDirs := dirsBetween(projectRoot, cwd)
	if len(searchDirs) == 0 {
		searchDirs = []string{cwd}
	}

	candidateNames := candidateDocFilenames(opts.FallbackFilenames)
	paths := make([]string, 0, len(searchDirs))
	for _, dir := range searchDirs {
		for _, name := range candidateNames {
			candidate := filepath.Join(dir, name)
			meta, err := os.Lstat(candidate)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return nil, err
			}
			mode := meta.Mode()
			if mode.IsRegular() || mode&os.ModeSymlink != 0 {
				paths = append(paths, candidate)
				break
			}
		}
	}
	return paths, nil
}

func ReadProjectDocs(opts ProjectDocOptions) (string, error) {
	limit := opts.MaxBytes
	if limit <= 0 {
		limit = defaultProjectDocMaxBytes
	}

	paths, err := DiscoverProjectDocPaths(opts)
	if err != nil {
		return "", err
	}
	if len(paths) == 0 {
		return "", nil
	}

	remaining := int64(limit)
	parts := make([]string, 0, len(paths))
	for _, path := range paths {
		if remaining <= 0 {
			break
		}
		f, err := os.Open(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", err
		}
		defer f.Close()

		reader := io.LimitReader(f, remaining)
		raw, err := io.ReadAll(reader)
		if err != nil {
			return "", err
		}
		if len(strings.TrimSpace(string(raw))) == 0 {
			continue
		}
		parts = append(parts, string(raw))
		remaining -= int64(len(raw))
	}

	if len(parts) == 0 {
		return "", nil
	}
	return strings.Join(parts, "\n\n"), nil
}

func MergeUserInstructionsWithProjectDocs(userInstructions string, projectDocs string) string {
	userInstructions = strings.TrimSpace(userInstructions)
	projectDocs = strings.TrimSpace(projectDocs)
	if userInstructions == "" {
		return projectDocs
	}
	if projectDocs == "" {
		return userInstructions
	}
	return userInstructions + projectDocSectionSeparator + projectDocs
}

func candidateDocFilenames(fallback []string) []string {
	out := []string{defaultProjectDocOverride, defaultProjectDocFilename}
	seen := map[string]struct{}{
		defaultProjectDocOverride: {},
		defaultProjectDocFilename: {},
	}
	for _, candidate := range fallback {
		trimmed := strings.TrimSpace(candidate)
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

func ValidateProjectDocOptions(opts ProjectDocOptions) error {
	if strings.TrimSpace(opts.Workspace) == "" && strings.TrimSpace(opts.CWD) == "" {
		return fmt.Errorf("workspace or cwd is required")
	}
	return nil
}
