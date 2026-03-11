package orchestrator

import "strings"

type workflowPreset struct {
	Name           string
	StrategyMode   string
	PolicyProfile  PolicyProfile
	Objective      Objective
	ContextBudget  int
	MaxParallelism int
}

var workflowPresets = map[string]workflowPreset{
	"workflow-apex-hardening": {
		Name:         "workflow-apex-hardening",
		StrategyMode: string(StrategyArena),
		PolicyProfile: PolicyProfile{
			Exfiltration: "strict",
			Secrets:      "strict",
			Filesystem:   "workspace",
			Network:      "restricted",
			Approvals:    "required",
		},
		Objective:      Objective{Quality: 0.65, Speed: 0.2, Cost: 0.15},
		ContextBudget:  36000,
		MaxParallelism: 8,
	},
	"workflow-speed-ship": {
		Name:         "workflow-speed-ship",
		StrategyMode: string(StrategyAdaptive),
		PolicyProfile: PolicyProfile{
			Exfiltration: "balanced",
			Secrets:      "balanced",
			Filesystem:   "workspace",
			Network:      "restricted",
			Approvals:    "on-risk",
		},
		Objective:      Objective{Quality: 0.35, Speed: 0.5, Cost: 0.15},
		ContextBudget:  28000,
		MaxParallelism: 12,
	},
	"workflow-cost-optimizer": {
		Name:         "workflow-cost-optimizer",
		StrategyMode: string(StrategyDeterministic),
		PolicyProfile: PolicyProfile{
			Exfiltration: "balanced",
			Secrets:      "strict",
			Filesystem:   "workspace",
			Network:      "restricted",
			Approvals:    "on-risk",
		},
		Objective:      Objective{Quality: 0.45, Speed: 0.15, Cost: 0.4},
		ContextBudget:  22000,
		MaxParallelism: 6,
	},
}

func applyWorkflowPreset(req *RunRequest) string {
	if req == nil {
		return ""
	}
	goal := strings.TrimSpace(req.Goal)
	if goal == "" {
		return ""
	}
	if !strings.HasPrefix(goal, "/workflow-") {
		return ""
	}

	parts := strings.Fields(goal)
	if len(parts) == 0 {
		return ""
	}
	workflowToken := strings.TrimPrefix(strings.TrimSpace(parts[0]), "/")
	preset, ok := workflowPresets[workflowToken]
	if !ok {
		return ""
	}

	req.Goal = strings.TrimSpace(strings.TrimPrefix(goal, parts[0]))
	if req.Goal == "" {
		req.Goal = "execute workflow " + preset.Name
	}
	if strings.TrimSpace(req.StrategyMode) == "" {
		req.StrategyMode = preset.StrategyMode
	}
	if strings.TrimSpace(req.PolicyProfile.Exfiltration) == "" {
		req.PolicyProfile = preset.PolicyProfile
	}
	if req.Objective.Quality <= 0 && req.Objective.Speed <= 0 && req.Objective.Cost <= 0 {
		req.Objective = preset.Objective
	}
	if req.ContextBudget <= 0 {
		req.ContextBudget = preset.ContextBudget
	}
	if req.MaxParallelism <= 0 {
		req.MaxParallelism = preset.MaxParallelism
	}
	return preset.Name
}
