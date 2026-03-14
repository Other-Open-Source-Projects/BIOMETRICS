package reviewer

import (
	"context"
	"strings"
	"time"

	"biometrics-cli/internal/contracts"
	"biometrics-cli/internal/runtime/actor"
)

func New() actor.Handler {
	return func(_ context.Context, env contracts.AgentEnvelope) contracts.AgentResult {
		coderOutput := strings.ToLower(env.Input["coder_output"])
		if strings.Contains(coderOutput, "panic") || strings.Contains(coderOutput, "fatal") {
			return contracts.AgentResult{
				RunID:      env.RunID,
				TaskID:     env.TaskID,
				Agent:      "reviewer",
				Success:    false,
				Error:      "reviewer detected critical terms in coder output",
				Summary:    "review failed",
				FinishedAt: time.Now().UTC(),
			}
		}

		return contracts.AgentResult{
			RunID:      env.RunID,
			TaskID:     env.TaskID,
			Agent:      "reviewer",
			Success:    true,
			Summary:    "review passed: no critical regressions detected",
			FinishedAt: time.Now().UTC(),
		}
	}
}
