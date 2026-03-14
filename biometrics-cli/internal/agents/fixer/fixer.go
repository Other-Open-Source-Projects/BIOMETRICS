package fixer

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
			Agent:      "fixer",
			Success:    false,
			Error:      "llm gateway is not configured",
			Summary:    "fixer execution failed",
			FinishedAt: time.Now().UTC(),
		}
	}

	reason := env.Input["failed_reason"]
	prompt := strings.TrimSpace(env.Prompt)
	if prompt == "" {
		prompt = fmt.Sprintf("Project: %s\nTask failure reason: %s\nApply minimal corrective action and summarize.", env.ProjectID, reason)
	}
	response, err := a.gateway.Execute(ctx, llm.Request{
		RunID:           env.RunID,
		TaskID:          env.TaskID,
		Agent:           "fixer",
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
			Agent:      "fixer",
			Success:    false,
			Error:      err.Error(),
			Summary:    "fixer execution failed",
			FinishedAt: time.Now().UTC(),
		}
	}

	return contracts.AgentResult{
		RunID:         env.RunID,
		TaskID:        env.TaskID,
		Agent:         "fixer",
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
