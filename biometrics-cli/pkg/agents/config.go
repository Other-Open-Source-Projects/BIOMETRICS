package agents

import "time"

// AgentConfig represents an AI agent configuration
type AgentConfig struct {
	ID           string            `yaml:"id" json:"id"`
	Name         string            `yaml:"name" json:"name"`
	Provider     string            `yaml:"provider" json:"provider"`
	Model        string            `yaml:"model" json:"model"`
	Capabilities []string          `yaml:"capabilities" json:"capabilities"`
	MaxRetries   int               `yaml:"max_retries" json:"max_retries"`
	Timeout      time.Duration     `yaml:"timeout" json:"timeout"`
	Tools        []string          `yaml:"tools" json:"tools"`
	Variables    map[string]string `yaml:"variables" json:"variables"`
}

// NewAgentConfig creates a new agent configuration
func NewAgentConfig(id, name, provider, model string) *AgentConfig {
	return &AgentConfig{
		ID:           id,
		Name:         name,
		Provider:     provider,
		Model:        model,
		Capabilities: []string{},
		MaxRetries:   3,
		Timeout:      10 * time.Minute,
		Tools:        []string{},
		Variables:    map[string]string{},
	}
}

// DefaultAgents returns default agent configurations
func DefaultAgents() []*AgentConfig {
	return []*AgentConfig{
		{
			ID:           "sisyphus",
			Name:         "Sisyphus - Main Coder",
			Provider:     "nvidia",
			Model:        "google/antigravity-gemini-3.1-pro",
			Capabilities: []string{"code", "refactor", "debug"},
			MaxRetries:   3,
			Timeout:      10 * time.Minute,
			Tools:        []string{"read", "write", "bash", "grep"},
		},
		{
			ID:           "prometheus",
			Name:         "Prometheus - Planner",
			Provider:     "nvidia",
			Model:        "google/antigravity-gemini-3.1-pro",
			Capabilities: []string{"planning", "architecture", "research"},
			MaxRetries:   2,
			Timeout:      15 * time.Minute,
			Tools:        []string{"read", "write", "glob"},
		},
		{
			ID:           "oracle",
			Name:         "Oracle - Architect",
			Provider:     "nvidia",
			Model:        "google/antigravity-gemini-3.1-pro",
			Capabilities: []string{"architecture", "review", "security"},
			MaxRetries:   2,
			Timeout:      20 * time.Minute,
			Tools:        []string{"read", "grep"},
		},
		{
			ID:           "atlas",
			Name:         "Atlas - Heavy Lifting",
			Provider:     "moonshot",
			Model:        "moonshotai/kimi-k2.5",
			Capabilities: []string{"implementation", "testing", "deployment"},
			MaxRetries:   3,
			Timeout:      30 * time.Minute,
			Tools:        []string{"read", "write", "bash", "test"},
		},
		{
			ID:           "librarian",
			Name:         "Librarian - Documentation",
			Provider:     "opencode-zen",
			Model:        "opencode/minimax-m2.5-free",
			Capabilities: []string{"documentation", "research"},
			MaxRetries:   2,
			Timeout:      10 * time.Minute,
			Tools:        []string{"read", "write"},
		},
	}
}

// Provider represents an LLM provider configuration
type Provider struct {
	Name     string `yaml:"name" json:"name"`
	Type     string `yaml:"type" json:"type"` // nvidia, moonshot, opencode-zen, etc.
	Endpoint string `yaml:"endpoint" json:"endpoint"`
	APIKey   string `yaml:"api_key" json:"-"`
}

// DefaultProviders returns default provider configurations
func DefaultProviders() []*Provider {
	return []*Provider{
		{
			Name:     "nvidia",
			Type:     "nvidia-nim",
			Endpoint: "https://integrate.api.nvidia.com/v1",
		},
		{
			Name:     "moonshot",
			Type:     "moonshot-ai",
			Endpoint: "https://api.moonshot.cn/v1",
		},
		{
			Name:     "opencode-zen",
			Type:     "opencode-zen",
			Endpoint: "https://api.opencode.ai/v1",
		},
	}
}
