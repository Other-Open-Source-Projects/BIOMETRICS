package codex

import (
	"context"
	"fmt"
	"strings"
	"time"

	"biometrics-cli/internal/auth/codexbroker"
	"biometrics-cli/internal/contracts"
	"biometrics-cli/internal/executor"
	"biometrics-cli/internal/llm"
	llmerrors "biometrics-cli/internal/llm/errors"
)

type Provider struct {
	exec         executor.Adapter
	broker       codexbroker.Interface
	defaultModel string
}

func New(exec executor.Adapter, broker codexbroker.Interface, defaultModel string) *Provider {
	return &Provider{
		exec:         exec,
		broker:       broker,
		defaultModel: strings.TrimSpace(defaultModel),
	}
}

func (p *Provider) ID() string {
	return "codex"
}

func (p *Provider) Name() string {
	return "OpenAI Codex"
}

func (p *Provider) Description() string {
	return "OpenAI Codex provider (primary coding runtime)"
}

func (p *Provider) DefaultModelID() string {
	return strings.TrimSpace(p.defaultModel)
}

func (p *Provider) SupportedModels() []contracts.ModelOption {
	defaultModel := strings.TrimSpace(p.defaultModel)
	if defaultModel == "" {
		return []contracts.ModelOption{}
	}
	return []contracts.ModelOption{
		{
			ID:      defaultModel,
			Name:    "Configured Codex Model",
			Default: true,
		},
	}
}

func (p *Provider) Available(ctx context.Context) bool {
	if p.broker == nil {
		return p.exec != nil
	}
	status, err := p.broker.Status(ctx)
	if err != nil {
		return false
	}
	return status.Ready && status.LoggedIn
}

func (p *Provider) Generate(ctx context.Context, req llm.Request) (llm.Response, error) {
	if p.exec == nil {
		return llm.Response{}, &llmerrors.ProviderError{
			Class:   llmerrors.ClassProviderUnavailable,
			Message: "codex provider executor is not configured",
		}
	}
	if p.broker != nil {
		status, err := p.broker.Status(ctx)
		if err != nil {
			return llm.Response{}, &llmerrors.ProviderError{
				Class:   llmerrors.ClassProviderUnavailable,
				Message: fmt.Sprintf("codex auth broker status failed: %v", err),
				Cause:   err,
			}
		}
		if !status.Ready || !status.LoggedIn {
			return llm.Response{}, &llmerrors.ProviderError{
				Class:   llmerrors.ClassProviderUnavailable,
				Message: "codex authentication is not ready; run codex login",
			}
		}
	}

	modelID := strings.TrimSpace(req.ModelID)
	if modelID == "" {
		modelID = p.defaultModel
	}

	prompt := req.Prompt
	if modelID != "" {
		prompt = fmt.Sprintf("[provider=codex model=%s]\n%s", modelID, req.Prompt)
	} else {
		prompt = "[provider=codex]\n" + req.Prompt
	}

	started := time.Now()
	output, err := p.exec.Execute(ctx, req.RunID, req.Agent, prompt, req.ProjectID)
	if err != nil {
		class, _ := llmerrors.Classify(err)
		return llm.Response{}, &llmerrors.ProviderError{
			Class:   class,
			Message: err.Error(),
			Cause:   err,
		}
	}
	latency := time.Since(started).Milliseconds()
	if latency < 0 {
		latency = 0
	}

	return llm.Response{
		Output:    output,
		Provider:  "codex",
		ModelID:   modelID,
		LatencyMs: latency,
		TokensIn:  roughTokenCount(prompt),
		TokensOut: roughTokenCount(output),
		CreatedAt: time.Now().UTC(),
	}, nil
}

func roughTokenCount(value string) int {
	fields := strings.Fields(strings.TrimSpace(value))
	if len(fields) == 0 {
		return 0
	}
	return len(fields)
}
