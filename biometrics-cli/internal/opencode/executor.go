package opencode

import (
	"biometrics-cli/internal/telemetry"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

type Executor struct {
	logger *slog.Logger
}

func NewExecutor(logger *slog.Logger) *Executor {
	return &Executor{logger: logger}
}

// RunAgent startet den OpenCode Prozess. Es MUSS SysProcAttr für Process Groups nutzen!
func (e *Executor) RunAgent(ctx context.Context, req AgentRequest) AgentResult {
	telemetry.LogWithTrace(ctx, e.logger, slog.LevelInfo, "Starting OpenCode Agent",
		slog.String("model", req.Model),
		slog.String("project", req.ProjectID),
	)

	// Command Aufbau
	args := []string{"run"}
	if strings.TrimSpace(req.Model) != "" {
		args = append(args, "--model", strings.TrimSpace(req.Model))
	}
	if dir := resolveOpenCodeRunDir(); dir != "" {
		args = append(args, "--dir", dir)
	}
	args = append(args, req.Prompt)
	cmd := exec.CommandContext(ctx, "opencode", args...)

	// PFLICHT: Process Group ID setzen, damit wir den ganzen Tree killen können
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// Environment Variablen setzen (Mandat 0.38 - Project Isolation)
	cmd.Env = append(cmd.Environ(), fmt.Sprintf("PROJECT_ID=%s", req.ProjectID))

	// Stdout/Stderr Streaming (siehe stream.go)
	outChan := e.streamOutput(cmd)

	err := cmd.Start()
	if err != nil {
		return AgentResult{Success: false, Error: fmt.Errorf("failed to start agent: %w", err)}
	}

	// Logge Output asynchron
	go func() {
		for line := range outChan {
			telemetry.LogWithTrace(ctx, e.logger, slog.LevelDebug, "Agent Output", slog.String("line", line))
		}
	}()

	err = cmd.Wait()

	// Cleanup: Falls Context canceled wurde, kille die GANZE Process Group!
	if ctx.Err() != nil {
		syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		return AgentResult{Success: false, Error: ctx.Err()}
	}

	if err != nil {
		return AgentResult{Success: false, Error: err}
	}

	return AgentResult{Success: true, Output: "Agent finished successfully"}
}

func resolveOpenCodeRunDir() string {
	for _, candidate := range []string{
		strings.TrimSpace(os.Getenv("BIOMETRICS_OPENCODE_DIR")),
		strings.TrimSpace(os.Getenv("BIOMETRICS_WORKSPACE")),
	} {
		if candidate == "" {
			continue
		}
		if stat, err := os.Stat(candidate); err == nil && stat.IsDir() {
			return candidate
		}
	}
	if cwd, err := os.Getwd(); err == nil {
		if stat, statErr := os.Stat(cwd); statErr == nil && stat.IsDir() {
			return cwd
		}
	}
	return ""
}
