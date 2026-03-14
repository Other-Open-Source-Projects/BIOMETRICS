package scoper

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

func (a *Agent) handle(_ context.Context, env contracts.AgentEnvelope) contracts.AgentResult {
	target := filepath.Join(a.workspace, env.ProjectID)
	if _, err := os.Stat(target); err != nil {
		target = a.workspace
	}

	entries, err := os.ReadDir(target)
	if err != nil {
		return contracts.AgentResult{
			RunID:      env.RunID,
			TaskID:     env.TaskID,
			Agent:      "scoper",
			Success:    false,
			Error:      err.Error(),
			Summary:    "failed to scope project",
			FinishedAt: time.Now().UTC(),
		}
	}

	names := make([]string, 0, 12)
	for _, entry := range entries {
		names = append(names, entry.Name())
		if len(names) >= 12 {
			break
		}
	}

	summary := fmt.Sprintf("scoped workspace %s (%d entries): %s", target, len(entries), strings.Join(names, ", "))
	return contracts.AgentResult{
		RunID:      env.RunID,
		TaskID:     env.TaskID,
		Agent:      "scoper",
		Success:    true,
		Summary:    summary,
		FinishedAt: time.Now().UTC(),
	}
}
