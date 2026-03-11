package router

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"biometrics-cli/internal/contracts"
	"biometrics-cli/internal/llm"
	llmerrors "biometrics-cli/internal/llm/errors"
	llmmetrics "biometrics-cli/internal/llm/metrics"
	llmpolicy "biometrics-cli/internal/llm/policy"
)

type EventEmitter func(contracts.Event)

type catalogProvider interface {
	DefaultModelID() string
	SupportedModels() []contracts.ModelOption
	Description() string
}

type Router struct {
	policy llmpolicy.RoutingPolicy

	providers map[string]llm.Provider
	emit      EventEmitter
	redact    func(string) string
	metrics   llmmetrics.Counters
}

func New(policy llmpolicy.RoutingPolicy, emitter EventEmitter, redactor func(string) string) *Router {
	if policy.DefaultPrimary == "" {
		policy = llmpolicy.Default()
	}
	if redactor == nil {
		redactor = func(value string) string { return value }
	}
	return &Router{
		policy:    policy,
		providers: make(map[string]llm.Provider),
		emit:      emitter,
		redact:    redactor,
	}
}

func (r *Router) Register(provider llm.Provider) {
	if provider == nil {
		return
	}
	id := llmpolicy.NormalizeProviderID(provider.ID())
	if id == "" {
		return
	}
	r.providers[id] = provider
}

func (r *Router) Execute(ctx context.Context, req llm.Request) (llm.Response, error) {
	chain := r.resolveChain(req.ModelPreference, req.FallbackChain)
	if len(chain) == 0 {
		return llm.Response{}, fmt.Errorf("llm router has no providers configured")
	}

	trail := make([]contracts.ProviderAttempt, 0, len(chain))
	hops := 0
	for idx, providerID := range chain {
		provider, ok := r.providers[providerID]
		if !ok {
			trail = append(trail, contracts.ProviderAttempt{
				Provider:      providerID,
				Status:        "failed",
				ErrorClass:    string(llmerrors.ClassProviderUnavailable),
				ErrorMessage:  r.redact("provider is not configured"),
				FallbackIndex: idx,
			})
			if !r.canFallback(idx, len(chain), llmerrors.ClassProviderUnavailable, hops) {
				r.emitFallbackExhausted(req.RunID, trail)
				return llm.Response{}, fmt.Errorf("model fallback exhausted: provider %q is not configured", providerID)
			}
			r.emitFallbackTriggered(req.RunID, providerID, chain[idx+1], string(llmerrors.ClassProviderUnavailable))
			r.metrics.IncFallbackTriggered()
			hops++
			continue
		}

		if idx == 0 {
			r.emitModelSelected(req.RunID, providerID, req.ModelID)
			r.metrics.IncSelected()
		}

		if !provider.Available(ctx) {
			trail = append(trail, contracts.ProviderAttempt{
				Provider:      providerID,
				ModelID:       strings.TrimSpace(req.ModelID),
				Status:        "failed",
				ErrorClass:    string(llmerrors.ClassProviderUnavailable),
				ErrorMessage:  r.redact("provider is not ready"),
				FallbackIndex: idx,
			})
			if !r.canFallback(idx, len(chain), llmerrors.ClassProviderUnavailable, hops) {
				r.emitFallbackExhausted(req.RunID, trail)
				return llm.Response{}, fmt.Errorf("model fallback exhausted: provider %q is not ready", providerID)
			}
			r.emitFallbackTriggered(req.RunID, providerID, chain[idx+1], string(llmerrors.ClassProviderUnavailable))
			r.metrics.IncFallbackTriggered()
			hops++
			continue
		}

		started := time.Now()
		response, err := provider.Generate(ctx, req)
		latency := time.Since(started).Milliseconds()
		if latency < 0 {
			latency = 0
		}
		if err == nil {
			response.Provider = providerID
			if strings.TrimSpace(response.ModelID) == "" {
				response.ModelID = strings.TrimSpace(req.ModelID)
			}
			if response.LatencyMs <= 0 {
				response.LatencyMs = latency
			}
			trail = append(trail, contracts.ProviderAttempt{
				Provider:      providerID,
				ModelID:       response.ModelID,
				Status:        "succeeded",
				LatencyMs:     response.LatencyMs,
				FallbackIndex: idx,
			})
			response.ProviderTrail = trail
			return response, nil
		}

		class, message := llmerrors.Classify(err)
		trail = append(trail, contracts.ProviderAttempt{
			Provider:      providerID,
			ModelID:       strings.TrimSpace(req.ModelID),
			Status:        "failed",
			ErrorClass:    string(class),
			ErrorMessage:  r.redact(message),
			LatencyMs:     latency,
			FallbackIndex: idx,
		})

		if !r.canFallback(idx, len(chain), class, hops) {
			r.emitFallbackExhausted(req.RunID, trail)
			return llm.Response{}, fmt.Errorf("model fallback exhausted: %s", r.redact(message))
		}

		r.emitFallbackTriggered(req.RunID, providerID, chain[idx+1], string(class))
		r.metrics.IncFallbackTriggered()
		hops++
	}

	r.emitFallbackExhausted(req.RunID, trail)
	return llm.Response{}, fmt.Errorf("model fallback exhausted without successful provider")
}

func (r *Router) ModelCatalog(ctx context.Context) contracts.ModelCatalog {
	ids := make([]string, 0, len(r.providers))
	for id := range r.providers {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	providers := make([]contracts.ModelProvider, 0, len(ids))
	for _, id := range ids {
		provider := r.providers[id]
		available := provider.Available(ctx)
		description := "local-first provider adapter"
		modelID := ""
		models := []contracts.ModelOption{}
		if c, ok := provider.(catalogProvider); ok {
			description = strings.TrimSpace(c.Description())
			if description == "" {
				description = "local-first provider adapter"
			}
			modelID = strings.TrimSpace(c.DefaultModelID())
			models = c.SupportedModels()
		}
		providers = append(providers, contracts.ModelProvider{
			ID:          id,
			Name:        provider.Name(),
			Status:      ternaryStatus(available, "ready", "unavailable"),
			Default:     id == r.policy.DefaultPrimary,
			Available:   available,
			ModelID:     modelID,
			Models:      models,
			Description: description,
		})
	}
	return contracts.ModelCatalog{
		DefaultPrimary: r.policy.DefaultPrimary,
		DefaultChain:   append([]string{}, r.policy.DefaultFallback...),
		Providers:      providers,
	}
}

func (r *Router) MetricsSnapshot() map[string]int64 {
	return r.metrics.Snapshot()
}

func (r *Router) resolveChain(primary string, fallback []string) []string {
	selectedPrimary := llmpolicy.NormalizeProviderID(primary)
	if selectedPrimary == "" {
		selectedPrimary = llmpolicy.NormalizeProviderID(r.policy.DefaultPrimary)
	}

	baseFallback := fallback
	if len(baseFallback) == 0 {
		baseFallback = r.policy.DefaultFallback
	}

	out := make([]string, 0, 1+len(baseFallback))
	seen := map[string]struct{}{}
	if selectedPrimary != "" {
		out = append(out, selectedPrimary)
		seen[selectedPrimary] = struct{}{}
	}
	for _, raw := range baseFallback {
		normalized := llmpolicy.NormalizeProviderID(raw)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		out = append(out, normalized)
		seen[normalized] = struct{}{}
	}
	return out
}

func (r *Router) canFallback(index, total int, class llmerrors.Class, hops int) bool {
	if index >= total-1 {
		return false
	}
	if !llmerrors.IsRecoverable(class) {
		return false
	}
	return hops < r.policy.MaxFallbackHops
}

func (r *Router) emitModelSelected(runID, provider, modelID string) {
	if r.emit == nil {
		return
	}
	payload := map[string]string{
		"provider": provider,
	}
	if strings.TrimSpace(modelID) != "" {
		payload["model_id"] = strings.TrimSpace(modelID)
	}
	r.emit(contracts.Event{
		RunID:   runID,
		Type:    "model.selected",
		Source:  "llm.router",
		Payload: payload,
	})
}

func (r *Router) emitFallbackTriggered(runID, fromProvider, toProvider, class string) {
	if r.emit == nil {
		return
	}
	r.emit(contracts.Event{
		RunID:  runID,
		Type:   "model.fallback.triggered",
		Source: "llm.router",
		Payload: map[string]string{
			"from_provider": fromProvider,
			"to_provider":   toProvider,
			"error_class":   class,
		},
	})
}

func (r *Router) emitFallbackExhausted(runID string, trail []contracts.ProviderAttempt) {
	r.metrics.IncFallbackExhausted()
	if r.emit == nil {
		return
	}
	parts := make([]string, 0, len(trail))
	for _, item := range trail {
		state := item.Provider + ":" + item.Status
		if item.ErrorClass != "" {
			state += ":" + item.ErrorClass
		}
		parts = append(parts, state)
	}
	r.emit(contracts.Event{
		RunID:  runID,
		Type:   "model.fallback.exhausted",
		Source: "llm.router",
		Payload: map[string]string{
			"trail": strings.Join(parts, ","),
		},
	})
}

func ternaryStatus(ok bool, trueValue, falseValue string) string {
	if ok {
		return trueValue
	}
	return falseValue
}
