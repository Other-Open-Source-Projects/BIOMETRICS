package policy

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestValidatePathAllowsInsideRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "workspace")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("mkdir root: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "project"), 0o755); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}

	engine := Default()
	validated, err := engine.ValidatePath(root, "project")
	if err != nil {
		t.Fatalf("validate path: %v", err)
	}
	if validated != filepath.Join(root, "project") {
		t.Fatalf("unexpected validated path: %s", validated)
	}
}

func TestValidatePathBlocksPrefixBypass(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "root")
	sibling := filepath.Join(base, "root2")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("mkdir root: %v", err)
	}
	if err := os.MkdirAll(sibling, 0o755); err != nil {
		t.Fatalf("mkdir sibling: %v", err)
	}

	engine := Default()
	if _, err := engine.ValidatePath(root, "../root2/secret.txt"); err == nil {
		t.Fatal("expected traversal block for prefix-bypass path")
	}
}

func TestValidatePathBlocksSymlinkEscape(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink behavior differs on windows CI")
	}

	base := t.TempDir()
	root := filepath.Join(base, "workspace")
	outside := filepath.Join(base, "outside")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("mkdir root: %v", err)
	}
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatalf("mkdir outside: %v", err)
	}
	outsideFile := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(outsideFile, []byte("secret"), 0o644); err != nil {
		t.Fatalf("write outside file: %v", err)
	}

	linkPath := filepath.Join(root, "escape-link")
	if err := os.Symlink(outsideFile, linkPath); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	engine := Default()
	if _, err := engine.ValidatePath(root, "escape-link"); err == nil {
		t.Fatal("expected symlink escape to be blocked")
	}
}

func TestValidatePathAllowsSymlinkInsideRoot(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink behavior differs on windows CI")
	}

	base := t.TempDir()
	root := filepath.Join(base, "workspace")
	targetDir := filepath.Join(root, "target")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatalf("mkdir target: %v", err)
	}
	targetFile := filepath.Join(targetDir, "file.txt")
	if err := os.WriteFile(targetFile, []byte("ok"), 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}

	linkPath := filepath.Join(root, "internal-link")
	if err := os.Symlink(targetFile, linkPath); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	engine := Default()
	validated, err := engine.ValidatePath(root, "internal-link")
	if err != nil {
		t.Fatalf("expected symlink inside root to be allowed, got error: %v", err)
	}
	if validated != linkPath {
		t.Fatalf("unexpected validated path: %s", validated)
	}
}

func TestRedactMasksAssignmentAndTokenPatterns(t *testing.T) {
	engine := Default()
	input := "api_key=abc123 token: xyz456 secret = qwerty id_token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.aaaaaaaaaaaaaaaaaaaa.bbbbbbbbbbbbbbbbbbbb authorization=Bearer super-secret-token cookie=session=topsecret nvapi-exampleexampleexample12 glpat-exampleexampleexample12 ghp_exampleexampleexample12"
	redacted := engine.Redact(input)

	if redacted == input {
		t.Fatal("expected redaction to modify input")
	}
	if containsAny(
		redacted,
		"abc123",
		"xyz456",
		"qwerty",
		"super-secret-token",
		"topsecret",
		"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.aaaaaaaaaaaaaaaaaaaa.bbbbbbbbbbbbbbbbbbbb",
		"nvapi-exampleexampleexample12",
		"glpat-exampleexampleexample12",
		"ghp_exampleexampleexample12",
	) {
		t.Fatalf("redaction leaked sensitive value: %s", redacted)
	}
}

func containsAny(text string, needles ...string) bool {
	for _, needle := range needles {
		if needle != "" && strings.Contains(text, needle) {
			return true
		}
	}
	return false
}
