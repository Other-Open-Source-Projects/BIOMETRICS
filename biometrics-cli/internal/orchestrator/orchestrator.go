package orchestrator

// Deprecated: This orchestrator is legacy and not part of the BIOMETRICS V3 control-plane runtime.
// Prefer `biometrics-cli/cmd/controlplane` scheduling and runtime orchestration.

import (
	"biometrics-cli/internal/metrics"
	"biometrics-cli/internal/state"
	"biometrics-cli/internal/tracker"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

type AgentConfig struct {
	Name          string   `json:"name"`
	Model         string   `json:"model"`
	MaxConcurrent int      `json:"max_concurrent"`
	Priority      int      `json:"priority"`
	Skills        []string `json:"skills"`
	Enabled       bool     `json:"enabled"`
}

type TodoTask struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Priority    string    `json:"priority"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	Agent       string    `json:"agent"`
}

type Orchestrator struct {
	agents       map[string]*AgentConfig
	todos        []TodoTask
	modelTracker *tracker.ModelTracker
	autoMode     bool
	cycleCount   int
}

var DefaultOrchestrator = &Orchestrator{
	agents:   make(map[string]*AgentConfig),
	todos:    make([]TodoTask, 0),
	autoMode: true,
}

func Init() {
	DefaultOrchestrator.agents = map[string]*AgentConfig{
		"sisyphus": {
			Name:          "sisyphus",
			Model:         "google/antigravity-gemini-3.1-pro",
			MaxConcurrent: 1,
			Priority:      10,
			Skills:        []string{"coding", "implementation"},
			Enabled:       true,
		},
		"prometheus": {
			Name:          "prometheus",
			Model:         "google/antigravity-gemini-3.1-pro",
			MaxConcurrent: 1,
			Priority:      9,
			Skills:        []string{"planning", "architecture"},
			Enabled:       true,
		},
		"atlas": {
			Name:          "atlas",
			Model:         "google/antigravity-gemini-3-flash",
			MaxConcurrent: 1,
			Priority:      8,
			Skills:        []string{"heavy-lifting", "execution"},
			Enabled:       true,
		},
		"librarian": {
			Name:          "librarian",
			Model:         "opencode/minimax-m2.5-free",
			MaxConcurrent: 5,
			Priority:      7,
			Skills:        []string{"research", "documentation"},
			Enabled:       true,
		},
		"explore": {
			Name:          "explore",
			Model:         "opencode/minimax-m2.5-free",
			MaxConcurrent: 10,
			Priority:      6,
			Skills:        []string{"discovery", "scanning"},
			Enabled:       true,
		},
	}

	DefaultOrchestrator.modelTracker = tracker.NewModelTracker()
	state.GlobalState.Log("INFO", "Orchestrator initialized with agents: sisyphus, prometheus, atlas, librarian, explore")
}

func (o *Orchestrator) GetAgentForTask(taskType string) *AgentConfig {
	skills := map[string][]string{
		"coding":        {"sisyphus", "atlas"},
		"planning":      {"prometheus"},
		"research":      {"librarian", "explore"},
		"documentation": {"librarian"},
		"heavy":         {"atlas"},
		"default":       {"sisyphus", "prometheus", "atlas"},
	}

	candidates := skills["default"]
	if agents, ok := skills[taskType]; ok {
		candidates = agents
	}

	for _, agentName := range candidates {
		if agent, ok := o.agents[agentName]; ok && agent.Enabled {
			if o.canRunAgent(agent) {
				return agent
			}
		}
	}

	return o.agents["sisyphus"]
}

func (o *Orchestrator) canRunAgent(agent *AgentConfig) bool {
	count := o.getRunningAgentCount(agent.Name)
	return count < agent.MaxConcurrent
}

func (o *Orchestrator) getRunningAgentCount(agentName string) int {
	count := 0
	for _, todo := range o.todos {
		if todo.Agent == agentName && todo.Status == "running" {
			count++
		}
	}
	return count
}

func (o *Orchestrator) AutoCreateTodos() {
	if len(o.todos) > 0 {
		return
	}

	idleTasks := []TodoTask{
		{
			ID:          fmt.Sprintf("auto-%d", time.Now().Unix()),
			Title:       "Check system health status",
			Description: "Run health checks on all services",
			Priority:    "medium",
			Status:      "pending",
			CreatedAt:   time.Now(),
			Agent:       "sisyphus",
		},
		{
			ID:          fmt.Sprintf("auto-%d", time.Now().Unix()+1),
			Title:       "Review active sessions",
			Description: "Check for idle or stuck sessions",
			Priority:    "low",
			Status:      "pending",
			CreatedAt:   time.Now(),
			Agent:       "explore",
		},
		{
			ID:          fmt.Sprintf("auto-%d", time.Now().Unix()+2),
			Title:       "Update documentation",
			Description: "Review and update project docs",
			Priority:    "low",
			Status:      "pending",
			CreatedAt:   time.Now(),
			Agent:       "librarian",
		},
	}

	o.todos = append(o.todos, idleTasks...)
	metrics.TasksCreatedTotal.Add(float64(len(idleTasks)))
	state.GlobalState.Log("INFO", fmt.Sprintf("Auto-created %d idle tasks", len(idleTasks)))
}

func (o *Orchestrator) GetNextTask() *TodoTask {
	for i := range o.todos {
		if o.todos[i].Status == "pending" {
			o.todos[i].Status = "running"
			return &o.todos[i]
		}
	}
	return nil
}

func (o *Orchestrator) CompleteTask(taskID string) {
	for i := range o.todos {
		if o.todos[i].ID == taskID {
			o.todos[i].Status = "completed"
			metrics.TasksCompletedTotal.Inc()
			state.GlobalState.Log("INFO", fmt.Sprintf("Task completed: %s", taskID))
			return
		}
	}
}

func (o *Orchestrator) FailTask(taskID string, err string) {
	for i := range o.todos {
		if o.todos[i].ID == taskID {
			o.todos[i].Status = "failed"
			metrics.TasksFailedTotal.Inc()
			state.GlobalState.Log("ERROR", fmt.Sprintf("Task failed: %s - %s", taskID, err))
			return
		}
	}
}

func (o *Orchestrator) GetStats() map[string]interface{} {
	pending := 0
	running := 0
	completed := 0
	failed := 0

	for _, t := range o.todos {
		switch t.Status {
		case "pending":
			pending++
		case "running":
			running++
		case "completed":
			completed++
		case "failed":
			failed++
		}
	}

	return map[string]interface{}{
		"total_tasks":   len(o.todos),
		"pending":       pending,
		"running":       running,
		"completed":     completed,
		"failed":        failed,
		"cycle_count":   o.cycleCount,
		"auto_mode":     o.autoMode,
		"active_agents": o.getActiveAgents(),
	}
}

func (o *Orchestrator) getActiveAgents() []string {
	active := make(map[string]bool)
	for _, t := range o.todos {
		if t.Status == "running" {
			active[t.Agent] = true
		}
	}

	agents := make([]string, 0, len(active))
	for agent := range active {
		agents = append(agents, agent)
	}
	return agents
}

func (o *Orchestrator) RunCycle() {
	o.cycleCount++
	metrics.CyclesTotal.Inc()

	if o.autoMode {
		o.AutoCreateTodos()
	}

	task := o.GetNextTask()
	if task == nil {
		state.GlobalState.Log("INFO", "No tasks to process")
		time.Sleep(30 * time.Second)
		return
	}

	agent := o.GetAgentForTask("default")
	if agent == nil {
		o.FailTask(task.ID, "No available agent")
		return
	}

	state.GlobalState.Log("INFO", fmt.Sprintf("Executing task %s with agent %s", task.ID, agent.Name))

	args := []string{"run", "--agent", agent.Name}
	if strings.TrimSpace(agent.Model) != "" {
		args = append(args, "--model", strings.TrimSpace(agent.Model))
	}
	if dir := resolveOpenCodeRunDir(); dir != "" {
		args = append(args, "--dir", dir)
	}
	args = append(args, task.Description)
	cmd := exec.Command("opencode", args...)
	cmd.Start()

	time.Sleep(10 * time.Second)
	o.CompleteTask(task.ID)
}

func (o *Orchestrator) Start(autoMode bool) {
	o.autoMode = autoMode
	state.GlobalState.Log("INFO", "Starting orchestrator in auto mode")

	for {
		o.RunCycle()
		time.Sleep(60 * time.Second)
	}
}

func init() {
	Init()
}

func resolveOpenCodeRunDir() string {
	for _, candidate := range []string{
		strings.TrimSpace(os.Getenv("BIOMETRICS_OPENCODE_DIR")),
		strings.TrimSpace(os.Getenv("BIOMETRICS_WORKSPACE")),
	} {
		if candidate == "" {
			continue
		}
		if stat, err := os.Stat(candidate); err == nil && stat.IsDir() {
			return candidate
		}
	}
	if cwd, err := os.Getwd(); err == nil {
		if stat, statErr := os.Stat(cwd); statErr == nil && stat.IsDir() {
			return cwd
		}
	}
	return ""
}
