package skillops

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"biometrics-cli/internal/skillkit"
)

type Runner struct {
	workspace string
	codexHome string
	pythonBin string
}

func New(workspace, codexHome string) *Runner {
	workspace = strings.TrimSpace(workspace)
	if workspace == "" {
		workspace = "."
	}
	if strings.TrimSpace(codexHome) == "" {
		if env := strings.TrimSpace(os.Getenv("CODEX_HOME")); env != "" {
			codexHome = env
		} else {
			home, _ := os.UserHomeDir()
			codexHome = filepath.Join(home, ".codex")
		}
	}

	python := "python3"
	if _, err := exec.LookPath("python3"); err != nil {
		python = "python"
	}

	return &Runner{
		workspace: workspace,
		codexHome: codexHome,
		pythonBin: python,
	}
}

func (r *Runner) List(ctx context.Context, experimental bool) (skillkit.OperationResult, error) {
	script := filepath.Join(r.workspace, "third_party", "openai-skills", ".system", "skill-installer", "scripts", "list-skills.py")
	args := []string{script, "--format", "json"}
	if experimental {
		args = append(args, "--path", "skills/.experimental")
	}
	return r.exec(ctx, "skill.list", r.pythonBin, args...)
}

func (r *Runner) Install(ctx context.Context, req skillkit.InstallRequest) (skillkit.OperationResult, error) {
	script := filepath.Join(r.workspace, "third_party", "openai-skills", ".system", "skill-installer", "scripts", "install-skill-from-github.py")
	args := []string{script}

	dest := strings.TrimSpace(req.Dest)
	if dest == "" {
		dest = filepath.Join(r.codexHome, "skills")
	}
	if !filepath.IsAbs(dest) {
		dest = filepath.Join(r.workspace, dest)
	}
	args = append(args, "--dest", dest)

	if method := strings.TrimSpace(req.Method); method != "" {
		args = append(args, "--method", method)
	}
	if ref := strings.TrimSpace(req.Ref); ref != "" {
		args = append(args, "--ref", ref)
	}
	if name := strings.TrimSpace(req.Name); name != "" {
		args = append(args, "--name", name)
	}

	if url := strings.TrimSpace(req.URL); url != "" {
		args = append(args, "--url", url)
	} else {
		repo := strings.TrimSpace(req.Repo)
		paths := append([]string{}, req.Paths...)
		if repo == "" && req.Name != "" && len(paths) == 0 {
			repo = "openai/skills"
			prefix := "skills/.curated"
			if req.Experimental {
				prefix = "skills/.experimental"
			}
			paths = []string{filepath.ToSlash(filepath.Join(prefix, req.Name))}
		}
		if repo != "" {
			args = append(args, "--repo", repo)
		}
		if len(paths) > 0 {
			args = append(args, "--path")
			for _, p := range paths {
				trimmed := strings.TrimSpace(p)
				if trimmed == "" {
					continue
				}
				args = append(args, filepath.ToSlash(trimmed))
			}
		}
	}

	return r.exec(ctx, "skill.install", r.pythonBin, args...)
}

func (r *Runner) Create(ctx context.Context, req skillkit.CreateRequest) (skillkit.OperationResult, error) {
	script := filepath.Join(r.workspace, "third_party", "openai-skills", ".system", "skill-creator", "scripts", "init_skill.py")

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return skillkit.OperationResult{Status: "failed", Message: "name is required"}, fmt.Errorf("name is required")
	}

	baseAbs, err := filepath.Abs(filepath.Join(r.workspace, ".codex", "skills"))
	if err != nil {
		return skillkit.OperationResult{Status: "failed", Message: err.Error()}, err
	}
	if err := os.MkdirAll(baseAbs, 0o755); err != nil {
		return skillkit.OperationResult{Status: "failed", Message: err.Error()}, err
	}

	args := []string{script, name, "--path", baseAbs}
	if len(req.Resources) > 0 {
		resources := make([]string, 0, len(req.Resources))
		for _, entry := range req.Resources {
			trimmed := strings.TrimSpace(entry)
			if trimmed == "" {
				continue
			}
			resources = append(resources, trimmed)
		}
		if len(resources) > 0 {
			args = append(args, "--resources", strings.Join(resources, ","))
		}
	}
	if req.Examples {
		args = append(args, "--examples")
	}

	if len(req.Interface) > 0 {
		keys := make([]string, 0, len(req.Interface))
		for key := range req.Interface {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			value := strings.TrimSpace(req.Interface[key])
			if value == "" {
				continue
			}
			args = append(args, "--interface", key+"="+value)
		}
	}

	result, err := r.exec(ctx, "skill.create", r.pythonBin, args...)
	if err != nil {
		return result, err
	}

	if req.Validate {
		skillPath := filepath.Join(baseAbs, name)
		validateResult, validateErr := r.Validate(ctx, skillPath)
		result.Output = append(result.Output, validateResult.Output...)
		if validateErr != nil {
			return skillkit.OperationResult{
				Status:  "failed",
				Message: "skill created but validation failed",
				Output:  result.Output,
			}, validateErr
		}
	}

	return result, nil
}

func (r *Runner) Validate(ctx context.Context, skillPath string) (skillkit.OperationResult, error) {
	script := filepath.Join(r.workspace, "third_party", "openai-skills", ".system", "skill-creator", "scripts", "quick_validate.py")
	return r.exec(ctx, "skill.validate", r.pythonBin, script, skillPath)
}

func (r *Runner) exec(ctx context.Context, name string, cmd string, args ...string) (skillkit.OperationResult, error) {
	c := exec.CommandContext(ctx, cmd, args...)
	c.Dir = r.workspace
	c.Env = append(os.Environ(), "CODEX_HOME="+r.codexHome)

	stdout, err := c.StdoutPipe()
	if err != nil {
		return skillkit.OperationResult{Status: "failed", Message: err.Error()}, err
	}
	stderr, err := c.StderrPipe()
	if err != nil {
		return skillkit.OperationResult{Status: "failed", Message: err.Error()}, err
	}

	if err := c.Start(); err != nil {
		return skillkit.OperationResult{Status: "failed", Message: err.Error()}, err
	}

	lines := make([]string, 0, 32)
	collect := func(scanner *bufio.Scanner) {
		for scanner.Scan() {
			text := strings.TrimSpace(scanner.Text())
			if text != "" {
				lines = append(lines, text)
			}
		}
	}
	collect(bufio.NewScanner(stdout))
	collect(bufio.NewScanner(stderr))

	if err := c.Wait(); err != nil {
		message := fmt.Sprintf("%s failed", name)
		if len(lines) > 0 {
			message = lines[len(lines)-1]
		}
		return skillkit.OperationResult{Status: "failed", Message: message, Output: lines}, err
	}

	message := fmt.Sprintf("%s succeeded", name)
	if len(lines) > 0 {
		message = lines[len(lines)-1]
	}
	return skillkit.OperationResult{Status: "ok", Message: message, Output: lines}, nil
}
