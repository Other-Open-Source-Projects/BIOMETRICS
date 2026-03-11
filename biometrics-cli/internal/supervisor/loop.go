package supervisor

// Deprecated: This loop implementation is legacy and not part of the BIOMETRICS V3 control-plane runtime.
// Prefer `biometrics-cli/cmd/controlplane` scheduling and `biometrics-cli/internal/executor/opencode` for OpenCode execution.

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"biometrics-cli/internal/project"
	"biometrics-cli/pkg/logging"
)

// BiometricsLoop represents the autonomous execution engine for BIOMETRICS.
type BiometricsLoop struct {
	projectID string
	logger    *logging.AdvancedLogger
	retries   map[string]int
}

// NewBiometricsLoop creates a new BiometricsLoop instance.
func NewBiometricsLoop(projectID string, logger *logging.AdvancedLogger) *BiometricsLoop {
	return &BiometricsLoop{
		projectID: projectID,
		logger:    logger,
		retries:   make(map[string]int),
	}
}

// Start begins the autonomous execution loop.
func (l *BiometricsLoop) Start(ctx context.Context) error {
	l.logger.Info("Starting Biometrics Loop", logging.String("project", l.projectID))

	for {
		select {
		case <-ctx.Done():
			l.logger.Info("Biometrics Loop stopping", logging.Err(ctx.Err()))
			return ctx.Err()
		default:
			// Fetch next pending task
			task, err := project.GetNextTask(l.projectID)
			if err != nil {
				// No pending tasks, wait and retry
				time.Sleep(15 * time.Second)
				continue
			}

			l.logger.Info("Found pending task",
				logging.String("id", task.ID),
				logging.String("description", task.Description))

			// Execute task with exponential backoff for retries
			if err := l.executeWithRetry(ctx, task); err != nil {
				l.logger.Error("Task failed permanently (DLQ)",
					logging.String("id", task.ID),
					logging.Err(err))
				// In a real system, we would move this to a DLQ state in boulder.json
				continue
			}

			// Mark task as completed
			if err := project.MarkTaskCompleted(l.projectID, task.ID); err != nil {
				l.logger.Error("Failed to update task status",
					logging.String("id", task.ID),
					logging.Err(err))
			}

			l.logger.Info("Task completed successfully", logging.String("id", task.ID))
		}
	}
}

func (l *BiometricsLoop) executeWithRetry(ctx context.Context, task *project.Task) error {
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		if err := l.executeTask(ctx, task); err == nil {
			return nil
		} else {
			l.logger.Warn("Task execution attempt failed",
				logging.String("id", task.ID),
				logging.Int("attempt", i+1),
				logging.Err(err))

			// Exponential backoff
			backoff := time.Duration(1<<uint(i)) * time.Second
			time.Sleep(backoff)
		}
	}
	return fmt.Errorf("failed after %d attempts", maxRetries)
}

func (l *BiometricsLoop) executeTask(ctx context.Context, task *project.Task) error {
	// Inject X-Trace-ID for observability
	traceID := fmt.Sprintf("trace-%d", time.Now().UnixNano())
	l.logger.Info("Spawning agent process", logging.String("trace_id", traceID))

	// Prepare opencode command
	args := []string{"run", "--agent", "sisyphus"}
	if dir := resolveOpenCodeRunDir(); dir != "" {
		args = append(args, "--dir", dir)
	}
	args = append(args, task.Description)
	cmd := exec.CommandContext(ctx, "opencode", args...)
	cmd.Env = append(os.Environ(), "X_TRACE_ID="+traceID)

	// Connect to stdin/stdout/stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to open stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to open stdout pipe: %w", err)
	}

	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start agent process: %w", err)
	}

	// Channel to signal completion of output handling
	done := make(chan bool)

	// Async output handler for proactivity and quality gate injection
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Println(line)

			// MANDATE 0.37: Auto-inject Quality Gate trigger
			// We look for common "ready" patterns from the agent
			if strings.Contains(line, "I have completed") ||
				strings.Contains(line, "Task done") ||
				strings.Contains(line, "Changes applied") {
				l.logger.Info("Injecting MANDATE 0.37 Quality Gate")
				io.WriteString(stdin, "Sicher? Führe eine vollständige Selbstreflexion durch. Prüfe jede deiner Aussagen, verifiziere, ob ALLE Restriktionen des Initial-Prompts exakt eingehalten wurden. Stelle alles Fehlende fertig.\n")
			}
		}
		done <- true
	}()

	// Wait for process to exit
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("agent process exited with error: %w", err)
	}

	<-done

	// Hard Auto-Verification
	return l.verifyCompliance()
}

func (l *BiometricsLoop) verifyCompliance() error {
	l.logger.Info("Running auto-verification (go vet & go test)")

	// Go Vet
	vetCmd := exec.Command("go", "vet", "./...")
	if out, err := vetCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("go vet failed:\n%s", string(out))
	}

	// Go Test
	testCmd := exec.Command("go", "test", "./...")
	if out, err := testCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("go test failed:\n%s", string(out))
	}

	l.logger.Info("Compliance check PASSED")
	return nil
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
