package paths

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

func homeDir() string {
	if h := strings.TrimSpace(os.Getenv("HOME")); h != "" {
		return h
	}
	h, _ := os.UserHomeDir()
	return h
}

func expandHome(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return p
	}
	if strings.HasPrefix(p, "~"+string(filepath.Separator)) || p == "~" {
		return filepath.Join(homeDir(), strings.TrimPrefix(p, "~"))
	}
	return p
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// SisyphusDir returns the base directory for local BIOMETRICS scratch/state.
// Override with BIOMETRICS_SISYPHUS_DIR (preferred) or SISYPHUS_DIR.
func SisyphusDir() string {
	if v := strings.TrimSpace(os.Getenv("BIOMETRICS_SISYPHUS_DIR")); v != "" {
		return expandHome(v)
	}
	if v := strings.TrimSpace(os.Getenv("SISYPHUS_DIR")); v != "" {
		return expandHome(v)
	}
	return filepath.Join(homeDir(), ".sisyphus")
}

func SisyphusSessionsPath() string {
	return filepath.Join(SisyphusDir(), "sessions.json")
}

func SisyphusPromptHistoryPath() string {
	return filepath.Join(SisyphusDir(), "prompt_history.json")
}

func SisyphusDBPath(filename string) string {
	return filepath.Join(SisyphusDir(), filename)
}

func SisyphusPlansDir(projectID string) string {
	return filepath.Join(SisyphusDir(), "plans", projectID)
}

// ProjectsDir returns the default parent directory for local repos when only a project ID is known.
// Override with BIOMETRICS_PROJECTS_DIR.
func ProjectsDir() string {
	if v := strings.TrimSpace(os.Getenv("BIOMETRICS_PROJECTS_DIR")); v != "" {
		return expandHome(v)
	}
	return filepath.Join(homeDir(), "dev")
}

// FindRepoRoot walks up from cwd until it finds a directory containing `biometrics-cli/go.mod`.
// Override with BIOMETRICS_WORKSPACE to force a fixed repo root.
func FindRepoRoot() (string, error) {
	if v := strings.TrimSpace(os.Getenv("BIOMETRICS_WORKSPACE")); v != "" {
		root := expandHome(v)
		if fileExists(filepath.Join(root, "biometrics-cli", "go.mod")) {
			return root, nil
		}
		return "", errors.New("BIOMETRICS_WORKSPACE does not look like a BIOMETRICS repo root")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	dir := cwd
	for {
		if fileExists(filepath.Join(dir, "biometrics-cli", "go.mod")) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", errors.New("could not find BIOMETRICS repo root from cwd")
}
