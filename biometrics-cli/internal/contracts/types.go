package contracts

import (
	"context"
	"strings"
	"time"
)

type RunStatus string

const (
	RunQueued    RunStatus = "queued"
	RunRunning   RunStatus = "running"
	RunPaused    RunStatus = "paused"
	RunCancelled RunStatus = "cancelled"
	RunCompleted RunStatus = "completed"
	RunFailed    RunStatus = "failed"
)

type RunMode string

const (
	RunModeAutonomous RunMode = "autonomous"
	RunModeSupervised RunMode = "supervised"
)

func NormalizeRunMode(raw string) RunMode {
	mode := strings.ToLower(strings.TrimSpace(raw))
	switch RunMode(mode) {
	case RunModeSupervised:
		return RunModeSupervised
	default:
		return RunModeAutonomous
	}
}

func IsValidRunMode(raw string) bool {
	mode := strings.ToLower(strings.TrimSpace(raw))
	return mode == string(RunModeAutonomous) || mode == string(RunModeSupervised)
}

type SchedulerMode string

const (
	SchedulerModeDAGParallelV1 SchedulerMode = "dag_parallel_v1"
	SchedulerModeSerial        SchedulerMode = "serial"
)

type TaskStatus string

const (
	TaskPending   TaskStatus = "pending"
	TaskRunning   TaskStatus = "running"
	TaskCompleted TaskStatus = "completed"
	TaskFailed    TaskStatus = "failed"
	TaskSkipped   TaskStatus = "skipped"
	TaskCancelled TaskStatus = "cancelled"
)

type TaskLifecycleState string

const (
	TaskLifecycleBlocked   TaskLifecycleState = "blocked"
	TaskLifecycleReady     TaskLifecycleState = "ready"
	TaskLifecycleRunning   TaskLifecycleState = "running"
	TaskLifecycleRetrying  TaskLifecycleState = "retrying"
	TaskLifecycleCompleted TaskLifecycleState = "completed"
	TaskLifecycleFailed    TaskLifecycleState = "failed"
	TaskLifecycleSkipped   TaskLifecycleState = "skipped"
	TaskLifecycleCancelled TaskLifecycleState = "cancelled"
)

type Run struct {
	ID                        string        `json:"id"`
	ProjectID                 string        `json:"project_id"`
	Goal                      string        `json:"goal"`
	Mode                      string        `json:"mode"`
	Skills                    []string      `json:"skills,omitempty"`
	SkillSelectionMode        string        `json:"skill_selection_mode,omitempty"`
	Status                    RunStatus     `json:"status"`
	Error                     string        `json:"error,omitempty"`
	SchedulerMode             SchedulerMode `json:"scheduler_mode,omitempty"`
	MaxParallelism            int           `json:"max_parallelism,omitempty"`
	FallbackTriggered         bool          `json:"fallback_triggered,omitempty"`
	ModelPreference           string        `json:"model_preference,omitempty"`
	FallbackChain             []string      `json:"fallback_chain,omitempty"`
	ModelID                   string        `json:"model_id,omitempty"`
	ContextBudget             int           `json:"context_budget,omitempty"`
	BlueprintProfile          string        `json:"blueprint_profile,omitempty"`
	BlueprintModules          []string      `json:"blueprint_modules,omitempty"`
	Bootstrap                 bool          `json:"bootstrap,omitempty"`
	OptimizerRecommendationID string        `json:"optimizer_recommendation_id,omitempty"`
	OptimizerConfidence       string        `json:"optimizer_confidence,omitempty"`
	CreatedAt                 time.Time     `json:"created_at"`
	UpdatedAt                 time.Time     `json:"updated_at"`
}

type Task struct {
	ID             string             `json:"id"`
	RunID          string             `json:"run_id"`
	Name           string             `json:"name"`
	Agent          string             `json:"agent"`
	Status         TaskStatus         `json:"status"`
	LifecycleState TaskLifecycleState `json:"lifecycle_state,omitempty"`
	DependsOn      []string           `json:"depends_on"`
	Priority       int                `json:"priority,omitempty"`
	MaxAttempts    int                `json:"max_attempts,omitempty"`
	Attempt        int                `json:"attempt"`
	LastError      string             `json:"last_error,omitempty"`
	CreatedAt      time.Time          `json:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at"`
	StartedAt      *time.Time         `json:"started_at,omitempty"`
	FinishedAt     *time.Time         `json:"finished_at,omitempty"`
	CompletedAt    *time.Time         `json:"completed_at,omitempty"`
}

type TaskAttempt struct {
	ID            string            `json:"id"`
	RunID         string            `json:"run_id"`
	TaskID        string            `json:"task_id"`
	Agent         string            `json:"agent"`
	Status        string            `json:"status"`
	Log           string            `json:"log,omitempty"`
	Error         string            `json:"error,omitempty"`
	Provider      string            `json:"provider,omitempty"`
	ModelID       string            `json:"model_id,omitempty"`
	FallbackIndex int               `json:"fallback_index,omitempty"`
	LatencyMs     int64             `json:"latency_ms,omitempty"`
	TokensIn      int               `json:"tokens_in,omitempty"`
	TokensOut     int               `json:"tokens_out,omitempty"`
	ProviderTrail []ProviderAttempt `json:"provider_trail,omitempty"`
	StartedAt     time.Time         `json:"started_at"`
	FinishedAt    time.Time         `json:"finished_at"`
}

type Artifact struct {
	Type        string    `json:"type"`
	Path        string    `json:"path"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type RunPolicy struct {
	AutonomousDefault bool     `json:"autonomous_default"`
	AllowedCommands   []string `json:"allowed_commands"`
	SecretRedaction   bool     `json:"secret_redaction"`
	AllowFSWrite      bool     `json:"allow_fs_write"`
	AllowGitCommit    bool     `json:"allow_git_commit"`
}

type AgentEnvelope struct {
	RunID              string            `json:"run_id"`
	TaskID             string            `json:"task_id"`
	TaskName           string            `json:"task_name"`
	ProjectID          string            `json:"project_id"`
	Goal               string            `json:"goal"`
	Prompt             string            `json:"prompt"`
	Attempt            int               `json:"attempt"`
	Skills             []string          `json:"skills,omitempty"`
	SkillSelectionMode string            `json:"skill_selection_mode,omitempty"`
	ModelPreference    string            `json:"model_preference,omitempty"`
	FallbackChain      []string          `json:"fallback_chain,omitempty"`
	ModelID            string            `json:"model_id,omitempty"`
	ContextBudget      int               `json:"context_budget,omitempty"`
	BlueprintProfile   string            `json:"blueprint_profile,omitempty"`
	BlueprintModules   []string          `json:"blueprint_modules,omitempty"`
	Bootstrap          bool              `json:"bootstrap,omitempty"`
	Input              map[string]string `json:"input,omitempty"`
	Ctx                context.Context   `json:"-"`
	ResponseCh         chan AgentResult  `json:"-"`
	DispatchedAt       time.Time         `json:"dispatched_at"`
}

type AgentResult struct {
	RunID         string            `json:"run_id"`
	TaskID        string            `json:"task_id"`
	Agent         string            `json:"agent"`
	Success       bool              `json:"success"`
	Summary       string            `json:"summary"`
	Error         string            `json:"error,omitempty"`
	Artifacts     []Artifact        `json:"artifacts,omitempty"`
	Provider      string            `json:"provider,omitempty"`
	ModelID       string            `json:"model_id,omitempty"`
	LatencyMs     int64             `json:"latency_ms,omitempty"`
	TokensIn      int               `json:"tokens_in,omitempty"`
	TokensOut     int               `json:"tokens_out,omitempty"`
	ProviderTrail []ProviderAttempt `json:"provider_trail,omitempty"`
	FinishedAt    time.Time         `json:"finished_at"`
}

type Event struct {
	ID        string            `json:"id"`
	RunID     string            `json:"run_id,omitempty"`
	Type      string            `json:"type"`
	Source    string            `json:"source"`
	Payload   map[string]string `json:"payload,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
}

type ProviderAttempt struct {
	Provider      string `json:"provider"`
	ModelID       string `json:"model_id,omitempty"`
	Status        string `json:"status"`
	ErrorClass    string `json:"error_class,omitempty"`
	ErrorMessage  string `json:"error_message,omitempty"`
	LatencyMs     int64  `json:"latency_ms,omitempty"`
	FallbackIndex int    `json:"fallback_index,omitempty"`
}

type ModelProvider struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Status      string        `json:"status"`
	Default     bool          `json:"default"`
	Available   bool          `json:"available"`
	ModelID     string        `json:"model_id,omitempty"`
	Models      []ModelOption `json:"models,omitempty"`
	Description string        `json:"description,omitempty"`
}

type ModelOption struct {
	ID      string `json:"id"`
	Name    string `json:"name,omitempty"`
	Default bool   `json:"default,omitempty"`
}

type ModelCatalog struct {
	DefaultPrimary string          `json:"default_primary"`
	DefaultChain   []string        `json:"default_chain"`
	Providers      []ModelProvider `json:"providers"`
}

type CodexAuthStatus struct {
	Ready       bool      `json:"ready"`
	LoggedIn    bool      `json:"logged_in"`
	User        string    `json:"user,omitempty"`
	LastError   string    `json:"last_error,omitempty"`
	LastChecked time.Time `json:"last_checked,omitempty"`
}

type WorkPackage struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	DependsOn []string `json:"depends_on,omitempty"`
	Priority  int      `json:"priority"`
	Scope     []string `json:"scope,omitempty"`
}

type PlannerPlan struct {
	Version      int           `json:"version"`
	WorkPackages []WorkPackage `json:"work_packages"`
}

type TaskGraphNode struct {
	ID             string             `json:"id"`
	Name           string             `json:"name"`
	Agent          string             `json:"agent"`
	DependsOn      []string           `json:"depends_on,omitempty"`
	Priority       int                `json:"priority"`
	Status         TaskStatus         `json:"status"`
	LifecycleState TaskLifecycleState `json:"lifecycle_state"`
}

type TaskGraphEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type TaskGraph struct {
	RunID        string          `json:"run_id"`
	Nodes        []TaskGraphNode `json:"nodes"`
	Edges        []TaskGraphEdge `json:"edges"`
	CriticalPath []string        `json:"critical_path,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
}
