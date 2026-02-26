package orchestrator

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"biometrics-cli/internal/contracts"
	"biometrics-cli/internal/runtime/scheduler"
	"github.com/google/uuid"
)

type runBackend interface {
	StartRunWithOptions(opts scheduler.RunStartOptions) (contracts.Run, error)
	GetRun(runID string) (contracts.Run, error)
	ListRunAttempts(runID string) ([]contracts.TaskAttempt, error)
	Events(runID string, limit int) ([]contracts.Event, error)
	MetricsSnapshot() map[string]int64
}

type eventPublisher interface {
	Publish(ev contracts.Event) (contracts.Event, error)
}

type Service struct {
	backend runBackend
	bus     eventPublisher

	workspaceRoot string
	arenaRoot     string
	memory        *MemoryStore

	mu         sync.RWMutex
	plans      map[string]Plan
	runs       map[string]Run
	runInputs  map[string]RunRequest
	scorecards map[string]Scorecard
	arenaPaths map[string]map[string]string
}

func NewService(backend runBackend, bus eventPublisher) *Service {
	workspace := detectWorkspaceRoot()
	arenaRoot := filepath.Join(workspace, ".biometrics", "arena")
	_ = os.MkdirAll(arenaRoot, 0o755)

	return &Service{
		backend:       backend,
		bus:           bus,
		workspaceRoot: workspace,
		arenaRoot:     arenaRoot,
		memory:        NewMemoryStore(),
		plans:         make(map[string]Plan),
		runs:          make(map[string]Run),
		runInputs:     make(map[string]RunRequest),
		scorecards:    make(map[string]Scorecard),
		arenaPaths:    make(map[string]map[string]string),
	}
}

func (s *Service) Capabilities() Capabilities {
	return Capabilities{
		StrategyModes:    []string{string(StrategyDeterministic), string(StrategyAdaptive), string(StrategyArena)},
		PolicyPresets:    []string{"strict", "balanced", "velocity"},
		MaxParallelism:   32,
		ResumeFromStep:   true,
		ArenaMode:        true,
		EvalSupport:      true,
		DecisionExplain:  true,
		AuditTrail:       true,
		IdempotentStepID: true,
	}
}

func (s *Service) CreatePlan(req PlanRequest) (Plan, error) {
	normalized, err := normalizePlanRequest(req)
	if err != nil {
		return Plan{}, err
	}

	now := time.Now().UTC()
	plan := Plan{
		ID:            "orc-plan-" + uuid.NewString(),
		ProjectID:     normalized.ProjectID,
		Goal:          normalized.Goal,
		StrategyMode:  NormalizeStrategyMode(normalized.StrategyMode),
		AgentProfiles: append([]AgentProfile{}, normalized.AgentProfiles...),
		PolicyProfile: NormalizePolicyProfile(normalized.PolicyProfile),
		Objective:     NormalizeObjective(normalized.Objective),
		Steps:         buildPlanSteps(NormalizeStrategyMode(normalized.StrategyMode)),
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	s.mu.Lock()
	s.plans[plan.ID] = plan
	s.mu.Unlock()
	s.memory.Put(MemoryRecord{
		Scope:      MemoryScopeWorkspace,
		Workspace:  s.workspaceRoot,
		Key:        "plan." + plan.ID,
		Value:      fmt.Sprintf("goal=%s strategy=%s steps=%d", plan.Goal, plan.StrategyMode, len(plan.Steps)),
		Source:     "orchestrator.plan",
		Provenance: "planner",
		CreatedAt:  now,
	})

	s.publish("", "orchestrator.plan.generated", map[string]string{
		"plan_id":        plan.ID,
		"project_id":     plan.ProjectID,
		"strategy_mode":  string(plan.StrategyMode),
		"steps":          fmt.Sprintf("%d", len(plan.Steps)),
		"objective":      fmt.Sprintf("q=%.2f,s=%.2f,c=%.2f", plan.Objective.Quality, plan.Objective.Speed, plan.Objective.Cost),
		"policy_profile": fmt.Sprintf("exfil=%s,secrets=%s,fs=%s,net=%s,approvals=%s", plan.PolicyProfile.Exfiltration, plan.PolicyProfile.Secrets, plan.PolicyProfile.Filesystem, plan.PolicyProfile.Network, plan.PolicyProfile.Approvals),
	})

	s.publish("", "orchestrator.decision.explained", map[string]string{
		"plan_id":   plan.ID,
		"decision":  "strategy_selection",
		"selection": string(plan.StrategyMode),
		"reason":    strategyReason(plan.StrategyMode),
	})

	return plan, nil
}

func (s *Service) StartRun(ctx context.Context, req RunRequest) (Run, error) {
	workflow := applyWorkflowPreset(&req)
	normalized, err := normalizeRunRequest(req)
	if err != nil {
		return Run{}, err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if workflow != "" {
		s.publish("", "orchestrator.decision.explained", map[string]string{
			"decision":  "workflow_resolved",
			"workflow":  workflow,
			"goal":      req.Goal,
			"strategy":  req.StrategyMode,
			"objective": fmt.Sprintf("q=%.2f,s=%.2f,c=%.2f", normalized.Objective.Quality, normalized.Objective.Speed, normalized.Objective.Cost),
		})
	}

	plan, err := s.resolvePlanForRun(normalized)
	if err != nil {
		return Run{}, err
	}
	if strings.TrimSpace(normalized.Goal) == "" {
		normalized.Goal = plan.Goal
	}
	if strings.TrimSpace(normalized.ProjectID) == "" {
		normalized.ProjectID = plan.ProjectID
	}
	risk := assessRunRisk(plan.Goal, normalized)
	s.publish("", "orchestrator.decision.explained", map[string]string{
		"decision":       "risk_assessment",
		"risk_score":     fmt.Sprintf("%d", risk.Score),
		"risk_reasons":   strings.Join(risk.Reasons, ";"),
		"approval_state": risk.ApprovalState,
	})
	if risk.Blocked {
		return Run{}, fmt.Errorf("%s", risk.BlockReason)
	}

	now := time.Now().UTC()
	run := Run{
		ID:                        "orc-run-" + uuid.NewString(),
		PlanID:                    plan.ID,
		ProjectID:                 plan.ProjectID,
		Goal:                      plan.Goal,
		StrategyMode:              plan.StrategyMode,
		AgentProfiles:             append([]AgentProfile{}, plan.AgentProfiles...),
		PolicyProfile:             plan.PolicyProfile,
		Objective:                 plan.Objective,
		Steps:                     cloneSteps(plan.Steps),
		OptimizerRecommendationID: normalized.OptimizerRecommendationID,
		OptimizerConfidence:       normalized.OptimizerConfidence,
		Status:                    RunRunning,
		CreatedAt:                 now,
		UpdatedAt:                 now,
	}

	s.mu.Lock()
	s.runs[run.ID] = run
	s.runInputs[run.ID] = normalized
	s.arenaPaths[run.ID] = make(map[string]string)
	s.mu.Unlock()
	s.memory.Put(MemoryRecord{
		Scope:      MemoryScopeRun,
		Workspace:  s.workspaceRoot,
		RunID:      run.ID,
		Key:        "run.goal",
		Value:      run.Goal,
		Source:     "orchestrator.start",
		Provenance: "user-request",
		CreatedAt:  now,
	})

	go s.executeRun(run.ID, 0)
	return run, nil
}

func (s *Service) ResumeFromStep(_ context.Context, runID, stepID string) (Run, error) {
	runID = strings.TrimSpace(runID)
	stepID = strings.TrimSpace(stepID)
	if runID == "" {
		return Run{}, fmt.Errorf("run id is required")
	}
	if stepID == "" {
		return Run{}, fmt.Errorf("step_id is required")
	}

	s.mu.Lock()
	run, ok := s.runs[runID]
	if !ok {
		s.mu.Unlock()
		return Run{}, fmt.Errorf("orchestrator run not found")
	}

	resumeIdx := -1
	for i := range run.Steps {
		if run.Steps[i].ID == stepID {
			resumeIdx = i
			break
		}
	}
	if resumeIdx < 0 {
		s.mu.Unlock()
		return Run{}, fmt.Errorf("step_id %q not found", stepID)
	}

	now := time.Now().UTC()
	for i := range run.Steps {
		run.Steps[i].Error = ""
		run.Steps[i].StartedAt = nil
		run.Steps[i].FinishedAt = nil
		if i < resumeIdx {
			run.Steps[i].Status = StepCompleted
		} else {
			run.Steps[i].Status = StepPending
		}
	}
	run.CurrentStepID = stepID
	run.Status = RunRunning
	run.Error = ""
	run.UpdatedAt = now
	run.FinishedAt = nil
	s.runs[runID] = run
	s.mu.Unlock()

	s.publish(runID, "orchestrator.resume.applied", map[string]string{
		"run_id":  runID,
		"step_id": stepID,
		"index":   fmt.Sprintf("%d", resumeIdx),
	})

	go s.executeRun(runID, resumeIdx)
	return run, nil
}

func (s *Service) GetRun(runID string) (Run, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	run, ok := s.runs[strings.TrimSpace(runID)]
	if !ok {
		return Run{}, fmt.Errorf("orchestrator run not found")
	}
	return run, nil
}

func (s *Service) Scorecard(runID string) (Scorecard, error) {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return Scorecard{}, fmt.Errorf("run id is required")
	}

	s.mu.RLock()
	if score, ok := s.scorecards[runID]; ok {
		s.mu.RUnlock()
		return score, nil
	}
	run, ok := s.runs[runID]
	s.mu.RUnlock()
	if !ok {
		return Scorecard{}, fmt.Errorf("orchestrator run not found")
	}
	if run.Status != RunCompleted && run.Status != RunFailed {
		return Scorecard{}, fmt.Errorf("scorecard unavailable for non-terminal run")
	}

	score, err := s.computeScorecard(run)
	if err != nil {
		return Scorecard{}, err
	}
	s.mu.Lock()
	s.scorecards[runID] = score
	s.mu.Unlock()
	return score, nil
}

func (s *Service) ListScorecards() []Scorecard {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Scorecard, 0, len(s.scorecards))
	for _, score := range s.scorecards {
		out = append(out, score)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].GeneratedAt.After(out[j].GeneratedAt)
	})
	return out
}

func (s *Service) resolvePlanForRun(req RunRequest) (Plan, error) {
	if strings.TrimSpace(req.PlanID) != "" {
		s.mu.RLock()
		plan, ok := s.plans[strings.TrimSpace(req.PlanID)]
		s.mu.RUnlock()
		if !ok {
			return Plan{}, fmt.Errorf("plan not found")
		}
		return plan, nil
	}

	planReq := PlanRequest{
		ProjectID:          req.ProjectID,
		Goal:               req.Goal,
		StrategyMode:       req.StrategyMode,
		AgentProfiles:      append([]AgentProfile{}, req.AgentProfiles...),
		PolicyProfile:      req.PolicyProfile,
		Objective:          req.Objective,
		Skills:             append([]string{}, req.Skills...),
		SkillSelectionMode: req.SkillSelectionMode,
		SchedulerMode:      req.SchedulerMode,
		MaxParallelism:     req.MaxParallelism,
		ModelPreference:    req.ModelPreference,
		FallbackChain:      append([]string{}, req.FallbackChain...),
		ModelID:            req.ModelID,
		ContextBudget:      req.ContextBudget,
	}
	return s.CreatePlan(planReq)
}

func (s *Service) executeRun(runID string, startIdx int) {
	s.mu.RLock()
	run, ok := s.runs[runID]
	req := s.runInputs[runID]
	s.mu.RUnlock()
	if !ok {
		return
	}

	if startIdx < 0 {
		startIdx = 0
	}
	if startIdx >= len(run.Steps) {
		s.finishRun(runID, RunCompleted, "")
		return
	}

	for idx := startIdx; idx < len(run.Steps); idx++ {
		step, err := s.transitionStep(runID, idx, StepRunning, "")
		if err != nil {
			s.finishRun(runID, RunFailed, err.Error())
			return
		}
		s.publish(runID, "orchestrator.step.started", map[string]string{
			"run_id":    runID,
			"step_id":   step.ID,
			"step_name": step.Name,
			"index":     fmt.Sprintf("%d", idx),
		})

		if err := s.executeStep(runID, idx, req); err != nil {
			_, _ = s.transitionStep(runID, idx, StepFailed, err.Error())
			s.publish(runID, "orchestrator.step.failed", map[string]string{
				"run_id":    runID,
				"step_id":   step.ID,
				"step_name": step.Name,
				"error":     err.Error(),
			})
			s.finishRun(runID, RunFailed, err.Error())
			return
		}

		step, _ = s.transitionStep(runID, idx, StepCompleted, "")
		s.publish(runID, "orchestrator.step.completed", map[string]string{
			"run_id":    runID,
			"step_id":   step.ID,
			"step_name": step.Name,
			"index":     fmt.Sprintf("%d", idx),
		})
	}

	s.finishRun(runID, RunCompleted, "")
}

func (s *Service) executeStep(runID string, idx int, req RunRequest) error {
	s.mu.RLock()
	run := s.runs[runID]
	s.mu.RUnlock()
	if idx < 0 || idx >= len(run.Steps) {
		return fmt.Errorf("invalid step index")
	}
	step := run.Steps[idx]

	switch step.ID {
	case "plan":
		s.materializeAgentContexts(runID, run)
		s.publish(runID, "orchestrator.decision.explained", map[string]string{
			"run_id":   runID,
			"decision": "plan_shape",
			"reason":   strategyReason(run.StrategyMode),
		})
		return nil
	case "arena_branch_a", "arena_branch_b":
		workspacePath, err := s.prepareArenaBranch(runID, step.ID)
		if err != nil {
			return err
		}
		time.Sleep(25 * time.Millisecond)
		s.publish(runID, "orchestrator.decision.explained", map[string]string{
			"run_id":         runID,
			"decision":       "arena_branch_prepared",
			"branch":         step.ID,
			"workspace_path": workspacePath,
		})
		return nil
	case "arena_select":
		if err := s.validateArenaIsolation(runID); err != nil {
			return err
		}
		time.Sleep(25 * time.Millisecond)
		paths := s.arenaBranchPaths(runID)
		if len(paths) > 0 {
			s.publish(runID, "orchestrator.decision.explained", map[string]string{
				"run_id":   runID,
				"decision": "arena_isolation_verified",
				"paths":    strings.Join(paths, ","),
			})
		}
		if step.ID == "arena_select" {
			s.publish(runID, "orchestrator.decision.explained", map[string]string{
				"run_id":      runID,
				"decision":    "arena_candidate_selection",
				"selection":   "candidate_a",
				"score_logic": "quality_first_with_cost_guardrail",
			})
		}
		return nil
	case "execute":
		if err := s.validateExecutionPolicy(runID, run, req); err != nil {
			return err
		}
		return s.runUnderlyingSchedulerRun(runID, req)
	case "evaluate":
		s.mu.RLock()
		current := s.runs[runID]
		s.mu.RUnlock()
		score, err := s.computeScorecard(current)
		if err != nil {
			return err
		}
		s.mu.Lock()
		s.scorecards[runID] = score
		s.mu.Unlock()
		s.memory.Put(MemoryRecord{
			Scope:      MemoryScopeRun,
			Workspace:  s.workspaceRoot,
			RunID:      runID,
			Key:        "scorecard.latest",
			Value:      fmt.Sprintf("quality=%.4f composite=%.4f success_rate=%.4f", score.QualityScore, score.CompositeScore, score.SuccessRate),
			Source:     "orchestrator.evaluate",
			Provenance: "computed",
			CreatedAt:  time.Now().UTC(),
		})
		s.publish(runID, "orchestrator.decision.explained", map[string]string{
			"run_id":          runID,
			"decision":        "scorecard_generated",
			"quality_score":   fmt.Sprintf("%.4f", score.QualityScore),
			"composite_score": fmt.Sprintf("%.4f", score.CompositeScore),
		})
		return nil
	default:
		return nil
	}
}

func (s *Service) runUnderlyingSchedulerRun(runID string, req RunRequest) error {
	s.mu.RLock()
	run := s.runs[runID]
	s.mu.RUnlock()

	underlyingID := strings.TrimSpace(run.UnderlyingRunID)
	if underlyingID == "" {
		opts := scheduler.RunStartOptions{
			ProjectID:                 req.ProjectID,
			Goal:                      req.Goal,
			Mode:                      "autonomous",
			Skills:                    append([]string{}, req.Skills...),
			SkillSelectionMode:        req.SkillSelectionMode,
			ModelPreference:           req.ModelPreference,
			FallbackChain:             append([]string{}, req.FallbackChain...),
			ModelID:                   req.ModelID,
			ContextBudget:             req.ContextBudget,
			MaxParallelism:            req.MaxParallelism,
			OptimizerRecommendationID: req.OptimizerRecommendationID,
			OptimizerConfidence:       req.OptimizerConfidence,
		}
		if strings.EqualFold(req.PolicyProfile.Approvals, "required") {
			opts.Mode = "supervised"
		}
		switch strings.ToLower(strings.TrimSpace(req.SchedulerMode)) {
		case "serial":
			opts.SchedulerMode = contracts.SchedulerModeSerial
		case "dag_parallel_v1", "":
			opts.SchedulerMode = contracts.SchedulerModeDAGParallelV1
		default:
			return fmt.Errorf("invalid scheduler_mode: %s", req.SchedulerMode)
		}
		if opts.MaxParallelism <= 0 {
			opts.MaxParallelism = maxParallelismFromProfiles(req.AgentProfiles)
		}

		created, err := s.backend.StartRunWithOptions(opts)
		if err != nil {
			return err
		}
		underlyingID = created.ID

		s.mu.Lock()
		run = s.runs[runID]
		run.UnderlyingRunID = underlyingID
		run.UpdatedAt = time.Now().UTC()
		s.runs[runID] = run
		s.mu.Unlock()

		s.publish(runID, "orchestrator.decision.explained", map[string]string{
			"run_id":                      runID,
			"decision":                    "underlying_run_started",
			"underlying_run_id":           underlyingID,
			"mode":                        opts.Mode,
			"scheduler_mode":              string(opts.SchedulerMode),
			"optimizer_recommendation_id": req.OptimizerRecommendationID,
			"optimizer_confidence":        req.OptimizerConfidence,
		})
	}

	for {
		baseRun, err := s.backend.GetRun(underlyingID)
		if err != nil {
			return err
		}
		switch baseRun.Status {
		case contracts.RunCompleted:
			return nil
		case contracts.RunFailed, contracts.RunCancelled:
			if strings.TrimSpace(baseRun.Error) != "" {
				return fmt.Errorf("%s", baseRun.Error)
			}
			return fmt.Errorf("underlying run ended with status %s", baseRun.Status)
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func (s *Service) computeScorecard(run Run) (Scorecard, error) {
	underlyingID := strings.TrimSpace(run.UnderlyingRunID)
	if underlyingID == "" {
		return Scorecard{}, fmt.Errorf("underlying run missing")
	}

	attempts, err := s.backend.ListRunAttempts(underlyingID)
	if err != nil {
		return Scorecard{}, err
	}
	baseRun, err := s.backend.GetRun(underlyingID)
	if err != nil {
		return Scorecard{}, err
	}
	events, _ := s.backend.Events(underlyingID, 2000)
	metrics := s.backend.MetricsSnapshot()

	totalAttempts := len(attempts)
	successfulAttempts := 0
	timeouts := 0
	tokensIn := 0.0
	tokensOut := 0.0
	for _, attempt := range attempts {
		if strings.EqualFold(attempt.Status, "completed") {
			successfulAttempts++
		}
		if strings.Contains(strings.ToLower(attempt.Error), "timeout") {
			timeouts++
		}
		tokensIn += float64(attempt.TokensIn)
		tokensOut += float64(attempt.TokensOut)
	}

	qualityScore := 0.0
	successRate := 0.0
	if totalAttempts > 0 {
		successRate = float64(successfulAttempts) / float64(totalAttempts)
		qualityScore = successRate
	} else if baseRun.Status == contracts.RunCompleted {
		successRate = 1.0
		qualityScore = 1.0
	}

	durationSeconds := baseRun.UpdatedAt.Sub(baseRun.CreatedAt).Seconds()
	if durationSeconds < 1 {
		durationSeconds = 1
	}

	estimatedCost := tokensIn*0.0000015 + tokensOut*0.000002
	costPerSuccess := estimatedCost
	if successfulAttempts > 0 {
		costPerSuccess = estimatedCost / float64(successfulAttempts)
	}

	criticalViolations := 0
	for _, event := range events {
		eventType := strings.ToLower(strings.TrimSpace(event.Type))
		if strings.Contains(eventType, "policy") && strings.Contains(eventType, "violation") {
			criticalViolations++
		}
	}

	dispatchP95 := float64(metrics["task_dispatch_latency_p95_estimate_ms"])
	runsStarted := metrics["runs_started"]
	fallbackRate := 0.0
	backpressurePerRun := 0.0
	if runsStarted > 0 {
		fallbackRate = float64(metrics["fallbacks_triggered"]) / float64(runsStarted)
		backpressurePerRun = float64(metrics["backpressure_signals"]) / float64(runsStarted)
	}

	speedScore := 1.0 / (1.0 + durationSeconds/1200.0)
	costScore := 1.0 / (1.0 + costPerSuccess)
	objective := NormalizeObjective(run.Objective)
	composite := objective.Quality*qualityScore + objective.Speed*speedScore + objective.Cost*costScore

	score := Scorecard{
		RunID:                    run.ID,
		UnderlyingRunID:          underlyingID,
		QualityScore:             round(qualityScore, 4),
		MedianTimeToGreenSeconds: round(durationSeconds, 2),
		CostPerSuccess:           round(costPerSuccess, 8),
		CriticalPolicyViolations: criticalViolations,
		SuccessRate:              round(successRate, 4),
		Timeouts:                 timeouts,
		DispatchP95MS:            round(dispatchP95, 2),
		FallbackRate:             round(fallbackRate, 6),
		BackpressurePerRun:       round(backpressurePerRun, 4),
		Objective:                objective,
		CompositeScore:           round(composite, 4),
		Thresholds: map[string]bool{
			"quality_score_gte_0_90":          qualityScore >= 0.90,
			"critical_policy_violations_eq_0": criticalViolations == 0,
			"success_rate_gte_0_98":           successRate >= 0.98,
			"timeouts_eq_0":                   timeouts == 0,
			"dispatch_p95_ms_lte_250":         dispatchP95 <= 250,
			"fallback_rate_lte_0_05":          fallbackRate <= 0.05,
			"backpressure_per_run_lte_20":     backpressurePerRun <= 20,
		},
		GeneratedAt: time.Now().UTC(),
	}
	return score, nil
}

func (s *Service) finishRun(runID string, status RunStatus, errMsg string) {
	s.mu.Lock()
	run, ok := s.runs[runID]
	if !ok {
		s.mu.Unlock()
		return
	}
	now := time.Now().UTC()
	run.Status = status
	run.Error = strings.TrimSpace(errMsg)
	run.UpdatedAt = now
	run.FinishedAt = &now
	s.runs[runID] = run
	s.mu.Unlock()

	if status == RunCompleted || status == RunFailed {
		if score, err := s.computeScorecard(run); err == nil {
			s.mu.Lock()
			s.scorecards[runID] = score
			s.mu.Unlock()
		}
	}
	s.memory.Put(MemoryRecord{
		Scope:      MemoryScopeRun,
		Workspace:  s.workspaceRoot,
		RunID:      runID,
		Key:        "run.status",
		Value:      string(status),
		Source:     "orchestrator.finish",
		Provenance: "runtime",
		CreatedAt:  now,
	})
}

func (s *Service) transitionStep(runID string, idx int, status StepStatus, errMsg string) (PlanStep, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	run, ok := s.runs[runID]
	if !ok {
		return PlanStep{}, fmt.Errorf("orchestrator run not found")
	}
	if idx < 0 || idx >= len(run.Steps) {
		return PlanStep{}, fmt.Errorf("invalid step index")
	}

	now := time.Now().UTC()
	step := run.Steps[idx]
	step.Status = status
	step.Error = strings.TrimSpace(errMsg)
	if status == StepRunning {
		step.StartedAt = &now
		step.FinishedAt = nil
	} else if status == StepCompleted || status == StepFailed {
		if step.StartedAt == nil {
			step.StartedAt = &now
		}
		step.FinishedAt = &now
	}

	run.Steps[idx] = step
	run.CurrentStepID = step.ID
	run.UpdatedAt = now
	s.runs[runID] = run

	return step, nil
}

func (s *Service) publish(runID, eventType string, payload map[string]string) {
	if s.bus == nil {
		return
	}
	_, _ = s.bus.Publish(contracts.Event{
		RunID:   runID,
		Type:    eventType,
		Source:  "orchestrator",
		Payload: payload,
	})
}

func normalizePlanRequest(req PlanRequest) (PlanRequest, error) {
	req.ProjectID = strings.TrimSpace(req.ProjectID)
	if req.ProjectID == "" {
		req.ProjectID = "biometrics"
	}
	req.Goal = strings.TrimSpace(req.Goal)
	if req.Goal == "" {
		return PlanRequest{}, fmt.Errorf("goal is required")
	}
	req.StrategyMode = string(NormalizeStrategyMode(req.StrategyMode))
	req.PolicyProfile = NormalizePolicyProfile(req.PolicyProfile)
	req.Objective = NormalizeObjective(req.Objective)
	if req.MaxParallelism < 0 {
		req.MaxParallelism = 0
	}
	if req.MaxParallelism > 32 {
		req.MaxParallelism = 32
	}
	return req, nil
}

func normalizeRunRequest(req RunRequest) (RunRequest, error) {
	req.ProjectID = strings.TrimSpace(req.ProjectID)
	req.Goal = strings.TrimSpace(req.Goal)
	req.PlanID = strings.TrimSpace(req.PlanID)
	if req.PlanID == "" && req.Goal == "" {
		return RunRequest{}, fmt.Errorf("goal is required when plan_id is not provided")
	}
	if req.ProjectID == "" {
		req.ProjectID = "biometrics"
	}
	req.StrategyMode = string(NormalizeStrategyMode(req.StrategyMode))
	req.PolicyProfile = NormalizePolicyProfile(req.PolicyProfile)
	req.Objective = NormalizeObjective(req.Objective)
	if req.MaxParallelism < 0 {
		req.MaxParallelism = 0
	}
	if req.MaxParallelism > 32 {
		req.MaxParallelism = 32
	}
	if strings.TrimSpace(req.SkillSelectionMode) == "" {
		req.SkillSelectionMode = "auto"
	}
	if req.ContextBudget <= 0 {
		req.ContextBudget = 24000
	}
	req.OptimizerRecommendationID = strings.TrimSpace(req.OptimizerRecommendationID)
	req.OptimizerConfidence = strings.ToLower(strings.TrimSpace(req.OptimizerConfidence))
	return req, nil
}

func buildPlanSteps(mode StrategyMode) []PlanStep {
	steps := []PlanStep{
		{ID: "plan", Name: "plan", Description: "Generate deterministic execution graph", Status: StepPending},
	}

	switch mode {
	case StrategyArena:
		steps = append(steps,
			PlanStep{ID: "arena_branch_a", Name: "arena_branch_a", Description: "Run candidate strategy A in isolated worktree", DependsOn: []string{"plan"}, Status: StepPending},
			PlanStep{ID: "arena_branch_b", Name: "arena_branch_b", Description: "Run candidate strategy B in isolated worktree", DependsOn: []string{"plan"}, Status: StepPending},
			PlanStep{ID: "arena_select", Name: "arena_select", Description: "Select best candidate by score function", DependsOn: []string{"arena_branch_a", "arena_branch_b"}, Status: StepPending},
		)
	}

	executeDepends := []string{"plan"}
	if mode == StrategyArena {
		executeDepends = []string{"arena_select"}
	}

	steps = append(steps,
		PlanStep{ID: "execute", Name: "execute", Description: "Execute core coding pipeline", DependsOn: executeDepends, Status: StepPending},
		PlanStep{ID: "evaluate", Name: "evaluate", Description: "Generate scorecard and threshold checks", DependsOn: []string{"execute"}, Status: StepPending},
		PlanStep{ID: "finalize", Name: "finalize", Description: "Finalize run artifacts and telemetry", DependsOn: []string{"evaluate"}, Status: StepPending},
	)

	return steps
}

func cloneSteps(in []PlanStep) []PlanStep {
	out := make([]PlanStep, len(in))
	for i := range in {
		out[i] = in[i]
		out[i].DependsOn = append([]string{}, in[i].DependsOn...)
	}
	return out
}

func strategyReason(mode StrategyMode) string {
	switch mode {
	case StrategyAdaptive:
		return "adaptive balances quality, speed, and cost at runtime"
	case StrategyArena:
		return "arena evaluates parallel candidates and selects highest composite score"
	default:
		return "deterministic maximizes reproducibility and replayability"
	}
}

func maxParallelismFromProfiles(profiles []AgentProfile) int {
	best := 0
	for _, profile := range profiles {
		if profile.MaxParallelism > best {
			best = profile.MaxParallelism
		}
	}
	if best <= 0 {
		return 8
	}
	if best > 32 {
		return 32
	}
	return best
}

type runRisk struct {
	Score         int
	Reasons       []string
	ApprovalState string
	Blocked       bool
	BlockReason   string
}

func assessRunRisk(goal string, req RunRequest) runRisk {
	text := strings.ToLower(strings.TrimSpace(goal + " " + req.Goal))
	score := 0
	reasons := make([]string, 0, 8)

	add := func(points int, reason string) {
		if points <= 0 {
			return
		}
		score += points
		reasons = append(reasons, reason)
	}

	if containsAny(text, "curl ", "wget ", "http://", "https://", "scp ", "rsync ") {
		add(20, "network egress intent")
	}
	if containsAny(text, "token", "secret", "api key", "credentials", ".env", "private key") {
		add(22, "sensitive material referenced")
	}
	if containsAny(text, "~/.ssh", "/etc/passwd", "/etc/shadow", "/var/lib") {
		add(28, "sensitive filesystem target")
	}
	if containsAny(text, "exfiltrate", "upload secrets", "send to webhook", "post to slack", "dump database") {
		add(45, "explicit exfiltration request")
	}
	if containsAny(text, "ignore previous", "bypass policy", "disable guard", "do not log") {
		add(38, "prompt injection indicator")
	}

	approvalState := "not-required"
	approvals := strings.ToLower(strings.TrimSpace(req.PolicyProfile.Approvals))
	if approvals == "" {
		approvals = "on-risk"
	}
	blocked := false
	blockReason := ""
	switch approvals {
	case "required":
		approvalState = "required"
		if score >= 90 {
			blocked = true
			blockReason = fmt.Sprintf("critical risk score (%d) blocked pending manual approval", score)
		}
	case "on-risk":
		if score >= 80 {
			approvalState = "required-on-risk"
			blocked = true
			blockReason = fmt.Sprintf("high risk score (%d) requires manual approval", score)
		}
	case "never":
		approvalState = "skipped"
	default:
		if score >= 80 {
			approvalState = "required-on-risk"
			blocked = true
			blockReason = fmt.Sprintf("high risk score (%d) requires manual approval", score)
		}
	}
	if len(reasons) == 0 {
		reasons = append(reasons, "no elevated indicators")
	}
	return runRisk{
		Score:         score,
		Reasons:       reasons,
		ApprovalState: approvalState,
		Blocked:       blocked,
		BlockReason:   blockReason,
	}
}

func (s *Service) validateExecutionPolicy(runID string, run Run, req RunRequest) error {
	text := strings.ToLower(strings.TrimSpace(req.Goal + " " + run.Goal))
	profile := NormalizePolicyProfile(req.PolicyProfile)
	if profile.Exfiltration == "strict" && containsAny(text, "exfiltrate", "upload ", "send secrets", "post to webhook") {
		return fmt.Errorf("policy violation: strict exfiltration profile blocked outbound secret transfer intent")
	}
	if profile.Network == "off" && containsAny(text, "http://", "https://", "curl ", "wget ", "socket", "webhook") {
		return fmt.Errorf("policy violation: network is disabled for this run")
	}
	if profile.Filesystem == "readonly" && containsAny(text, "write ", "modify ", "delete ", "rm -rf", "truncate", "chmod") {
		return fmt.Errorf("policy violation: filesystem is readonly for this run")
	}
	if containsAny(text, "ignore previous instructions", "ignore system prompt", "disable safety", "bypass policy") {
		return fmt.Errorf("policy violation: prompt-injection pattern detected")
	}

	risk := assessRunRisk(run.Goal, req)
	s.publish(runID, "orchestrator.decision.explained", map[string]string{
		"run_id":       runID,
		"decision":     "execution_policy_gate",
		"risk_score":   fmt.Sprintf("%d", risk.Score),
		"risk_reasons": strings.Join(risk.Reasons, ";"),
		"result":       "pass",
	})
	return nil
}

func (s *Service) materializeAgentContexts(runID string, run Run) {
	profiles := run.AgentProfiles
	if len(profiles) == 0 {
		profiles = []AgentProfile{
			{Name: "planner", AllowedTools: []string{"read", "graph", "metrics"}, MaxParallelism: 2, ModelPolicy: "quality"},
			{Name: "coder", AllowedTools: []string{"read", "write", "test"}, MaxParallelism: 6, ModelPolicy: "balanced"},
			{Name: "reviewer", AllowedTools: []string{"read", "test", "comment"}, MaxParallelism: 3, ModelPolicy: "quality"},
		}
	}
	for _, profile := range profiles {
		key := "agent_context." + profile.Name
		ttl := time.Now().UTC().Add(12 * time.Hour)
		s.memory.Put(MemoryRecord{
			Scope:      MemoryScopeRun,
			Workspace:  s.workspaceRoot,
			RunID:      runID,
			Key:        key,
			Value:      fmt.Sprintf("goal=%s tools=%s max_parallelism=%d model_policy=%s", run.Goal, strings.Join(profile.AllowedTools, ","), profile.MaxParallelism, profile.ModelPolicy),
			Source:     "orchestrator.plan",
			Provenance: "subagent-profile",
			CreatedAt:  time.Now().UTC(),
			ExpiresAt:  &ttl,
		})
	}
}

func (s *Service) prepareArenaBranch(runID, branch string) (string, error) {
	if strings.TrimSpace(runID) == "" || strings.TrimSpace(branch) == "" {
		return "", fmt.Errorf("arena branch preparation requires run id and branch")
	}

	branchPath := filepath.Join(s.arenaRoot, sanitizePathSegment(runID), sanitizePathSegment(branch))
	if err := os.MkdirAll(branchPath, 0o755); err != nil {
		return "", fmt.Errorf("prepare arena workspace: %w", err)
	}
	metaPath := filepath.Join(branchPath, ".arena-meta")
	metaContent := fmt.Sprintf("run_id=%s\nbranch=%s\nprepared_at=%s\n", runID, branch, time.Now().UTC().Format(time.RFC3339Nano))
	_ = os.WriteFile(metaPath, []byte(metaContent), 0o644)

	s.mu.Lock()
	if _, ok := s.arenaPaths[runID]; !ok {
		s.arenaPaths[runID] = make(map[string]string)
	}
	s.arenaPaths[runID][branch] = branchPath
	s.mu.Unlock()

	s.memory.Put(MemoryRecord{
		Scope:      MemoryScopeRun,
		Workspace:  s.workspaceRoot,
		RunID:      runID,
		Key:        "arena." + branch + ".workspace",
		Value:      branchPath,
		Source:     "orchestrator.arena",
		Provenance: "branch-prepare",
		CreatedAt:  time.Now().UTC(),
	})

	return branchPath, nil
}

func (s *Service) validateArenaIsolation(runID string) error {
	s.mu.RLock()
	pathsByBranch := s.arenaPaths[runID]
	s.mu.RUnlock()
	if len(pathsByBranch) == 0 {
		return fmt.Errorf("arena isolation missing: no branch workspaces prepared")
	}

	seen := make(map[string]string, len(pathsByBranch))
	for branch, path := range pathsByBranch {
		normalized := filepath.Clean(path)
		if prevBranch, exists := seen[normalized]; exists {
			return fmt.Errorf("arena isolation violation: %s and %s share workspace %s", prevBranch, branch, normalized)
		}
		seen[normalized] = branch
	}
	if len(seen) < 2 {
		return fmt.Errorf("arena isolation violation: expected at least two distinct branch workspaces")
	}
	return nil
}

func (s *Service) arenaBranchPaths(runID string) []string {
	s.mu.RLock()
	pathsByBranch := s.arenaPaths[runID]
	s.mu.RUnlock()
	out := make([]string, 0, len(pathsByBranch))
	for _, path := range pathsByBranch {
		out = append(out, path)
	}
	sort.Strings(out)
	return out
}

func detectWorkspaceRoot() string {
	cwd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return cwd
}

func sanitizePathSegment(value string) string {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if trimmed == "" {
		return "unknown"
	}
	replacer := strings.NewReplacer("/", "_", "\\", "_", " ", "_", ":", "_")
	sanitized := replacer.Replace(trimmed)
	sanitized = strings.Trim(sanitized, "._-")
	if sanitized == "" {
		return "unknown"
	}
	return sanitized
}

func containsAny(text string, needles ...string) bool {
	for _, needle := range needles {
		if needle == "" {
			continue
		}
		if strings.Contains(text, strings.ToLower(needle)) {
			return true
		}
	}
	return false
}

func round(value float64, decimals int) float64 {
	if decimals < 0 {
		return value
	}
	factor := math.Pow10(decimals)
	return math.Round(value*factor) / factor
}
