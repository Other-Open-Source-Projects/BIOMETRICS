package swarm

import (
	"context"
	"log/slog"
	"path/filepath"
	"time"

	"biometrics-cli/internal/collision"
	"biometrics-cli/internal/git"
	"biometrics-cli/internal/opencode"
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
		watchdogCtx, cancel := recovery.StartWatchdog(d.logger, task.ID, 45*time.Minute)
		defer cancel()

		d.logger.Info("Swarm Agent Dispatched", slog.String("project", projID), slog.String("task_id", task.ID), slog.String("trace_id", traceID))

		// 2. Model Lock (Wir nutzen Qwen für Build-Tasks)
		model := "qwen-3.5"
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

		// Projekt-Pfad ermitteln (vereinfacht: /Users/jeremy/dev/PROJEKTNAME)
		projectPath := filepath.Join("/Users/jeremy/dev", projID)
		git.AutoCommit(watchdogCtx, d.logger, projectPath, task.ID)
	}()
}
