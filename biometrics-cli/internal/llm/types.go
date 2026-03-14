package llm

import (
	"context"
	"time"

	"biometrics-cli/internal/contracts"
)

type Request struct {
	RunID           string
	TaskID          string
	Agent           string
	TaskName        string
	ProjectID       string
	Prompt          string
	ModelPreference string
	FallbackChain   []string
	ModelID         string
	ContextBudget   int
}

type Response struct {
	Output        string
	Provider      string
	ModelID       string
	LatencyMs     int64
	TokensIn      int
	TokensOut     int
	ProviderTrail []contracts.ProviderAttempt
	CreatedAt     time.Time
}

type Provider interface {
	ID() string
	Name() string
	Available(context.Context) bool
	Generate(context.Context, Request) (Response, error)
}

type Gateway interface {
	Execute(context.Context, Request) (Response, error)
	ModelCatalog(context.Context) contracts.ModelCatalog
	MetricsSnapshot() map[string]int64
}
