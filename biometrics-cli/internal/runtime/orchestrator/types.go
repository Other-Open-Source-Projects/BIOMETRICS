package orchestrator

import (
	"math"
	"strings"
	"time"
)

type StrategyMode string

const (
	StrategyDeterministic StrategyMode = "deterministic"
	StrategyAdaptive      StrategyMode = "adaptive"
	StrategyArena         StrategyMode = "arena"
)

func NormalizeStrategyMode(raw string) StrategyMode {
	switch StrategyMode(strings.ToLower(strings.TrimSpace(raw))) {
	case StrategyAdaptive:
		return StrategyAdaptive
	case StrategyArena:
		return StrategyArena
	default:
		return StrategyDeterministic
	}
}

type AgentProfile struct {
	Name           string   `json:"name"`
	AllowedTools   []string `json:"allowed_tools,omitempty"`
	MaxParallelism int      `json:"max_parallelism,omitempty"`
	ModelPolicy    string   `json:"model_policy,omitempty"`
}

type PolicyProfile struct {
	Exfiltration string `json:"exfiltration,omitempty"`
	Secrets      string `json:"secrets,omitempty"`
	Filesystem   string `json:"filesystem,omitempty"`
	Network      string `json:"network,omitempty"`
	Approvals    string `json:"approvals,omitempty"`
}

func NormalizePolicyProfile(raw PolicyProfile) PolicyProfile {
	out := PolicyProfile{
		Exfiltration: strings.ToLower(strings.TrimSpace(raw.Exfiltration)),
		Secrets:      strings.ToLower(strings.TrimSpace(raw.Secrets)),
		Filesystem:   strings.ToLower(strings.TrimSpace(raw.Filesystem)),
		Network:      strings.ToLower(strings.TrimSpace(raw.Network)),
		Approvals:    strings.ToLower(strings.TrimSpace(raw.Approvals)),
	}
	if out.Exfiltration == "" {
		out.Exfiltration = "balanced"
	}
	if out.Secrets == "" {
		out.Secrets = "balanced"
	}
	if out.Filesystem == "" {
		out.Filesystem = "workspace"
	}
	if out.Network == "" {
		out.Network = "restricted"
	}
	if out.Approvals == "" {
		out.Approvals = "on-risk"
	}
	return out
}

type Objective struct {
	Quality float64 `json:"quality"`
	Speed   float64 `json:"speed"`
	Cost    float64 `json:"cost"`
}

func NormalizeObjective(raw Objective) Objective {
	quality := math.Max(0, raw.Quality)
	speed := math.Max(0, raw.Speed)
	cost := math.Max(0, raw.Cost)

	sum := quality + speed + cost
	if sum <= 0 {
		return Objective{Quality: 0.5, Speed: 0.3, Cost: 0.2}
	}
	return Objective{
		Quality: quality / sum,
		Speed:   speed / sum,
		Cost:    cost / sum,
	}
}

type StepStatus string

const (
	StepPending   StepStatus = "pending"
	StepRunning   StepStatus = "running"
	StepCompleted StepStatus = "completed"
	StepFailed    StepStatus = "failed"
)

type PlanStep struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	DependsOn   []string   `json:"depends_on,omitempty"`
	Status      StepStatus `json:"status"`
	Error       string     `json:"error,omitempty"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	FinishedAt  *time.Time `json:"finished_at,omitempty"`
}

type Plan struct {
	ID            string         `json:"id"`
	ProjectID     string         `json:"project_id"`
	Goal          string         `json:"goal"`
	StrategyMode  StrategyMode   `json:"strategy_mode"`
	AgentProfiles []AgentProfile `json:"agent_profiles,omitempty"`
	PolicyProfile PolicyProfile  `json:"policy_profile"`
	Objective     Objective      `json:"objective"`
	Steps         []PlanStep     `json:"steps"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

type PlanRequest struct {
	ProjectID          string         `json:"project_id,omitempty"`
	Goal               string         `json:"goal"`
	StrategyMode       string         `json:"strategy_mode,omitempty"`
	AgentProfiles      []AgentProfile `json:"agent_profiles,omitempty"`
	PolicyProfile      PolicyProfile  `json:"policy_profile,omitempty"`
	Objective          Objective      `json:"objective,omitempty"`
	Skills             []string       `json:"skills,omitempty"`
	SkillSelectionMode string         `json:"skill_selection_mode,omitempty"`
	SchedulerMode      string         `json:"scheduler_mode,omitempty"`
	MaxParallelism     int            `json:"max_parallelism,omitempty"`
	ModelPreference    string         `json:"model_preference,omitempty"`
	FallbackChain      []string       `json:"fallback_chain,omitempty"`
	ModelID            string         `json:"model_id,omitempty"`
	ContextBudget      int            `json:"context_budget,omitempty"`
}

type RunStatus string

const (
	RunQueued    RunStatus = "queued"
	RunRunning   RunStatus = "running"
	RunPaused    RunStatus = "paused"
	RunCancelled RunStatus = "cancelled"
	RunCompleted RunStatus = "completed"
	RunFailed    RunStatus = "failed"
)

type RunRequest struct {
	PlanID                    string         `json:"plan_id,omitempty"`
	ProjectID                 string         `json:"project_id,omitempty"`
	Goal                      string         `json:"goal,omitempty"`
	StrategyMode              string         `json:"strategy_mode,omitempty"`
	AgentProfiles             []AgentProfile `json:"agent_profiles,omitempty"`
	PolicyProfile             PolicyProfile  `json:"policy_profile,omitempty"`
	Objective                 Objective      `json:"objective,omitempty"`
	Skills                    []string       `json:"skills,omitempty"`
	SkillSelectionMode        string         `json:"skill_selection_mode,omitempty"`
	SchedulerMode             string         `json:"scheduler_mode,omitempty"`
	MaxParallelism            int            `json:"max_parallelism,omitempty"`
	ModelPreference           string         `json:"model_preference,omitempty"`
	FallbackChain             []string       `json:"fallback_chain,omitempty"`
	ModelID                   string         `json:"model_id,omitempty"`
	ContextBudget             int            `json:"context_budget,omitempty"`
	OptimizerRecommendationID string         `json:"optimizer_recommendation_id,omitempty"`
	OptimizerConfidence       string         `json:"optimizer_confidence,omitempty"`
}

type Run struct {
	ID                        string         `json:"id"`
	PlanID                    string         `json:"plan_id"`
	UnderlyingRunID           string         `json:"underlying_run_id,omitempty"`
	ProjectID                 string         `json:"project_id"`
	Goal                      string         `json:"goal"`
	StrategyMode              StrategyMode   `json:"strategy_mode"`
	AgentProfiles             []AgentProfile `json:"agent_profiles,omitempty"`
	PolicyProfile             PolicyProfile  `json:"policy_profile"`
	Objective                 Objective      `json:"objective"`
	Steps                     []PlanStep     `json:"steps"`
	CurrentStepID             string         `json:"current_step_id,omitempty"`
	OptimizerRecommendationID string         `json:"optimizer_recommendation_id,omitempty"`
	OptimizerConfidence       string         `json:"optimizer_confidence,omitempty"`
	Status                    RunStatus      `json:"status"`
	Error                     string         `json:"error,omitempty"`
	CreatedAt                 time.Time      `json:"created_at"`
	UpdatedAt                 time.Time      `json:"updated_at"`
	FinishedAt                *time.Time     `json:"finished_at,omitempty"`
}

type ScoreComparison struct {
	QualityDelta                  float64 `json:"quality_delta"`
	TimeToGreenImprovementPercent float64 `json:"time_to_green_improvement_percent"`
	CostImprovementPercent        float64 `json:"cost_improvement_percent"`
	CompositeDelta                float64 `json:"composite_delta"`
}

type Scorecard struct {
	RunID                    string                     `json:"run_id"`
	UnderlyingRunID          string                     `json:"underlying_run_id,omitempty"`
	DatasetID                string                     `json:"dataset_id,omitempty"`
	EvidencePaths            []string                   `json:"evidence_paths,omitempty"`
	Comparison               map[string]ScoreComparison `json:"comparison,omitempty"`
	QualityScore             float64                    `json:"quality_score"`
	MedianTimeToGreenSeconds float64                    `json:"median_time_to_green_seconds"`
	CostPerSuccess           float64                    `json:"cost_per_success"`
	CriticalPolicyViolations int                        `json:"critical_policy_violations"`
	SuccessRate              float64                    `json:"success_rate"`
	Timeouts                 int                        `json:"timeouts"`
	DispatchP95MS            float64                    `json:"dispatch_p95_ms"`
	FallbackRate             float64                    `json:"fallback_rate"`
	BackpressurePerRun       float64                    `json:"backpressure_per_run"`
	Objective                Objective                  `json:"objective"`
	CompositeScore           float64                    `json:"composite_score"`
	Thresholds               map[string]bool            `json:"thresholds"`
	GeneratedAt              time.Time                  `json:"generated_at"`
}

type Capabilities struct {
	StrategyModes    []string `json:"strategy_modes"`
	PolicyPresets    []string `json:"policy_presets"`
	MaxParallelism   int      `json:"max_parallelism"`
	ResumeFromStep   bool     `json:"resume_from_step"`
	ArenaMode        bool     `json:"arena_mode"`
	EvalSupport      bool     `json:"eval_support"`
	DecisionExplain  bool     `json:"decision_explain"`
	AuditTrail       bool     `json:"audit_trail"`
	IdempotentStepID bool     `json:"idempotent_step_ids"`
}
