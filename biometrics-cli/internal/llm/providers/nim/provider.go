package nim

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"biometrics-cli/internal/contracts"
	"biometrics-cli/internal/executor"
	"biometrics-cli/internal/llm"
	llmerrors "biometrics-cli/internal/llm/errors"
)

const defaultNIMModel = "nvidia-nim/qwen-3.5-397b"

var supportedNIMModels = []contracts.ModelOption{
	{ID: "nvidia-nim/qwen-3.5-397b", Name: "Qwen 3.5 397B (NVIDIA NIM)", Default: true},
}

var nimAliases = map[string]string{
	"nvidia-nim/qwen-3.5-397b": "nvidia-nim/qwen-3.5-397b",
	"qwen-3.5-397b":            "nvidia-nim/qwen-3.5-397b",
	"qwen3.5-397b-a17b":        "nvidia-nim/qwen-3.5-397b",
	"qwen/qwen3.5-397b-a17b":   "nvidia-nim/qwen-3.5-397b",
}

type Provider struct {
	exec         executor.Adapter
	defaultModel string
}

func New(exec executor.Adapter, defaultModel string) *Provider {
	normalizedDefault := normalizeNIMModelID(defaultModel)
	if normalizedDefault == "" {
		normalizedDefault = defaultNIMModel
	}
	return &Provider{
		exec:         exec,
		defaultModel: normalizedDefault,
	}
}

func (p *Provider) ID() string {
	return "nim"
}

func (p *Provider) Name() string {
	return "NVIDIA NIM"
}

func (p *Provider) Description() string {
	return "NVIDIA NIM provider (Qwen 3.5 397B)"
}

func (p *Provider) DefaultModelID() string {
	return strings.TrimSpace(p.defaultModel)
}

func (p *Provider) SupportedModels() []contracts.ModelOption {
	out := make([]contracts.ModelOption, len(supportedNIMModels))
	copy(out, supportedNIMModels)
	defaultModel := strings.TrimSpace(p.defaultModel)
	for i := range out {
		out[i].Default = out[i].ID == defaultModel
	}
	return out
}

func (p *Provider) Available(_ context.Context) bool {
	if !envEnabled("BIOMETRICS_NIM_ENABLED", true) {
		return false
	}
	return p.exec != nil
}

func (p *Provider) Generate(ctx context.Context, req llm.Request) (llm.Response, error) {
	if !p.Available(ctx) {
		return llm.Response{}, &llmerrors.ProviderError{
			Class:   llmerrors.ClassProviderUnavailable,
			Message: "nim provider is disabled",
		}
	}

	modelID := normalizeNIMModelID(req.ModelID)
	if modelID == "" {
		modelID = p.defaultModel
	}

	var prompt string
	if modelID != "" {
		prompt = fmt.Sprintf("[provider=nim model=%s]\n%s", modelID, req.Prompt)
	} else {
		prompt = "[provider=nim]\n" + req.Prompt
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
		Provider:  "nim",
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

func envEnabled(key string, defaultValue bool) bool {
	raw := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	switch raw {
	case "":
		return defaultValue
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return defaultValue
	}
}

func normalizeNIMModelID(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	normalized := strings.ToLower(trimmed)
	if alias, ok := nimAliases[normalized]; ok {
		return alias
	}
	return trimmed
}
