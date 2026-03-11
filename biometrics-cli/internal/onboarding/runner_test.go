package onboarding

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDoctorRunStepDoesNotWriteArtifacts(t *testing.T) {
	workspace := testWorkspace(t)
	runner, err := NewRunner(Options{Workspace: workspace, Doctor: true, Out: io.Discard})
	if err != nil {
		t.Fatalf("new runner: %v", err)
	}

	if err := runner.initState(); err != nil {
		t.Fatalf("init state: %v", err)
	}

	err = runner.runStep(context.Background(), step{
		ID:          "doctor-noop",
		Description: "doctor noop",
		Run: func(context.Context) *StepError {
			return nil
		},
	})
	if err != nil {
		t.Fatalf("run step: %v", err)
	}

	for _, path := range []string{runner.statePath, runner.reportPath, runner.eventsPath} {
		if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
			t.Fatalf("expected no artifact write in doctor mode for %s", path)
		}
	}
}

func TestStepExposeCommandPathMissingCreatesWarning(t *testing.T) {
	workspace := testWorkspace(t)
	binaryPath := filepath.Join(workspace, "bin", "biometrics-onboard")
	if err := os.WriteFile(binaryPath, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write onboard binary: %v", err)
	}

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", "/usr/bin:/bin")

	runner, err := NewRunner(Options{Workspace: workspace, Out: io.Discard})
	if err != nil {
		t.Fatalf("new runner: %v", err)
	}

	if err := runner.stepExposeCommand(context.Background()); err != nil {
		t.Fatalf("stepExposeCommand returned unexpected error: %v", err)
	}

	if len(runner.warnings) == 0 {
		t.Fatalf("expected PATH warning when ~/.local/bin is missing from PATH")
	}
	if !strings.Contains(runner.warnings[0], "not in PATH") {
		t.Fatalf("expected PATH warning, got %q", runner.warnings[0])
	}

	linkPath := filepath.Join(home, ".local", "bin", "biometrics-onboard")
	if _, err := os.Lstat(linkPath); err != nil {
		t.Fatalf("expected symlink to be created, got: %v", err)
	}
}

func TestResumeInitStateRetainsCompletedStep(t *testing.T) {
	workspace := testWorkspace(t)

	initial, err := NewRunner(Options{Workspace: workspace, Out: io.Discard})
	if err != nil {
		t.Fatalf("new runner: %v", err)
	}
	if err := initial.initState(); err != nil {
		t.Fatalf("init state: %v", err)
	}
	state := initial.ensureStepState("preflight")
	state.Status = stepStatusCompleted
	state.Attempts = 1
	if err := initial.persistState(); err != nil {
		t.Fatalf("persist state: %v", err)
	}

	resumed, err := NewRunner(Options{Workspace: workspace, Resume: true, Out: io.Discard})
	if err != nil {
		t.Fatalf("new resumed runner: %v", err)
	}
	if err := resumed.initState(); err != nil {
		t.Fatalf("resume init state: %v", err)
	}
	if !resumed.stepCompleted("preflight") {
		t.Fatalf("expected resumed state to keep completed preflight step")
	}
}

func TestPersistReportIncludesWarnings(t *testing.T) {
	workspace := testWorkspace(t)
	runner, err := NewRunner(Options{Workspace: workspace, Out: io.Discard})
	if err != nil {
		t.Fatalf("new runner: %v", err)
	}
	if err := runner.initState(); err != nil {
		t.Fatalf("init state: %v", err)
	}

	runner.addWarning("path missing", "export PATH=\"$HOME/.local/bin:$PATH\"")
	runner.ensureStepState("noop").Status = stepStatusCompleted

	steps := []step{{ID: "noop", Description: "noop"}}
	if err := runner.persistReport(steps, "", nil); err != nil {
		t.Fatalf("persist report: %v", err)
	}

	raw, err := os.ReadFile(runner.reportPath)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	var report struct {
		Warnings []string `json:"warnings"`
	}
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("decode report: %v", err)
	}
	if len(report.Warnings) == 0 {
		t.Fatalf("expected warnings in report")
	}
}

func TestStepSkillsInstallsSystemBundles(t *testing.T) {
	workspace := testWorkspace(t)
	home := t.TempDir()
	t.Setenv("HOME", home)

	runner, err := NewRunner(Options{Workspace: workspace, Out: io.Discard})
	if err != nil {
		t.Fatalf("new runner: %v", err)
	}

	if err := runner.stepSkills(context.Background()); err != nil {
		t.Fatalf("stepSkills failed: %v", err)
	}

	for _, path := range []string{
		filepath.Join(workspace, ".codex", "skills"),
		filepath.Join(workspace, ".agents", "skills"),
		filepath.Join(home, ".codex", "skills", ".system", "skill-creator", "SKILL.md"),
		filepath.Join(home, ".codex", "skills", ".system", "skill-installer", "SKILL.md"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s to exist: %v", path, err)
		}
	}
}

func testWorkspace(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	for _, path := range []string{
		filepath.Join(root, "biometrics-cli", "web-v3"),
		filepath.Join(root, "scripts"),
		filepath.Join(root, "bin"),
		filepath.Join(root, "third_party", "openai-skills", ".system", "skill-creator"),
		filepath.Join(root, "third_party", "openai-skills", ".system", "skill-installer"),
	} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", path, err)
		}
	}

	files := map[string]string{
		filepath.Join(root, "biometrics-cli", "go.mod"):                 "module biometrics-cli\n\ngo 1.24\n",
		filepath.Join(root, "biometrics-cli", "web-v3", "package.json"): "{\"name\":\"test\"}\n",
		filepath.Join(root, "scripts", "init-env.sh"):                   "#!/usr/bin/env bash\nexit 0\n",
		filepath.Join(root, ".env"):                                     "DUMMY=1\n",
		filepath.Join(root, "third_party", "openai-skills", ".system", "skill-creator", "SKILL.md"):  "---\nname: skill-creator\ndescription: create skills.\n---\n",
		filepath.Join(root, "third_party", "openai-skills", ".system", "skill-installer", "SKILL.md"): "---\nname: skill-installer\ndescription: install skills.\n---\n",
	}
	for path, contents := range files {
		if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	if err := os.Chmod(filepath.Join(root, "scripts", "init-env.sh"), 0o755); err != nil {
		t.Fatalf("chmod init-env: %v", err)
	}

	return root
}
