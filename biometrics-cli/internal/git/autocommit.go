package git

import (
	"biometrics-cli/internal/telemetry"
	"context"
	"fmt"
	"log/slog"
	"os/exec"
)

// AutoCommit führt git add, commit und push im angegebenen Verzeichnis aus
func AutoCommit(ctx context.Context, logger *slog.Logger, projectPath string, taskID string) error {
	telemetry.LogWithTrace(ctx, logger, slog.LevelInfo, "Starting Auto-Commit", slog.String("project", projectPath))

	// 1. Git Add
	cmdAdd := exec.CommandContext(ctx, "git", "add", "-A")
	cmdAdd.Dir = projectPath
	if err := cmdAdd.Run(); err != nil {
		return fmt.Errorf("git add failed: %w", err)
	}

	// 2. Git Commit
	msg := fmt.Sprintf("feat: Auto-completed task %s via BIOMETRICS Swarm", taskID)
	cmdCommit := exec.CommandContext(ctx, "git", "commit", "-m", msg)
	cmdCommit.Dir = projectPath
	// Ignore error on commit (might be nothing to commit)
	cmdCommit.Run()

	// 3. Git Push
	cmdPush := exec.CommandContext(ctx, "git", "push", "origin", "main")
	cmdPush.Dir = projectPath
	if err := cmdPush.Run(); err != nil {
		telemetry.LogWithTrace(ctx, logger, slog.LevelWarn, "Git push failed (might need manual intervention)", slog.String("error", err.Error()))
		// We don't return error here to not fail the task just because of push
	}

	telemetry.LogWithTrace(ctx, logger, slog.LevelInfo, "Auto-Commit successful")
	return nil
}
