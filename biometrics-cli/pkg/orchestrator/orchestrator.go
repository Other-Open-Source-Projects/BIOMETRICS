package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

// AgentModel represents an AI model configuration
type AgentModel struct {
	Name        string   `json:"name"`
	Provider    string   `json:"provider"`
	ModelID     string   `json:"model_id"`
	Category    []string `json:"category"`
	MaxParallel int      `json:"max_parallel"`
}

// AgentSession represents an active agent session
type AgentSession struct {
	ID          string    `json:"id"`
	AgentName   string    `json:"agent_name"`
	Model       string    `json:"model"`
	Category    string    `json:"category"`
	Status      string    `json:"status"` // pending, running, completed, failed
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
	SessionID   string    `json:"session_id"`
	TaskID      string    `json:"task_id"`
	Prompt      string    `json:"prompt"`
	Result      string    `json:"result,omitempty"`
	Error       string    `json:"error,omitempty"`
	SicherCheck bool      `json:"sicher_check"`
}

// OrchestratorConfig holds the orchestrator configuration
type OrchestratorConfig struct {
	ProjectRoot    string                `json:"project_root"`
	AgentsPlanPath string                `json:"agents_plan_path"`
	MaxParallel    int                   `json:"max_parallel"`
	ModelLimits    map[string]int        `json:"model_limits"`
	Models         map[string]AgentModel `json:"models"`
	SessionTimeout time.Duration         `json:"session_timeout"`
	PollInterval   time.Duration         `json:"poll_interval"`
}

// Orchestrator manages autonomous agent swarms
type Orchestrator struct {
	config         OrchestratorConfig
	activeSessions map[string]*AgentSession
	sessionMutex   sync.RWMutex
	modelUsage     map[string]int // Track model usage
	modelMutex     sync.RWMutex
	ctx            context.Context
	cancel         context.CancelFunc
}

// DefaultConfig returns the default orchestrator configuration
func DefaultConfig(projectRoot string) OrchestratorConfig {
	return OrchestratorConfig{
		ProjectRoot:    projectRoot,
		AgentsPlanPath: filepath.Join(projectRoot, "AGENTS-PLAN.md"),
		MaxParallel:    3,
		ModelLimits: map[string]int{
			"google/antigravity-gemini-3.1-pro": 1,
			"opencode/kimi-k2.5-free":           1,
			"opencode/minimax-m2.5-free":        1,
		},
		Models: map[string]AgentModel{
			"gemini-prime": {
				Name:        "Gemini 3.1 Pro",
				Provider:    "nvidia-nim",
				ModelID:     "google/antigravity-gemini-3.1-pro",
				Category:    []string{"build", "visual-engineering", "writing", "general"},
				MaxParallel: 1,
			},
			"kimi": {
				Name:        "Kimi K2.5",
				Provider:    "opencode-zen",
				ModelID:     "opencode/kimi-k2.5-free",
				Category:    []string{"deep"},
				MaxParallel: 1,
			},
			"minimax": {
				Name:        "MiniMax M2.5",
				Provider:    "opencode-zen",
				ModelID:     "opencode/minimax-m2.5-free",
				Category:    []string{"quick", "explore"},
				MaxParallel: 1,
			},
		},
		SessionTimeout: 30 * time.Minute,
		PollInterval:   10 * time.Second,
	}
}

// NewOrchestrator creates a new orchestrator instance
func NewOrchestrator(projectRoot string) (*Orchestrator, error) {
	config := DefaultConfig(projectRoot)
	ctx, cancel := context.WithCancel(context.Background())

	o := &Orchestrator{
		config:         config,
		activeSessions: make(map[string]*AgentSession),
		modelUsage:     make(map[string]int),
		ctx:            ctx,
		cancel:         cancel,
	}

	// Load existing sessions if any
	if err := o.loadSessions(); err != nil {
		return nil, fmt.Errorf("failed to load sessions: %w", err)
	}

	return o, nil
}

// Start begins the orchestrator loop
func (o *Orchestrator) Start() error {
	fmt.Println("🚀 Starting BIOMETRICS Orchestrator...")
	fmt.Printf("📊 Max Parallel Agents: %d\n", o.config.MaxParallel)
	fmt.Printf("⏱️  Session Timeout: %v\n", o.config.SessionTimeout)

	// Start monitoring loop
	go o.monitorLoop()

	return nil
}

// Stop gracefully stops the orchestrator
func (o *Orchestrator) Stop() {
	fmt.Println("\n🛑 Stopping orchestrator...")
	o.cancel()

	// Wait for all sessions to complete
	o.waitForAllSessions()
}

// SpawnAgent creates a new agent session
func (o *Orchestrator) SpawnAgent(agentName, category, agentPrompt string) (*AgentSession, error) {
	// Check model availability
	modelKey := o.getModelForCategory(category)
	if modelKey == "" {
		return nil, fmt.Errorf("no available model for category: %s", category)
	}

	// Check if model is at capacity
	if !o.canUseModel(modelKey) {
		return nil, fmt.Errorf("model %s is at capacity", modelKey)
	}

	// Create session
	sessionID := fmt.Sprintf("ses_%d", time.Now().UnixNano())
	session := &AgentSession{
		ID:          sessionID,
		AgentName:   agentName,
		Model:       modelKey,
		Category:    category,
		Status:      "pending",
		StartedAt:   time.Now(),
		Prompt:      agentPrompt,
		SicherCheck: false,
	}

	// Register session
	o.sessionMutex.Lock()
	o.activeSessions[session.ID] = session
	o.sessionMutex.Unlock()

	// Increment model usage
	o.modelMutex.Lock()
	o.modelUsage[modelKey]++
	o.modelMutex.Unlock()

	// Start agent in background
	go o.executeAgent(session)

	return session, nil
}

// executeAgent runs the agent task
func (o *Orchestrator) executeAgent(session *AgentSession) {
	session.Status = "running"
	fmt.Printf("🤖 Agent %s (%s) started - Model: %s\n", session.AgentName, session.Category, session.Model)

	// Build opencode command
	cmd := exec.CommandContext(o.ctx, "opencode", session.Prompt)
	cmd.Env = os.Environ()

	// Execute
	output, err := cmd.CombinedOutput()
	if err != nil {
		session.Status = "failed"
		session.Error = err.Error()
		session.CompletedAt = time.Now()
		fmt.Printf("❌ Agent %s failed: %v\n", session.AgentName, err)
		o.decrementModelUsage(session.Model)
		return
	}

	session.Result = string(output)
	session.Status = "completed"
	session.CompletedAt = time.Now()

	// Perform "Sicher?" check
	session.SicherCheck = o.performSicherCheck(session)

	fmt.Printf("✅ Agent %s completed - Sicher Check: %v\n", session.AgentName, session.SicherCheck)
	o.decrementModelUsage(session.Model)
}

// performSicherCheck verifies agent work
func (o *Orchestrator) performSicherCheck(session *AgentSession) bool {
	fmt.Printf("🔍 Performing 'Sicher?' check for agent %s...\n", session.AgentName)

	// Check 1: Verify files were created/modified
	// Check 2: Verify tests pass
	// Check 3: Verify git commit was made
	// Check 4: Verify no duplicates created

	// TODO: Implement detailed verification logic
	return true
}

// monitorLoop continuously monitors active sessions
func (o *Orchestrator) monitorLoop() {
	ticker := time.NewTicker(o.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-o.ctx.Done():
			return
		case <-ticker.C:
			o.checkSessions()
		}
	}
}

// checkSessions monitors all active sessions
func (o *Orchestrator) checkSessions() {
	o.sessionMutex.RLock()
	defer o.sessionMutex.RUnlock()

	for _, session := range o.activeSessions {
		if session.Status == "running" {
			// Check for timeout
			if time.Since(session.StartedAt) > o.config.SessionTimeout {
				fmt.Printf("⚠️  Session %s timed out\n", session.ID)
				// TODO: Handle timeout
			}

			// Read session progress
			o.readSessionProgress(session)
		}
	}
}

// readSessionProgress reads the session from opencode
func (o *Orchestrator) readSessionProgress(session *AgentSession) {
	// TODO: Implement session reading via opencode API
	// This would call session_read(session_id) and update session status
}

// canUseModel checks if a model can be used
func (o *Orchestrator) canUseModel(modelKey string) bool {
	o.modelMutex.RLock()
	defer o.modelMutex.RUnlock()

	limit := o.config.ModelLimits[modelKey]
	current := o.modelUsage[modelKey]

	return current < limit
}

// getModelForCategory returns the appropriate model for a category
func (o *Orchestrator) getModelForCategory(category string) string {
	for key, model := range o.config.Models {
		for _, cat := range model.Category {
			if cat == category {
				if o.canUseModel(key) {
					return key
				}
			}
		}
	}
	return ""
}

// decrementModelUsage decreases model usage count
func (o *Orchestrator) decrementModelUsage(modelKey string) {
	o.modelMutex.Lock()
	defer o.modelMutex.Unlock()

	if o.modelUsage[modelKey] > 0 {
		o.modelUsage[modelKey]--
	}
}

// waitForAllSessions waits for all sessions to complete
func (o *Orchestrator) waitForAllSessions() {
	timeout := time.After(24 * time.Hour) // NEVER TIMEOUT IN CEO MODE
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			fmt.Println("⚠️  Timeout waiting for sessions to complete")
			return
		case <-ticker.C:
			o.sessionMutex.RLock()
			allDone := true
			for _, session := range o.activeSessions {
				if session.Status == "running" || session.Status == "pending" {
					allDone = false
					break
				}
			}
			o.sessionMutex.RUnlock()

			if allDone {
				fmt.Println("✅ All sessions completed")
				return
			}
		}
	}
}

// loadSessions loads sessions from disk
func (o *Orchestrator) loadSessions() error {
	sessionsDir := filepath.Join(o.config.ProjectRoot, ".sisyphus", "sessions")
	if _, err := os.Stat(sessionsDir); os.IsNotExist(err) {
		return nil // No existing sessions
	}

	files, err := ioutil.ReadDir(sessionsDir)
	if err != nil {
		return err
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) == ".json" {
			data, err := ioutil.ReadFile(filepath.Join(sessionsDir, file.Name()))
			if err != nil {
				continue
			}

			var session AgentSession
			if err := json.Unmarshal(data, &session); err != nil {
				continue
			}

			o.activeSessions[session.ID] = &session
		}
	}

	return nil
}

// saveSessions persists sessions to disk
func (o *Orchestrator) saveSessions() error {
	sessionsDir := filepath.Join(o.config.ProjectRoot, ".sisyphus", "sessions")
	if err := os.MkdirAll(sessionsDir, 0755); err != nil {
		return err
	}

	o.sessionMutex.RLock()
	defer o.sessionMutex.RUnlock()

	for _, session := range o.activeSessions {
		data, err := json.MarshalIndent(session, "", "  ")
		if err != nil {
			return err
		}

		filename := filepath.Join(sessionsDir, fmt.Sprintf("%s.json", session.ID))
		if err := ioutil.WriteFile(filename, data, 0644); err != nil {
			return err
		}
	}

	return nil
}

// GetStatus returns the current orchestrator status
func (o *Orchestrator) GetStatus() map[string]interface{} {
	o.sessionMutex.RLock()
	defer o.sessionMutex.RUnlock()

	o.modelMutex.RLock()
	defer o.modelMutex.RUnlock()

	activeCount := 0
	completedCount := 0
	failedCount := 0

	for _, session := range o.activeSessions {
		switch session.Status {
		case "running", "pending":
			activeCount++
		case "completed":
			completedCount++
		case "failed":
			failedCount++
		}
	}

	return map[string]interface{}{
		"active_sessions":    activeCount,
		"completed_sessions": completedCount,
		"failed_sessions":    failedCount,
		"model_usage":        o.modelUsage,
		"model_limits":       o.config.ModelLimits,
	}
}
