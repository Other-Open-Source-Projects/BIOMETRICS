package integrator

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"biometrics-cli/internal/contracts"
	"biometrics-cli/internal/runtime/actor"
)

func New() actor.Handler {
	return func(_ context.Context, env contracts.AgentEnvelope) contracts.AgentResult {
		artifact := contracts.Artifact{
			Type:        "summary",
			Path:        filepath.Join("runs", env.RunID, "summary.md"),
			Description: "Run integration summary",
			CreatedAt:   time.Now().UTC(),
		}

		return contracts.AgentResult{
			RunID:      env.RunID,
			TaskID:     env.TaskID,
			Agent:      "integrator",
			Success:    true,
			Summary:    fmt.Sprintf("integration finished for run %s", env.RunID),
			Artifacts:  []contracts.Artifact{artifact},
			FinishedAt: time.Now().UTC(),
		}
	}
}
