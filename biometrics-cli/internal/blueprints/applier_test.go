package blueprints

import (
	"path/filepath"
	"testing"
)

func TestApplierIsIdempotent(t *testing.T) {
	workspace := t.TempDir()
	writeValidCatalogFixture(t, workspace)

	projectPath := filepath.Join(workspace, "sample")
	mustWrite(t, filepath.Join(projectPath, "go.mod"), "module sample\n")

	registry, err := NewRegistry(workspace, "")
	if err != nil {
		t.Fatalf("new registry: %v", err)
	}
	applier := NewApplier(workspace, registry)

	first, err := applier.Apply(ApplyOptions{ProjectID: "sample", ProfileID: "universal-2026", ModuleIDs: []string{"engine"}})
	if err != nil {
		t.Fatalf("first apply: %v", err)
	}
	if len(first.AppliedModules) != 1 || first.AppliedModules[0] != "engine" {
		t.Fatalf("unexpected first applied modules: %+v", first.AppliedModules)
	}

	second, err := applier.Apply(ApplyOptions{ProjectID: "sample", ProfileID: "universal-2026", ModuleIDs: []string{"engine"}})
	if err != nil {
		t.Fatalf("second apply: %v", err)
	}
	if len(second.AppliedModules) != 0 {
		t.Fatalf("expected no newly applied modules, got %+v", second.AppliedModules)
	}
	if len(second.SkippedModules) != 1 || second.SkippedModules[0] != "engine" {
		t.Fatalf("expected engine to be skipped, got %+v", second.SkippedModules)
	}
}
