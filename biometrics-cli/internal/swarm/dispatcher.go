package swarm

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"biometrics-cli/internal/collision"
	"biometrics-cli/internal/git"
	"biometrics-cli/internal/opencode"
	"biometrics-cli/internal/paths"
	"biometrics-cli/internal/project"
	"biometrics-cli/internal/prompt"
	"biometrics-cli/internal/quality"
	"biometrics-cli/internal/recovery"
	"biometrics-cli/internal/telemetry"
)

type Dispatcher struct {
	logger    *slog.Logger
	modelPool *collision.ModelPool
	executor  *opencode.Executor
}

func NewDispatcher(logger *slog.Logger, pool *collision.ModelPool, exec *opencode.Executor) *Dispatcher {
	return &Dispatcher{
		logger:    logger,
		modelPool: pool,
		executor:  exec,
	}
}

// Dispatch startet einen Task asynchron in einer Goroutine
func (d *Dispatcher) Dispatch(globalCtx context.Context, projID string, task *project.Task) {
	go func() {
		// 1. Eigener Context mit TraceID und Watchdog (z.B. 45 Minuten Timeout)
		_, traceID := telemetry.InjectTraceID(context.Background())
		watchdogCtx, cancel := recovery.StartWatchdog(d.logger, task.ID, readWatchdogTimeout())
		defer cancel()

		d.logger.Info("Swarm Agent Dispatched", slog.String("project", projID), slog.String("task_id", task.ID), slog.String("trace_id", traceID))

		// 2. Model Lock (Wir nutzen Gemini für Build-Tasks)
		model := "google/antigravity-gemini-3.1-pro"
		if err := d.modelPool.Acquire(watchdogCtx, model); err != nil {
			d.logger.Error("Failed to acquire model", slog.String("task_id", task.ID))
			return
		}
		defer d.modelPool.Release(model)

		// 3. Prompt Generierung
		taskPrompt := prompt.GenerateEnterprisePrompt(projID, "active_plan.md", task.ID, task.Description)
		req := opencode.AgentRequest{
			ProjectID: projID,
			Model:     model,
			Prompt:    taskPrompt,
			Category:  "build",
		}

		// 4. Ausführung
		result := d.executor.RunAgent(watchdogCtx, req)
		if !result.Success {
			d.logger.Error("Task failed", slog.String("task_id", task.ID), slog.String("error", result.Error.Error()))
			return
		}

		// 5. Quality Gate
		if err := quality.EnforceQualityGate(watchdogCtx, d.executor, req); err != nil {
			d.logger.Error("Quality Gate failed", slog.String("task_id", task.ID))
			return
		}

		// 6. Abschluss & Auto-Commit
		project.MarkTaskCompleted(projID, task.ID)
		d.logger.Info("Task completed successfully", slog.String("task_id", task.ID))

		// Projekt-Pfad ermitteln (default: $HOME/dev/<PROJEKTNAME>, override: BIOMETRICS_PROJECTS_DIR)
		projectPath := filepath.Join(paths.ProjectsDir(), projID)
		git.AutoCommit(watchdogCtx, d.logger, projectPath, task.ID)
	}()
}

func readWatchdogTimeout() time.Duration {
	raw := os.Getenv("BIOMETRICS_WATCHDOG_TIMEOUT")
	if raw == "" {
		return 0
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0
	}
	return d
}
