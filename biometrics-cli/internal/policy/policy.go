package policy

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"biometrics-cli/internal/contracts"
)

type Engine struct {
	policy      contracts.RunPolicy
	secretRules []redactionRule
}

type redactionRule struct {
	re      *regexp.Regexp
	replace string
}

func Default() *Engine {
	return &Engine{
		policy: contracts.RunPolicy{
			AutonomousDefault: true,
			AllowedCommands: []string{
				"go test",
				"go vet",
				"go fmt",
				"git status",
				"git diff",
				"opencode",
			},
			SecretRedaction: true,
			AllowFSWrite:    true,
			AllowGitCommit:  false,
		},
		secretRules: []redactionRule{
			{re: regexp.MustCompile(`(?i)(api[_-]?key\s*[=:]\s*)([^\s"']+)`), replace: `$1[REDACTED]`},
			{re: regexp.MustCompile(`(?i)(token\s*[=:]\s*)([^\s"']+)`), replace: `$1[REDACTED]`},
			{re: regexp.MustCompile(`(?i)((id_token|access_token)\s*[=:]\s*)([^\s"']+)`), replace: `$1[REDACTED]`},
			{re: regexp.MustCompile(`(?i)(authorization\s*[=:]\s*bearer\s+)([^\s"']+)`), replace: `$1[REDACTED]`},
			{re: regexp.MustCompile(`(?i)(authorization\s*[=:]\s*)([^\s"']+)`), replace: `$1[REDACTED]`},
			{re: regexp.MustCompile(`(?i)(cookie\s*[=:]\s*)([^\s"']+)`), replace: `$1[REDACTED]`},
			{re: regexp.MustCompile(`(?i)(bearer\s+)([A-Za-z0-9._-]{20,})`), replace: `$1[REDACTED]`},
			{re: regexp.MustCompile(`(?i)(secret\s*[=:]\s*)([^\s"']+)`), replace: `$1[REDACTED]`},
			{re: regexp.MustCompile(`eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}`), replace: `[REDACTED]`},
			{re: regexp.MustCompile(`(?i)nvapi-[A-Za-z0-9_-]{20,}`), replace: `[REDACTED]`},
			{re: regexp.MustCompile(`(?i)sk-proj-[A-Za-z0-9_-]{20,}`), replace: `[REDACTED]`},
			{re: regexp.MustCompile(`(?i)sk-live-[A-Za-z0-9_-]{20,}`), replace: `[REDACTED]`},
			{re: regexp.MustCompile(`(?i)ghp_[A-Za-z0-9]{20,}`), replace: `[REDACTED]`},
			{re: regexp.MustCompile(`(?i)glpat-[A-Za-z0-9_-]{20,}`), replace: `[REDACTED]`},
		},
	}
}

func (e *Engine) Policy() contracts.RunPolicy {
	return e.policy
}

func (e *Engine) AllowedCommand(command string) bool {
	trimmed := strings.TrimSpace(command)
	for _, allowed := range e.policy.AllowedCommands {
		if strings.HasPrefix(trimmed, allowed) {
			return true
		}
	}
	return false
}

func (e *Engine) Redact(input string) string {
	if !e.policy.SecretRedaction {
		return input
	}
	out := input
	for _, rule := range e.secretRules {
		out = rule.re.ReplaceAllString(out, rule.replace)
	}
	return out
}

func (e *Engine) ValidatePath(root, target string) (string, error) {
	cleanRoot, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("root path: %w", err)
	}
	rootResolved, err := filepath.EvalSymlinks(cleanRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("root path does not exist")
		}
		return "", fmt.Errorf("resolve root path: %w", err)
	}

	joined := filepath.Join(cleanRoot, target)
	cleanPath, err := filepath.Abs(joined)
	if err != nil {
		return "", fmt.Errorf("target path: %w", err)
	}
	if !isWithinRoot(cleanRoot, cleanPath) {
		return "", fmt.Errorf("path traversal blocked")
	}

	resolvedPath, err := resolvePathForPolicy(cleanPath)
	if err != nil {
		return "", fmt.Errorf("resolve target path: %w", err)
	}
	if !isWithinRoot(rootResolved, resolvedPath) {
		return "", fmt.Errorf("path traversal blocked")
	}

	return cleanPath, nil
}

func resolvePathForPolicy(path string) (string, error) {
	resolved, err := filepath.EvalSymlinks(path)
	if err == nil {
		return resolved, nil
	}
	if !os.IsNotExist(err) {
		return "", err
	}

	parent := filepath.Dir(path)
	resolvedParent, parentErr := filepath.EvalSymlinks(parent)
	if parentErr == nil {
		return filepath.Join(resolvedParent, filepath.Base(path)), nil
	}
	if os.IsNotExist(parentErr) {
		return path, nil
	}
	return "", parentErr
}

func isWithinRoot(root, candidate string) bool {
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
