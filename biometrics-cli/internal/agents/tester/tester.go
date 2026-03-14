package tester

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"biometrics-cli/internal/contracts"
	"biometrics-cli/internal/runtime/actor"
)

type Agent struct {
	workspace string
}

func New(workspace string) actor.Handler {
	a := &Agent{workspace: workspace}
	return a.handle
}

func (a *Agent) handle(ctx context.Context, env contracts.AgentEnvelope) contracts.AgentResult {
	if os.Getenv("BIOMETRICS_ENABLE_REAL_TESTS") != "1" {
		return contracts.AgentResult{
			RunID:      env.RunID,
			TaskID:     env.TaskID,
			Agent:      "tester",
			Success:    true,
			Summary:    "simulated test pass (set BIOMETRICS_ENABLE_REAL_TESTS=1 for real execution)",
			FinishedAt: time.Now().UTC(),
		}
	}

	target := filepath.Join(a.workspace, env.ProjectID)
	if _, err := os.Stat(target); err != nil {
		target = a.workspace
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()

	cmd := exec.CommandContext(timeoutCtx, "go", "test", "./...")
	cmd.Dir = target
	var out bytes.Buffer
	var errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut

	err := cmd.Run()
	if err != nil {
		return contracts.AgentResult{
			RunID:      env.RunID,
			TaskID:     env.TaskID,
			Agent:      "tester",
			Success:    false,
			Error:      fmt.Sprintf("%v: %s", err, errOut.String()),
			Summary:    out.String(),
			FinishedAt: time.Now().UTC(),
		}
	}

	return contracts.AgentResult{
		RunID:      env.RunID,
		TaskID:     env.TaskID,
		Agent:      "tester",
		Success:    true,
		Summary:    out.String(),
		FinishedAt: time.Now().UTC(),
	}
}
