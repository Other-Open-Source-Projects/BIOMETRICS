package onboarding

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	stateVersion = 1

	stateStatusRunning = "running"
	stateStatusSuccess = "success"
	stateStatusFailed  = "failed"

	stepStatusPending   = "pending"
	stepStatusRunning   = "running"
	stepStatusCompleted = "completed"
	stepStatusFailed    = "failed"
)

// Options defines the onboarding runtime behavior.
type Options struct {
	Workspace      string
	Resume         bool
	Yes            bool
	NonInteractive bool
	Doctor         bool
	Out            io.Writer
}

type Runner struct {
	opts      Options
	workspace string
	cliDir    string
	webDir    string
	rootBin   string

	statePath  string
	reportPath string
	eventsPath string
	logDir     string

	state    stateFile
	warnings []string
	out      io.Writer
}

type step struct {
	ID          string
	Description string
	Run         func(context.Context) *StepError
}

// StepError provides a human-friendly remediation trail for hard failures.
type StepError struct {
	Message     string   `json:"message"`
	Remediation []string `json:"remediation,omitempty"`
}

func (e *StepError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

type stepState struct {
	Status      string   `json:"status"`
	Attempts    int      `json:"attempts"`
	StartedAt   string   `json:"started_at,omitempty"`
	FinishedAt  string   `json:"finished_at,omitempty"`
	Message     string   `json:"message,omitempty"`
	Remediation []string `json:"remediation,omitempty"`
}

type stateFile struct {
	Version    int                   `json:"version"`
	Workspace  string                `json:"workspace"`
	Mode       string                `json:"mode"`
	Status     string                `json:"status"`
	StartedAt  string                `json:"started_at"`
	UpdatedAt  string                `json:"updated_at"`
	FinishedAt string                `json:"finished_at,omitempty"`
	Steps      map[string]*stepState `json:"steps"`
}

type reportStep struct {
	ID          string   `json:"id"`
	Description string   `json:"description"`
	Status      string   `json:"status"`
	Message     string   `json:"message,omitempty"`
	Remediation []string `json:"remediation,omitempty"`
}

type reportFile struct {
	Version        int          `json:"version"`
	Workspace      string       `json:"workspace"`
	Mode           string       `json:"mode"`
	Status         string       `json:"status"`
	StartedAt      string       `json:"started_at"`
	FinishedAt     string       `json:"finished_at"`
	CompletedSteps int          `json:"completed_steps"`
	FailedStep     string       `json:"failed_step,omitempty"`
	FailedReason   string       `json:"failed_reason,omitempty"`
	Remediation    []string     `json:"remediation,omitempty"`
	Warnings       []string     `json:"warnings,omitempty"`
	Steps          []reportStep `json:"steps"`
}

type onboardEvent struct {
	Type      string            `json:"type"`
	StepID    string            `json:"step_id,omitempty"`
	Mode      string            `json:"mode"`
	Status    string            `json:"status"`
	Payload   map[string]string `json:"payload,omitempty"`
	CreatedAt string            `json:"created_at"`
}

type onboardingReportStatus struct {
	Status string `json:"status"`
}

type readinessProbe struct {
	Ready bool `json:"ready"`
}

// NewRunner creates a ready-to-run onboarding runner.
func NewRunner(opts Options) (*Runner, error) {
	workspace, cliDir, err := detectWorkspace(opts.Workspace)
	if err != nil {
		return nil, err
	}

	out := opts.Out
	if out == nil {
		out = os.Stdout
	}

	onboardDir := filepath.Join(workspace, ".biometrics", "onboard")
	r := &Runner{
		opts:       opts,
		out:        out,
		workspace:  workspace,
		cliDir:     cliDir,
		webDir:     filepath.Join(cliDir, "web-v3"),
		rootBin:    filepath.Join(workspace, "bin"),
		statePath:  filepath.Join(onboardDir, "state.json"),
		reportPath: filepath.Join(onboardDir, "report.json"),
		eventsPath: filepath.Join(onboardDir, "events.jsonl"),
		logDir:     filepath.Join(onboardDir, "logs"),
	}

	if !opts.Doctor {
		if err := os.MkdirAll(onboardDir, 0o755); err != nil {
			return nil, fmt.Errorf("create onboard dir: %w", err)
		}
		if err := os.MkdirAll(r.logDir, 0o755); err != nil {
			return nil, fmt.Errorf("create onboard log dir: %w", err)
		}
	}
	return r, nil
}

// Run executes the full onboarding flow (or doctor checks).
func (r *Runner) Run(ctx context.Context) error {
	if err := r.initState(); err != nil {
		return err
	}

	steps := r.steps()
	for _, current := range steps {
		if r.opts.Resume && r.stepCompleted(current.ID) {
			r.logf("skip step=%s (already completed)", current.ID)
			continue
		}

		if err := r.runStep(ctx, current); err != nil {
			serr := asStepError(err)
			r.state.Status = stateStatusFailed
			r.state.FinishedAt = nowUTC()
			r.state.UpdatedAt = nowUTC()
			_ = r.persistState()
			_ = r.persistReport(steps, current.ID, serr)
			return serr
		}
	}

	r.state.Status = stateStatusSuccess
	r.state.FinishedAt = nowUTC()
	r.state.UpdatedAt = nowUTC()
	if !r.opts.Doctor {
		if err := r.persistState(); err != nil {
			return err
		}
		if err := r.persistReport(steps, "", nil); err != nil {
			return err
		}
	}

	mode := "onboard"
	if r.opts.Doctor {
		mode = "doctor"
	}
	r.logf("%s completed successfully", mode)
	if len(r.warnings) > 0 {
		for _, warning := range r.warnings {
			r.logf("warning: %s", warning)
		}
	}
	if !r.opts.Doctor {
		r.logf("state:  %s", r.statePath)
		r.logf("report: %s", r.reportPath)
	} else {
		r.logf("doctor mode is non-mutating; no onboarding state/report artifacts were written")
	}
	return nil
}

func (r *Runner) steps() []step {
	return []step{
		{ID: "preflight", Description: "Validate host and workspace prerequisites", Run: r.stepPreflight},
		{ID: "toolchain", Description: "Ensure required toolchain components are installed", Run: r.stepToolchain},
		{ID: "opencode", Description: "Ensure opencode is installed and runnable", Run: r.stepOpenCode},
		{ID: "skills", Description: "Install and verify Codex-compatible system skills", Run: r.stepSkills},
		{ID: "env", Description: "Bootstrap .env from canonical template", Run: r.stepEnv},
		{ID: "deps", Description: "Install project dependencies", Run: r.stepDependencies},
		{ID: "build", Description: "Build control-plane and web UI assets", Run: r.stepBuild},
		{ID: "command", Description: "Expose biometrics-onboard command in user scope", Run: r.stepExposeCommand},
		{ID: "smoke", Description: "Run local control-plane smoke checks", Run: r.stepSmoke},
	}
}

func (r *Runner) runStep(ctx context.Context, current step) error {
	state := r.ensureStepState(current.ID)
	state.Status = stepStatusRunning
	state.Attempts++
	state.StartedAt = nowUTC()
	state.Message = ""
	state.Remediation = nil
	r.state.UpdatedAt = nowUTC()
	if err := r.persistState(); err != nil {
		return err
	}

	r.logf("step=%s started: %s", current.ID, current.Description)
	r.emitStepEvent("onboard.step.started", current.ID, stepStatusRunning, map[string]string{
		"description": current.Description,
		"attempt":     strconv.Itoa(state.Attempts),
	})
	if err := current.Run(ctx); err != nil {
		serr := asStepError(err)
		state.Status = stepStatusFailed
		state.FinishedAt = nowUTC()
		state.Message = serr.Message
		state.Remediation = append([]string{}, serr.Remediation...)
		r.state.UpdatedAt = nowUTC()
		_ = r.persistState()
		r.emitStepEvent("onboard.step.failed", current.ID, stepStatusFailed, map[string]string{
			"error": serr.Message,
		})
		r.logf("step=%s failed: %s", current.ID, serr.Message)
		return serr
	}

	state.Status = stepStatusCompleted
	state.FinishedAt = nowUTC()
	state.Message = "completed"
	state.Remediation = nil
	r.state.UpdatedAt = nowUTC()
	if err := r.persistState(); err != nil {
		return err
	}
	r.emitStepEvent("onboard.step.completed", current.ID, stepStatusCompleted, nil)
	r.logf("step=%s completed", current.ID)
	return nil
}

func (r *Runner) initState() error {
	if r.opts.Doctor {
		r.state = stateFile{
			Version:   stateVersion,
			Workspace: r.workspace,
			Mode:      r.mode(),
			Status:    stateStatusRunning,
			StartedAt: nowUTC(),
			UpdatedAt: nowUTC(),
			Steps:     map[string]*stepState{},
		}
		return nil
	}

	if r.opts.Resume {
		if err := r.loadState(); err != nil {
			return err
		}
		if r.state.Version == 0 {
			r.state = stateFile{
				Version:   stateVersion,
				Workspace: r.workspace,
				Mode:      r.mode(),
				Status:    stateStatusRunning,
				StartedAt: nowUTC(),
				UpdatedAt: nowUTC(),
				Steps:     map[string]*stepState{},
			}
		} else {
			r.state.Status = stateStatusRunning
			r.state.Mode = r.mode()
			r.state.UpdatedAt = nowUTC()
		}
		return r.persistState()
	}

	r.state = stateFile{
		Version:   stateVersion,
		Workspace: r.workspace,
		Mode:      r.mode(),
		Status:    stateStatusRunning,
		StartedAt: nowUTC(),
		UpdatedAt: nowUTC(),
		Steps:     map[string]*stepState{},
	}
	return r.persistState()
}

func (r *Runner) loadState() error {
	raw, err := os.ReadFile(r.statePath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read state file: %w", err)
	}
	var state stateFile
	if err := json.Unmarshal(raw, &state); err != nil {
		return fmt.Errorf("decode state file: %w", err)
	}
	if state.Steps == nil {
		state.Steps = map[string]*stepState{}
	}
	r.state = state
	return nil
}

func (r *Runner) persistState() error {
	if r.opts.Doctor {
		return nil
	}
	r.state.Workspace = r.workspace
	if r.state.Mode == "" {
		r.state.Mode = r.mode()
	}
	if r.state.Steps == nil {
		r.state.Steps = map[string]*stepState{}
	}
	raw, err := json.MarshalIndent(r.state, "", "  ")
	if err != nil {
		return fmt.Errorf("encode state file: %w", err)
	}
	if err := os.WriteFile(r.statePath, raw, 0o644); err != nil {
		return fmt.Errorf("write state file: %w", err)
	}
	return nil
}

func (r *Runner) persistReport(steps []step, failedStepID string, failure *StepError) error {
	if r.opts.Doctor {
		return nil
	}

	report := reportFile{
		Version:    stateVersion,
		Workspace:  r.workspace,
		Mode:       r.mode(),
		Status:     r.state.Status,
		StartedAt:  r.state.StartedAt,
		FinishedAt: nowUTC(),
		FailedStep: failedStepID,
		Warnings:   append([]string{}, r.warnings...),
	}
	if failure != nil {
		report.FailedReason = failure.Message
		report.Remediation = append([]string{}, failure.Remediation...)
	}

	for _, current := range steps {
		state := r.ensureStepState(current.ID)
		report.Steps = append(report.Steps, reportStep{
			ID:          current.ID,
			Description: current.Description,
			Status:      state.Status,
			Message:     state.Message,
			Remediation: append([]string{}, state.Remediation...),
		})
		if state.Status == stepStatusCompleted {
			report.CompletedSteps++
		}
	}

	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("encode report file: %w", err)
	}
	if err := os.WriteFile(r.reportPath, raw, 0o644); err != nil {
		return fmt.Errorf("write report file: %w", err)
	}
	return nil
}

func (r *Runner) ensureStepState(id string) *stepState {
	if r.state.Steps == nil {
		r.state.Steps = map[string]*stepState{}
	}
	if existing, ok := r.state.Steps[id]; ok && existing != nil {
		return existing
	}
	created := &stepState{Status: stepStatusPending}
	r.state.Steps[id] = created
	return created
}

func (r *Runner) stepCompleted(id string) bool {
	state, ok := r.state.Steps[id]
	return ok && state != nil && state.Status == stepStatusCompleted
}

func (r *Runner) mode() string {
	if r.opts.Doctor {
		return "doctor"
	}
	return "onboard"
}

func (r *Runner) stepPreflight(_ context.Context) *StepError {
	if runtime.GOOS != "darwin" && !parseBoolEnvDefaultFalse("BIOMETRICS_ONBOARD_ALLOW_NON_DARWIN") {
		return fail(
			"unsupported OS: onboarding is currently macOS/local-first",
			"Use macOS for V3.1 onboarding or run manual setup on other platforms.",
		)
	}

	if _, err := exec.LookPath("git"); err != nil {
		return fail(
			"git is required but was not found",
			"Install git: brew install git",
		)
	}

	if _, err := net.LookupHost("github.com"); err != nil {
		return fail(
			fmt.Sprintf("network/DNS preflight failed: %v", err),
			"Check internet connectivity and DNS resolution for github.com.",
		)
	}

	if ok, err := checkDiskSpace(r.workspace, 2*1024*1024*1024); err == nil && !ok {
		return fail(
			"insufficient free disk space (<2 GiB)",
			"Free disk space and rerun onboarding.",
		)
	}

	info, err := os.Stat(r.workspace)
	if err != nil {
		return fail(
			fmt.Sprintf("unable to stat workspace: %v", err),
			"Ensure the workspace path exists and is accessible.",
		)
	}
	if info.Mode().Perm()&0o200 == 0 {
		return fail(
			"workspace appears read-only",
			"Ensure the current user has write access to the repository.",
		)
	}
	return nil
}

func (r *Runner) stepToolchain(ctx context.Context) *StepError {
	if err := r.ensureHomebrew(ctx); err != nil {
		return err
	}
	for _, tool := range []struct {
		Name    string
		Formula string
	}{
		{Name: "go", Formula: "go"},
		{Name: "node", Formula: "node"},
		{Name: "pnpm", Formula: "pnpm"},
	} {
		if err := r.ensureTool(ctx, tool.Name, tool.Formula); err != nil {
			return err
		}
	}
	return nil
}

func (r *Runner) stepOpenCode(ctx context.Context) *StepError {
	if err := r.ensureTool(ctx, "opencode", "opencode"); err != nil {
		return err
	}
	if _, err := r.runCommand(ctx, commandSpec{
		Name: "opencode --version",
		Dir:  r.workspace,
		Cmd:  "opencode",
		Args: []string{"--version"},
	}); err != nil {
		return fail(
			fmt.Sprintf("failed to execute opencode after install: %v", err),
			"Reinstall opencode: brew reinstall opencode",
		)
	}
	return nil
}

func (r *Runner) stepSkills(_ context.Context) *StepError {
	sourceRoot := filepath.Join(r.workspace, "third_party", "openai-skills", ".system")
	if !dirExists(sourceRoot) {
		return fail(
			"missing third_party/openai-skills/.system",
			"Ensure repository checkout includes third_party/openai-skills system bundles.",
		)
	}

	required := []string{"skill-creator", "skill-installer"}
	codexHome := strings.TrimSpace(os.Getenv("CODEX_HOME"))
	if codexHome == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fail("unable to detect user home directory", "Set HOME and rerun onboarding.")
		}
		codexHome = filepath.Join(homeDir, ".codex")
	}

	workspaceSkillDir := filepath.Join(r.workspace, ".codex", "skills")
	workspaceAgentsSkillDir := filepath.Join(r.workspace, ".agents", "skills")
	systemDest := filepath.Join(codexHome, "skills", ".system")

	if r.opts.Doctor {
		for _, path := range []string{workspaceSkillDir, workspaceAgentsSkillDir, systemDest} {
			if !dirExists(path) {
				return fail(
					fmt.Sprintf("skills directory missing: %s", path),
					"Run onboarding without --doctor to initialize skills directories.",
				)
			}
		}
		for _, name := range required {
			skillPath := filepath.Join(systemDest, name, "SKILL.md")
			if !fileExists(skillPath) {
				return fail(
					fmt.Sprintf("missing system skill bundle: %s", skillPath),
					"Run onboarding without --doctor to reinstall system skills.",
				)
			}
		}
		return nil
	}

	for _, path := range []string{workspaceSkillDir, workspaceAgentsSkillDir, systemDest} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			return fail(
				fmt.Sprintf("unable to create skills directory %s: %v", path, err),
				"Ensure write access to workspace and CODEX_HOME paths.",
			)
		}
	}

	for _, name := range required {
		source := filepath.Join(sourceRoot, name)
		dest := filepath.Join(systemDest, name)
		if !dirExists(source) {
			return fail(
				fmt.Sprintf("missing source system skill bundle %s", source),
				"Restore third_party/openai-skills and rerun onboarding.",
			)
		}
		if err := os.RemoveAll(dest); err != nil {
			return fail(
				fmt.Sprintf("unable to refresh destination skill bundle %s: %v", dest, err),
				"Remove the destination path manually and rerun onboarding.",
			)
		}
		if err := copyDirRecursive(source, dest); err != nil {
			return fail(
				fmt.Sprintf("failed to install system skill bundle %s: %v", name, err),
				"Verify file permissions and rerun onboarding.",
			)
		}
	}

	return nil
}

func (r *Runner) stepEnv(ctx context.Context) *StepError {
	initScript := filepath.Join(r.workspace, "scripts", "init-env.sh")
	if _, err := os.Stat(initScript); err != nil {
		return fail(
			"missing scripts/init-env.sh",
			"Ensure repository integrity and rerun from repo root.",
		)
	}

	if r.opts.Doctor {
		if _, err := os.Stat(filepath.Join(r.workspace, ".env")); err != nil {
			return fail(
				".env is missing",
				"Run onboarding without --doctor, or run: ./scripts/init-env.sh",
			)
		}
		return nil
	}

	if _, err := r.runCommand(ctx, commandSpec{
		Name: "init-env",
		Dir:  r.workspace,
		Cmd:  initScript,
	}); err != nil {
		return fail(
			fmt.Sprintf("environment bootstrap failed: %v", err),
			"Run: ./scripts/init-env.sh",
		)
	}
	return nil
}

func (r *Runner) stepDependencies(ctx context.Context) *StepError {
	if r.opts.Doctor {
		if _, err := os.Stat(filepath.Join(r.cliDir, "go.mod")); err != nil {
			return fail(
				"missing biometrics-cli/go.mod",
				"Ensure clone is complete and rerun onboarding.",
			)
		}
		if _, err := os.Stat(filepath.Join(r.webDir, "package.json")); err != nil {
			return fail(
				"missing biometrics-cli/web-v3/package.json",
				"Ensure clone is complete and rerun onboarding.",
			)
		}
		return nil
	}

	if _, err := r.runCommand(ctx, commandSpec{
		Name: "go mod download",
		Dir:  r.cliDir,
		Cmd:  "go",
		Args: []string{"mod", "download"},
	}); err != nil {
		return fail(
			fmt.Sprintf("go module download failed: %v", err),
			"Run: cd biometrics-cli && go mod download",
		)
	}

	if _, err := r.runCommand(ctx, commandSpec{
		Name: "npm ci",
		Dir:  r.webDir,
		Cmd:  "npm",
		Args: []string{"ci"},
	}); err != nil {
		return fail(
			fmt.Sprintf("web dependency install failed: %v", err),
			"Run: cd biometrics-cli/web-v3 && npm ci",
		)
	}
	return nil
}

func (r *Runner) stepBuild(ctx context.Context) *StepError {
	controlplaneBinary := filepath.Join(r.rootBin, "biometrics-cli")
	onboardBinary := filepath.Join(r.rootBin, "biometrics-onboard")
	skillsBinary := filepath.Join(r.rootBin, "biometrics-skills")
	webDistIndex := filepath.Join(r.webDir, "dist", "index.html")

	if r.opts.Doctor {
		missing := make([]string, 0, 4)
		if _, err := os.Stat(controlplaneBinary); err != nil {
			missing = append(missing, controlplaneBinary)
		}
		if _, err := os.Stat(onboardBinary); err != nil {
			missing = append(missing, onboardBinary)
		}
		if _, err := os.Stat(skillsBinary); err != nil {
			missing = append(missing, skillsBinary)
		}
		if _, err := os.Stat(webDistIndex); err != nil {
			missing = append(missing, webDistIndex)
		}
		if len(missing) > 0 {
			return fail(
				"build artifacts missing in doctor mode",
				append([]string{"Run onboarding without --doctor to build missing artifacts."}, missing...)...,
			)
		}
		return nil
	}

	if err := os.MkdirAll(r.rootBin, 0o755); err != nil {
		return fail(
			fmt.Sprintf("unable to create bin directory: %v", err),
			"Ensure write permission to repository root.",
		)
	}

	if _, err := r.runCommand(ctx, commandSpec{
		Name: "build controlplane",
		Dir:  r.cliDir,
		Cmd:  "go",
		Args: []string{"build", "-o", controlplaneBinary, "./cmd/controlplane"},
	}); err != nil {
		return fail(
			fmt.Sprintf("controlplane build failed: %v", err),
			"Run: cd biometrics-cli && go build -o ../bin/biometrics-cli ./cmd/controlplane",
		)
	}

	if _, err := r.runCommand(ctx, commandSpec{
		Name: "build onboard binary",
		Dir:  r.cliDir,
		Cmd:  "go",
		Args: []string{"build", "-o", onboardBinary, "./cmd/onboard"},
	}); err != nil {
		return fail(
			fmt.Sprintf("onboard build failed: %v", err),
			"Run: cd biometrics-cli && go build -o ../bin/biometrics-onboard ./cmd/onboard",
		)
	}

	if _, err := r.runCommand(ctx, commandSpec{
		Name: "build skills binary",
		Dir:  r.cliDir,
		Cmd:  "go",
		Args: []string{"build", "-o", skillsBinary, "./cmd/skills"},
	}); err != nil {
		return fail(
			fmt.Sprintf("skills cli build failed: %v", err),
			"Run: cd biometrics-cli && go build -o ../bin/biometrics-skills ./cmd/skills",
		)
	}

	if _, err := r.runCommand(ctx, commandSpec{
		Name: "build web-v3",
		Dir:  r.webDir,
		Cmd:  "npm",
		Args: []string{"run", "build"},
	}); err != nil {
		return fail(
			fmt.Sprintf("web-v3 build failed: %v", err),
			"Run: cd biometrics-cli/web-v3 && npm run build",
		)
	}
	return nil
}

func (r *Runner) stepExposeCommand(_ context.Context) *StepError {
	onboardBinary := filepath.Join(r.rootBin, "biometrics-onboard")
	if _, err := os.Stat(onboardBinary); err != nil {
		return fail(
			"missing onboard binary for command exposure",
			"Run onboarding build step first, or build manually: cd biometrics-cli && go build -o ../bin/biometrics-onboard ./cmd/onboard",
		)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fail("unable to detect user home directory", "Set HOME and rerun onboarding.")
	}
	userBin := filepath.Join(homeDir, ".local", "bin")
	linkPath := filepath.Join(userBin, "biometrics-onboard")

	if r.opts.Doctor {
		info, err := os.Lstat(linkPath)
		if err != nil {
			r.addWarning(
				fmt.Sprintf("biometrics-onboard command link is missing at %s", linkPath),
				fmt.Sprintf("Create symlink manually: ln -sfn %s %s", onboardBinary, linkPath),
			)
		} else if info.Mode()&os.ModeSymlink == 0 {
			r.addWarning(
				fmt.Sprintf("biometrics-onboard path exists but is not a symlink: %s", linkPath),
				fmt.Sprintf("Replace with symlink: ln -sfn %s %s", onboardBinary, linkPath),
			)
		}
	} else {
		if err := os.MkdirAll(userBin, 0o755); err != nil {
			return fail(fmt.Sprintf("unable to create %s: %v", userBin, err), "Ensure user home is writable.")
		}
		if err := ensureSymlink(linkPath, onboardBinary); err != nil {
			return fail(
				fmt.Sprintf("unable to expose biometrics-onboard command: %v", err),
				fmt.Sprintf("Create symlink manually: ln -sfn %s %s", onboardBinary, linkPath),
			)
		}
	}

	pathEntries := strings.Split(os.Getenv("PATH"), string(os.PathListSeparator))
	inPath := false
	for _, entry := range pathEntries {
		if strings.TrimSpace(entry) == userBin {
			inPath = true
			break
		}
	}
	if !inPath {
		r.addWarning(
			fmt.Sprintf("%s is not in PATH", userBin),
			fmt.Sprintf("Add to shell profile: export PATH=\"%s:$PATH\"", userBin),
			"Reload shell after updating PATH.",
		)
	}

	return nil
}

func (r *Runner) stepSmoke(ctx context.Context) *StepError {
	if r.opts.Doctor {
		return nil
	}

	binary := filepath.Join(r.rootBin, "biometrics-cli")
	if _, err := os.Stat(binary); err != nil {
		return fail("control-plane binary missing for smoke test", "Run build step before smoke.")
	}

	port, err := freePort()
	if err != nil {
		return fail(
			fmt.Sprintf("unable to reserve smoke-test port: %v", err),
			"Ensure localhost networking is available.",
		)
	}

	cmd := exec.CommandContext(ctx, binary)
	cmd.Dir = r.workspace
	cmd.Env = append(os.Environ(),
		"PORT="+strconv.Itoa(port),
		"BIOMETRICS_BIND_ADDR=127.0.0.1",
		"BIOMETRICS_WORKSPACE="+r.workspace,
	)

	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output

	if err := cmd.Start(); err != nil {
		return fail(
			fmt.Sprintf("failed to start control-plane smoke instance: %v", err),
			"Inspect binary permissions and rerun onboarding.",
		)
	}
	defer func() {
		_ = cmd.Process.Signal(os.Interrupt)
		_, _ = waitCommand(cmd, 5*time.Second)
	}()

	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	if err := waitForHTTP(baseURL+"/health", 45*time.Second); err != nil {
		return fail(
			fmt.Sprintf("health endpoint did not become ready: %v", err),
			"Review smoke logs and ensure no conflicting process uses the selected port.",
			output.String(),
		)
	}

	if err := assertReady(baseURL + "/health/ready"); err != nil {
		return fail(
			fmt.Sprintf("readiness check failed: %v", err),
			"Review /health/ready payload and control-plane logs.",
		)
	}
	if err := assertJSONList(baseURL + "/api/v1/projects"); err != nil {
		return fail(
			fmt.Sprintf("projects endpoint check failed: %v", err),
			"Verify runtime store initialization and API routing.",
		)
	}
	if err := assertHTML(baseURL + "/"); err != nil {
		return fail(
			fmt.Sprintf("web UI root check failed: %v", err),
			"Ensure web-v3 dist build exists (npm run build).",
		)
	}

	return nil
}

func (r *Runner) ensureHomebrew(ctx context.Context) *StepError {
	if _, err := r.commandPath("brew"); err == nil {
		return nil
	}
	if r.opts.Doctor {
		return fail(
			"homebrew not found",
			"Install Homebrew: /bin/bash -c \"$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)\"",
		)
	}

	if err := r.requirePrivilege("Install Homebrew (required for automatic dependency installation)?"); err != nil {
		return fail(err.Error(), "Rerun with --yes to allow automatic installation.")
	}

	installCmd := `/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"`
	if _, err := r.runCommand(ctx, commandSpec{
		Name:       "install homebrew",
		Dir:        r.workspace,
		Cmd:        "bash",
		Args:       []string{"-lc", installCmd},
		Privileged: true,
	}); err != nil {
		return fail(
			fmt.Sprintf("homebrew install failed: %v", err),
			"Install Homebrew manually and rerun onboarding.",
			"/bin/bash -c \"$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)\"",
		)
	}

	if _, err := r.commandPath("brew"); err != nil {
		return fail(
			"homebrew still unavailable after installation attempt",
			"Ensure /opt/homebrew/bin or /usr/local/bin is in PATH.",
		)
	}
	return nil
}

func (r *Runner) ensureTool(ctx context.Context, name, formula string) *StepError {
	if _, err := r.commandPath(name); err == nil {
		return nil
	}

	if r.opts.Doctor {
		return fail(
			fmt.Sprintf("%s is not installed", name),
			fmt.Sprintf("Install: brew install %s", formula),
		)
	}

	if err := r.requirePrivilege(fmt.Sprintf("Install missing tool '%s' now?", name)); err != nil {
		return fail(err.Error(), "Rerun with --yes to allow automatic installation.")
	}

	if _, err := r.runCommand(ctx, commandSpec{
		Name:       "install " + name,
		Dir:        r.workspace,
		Cmd:        "brew",
		Args:       []string{"install", formula},
		Privileged: true,
	}); err != nil {
		return fail(
			fmt.Sprintf("failed to install %s: %v", name, err),
			fmt.Sprintf("Install manually: brew install %s", formula),
		)
	}
	if _, err := r.commandPath(name); err != nil {
		return fail(
			fmt.Sprintf("%s still unavailable after installation", name),
			fmt.Sprintf("Verify PATH and installation for %s.", name),
		)
	}
	return nil
}

func (r *Runner) commandPath(name string) (string, error) {
	if path, err := exec.LookPath(name); err == nil {
		return path, nil
	}

	// Ensure common Homebrew paths are available in this process.
	candidates := []string{"/opt/homebrew/bin", "/usr/local/bin"}
	pathEnv := os.Getenv("PATH")
	parts := strings.Split(pathEnv, string(os.PathListSeparator))
	existing := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		existing[part] = struct{}{}
	}
	updated := false
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err != nil {
			continue
		}
		if _, ok := existing[candidate]; ok {
			continue
		}
		parts = append([]string{candidate}, parts...)
		updated = true
	}
	if updated {
		_ = os.Setenv("PATH", strings.Join(parts, string(os.PathListSeparator)))
	}

	return exec.LookPath(name)
}

func (r *Runner) requirePrivilege(question string) error {
	if r.opts.Yes {
		return nil
	}
	if r.opts.NonInteractive {
		return fmt.Errorf("privileged action denied in non-interactive mode: %s", question)
	}

	fmt.Fprintf(r.out, "[onboard] %s [y/N]: ", question)
	reader := bufio.NewReader(os.Stdin)
	reply, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("unable to read confirmation: %w", err)
	}
	reply = strings.TrimSpace(strings.ToLower(reply))
	if reply == "y" || reply == "yes" {
		return nil
	}
	return fmt.Errorf("privileged action declined")
}

type commandSpec struct {
	Name       string
	Dir        string
	Cmd        string
	Args       []string
	Privileged bool
	Env        []string
}

func (r *Runner) runCommand(ctx context.Context, spec commandSpec) (string, error) {
	if spec.Cmd == "" {
		return "", fmt.Errorf("missing command for %s", spec.Name)
	}

	cmd := exec.CommandContext(ctx, spec.Cmd, spec.Args...)
	cmd.Dir = spec.Dir
	cmd.Env = append(os.Environ(), spec.Env...)

	var output bytes.Buffer
	multi := io.MultiWriter(r.out, &output)
	cmd.Stdout = multi
	cmd.Stderr = multi

	start := time.Now()
	if err := cmd.Run(); err != nil {
		return output.String(), fmt.Errorf("%s failed after %s: %w", spec.Name, time.Since(start).Round(time.Millisecond), err)
	}
	return output.String(), nil
}

func waitCommand(cmd *exec.Cmd, timeout time.Duration) (bool, error) {
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()
	select {
	case err := <-done:
		return true, err
	case <-time.After(timeout):
		return false, nil
	}
}

func waitForHTTP(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 3 * time.Second}
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(750 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for %s", url)
}

func assertReady(url string) error {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	var payload readinessProbe
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return err
	}
	if !payload.Ready {
		return fmt.Errorf("readiness=false")
	}
	return nil
}

func assertJSONList(url string) error {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	var payload []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return err
	}
	return nil
}

func assertHTML(url string) error {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8192))
	if err != nil {
		return err
	}
	content := strings.ToLower(string(body))
	if !strings.Contains(content, "<html") {
		return fmt.Errorf("response does not look like HTML")
	}
	return nil
}

func ensureSymlink(linkPath, target string) error {
	info, err := os.Lstat(linkPath)
	if err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			currentTarget, readErr := os.Readlink(linkPath)
			if readErr == nil {
				if absCurrent, _ := filepath.Abs(currentTarget); absCurrent == target {
					return nil
				}
			}
		}
		if removeErr := os.Remove(linkPath); removeErr != nil {
			return removeErr
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	return os.Symlink(target, linkPath)
}

func copyDirRecursive(source, destination string) error {
	info, err := os.Stat(source)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", source)
	}
	if err := os.MkdirAll(destination, info.Mode().Perm()); err != nil {
		return err
	}

	entries, err := os.ReadDir(source)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		src := filepath.Join(source, entry.Name())
		dst := filepath.Join(destination, entry.Name())

		if entry.IsDir() {
			if err := copyDirRecursive(src, dst); err != nil {
				return err
			}
			continue
		}

		entryInfo, err := entry.Info()
		if err != nil {
			return err
		}
		data, err := os.ReadFile(src)
		if err != nil {
			return err
		}
		if err := os.WriteFile(dst, data, entryInfo.Mode().Perm()); err != nil {
			return err
		}
	}
	return nil
}

func detectWorkspace(in string) (string, string, error) {
	if strings.TrimSpace(in) == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", "", err
		}
		in = cwd
	}
	start, err := filepath.Abs(in)
	if err != nil {
		return "", "", err
	}

	candidates := []string{start}
	parent := start
	for i := 0; i < 5; i++ {
		next := filepath.Dir(parent)
		if next == parent {
			break
		}
		candidates = append(candidates, next)
		parent = next
	}

	for _, candidate := range candidates {
		cliDir := filepath.Join(candidate, "biometrics-cli")
		if fileExists(filepath.Join(cliDir, "go.mod")) {
			return candidate, cliDir, nil
		}
		if fileExists(filepath.Join(candidate, "go.mod")) && dirExists(filepath.Join(candidate, "cmd")) {
			if filepath.Base(candidate) == "biometrics-cli" {
				return filepath.Dir(candidate), candidate, nil
			}
		}
	}

	return "", "", fmt.Errorf("unable to detect BIOMETRICS workspace from %s", start)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func freePort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		return 0, fmt.Errorf("unexpected listener addr type")
	}
	return addr.Port, nil
}

func fail(message string, remediation ...string) *StepError {
	cleaned := make([]string, 0, len(remediation))
	for _, item := range remediation {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		cleaned = append(cleaned, trimmed)
	}
	return &StepError{
		Message:     strings.TrimSpace(message),
		Remediation: cleaned,
	}
}

func asStepError(err error) *StepError {
	if err == nil {
		return nil
	}
	var serr *StepError
	if errors.As(err, &serr) && serr != nil {
		return serr
	}
	return fail(err.Error())
}

func (r *Runner) logf(format string, args ...interface{}) {
	fmt.Fprintf(r.out, "[onboard] %s\n", fmt.Sprintf(format, args...))
}

func (r *Runner) emitStepEvent(eventType, stepID, status string, payload map[string]string) {
	if r.opts.Doctor {
		return
	}
	event := onboardEvent{
		Type:      eventType,
		StepID:    stepID,
		Mode:      r.mode(),
		Status:    status,
		Payload:   payload,
		CreatedAt: nowUTC(),
	}
	raw, err := json.Marshal(event)
	if err != nil {
		r.logf("event-encode error: %v", err)
		return
	}
	f, err := os.OpenFile(r.eventsPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		r.logf("event-write error: %v", err)
		return
	}
	if _, err := f.Write(append(raw, '\n')); err != nil {
		r.logf("event-write error: %v", err)
	}
	if err := f.Close(); err != nil {
		r.logf("event-close error: %v", err)
	}
}

func (r *Runner) addWarning(message string, remediation ...string) {
	trimmedMessage := strings.TrimSpace(message)
	if trimmedMessage == "" {
		return
	}

	parts := []string{trimmedMessage}
	for _, item := range remediation {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		parts = append(parts, trimmed)
	}
	combined := strings.Join(parts, " | ")

	for _, existing := range r.warnings {
		if existing == combined {
			return
		}
	}
	r.warnings = append(r.warnings, combined)
}

func nowUTC() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// ReadLastStatus reads the latest onboarding report status from workspace.
func ReadLastStatus(workspace string) string {
	reportPath := filepath.Join(workspace, ".biometrics", "onboard", "report.json")
	raw, err := os.ReadFile(reportPath)
	if err != nil {
		return ""
	}
	var report onboardingReportStatus
	if err := json.Unmarshal(raw, &report); err != nil {
		return ""
	}
	return strings.TrimSpace(report.Status)
}

// ParseBoolEnvDefaultTrue returns false only for explicit false-like values.
func ParseBoolEnvDefaultTrue(key string) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	switch value {
	case "", "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return true
	}
}

func parseBoolEnvDefaultFalse(key string) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	switch value {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// SortRemediation returns a deterministic remediation list.
func SortRemediation(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	cloned := append([]string{}, values...)
	sort.Strings(cloned)
	return cloned
}
