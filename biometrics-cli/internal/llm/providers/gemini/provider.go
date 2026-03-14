package gemini

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

const defaultGeminiModel = "google/gemini-3.1-pro-preview"

var supportedGeminiModels = []contracts.ModelOption{
	{ID: "google/gemini-3.1-pro-preview", Name: "Gemini 3.1 Pro", Default: true},
	{ID: "google/gemini-3-pro", Name: "Gemini 3 Pro"},
	{ID: "google/gemini-3-flash", Name: "Gemini 3 Flash"},
}

var geminiAliases = map[string]string{
	"gemini-3.1-pro":                "google/gemini-3.1-pro-preview",
	"gemini-3.1-pro-preview":        "google/gemini-3.1-pro-preview",
	"google/gemini-3.1-pro":         "google/gemini-3.1-pro-preview",
	"google/gemini-3.1-pro-preview": "google/gemini-3.1-pro-preview",

	"gemini-3-pro":                "google/gemini-3-pro",
	"gemini-3-pro-preview":        "google/gemini-3-pro",
	"google/gemini-3-pro":         "google/gemini-3-pro",
	"google/gemini-3-pro-preview": "google/gemini-3-pro",

	"gemini-3-flash":                "google/gemini-3-flash",
	"gemini-3-flash-preview":        "google/gemini-3-flash",
	"google/gemini-3-flash":         "google/gemini-3-flash",
	"google/gemini-3-flash-preview": "google/gemini-3-flash",
}

type Provider struct {
	exec         executor.Adapter
	defaultModel string
}

func New(exec executor.Adapter, defaultModel string) *Provider {
	normalizedDefault := normalizeGeminiModelID(defaultModel)
	if normalizedDefault == "" {
		normalizedDefault = defaultGeminiModel
	}
	return &Provider{
		exec:         exec,
		defaultModel: normalizedDefault,
	}
}

func (p *Provider) ID() string {
	return "gemini"
}

func (p *Provider) Name() string {
	return "Google Gemini"
}

func (p *Provider) Description() string {
	return "Google Gemini provider (Gemini 3.1 Pro, Gemini 3 Pro, Gemini 3 Flash)"
}

func (p *Provider) DefaultModelID() string {
	return strings.TrimSpace(p.defaultModel)
}

func (p *Provider) SupportedModels() []contracts.ModelOption {
	out := make([]contracts.ModelOption, len(supportedGeminiModels))
	copy(out, supportedGeminiModels)
	defaultModel := strings.TrimSpace(p.defaultModel)
	for i := range out {
		out[i].Default = out[i].ID == defaultModel
	}
	return out
}

func (p *Provider) Available(_ context.Context) bool {
	if !envEnabled("BIOMETRICS_GEMINI_ENABLED", true) {
		return false
	}
	return p.exec != nil
}

func (p *Provider) Generate(ctx context.Context, req llm.Request) (llm.Response, error) {
	if !p.Available(ctx) {
		return llm.Response{}, &llmerrors.ProviderError{
			Class:   llmerrors.ClassProviderUnavailable,
			Message: "gemini provider is disabled",
		}
	}

	modelID := normalizeGeminiModelID(req.ModelID)
	if modelID == "" {
		modelID = p.defaultModel
	}

	var prompt string
	if modelID != "" {
		prompt = fmt.Sprintf("[provider=gemini model=%s]\n%s", modelID, req.Prompt)
	} else {
		prompt = "[provider=gemini]\n" + req.Prompt
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
		Provider:  "gemini",
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

func normalizeGeminiModelID(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	normalized := strings.ToLower(trimmed)
	if alias, ok := geminiAliases[normalized]; ok {
		return alias
	}
	return trimmed
}
