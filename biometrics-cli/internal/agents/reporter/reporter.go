package reporter

import (
	"context"
	"fmt"
	"time"

	"biometrics-cli/internal/contracts"
	"biometrics-cli/internal/runtime/actor"
)

func New() actor.Handler {
	return func(_ context.Context, env contracts.AgentEnvelope) contracts.AgentResult {
		return contracts.AgentResult{
			RunID:      env.RunID,
			TaskID:     env.TaskID,
			Agent:      "reporter",
			Success:    true,
			Summary:    fmt.Sprintf("report streamed for run %s", env.RunID),
			FinishedAt: time.Now().UTC(),
		}
	}
}
