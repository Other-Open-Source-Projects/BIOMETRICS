package coder

import (
	"context"
	"fmt"
	"strings"
	"time"

	"biometrics-cli/internal/contracts"
	"biometrics-cli/internal/llm"
	"biometrics-cli/internal/runtime/actor"
)

type Agent struct {
	gateway llm.Gateway
}

func New(gateway llm.Gateway) actor.Handler {
	a := &Agent{gateway: gateway}
	return a.handle
}

func (a *Agent) handle(ctx context.Context, env contracts.AgentEnvelope) contracts.AgentResult {
	if a.gateway == nil {
		return contracts.AgentResult{
			RunID:      env.RunID,
			TaskID:     env.TaskID,
			Agent:      "coder",
			Success:    false,
			Error:      "llm gateway is not configured",
			Summary:    "coder execution failed",
			FinishedAt: time.Now().UTC(),
		}
	}

	prompt := strings.TrimSpace(env.Prompt)
	if prompt == "" {
		prompt = fmt.Sprintf("Project: %s\nGoal: %s\nTask: %s\nReturn a concise execution summary.", env.ProjectID, env.Goal, env.TaskName)
	}

	response, err := a.gateway.Execute(ctx, llm.Request{
		RunID:           env.RunID,
		TaskID:          env.TaskID,
		Agent:           "coder",
		TaskName:        env.TaskName,
		ProjectID:       env.ProjectID,
		Prompt:          prompt,
		ModelPreference: env.ModelPreference,
		FallbackChain:   append([]string{}, env.FallbackChain...),
		ModelID:         env.ModelID,
		ContextBudget:   env.ContextBudget,
	})
	if err != nil {
		return contracts.AgentResult{
			RunID:      env.RunID,
			TaskID:     env.TaskID,
			Agent:      "coder",
			Success:    false,
			Error:      err.Error(),
			Summary:    "coder execution failed",
			FinishedAt: time.Now().UTC(),
		}
	}

	return contracts.AgentResult{
		RunID:         env.RunID,
		TaskID:        env.TaskID,
		Agent:         "coder",
		Success:       true,
		Summary:       response.Output,
		Provider:      response.Provider,
		ModelID:       response.ModelID,
		LatencyMs:     response.LatencyMs,
		TokensIn:      response.TokensIn,
		TokensOut:     response.TokensOut,
		ProviderTrail: response.ProviderTrail,
		FinishedAt:    time.Now().UTC(),
	}
}
