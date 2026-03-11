package planner

import (
	"context"
	"encoding/json"
	"time"

	"biometrics-cli/internal/contracts"
	"biometrics-cli/internal/planning"
	"biometrics-cli/internal/runtime/actor"
)

func New() actor.Handler {
	return func(_ context.Context, env contracts.AgentEnvelope) contracts.AgentResult {
		plan := planning.BuildPlan(env.Goal)
		raw, _ := json.Marshal(plan)
		summary := string(raw)

		return contracts.AgentResult{
			RunID:      env.RunID,
			TaskID:     env.TaskID,
			Agent:      "planner",
			Success:    true,
			Summary:    summary,
			FinishedAt: time.Now().UTC(),
		}
	}
}
