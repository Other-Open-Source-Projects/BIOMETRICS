package skillkit

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestLoaderBlocksSymlinkEscapeOutsideRoot(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink permissions are not guaranteed on windows test environments")
	}

	workspace := t.TempDir()
	outside := t.TempDir()

	writeSkillFixture(t, filepath.Join(workspace, ".codex", "skills", "safe-skill", "SKILL.md"), `---
name: safe-skill
description: Skill inside repository root.
---
`)
	writeSkillFixture(t, filepath.Join(outside, "outside-skill", "SKILL.md"), `---
name: outside-skill
description: Skill outside repository root.
---
`)

	root := filepath.Join(workspace, ".codex", "skills")
	if err := os.Symlink(filepath.Join(outside, "outside-skill"), filepath.Join(root, "escape")); err != nil {
		t.Fatalf("create symlink: %v", err)
	}

	manager, err := NewManager(ManagerOptions{Workspace: workspace, CWD: workspace})
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}

	skills, err := manager.List()
	if err != nil {
		t.Fatalf("list skills: %v", err)
	}
	seenSafe := false
	for _, skill := range skills {
		if skill.Name == "safe-skill" {
			seenSafe = true
		}
		if skill.Name == "outside-skill" {
			t.Fatalf("expected symlink-escaped skill to be blocked, got %+v", skill)
		}
	}
	if !seenSafe {
		t.Fatalf("expected safe skill to be loaded")
	}

	outcome, err := manager.Reload()
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	foundEscapeError := false
	for _, item := range outcome.Errors {
		if strings.Contains(item.Message, "escapes skill root") {
			foundEscapeError = true
			break
		}
	}
	if !foundEscapeError {
		t.Fatalf("expected loader error for symlink escape, got: %#v", outcome.Errors)
	}
}
