package skillkit

import (
	"os"
	"path/filepath"
	"testing"
)

func TestManagerSelectExplicitAndAuto(t *testing.T) {
	workspace := t.TempDir()
	writeSkillFixture(t, filepath.Join(workspace, ".codex", "skills", "release-ops", "SKILL.md"), `---
name: release-ops
description: Release gate validation and soak evidence workflow for GA closures.
---

# release-ops
`)
	writeSkillFixture(t, filepath.Join(workspace, ".codex", "skills", "doc-writer", "SKILL.md"), `---
name: doc-writer
description: Structured documentation writing and migration guide updates.
---

# doc-writer
`)

	manager, err := NewManager(ManagerOptions{Workspace: workspace, CWD: workspace})
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}

	explicit, err := manager.Select("run release checks", []string{"release-ops"}, SelectionModeExplicit)
	if err != nil {
		t.Fatalf("explicit select: %v", err)
	}
	if len(explicit.Selected) != 1 || explicit.Selected[0].Name != "release-ops" {
		t.Fatalf("unexpected explicit selection: %#v", explicit.Selected)
	}

	auto, err := manager.Select("Use $release-ops and prepare gate soak evidence", nil, SelectionModeAuto)
	if err != nil {
		t.Fatalf("auto select: %v", err)
	}
	if len(auto.Selected) == 0 {
		t.Fatalf("expected auto selection to include release-ops")
	}
}

func TestManagerSelectExplicitIgnoresGoalMentions(t *testing.T) {
	workspace := t.TempDir()
	writeSkillFixture(t, filepath.Join(workspace, ".codex", "skills", "release-ops", "SKILL.md"), `---
name: release-ops
description: Release gate validation and soak evidence workflow for GA closures.
---
`)
	writeSkillFixture(t, filepath.Join(workspace, ".codex", "skills", "doc-writer", "SKILL.md"), `---
name: doc-writer
description: Structured documentation writing and migration guide updates.
---
`)

	manager, err := NewManager(ManagerOptions{Workspace: workspace, CWD: workspace})
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}

	explicit, err := manager.Select("Use $doc-writer for this task", []string{"release-ops"}, SelectionModeExplicit)
	if err != nil {
		t.Fatalf("explicit select: %v", err)
	}
	if len(explicit.Selected) != 1 {
		t.Fatalf("unexpected explicit selection length: %#v", explicit.Selected)
	}
	if explicit.Selected[0].Name != "release-ops" {
		t.Fatalf("expected explicit selection to keep only requested skill, got: %#v", explicit.Selected)
	}
}

func TestEnableDisableWritesCodexConfig(t *testing.T) {
	workspace := t.TempDir()
	skillPath := filepath.Join(workspace, ".codex", "skills", "release-ops", "SKILL.md")
	writeSkillFixture(t, skillPath, `---
name: release-ops
description: Release workflow skill.
---
`)

	manager, err := NewManager(ManagerOptions{Workspace: workspace, CWD: workspace})
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}

	if _, err := manager.Disable("release-ops"); err != nil {
		t.Fatalf("disable skill: %v", err)
	}
	skill, err := manager.Get("release-ops")
	if err != nil {
		t.Fatalf("get disabled skill: %v", err)
	}
	if skill.Enabled {
		t.Fatalf("expected skill to be disabled")
	}

	if _, err := manager.Enable("release-ops"); err != nil {
		t.Fatalf("enable skill: %v", err)
	}
	skill, err = manager.Get("release-ops")
	if err != nil {
		t.Fatalf("get enabled skill: %v", err)
	}
	if !skill.Enabled {
		t.Fatalf("expected skill to be enabled")
	}

	if _, err := os.Stat(filepath.Join(workspace, ".codex", "config.toml")); err != nil {
		t.Fatalf("expected config.toml to exist: %v", err)
	}
}

func writeSkillFixture(t *testing.T, path string, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
