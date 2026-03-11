package policy

import "strings"

type RoutingPolicy struct {
	DefaultPrimary       string
	DefaultFallback      []string
	MaxFallbackHops      int
	DefaultContextBudget int
}

func Default() RoutingPolicy {
	return RoutingPolicy{
		DefaultPrimary:       "codex",
		DefaultFallback:      []string{"gemini", "nim"},
		MaxFallbackHops:      2,
		DefaultContextBudget: 24000,
	}
}

func NormalizeProviderID(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}
