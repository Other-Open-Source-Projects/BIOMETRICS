package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"biometrics-cli/internal/collision"
	"biometrics-cli/internal/opencode"
	"biometrics-cli/internal/project"
	"biometrics-cli/internal/quality"
	"biometrics-cli/internal/telemetry"
)

func main() {
	// 1. Setup Telemetry
	logger := telemetry.SetupLogger()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		logger.Info("Received shutdown signal, terminating process group...")
		cancel()
	}()

	logger.Info("=== BIOMETRICS 24/7 ENTERPRISE ORCHESTRATOR STARTING ===")

	// 2. Setup Components
	modelPool := collision.NewModelPool()
	executor := opencode.NewExecutor(logger)

	// Registrierte Projekte für Round-Robin
	scheduler := project.NewRotationScheduler([]string{"biometrics", "sin-solver", "simone-webshop"})

	// 3. Main Loop
	for {
		if ctx.Err() != nil {
			logger.Info("Context canceled, exiting main loop")
			break
		}

		// 3.1 Nächstes Projekt holen
		projectID := scheduler.NextProject()
		if projectID == "" {
			time.Sleep(5 * time.Second)
			continue
		}

		// 3.2 TraceID für diesen Cycle generieren
		cycleCtx, traceID := telemetry.InjectTraceID(ctx)
		logger.Info("Starting cycle", slog.String("project", projectID), slog.String("trace_id", traceID))

		// 3.3 Projekt-spezifisches Boulder laden
		projCtx, err := project.LoadProjectContext(projectID)
		if err != nil {
			telemetry.LogWithTrace(cycleCtx, logger, slog.LevelWarn, "Failed to load project context", slog.String("error", err.Error()))
			time.Sleep(2 * time.Second)
			continue
		}

		if projCtx.Boulder.ActivePlan == "" {
			telemetry.LogWithTrace(cycleCtx, logger, slog.LevelDebug, "No active plan for project", slog.String("project", projectID))
			time.Sleep(2 * time.Second)
			continue
		}

		// 3.4 Model Lock holen (Standardmäßig qwen-3.5 für Hauptaufgaben)
		model := "qwen-3.5"
		if err := modelPool.Acquire(cycleCtx, model); err != nil {
			telemetry.LogWithTrace(cycleCtx, logger, slog.LevelWarn, "Failed to acquire model lock", slog.String("model", model))
			time.Sleep(5 * time.Second)
			continue
		}

		// 3.5 Agent Ausführen
		req := opencode.AgentRequest{
			ProjectID: projectID,
			Model:     model,
			Prompt:    "Führe den nächsten Task aus dem aktiven Plan aus.",
			Category:  "build",
		}

		result := executor.RunAgent(cycleCtx, req)

		// Model Lock freigeben
		modelPool.Release(model)

		if !result.Success {
			telemetry.LogWithTrace(cycleCtx, logger, slog.LevelError, "Agent execution failed", slog.String("error", result.Error.Error()))
			time.Sleep(5 * time.Second)
			continue
		}

		// 3.6 Quality Gate (Sicher? Trigger)
		telemetry.LogWithTrace(cycleCtx, logger, slog.LevelInfo, "Enforcing Quality Gate...")
		if err := quality.EnforceQualityGate(cycleCtx, executor, req); err != nil {
			telemetry.LogWithTrace(cycleCtx, logger, slog.LevelError, "Quality Gate failed", slog.String("error", err.Error()))
		} else {
			telemetry.LogWithTrace(cycleCtx, logger, slog.LevelInfo, "Quality Gate passed successfully")
		}

		// Kurze Pause zwischen den Cycles
		time.Sleep(10 * time.Second)
	}

	logger.Info("Orchestrator shutdown complete.")
}
