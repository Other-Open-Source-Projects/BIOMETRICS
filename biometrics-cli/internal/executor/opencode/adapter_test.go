package opencode

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestExecuteRejectsEmptyPrompt(t *testing.T) {
	adapter := NewAdapter(
		WithBinary("definitely-missing-opencode-binary"),
		WithAutoInstall(false),
	)

	_, err := adapter.Execute(context.Background(), "run-1", "coder", "", "project-1")
	if err == nil {
		t.Fatal("expected error for empty prompt")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "empty prompt") {
		t.Fatalf("expected empty prompt error, got %v", err)
	}
}

func TestExecuteHardFailsWhenBinaryMissingAndAutoInstallDisabled(t *testing.T) {
	adapter := NewAdapter(
		WithBinary("definitely-missing-opencode-binary"),
		WithAutoInstall(false),
	)

	_, err := adapter.Execute(context.Background(), "run-1", "coder", "hello", "project-1")
	if err == nil {
		t.Fatal("expected hard failure when opencode binary is missing")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "not found") {
		t.Fatalf("expected binary-not-found message, got %v", err)
	}
}

func TestExecuteAutoInstallFailureEmitsFailedEvent(t *testing.T) {
	emitted := make([]string, 0, 2)
	adapter := NewAdapter(
		WithBinary("definitely-missing-opencode-binary"),
		WithAutoInstall(true),
		WithInstaller("definitely-missing-installer-command", "install", "opencode"),
		WithInstallEventHook(func(_ string, eventType string, _ map[string]string) {
			emitted = append(emitted, eventType)
		}),
	)

	_, err := adapter.Execute(context.Background(), "run-1", "coder", "hello", "project-1")
	if err == nil {
		t.Fatal("expected hard failure when installer command is unavailable")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "installer command") {
		t.Fatalf("expected installer-command failure detail, got %v", err)
	}
	if len(emitted) == 0 || emitted[len(emitted)-1] != "opencode.install.failed" {
		t.Fatalf("expected opencode.install.failed event, got %#v", emitted)
	}
}

func TestExtractRoutingMetadata(t *testing.T) {
	clean, meta := extractRoutingMetadata("[provider=nim model=nvidia-nim/qwen-3.5-397b]\nbuild a task")
	if clean != "build a task" {
		t.Fatalf("unexpected clean prompt: %q", clean)
	}
	if meta.providerID != "nim" {
		t.Fatalf("expected provider nim, got %q", meta.providerID)
	}
	if meta.modelID != "nvidia-nim/qwen-3.5-397b" {
		t.Fatalf("unexpected model id: %q", meta.modelID)
	}
}

func TestExtractRoutingMetadataIgnoresUnstructuredPrompt(t *testing.T) {
	prompt := "plain prompt without metadata"
	clean, meta := extractRoutingMetadata(prompt)
	if clean != prompt {
		t.Fatalf("expected unchanged prompt, got %q", clean)
	}
	if meta.providerID != "" || meta.modelID != "" {
		t.Fatalf("expected empty metadata, got %+v", meta)
	}
}

func TestUnsupportedModelFlagError(t *testing.T) {
	if !isUnsupportedModelFlagError(errors.New("unknown flag: --model")) {
		t.Fatalf("expected unsupported-model error to be detected")
	}
	if isUnsupportedModelFlagError(errors.New("network timeout")) {
		t.Fatalf("expected non-model error to be ignored")
	}
}
