package optimizer

import (
	"context"
	"database/sql"
	"fmt"
	"hash/fnv"
	"math"
	"sort"
	"strings"
	"time"

	"biometrics-cli/internal/contracts"
	"biometrics-cli/internal/evals"
	runtimeorchestrator "biometrics-cli/internal/runtime/orchestrator"
	store "biometrics-cli/internal/store/sqlite"
)

type eventPublisher interface {
	Publish(ev contracts.Event) (contracts.Event, error)
}

type orchestratorBackend interface {
	StartRun(ctx context.Context, req runtimeorchestrator.RunRequest) (runtimeorchestrator.Run, error)
	GetRun(runID string) (runtimeorchestrator.Run, error)
	ListScorecards() []runtimeorchestrator.Scorecard
}

type evalBackend interface {
	StartRun(ctx context.Context, req evals.RunRequest) (evals.Run, error)
	GetRun(runID string) (evals.Run, error)
}

type Service struct {
	store        *store.Store
	orchestrator orchestratorBackend
	evals        evalBackend
	bus          eventPublisher
}

const (
	defaultProjectID         = "biometrics"
	highConfidenceMinGap     = 0.025
	defaultWorkflowObjective = 0.65
)

type Objective struct {
	Quality float64 `json:"quality"`
	Speed   float64 `json:"speed"`
	Cost    float64 `json:"cost"`
}

type PredictedGates struct {
	QualityPass                 bool    `json:"quality_pass"`
	TimePass                    bool    `json:"time_pass"`
	CostPass                    bool    `json:"cost_pass"`
	RegressionPass              bool    `json:"regression_pass"`
	AllPass                     bool    `json:"all_pass"`
	GatePassCount               int     `json:"gate_pass_count"`
	PredictedQualityScore       float64 `json:"predicted_quality_score"`
	PredictedTimeImprovementPct float64 `json:"predicted_time_improvement_percent"`
	PredictedCostImprovementPct float64 `json:"predicted_cost_improvement_percent"`
	PredictedCostPerSuccess     float64 `json:"predicted_cost_per_success"`
	PredictedRegressionDetected bool    `json:"predicted_regression_detected"`
	PredictedCompositeScore     float64 `json:"predicted_composite_score"`
}

type Validation struct {
	ID               string    `json:"id"`
	RecommendationID string    `json:"recommendation_id"`
	EvalRunID        string    `json:"eval_run_id,omitempty"`
	Status           string    `json:"status"`
	QualityPass      bool      `json:"quality_pass"`
	TimePass         bool      `json:"time_pass"`
	CostPass         bool      `json:"cost_pass"`
	RegressionPass   bool      `json:"regression_pass"`
	AllPass          bool      `json:"all_pass"`
	Summary          string    `json:"summary,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type Recommendation struct {
	ID                   string         `json:"id"`
	ProjectID            string         `json:"project_id"`
	Goal                 string         `json:"goal"`
	StrategyMode         string         `json:"strategy_mode"`
	SchedulerMode        string         `json:"scheduler_mode"`
	MaxParallelism       int            `json:"max_parallelism"`
	ModelPreference      string         `json:"model_preference"`
	FallbackChain        []string       `json:"fallback_chain,omitempty"`
	ModelID              string         `json:"model_id,omitempty"`
	ContextBudget        int            `json:"context_budget"`
	Objective            Objective      `json:"objective"`
	Confidence           string         `json:"confidence"`
	PredictedGates       PredictedGates `json:"predicted_gates"`
	Rationale            string         `json:"rationale"`
	SourceScorecardRunID string         `json:"source_scorecard_run_id,omitempty"`
	Status               string         `json:"status"`
	AppliedRunID         string         `json:"applied_run_id,omitempty"`
	RejectedReason       string         `json:"rejected_reason,omitempty"`
	Validation           *Validation    `json:"validation,omitempty"`
	CreatedAt            time.Time      `json:"created_at"`
	UpdatedAt            time.Time      `json:"updated_at"`
}

type GenerateRequest struct {
	ProjectID string    `json:"project_id,omitempty"`
	Goal      string    `json:"goal"`
	Objective Objective `json:"objective,omitempty"`
}

type ListOptions struct {
	ProjectID string
	Status    string
	Limit     int
}

type ApplyResult struct {
	RecommendationID string                  `json:"recommendation_id"`
	Recommendation   Recommendation          `json:"recommendation,omitempty"`
	Run              runtimeorchestrator.Run `json:"run"`
}

type candidateConfig struct {
	StrategyMode    string
	SchedulerMode   string
	MaxParallelism  int
	ModelPreference string
	FallbackChain   []string
	ModelID         string
	ContextBudget   int
	Objective       Objective
	Rationale       string
}

type candidateScore struct {
	config    candidateConfig
	predicted PredictedGates
}

func NewService(store *store.Store, orchestrator orchestratorBackend, evals evalBackend, bus eventPublisher) *Service {
	return &Service{store: store, orchestrator: orchestrator, evals: evals, bus: bus}
}

func (s *Service) GenerateRecommendation(_ context.Context, req GenerateRequest) (Recommendation, error) {
	if s.store == nil {
		return Recommendation{}, fmt.Errorf("optimizer store is not configured")
	}
	if s.orchestrator == nil {
		return Recommendation{}, fmt.Errorf("optimizer orchestrator backend is not configured")
	}

	projectID := strings.TrimSpace(req.ProjectID)
	if projectID == "" {
		projectID = defaultProjectID
	}
	goal := strings.TrimSpace(req.Goal)
	if goal == "" {
		return Recommendation{}, fmt.Errorf("goal is required")
	}

	baseline := latestScorecardBaseline(s.orchestrator.ListScorecards())
	objective := normalizeObjective(req.Objective)
	candidates := buildCandidates(s.store, projectID, objective)
	if len(candidates) == 0 {
		return Recommendation{}, fmt.Errorf("optimizer has no candidates")
	}

	scoredCandidates := make([]candidateScore, 0, len(candidates))
	for _, candidate := range candidates {
		scoredCandidates = append(scoredCandidates, candidateScore{
			config:    candidate,
			predicted: predictGates(baseline, candidate),
		})
	}
	if len(scoredCandidates) == 0 {
		return Recommendation{}, fmt.Errorf("optimizer failed to score candidates")
	}

	sortScoredCandidates(scoredCandidates)

	best := scoredCandidates[0]
	var second *PredictedGates
	if len(scoredCandidates) > 1 {
		second = &scoredCandidates[1].predicted
	}
	confidence := confidenceFromPrediction(best.predicted, second)
	rationale := strings.TrimSpace(best.config.Rationale)
	if confidence == "low" {
		rationale = strings.TrimSpace(rationale + "; warning: low-confidence recommendation (projected gate coverage below 2/4).")
	}

	recRecord, err := s.store.CreateOptimizerRecommendation(store.OptimizerRecommendationRecord{
		ProjectID:       projectID,
		Goal:            goal,
		StrategyMode:    best.config.StrategyMode,
		SchedulerMode:   best.config.SchedulerMode,
		MaxParallelism:  best.config.MaxParallelism,
		ModelPreference: best.config.ModelPreference,
		FallbackChain:   append([]string{}, best.config.FallbackChain...),
		ModelID:         best.config.ModelID,
		ContextBudget:   best.config.ContextBudget,
		Objective: store.OptimizerObjectiveRecord{
			Quality: best.config.Objective.Quality,
			Speed:   best.config.Objective.Speed,
			Cost:    best.config.Objective.Cost,
		},
		Confidence: confidence,
		PredictedGates: store.OptimizerPredictedGatesRecord{
			QualityPass:                 best.predicted.QualityPass,
			TimePass:                    best.predicted.TimePass,
			CostPass:                    best.predicted.CostPass,
			RegressionPass:              best.predicted.RegressionPass,
			AllPass:                     best.predicted.AllPass,
			GatePassCount:               best.predicted.GatePassCount,
			PredictedQualityScore:       best.predicted.PredictedQualityScore,
			PredictedTimeImprovementPct: best.predicted.PredictedTimeImprovementPct,
			PredictedCostImprovementPct: best.predicted.PredictedCostImprovementPct,
			PredictedCostPerSuccess:     best.predicted.PredictedCostPerSuccess,
			PredictedRegressionDetected: best.predicted.PredictedRegressionDetected,
			PredictedCompositeScore:     best.predicted.PredictedCompositeScore,
		},
		Rationale:            rationale,
		SourceScorecardRunID: baseline.RunID,
		Status:               "generated",
	})
	if err != nil {
		return Recommendation{}, err
	}

	rec := recommendationFromRecord(recRecord, nil)
	s.publish("", "optimizer.recommendation.generated", map[string]string{
		"recommendation_id": rec.ID,
		"project_id":        rec.ProjectID,
		"strategy_mode":     rec.StrategyMode,
		"scheduler_mode":    rec.SchedulerMode,
		"gate_pass_count":   fmt.Sprintf("%d", rec.PredictedGates.GatePassCount),
		"confidence":        rec.Confidence,
		"all_pass":          fmt.Sprintf("%t", rec.PredictedGates.AllPass),
	})
	return rec, nil
}

func (s *Service) ListRecommendations(opts ListOptions) ([]Recommendation, error) {
	if s.store == nil {
		return nil, fmt.Errorf("optimizer store is not configured")
	}
	records, err := s.store.ListOptimizerRecommendations(store.ListOptimizerRecommendationsOptions{
		ProjectID: opts.ProjectID,
		Status:    strings.ToLower(strings.TrimSpace(opts.Status)),
		Limit:     opts.Limit,
	})
	if err != nil {
		return nil, err
	}

	out := make([]Recommendation, 0, len(records))
	for _, record := range records {
		validation, validationErr := s.store.GetOptimizerValidationByRecommendation(record.ID)
		if validationErr != nil && !errorsIsNoRows(validationErr) {
			return nil, validationErr
		}
		out = append(out, recommendationFromRecord(record, maybeValidation(validation, validationErr)))
	}
	return out, nil
}

func (s *Service) GetRecommendation(id string) (Recommendation, error) {
	if s.store == nil {
		return Recommendation{}, fmt.Errorf("optimizer store is not configured")
	}
	record, err := s.store.GetOptimizerRecommendation(id)
	if err != nil {
		return Recommendation{}, err
	}
	validation, validationErr := s.store.GetOptimizerValidationByRecommendation(record.ID)
	if validationErr != nil && !errorsIsNoRows(validationErr) {
		return Recommendation{}, validationErr
	}
	return recommendationFromRecord(record, maybeValidation(validation, validationErr)), nil
}

func (s *Service) ApplyRecommendation(ctx context.Context, recommendationID string) (ApplyResult, error) {
	if s.store == nil || s.orchestrator == nil {
		return ApplyResult{}, fmt.Errorf("optimizer is not configured")
	}
	recommendation, err := s.store.GetOptimizerRecommendation(recommendationID)
	if err != nil {
		return ApplyResult{}, err
	}

	switch recommendation.Status {
	case "applied":
		if recommendation.AppliedRunID == "" {
			return ApplyResult{}, fmt.Errorf("recommendation %q is applied but has no run binding", recommendation.ID)
		}
		run, runErr := s.orchestrator.GetRun(recommendation.AppliedRunID)
		if runErr != nil {
			return ApplyResult{}, runErr
		}
		rec := recommendationFromRecord(recommendation, nil)
		return ApplyResult{
			RecommendationID: rec.ID,
			Recommendation:   rec,
			Run:              run,
		}, nil
	case "rejected":
		return ApplyResult{}, fmt.Errorf("recommendation status %q is not applyable", recommendation.Status)
	case "generated":
		// apply flow below
	default:
		return ApplyResult{}, fmt.Errorf("recommendation status %q is not applyable", recommendation.Status)
	}

	run, err := s.orchestrator.StartRun(ctx, runtimeorchestrator.RunRequest{
		ProjectID:                 recommendation.ProjectID,
		Goal:                      recommendation.Goal,
		StrategyMode:              recommendation.StrategyMode,
		SchedulerMode:             recommendation.SchedulerMode,
		MaxParallelism:            recommendation.MaxParallelism,
		ModelPreference:           recommendation.ModelPreference,
		FallbackChain:             append([]string{}, recommendation.FallbackChain...),
		ModelID:                   recommendation.ModelID,
		ContextBudget:             recommendation.ContextBudget,
		OptimizerRecommendationID: recommendation.ID,
		OptimizerConfidence:       recommendation.Confidence,
		PolicyProfile: runtimeorchestrator.PolicyProfile{
			Exfiltration: "balanced",
			Secrets:      "balanced",
			Filesystem:   "workspace",
			Network:      "restricted",
			Approvals:    "on-risk",
		},
		Objective: runtimeorchestrator.Objective{
			Quality: recommendation.Objective.Quality,
			Speed:   recommendation.Objective.Speed,
			Cost:    recommendation.Objective.Cost,
		},
	})
	if err != nil {
		return ApplyResult{}, err
	}

	recommendation, err = s.store.UpdateOptimizerRecommendationStatus(recommendation.ID, "applied", run.ID, "")
	if err != nil {
		return ApplyResult{}, err
	}

	rec := recommendationFromRecord(recommendation, nil)
	s.publish(run.ID, "optimizer.recommendation.applied", map[string]string{
		"recommendation_id":   rec.ID,
		"orchestrator_run_id": run.ID,
		"strategy_mode":       rec.StrategyMode,
		"scheduler_mode":      rec.SchedulerMode,
		"confidence":          rec.Confidence,
	})

	go s.runValidation(rec)
	return ApplyResult{
		RecommendationID: rec.ID,
		Recommendation:   rec,
		Run:              run,
	}, nil
}

func (s *Service) RejectRecommendation(recommendationID, reason string) (Recommendation, error) {
	if s.store == nil {
		return Recommendation{}, fmt.Errorf("optimizer store is not configured")
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "rejected by operator"
	}
	current, err := s.store.GetOptimizerRecommendation(recommendationID)
	if err != nil {
		return Recommendation{}, err
	}
	if current.Status == "rejected" {
		return recommendationFromRecord(current, nil), nil
	}
	if current.Status == "applied" {
		return Recommendation{}, fmt.Errorf("recommendation status %q cannot be rejected", current.Status)
	}
	if current.Status != "generated" {
		return Recommendation{}, fmt.Errorf("recommendation status %q cannot be rejected", current.Status)
	}
	updated, err := s.store.UpdateOptimizerRecommendationStatus(recommendationID, "rejected", "", reason)
	if err != nil {
		return Recommendation{}, err
	}
	rec := recommendationFromRecord(updated, nil)
	s.publish("", "optimizer.recommendation.rejected", map[string]string{
		"recommendation_id": rec.ID,
		"reason":            reason,
	})
	return rec, nil
}

func (s *Service) runValidation(rec Recommendation) {
	if s.store == nil || s.evals == nil {
		return
	}
	validation := store.OptimizerValidationRecord{
		ID:               "opt-val-" + rec.ID,
		RecommendationID: rec.ID,
		Status:           "running",
		Summary:          "validation eval started",
	}
	stored, err := s.store.UpsertOptimizerValidation(validation)
	if err != nil {
		return
	}

	evalRun, err := s.evals.StartRun(context.Background(), evals.RunRequest{
		Name:                "optimizer-validation",
		CandidateStrategy:   rec.StrategyMode,
		BaselineStrategy:    "deterministic",
		SampleSize:          500,
		DatasetID:           "apex-suite-v1",
		Seed:                int64(seedFromString(rec.ID)),
		TasksLimit:          500,
		CompetitorBaselines: []string{"codex", "cursor"},
	})
	if err != nil {
		stored.Status = "failed"
		stored.Summary = "failed to start validation eval: " + err.Error()
		_, _ = s.store.UpsertOptimizerValidation(stored)
		s.publish("", "optimizer.validation.completed", map[string]string{
			"recommendation_id": rec.ID,
			"status":            stored.Status,
			"all_pass":          "false",
		})
		return
	}

	stored.EvalRunID = evalRun.ID
	stored.Status = "running"
	stored.Summary = "validation eval running"
	stored, _ = s.store.UpsertOptimizerValidation(stored)

	deadline := time.Now().Add(4 * time.Minute)
	for time.Now().Before(deadline) {
		current, err := s.evals.GetRun(evalRun.ID)
		if err != nil {
			time.Sleep(500 * time.Millisecond)
			continue
		}

		switch current.Status {
		case evals.RunCompleted:
			baselineCmp := current.Comparison[current.BaselineStrategy]
			stored.Status = "completed"
			stored.QualityPass = current.GatePackV1.Quality
			stored.TimePass = current.GatePackV1.Time
			stored.CostPass = current.GatePackV1.Cost
			stored.RegressionPass = current.GatePackV1.Regression
			stored.AllPass = current.GatePackV1.AllPass
			if !stored.QualityPass && !stored.TimePass && !stored.CostPass && !stored.RegressionPass && !stored.AllPass {
				stored.QualityPass = current.Metrics.QualityScore >= 0.90
				stored.TimePass = baselineCmp.TimeToGreenImprovementPercent >= 25.0
				stored.CostPass = baselineCmp.CostImprovementPercent >= 20.0
				stored.RegressionPass = !current.RegressionDetected
				stored.AllPass = stored.QualityPass && stored.TimePass && stored.CostPass && stored.RegressionPass
			}
			stored.Summary = fmt.Sprintf(
				"q=%.3f time=%.2f%% cost=%.2f%% regression=%t",
				current.Metrics.QualityScore,
				baselineCmp.TimeToGreenImprovementPercent,
				baselineCmp.CostImprovementPercent,
				current.RegressionDetected,
			)
			stored, _ = s.store.UpsertOptimizerValidation(stored)
			s.publish("", "optimizer.validation.completed", map[string]string{
				"recommendation_id": rec.ID,
				"status":            stored.Status,
				"eval_run_id":       stored.EvalRunID,
				"all_pass":          fmt.Sprintf("%t", stored.AllPass),
			})
			return
		case evals.RunFailed:
			stored.Status = "failed"
			stored.Summary = current.Error
			stored.AllPass = false
			stored, _ = s.store.UpsertOptimizerValidation(stored)
			s.publish("", "optimizer.validation.completed", map[string]string{
				"recommendation_id": rec.ID,
				"status":            stored.Status,
				"eval_run_id":       stored.EvalRunID,
				"all_pass":          "false",
			})
			return
		}
		time.Sleep(500 * time.Millisecond)
	}

	stored.Status = "failed"
	stored.Summary = "validation timeout"
	stored.AllPass = false
	stored, _ = s.store.UpsertOptimizerValidation(stored)
	s.publish("", "optimizer.validation.completed", map[string]string{
		"recommendation_id": rec.ID,
		"status":            stored.Status,
		"eval_run_id":       stored.EvalRunID,
		"all_pass":          "false",
	})
}

func recommendationFromRecord(record store.OptimizerRecommendationRecord, validation *Validation) Recommendation {
	return Recommendation{
		ID:              record.ID,
		ProjectID:       record.ProjectID,
		Goal:            record.Goal,
		StrategyMode:    record.StrategyMode,
		SchedulerMode:   record.SchedulerMode,
		MaxParallelism:  record.MaxParallelism,
		ModelPreference: record.ModelPreference,
		FallbackChain:   append([]string{}, record.FallbackChain...),
		ModelID:         record.ModelID,
		ContextBudget:   record.ContextBudget,
		Objective:       Objective{Quality: record.Objective.Quality, Speed: record.Objective.Speed, Cost: record.Objective.Cost},
		Confidence:      record.Confidence,
		PredictedGates: PredictedGates{
			QualityPass:                 record.PredictedGates.QualityPass,
			TimePass:                    record.PredictedGates.TimePass,
			CostPass:                    record.PredictedGates.CostPass,
			RegressionPass:              record.PredictedGates.RegressionPass,
			AllPass:                     record.PredictedGates.AllPass,
			GatePassCount:               record.PredictedGates.GatePassCount,
			PredictedQualityScore:       record.PredictedGates.PredictedQualityScore,
			PredictedTimeImprovementPct: record.PredictedGates.PredictedTimeImprovementPct,
			PredictedCostImprovementPct: record.PredictedGates.PredictedCostImprovementPct,
			PredictedCostPerSuccess:     record.PredictedGates.PredictedCostPerSuccess,
			PredictedRegressionDetected: record.PredictedGates.PredictedRegressionDetected,
			PredictedCompositeScore:     record.PredictedGates.PredictedCompositeScore,
		},
		Rationale:            record.Rationale,
		SourceScorecardRunID: record.SourceScorecardRunID,
		Status:               record.Status,
		AppliedRunID:         record.AppliedRunID,
		RejectedReason:       record.RejectedReason,
		Validation:           validation,
		CreatedAt:            record.CreatedAt,
		UpdatedAt:            record.UpdatedAt,
	}
}

func maybeValidation(record store.OptimizerValidationRecord, err error) *Validation {
	if err != nil {
		return nil
	}
	return &Validation{
		ID:               record.ID,
		RecommendationID: record.RecommendationID,
		EvalRunID:        record.EvalRunID,
		Status:           record.Status,
		QualityPass:      record.QualityPass,
		TimePass:         record.TimePass,
		CostPass:         record.CostPass,
		RegressionPass:   record.RegressionPass,
		AllPass:          record.AllPass,
		Summary:          record.Summary,
		CreatedAt:        record.CreatedAt,
		UpdatedAt:        record.UpdatedAt,
	}
}

func predictGates(baseline runtimeorchestrator.Scorecard, candidate candidateConfig) PredictedGates {
	baseQuality := baseline.QualityScore
	if baseQuality <= 0 {
		baseQuality = 0.82
	}
	baseTime := baseline.MedianTimeToGreenSeconds
	if baseTime <= 0 {
		baseTime = 640
	}
	baseCost := baseline.CostPerSuccess
	if baseCost <= 0 {
		baseCost = 0.005
	}

	strategy := strings.ToLower(candidate.StrategyMode)
	qualityDelta := map[string]float64{"deterministic": 0.02, "adaptive": 0.06, "arena": 0.09}[strategy]
	timeFactor := map[string]float64{"deterministic": 0.94, "adaptive": 0.82, "arena": 0.76}[strategy]
	costFactor := map[string]float64{"deterministic": 0.88, "adaptive": 0.79, "arena": 0.72}[strategy]
	if qualityDelta == 0 {
		qualityDelta = 0.04
	}
	if timeFactor == 0 {
		timeFactor = 0.86
	}
	if costFactor == 0 {
		costFactor = 0.82
	}

	schedulerFactor := 1.0
	if strings.EqualFold(candidate.SchedulerMode, "serial") {
		schedulerFactor = 1.23
		costFactor *= 0.92
	}

	parallelBoost := math.Min(float64(candidate.MaxParallelism), 12) * 0.008
	parallelCostPenalty := math.Min(float64(candidate.MaxParallelism), 12) * 0.01
	contextQualityBonus := math.Min(float64(candidate.ContextBudget)/100000.0, 0.08)
	modelQualityBonus := map[string]float64{"codex": 0.03, "gemini": 0.015, "nim": 0.01}[strings.ToLower(candidate.ModelPreference)]

	predictedQuality := clamp(baseQuality+qualityDelta+(candidate.Objective.Quality*0.05)+modelQualityBonus+contextQualityBonus-(parallelBoost*0.25), 0.45, 0.99)
	predictedTime := math.Max(120, baseTime*timeFactor*schedulerFactor*(1-(parallelBoost*0.8)))
	predictedCost := math.Max(0.0001, baseCost*costFactor*(1+parallelCostPenalty-(candidate.Objective.Cost*0.12)))

	timeImprovement := ((baseTime - predictedTime) / baseTime) * 100
	costImprovement := ((baseCost - predictedCost) / baseCost) * 100
	regressionDetected := predictedQuality < (baseQuality - 0.03)

	qualityPass := predictedQuality >= 0.90
	timePass := timeImprovement >= 25
	costPass := costImprovement >= 20
	regressionPass := !regressionDetected
	allPass := qualityPass && timePass && costPass && regressionPass
	gatePassCount := 0
	for _, pass := range []bool{qualityPass, timePass, costPass, regressionPass} {
		if pass {
			gatePassCount++
		}
	}

	speedScore := 1.0 / (1.0 + predictedTime/1200.0)
	costScore := 1.0 / (1.0 + predictedCost)
	composite := clamp((0.6*predictedQuality)+(0.25*speedScore)+(0.15*costScore), 0, 1)

	return PredictedGates{
		QualityPass:                 qualityPass,
		TimePass:                    timePass,
		CostPass:                    costPass,
		RegressionPass:              regressionPass,
		AllPass:                     allPass,
		GatePassCount:               gatePassCount,
		PredictedQualityScore:       round(predictedQuality, 4),
		PredictedTimeImprovementPct: round(timeImprovement, 4),
		PredictedCostImprovementPct: round(costImprovement, 4),
		PredictedCostPerSuccess:     round(predictedCost, 8),
		PredictedRegressionDetected: regressionDetected,
		PredictedCompositeScore:     round(composite, 4),
	}
}

func buildCandidates(db *store.Store, projectID string, objective Objective) []candidateConfig {
	baseObjective := normalizeObjective(objective)
	seeds := []candidateConfig{
		{
			StrategyMode:    "arena",
			SchedulerMode:   "dag_parallel_v1",
			MaxParallelism:  8,
			ModelPreference: "codex",
			FallbackChain:   []string{"gemini", "nim"},
			ContextBudget:   36000,
			Objective:       normalizeObjective(Objective{Quality: defaultWorkflowObjective, Speed: 0.2, Cost: 0.15}),
			Rationale:       "seed workflow-apex-hardening",
		},
	}

	if seed, ok := seedFromCurrentRunState(db, projectID, baseObjective); ok {
		seeds = append(seeds, seed)
	}
	if seed, ok := seedFromBestAppliedProfile(db, projectID, baseObjective); ok {
		seeds = append(seeds, seed)
	}

	out := make([]candidateConfig, 0, 128)
	seen := make(map[string]struct{}, 128)
	for _, seed := range seeds {
		for _, candidate := range mutateSeed(seed, baseObjective) {
			addCandidate(&out, seen, candidate)
		}
	}
	return out
}

func mutateSeed(seed candidateConfig, objective Objective) []candidateConfig {
	base := normalizeCandidate(seed)
	objectives := []Objective{
		base.Objective,
		objective,
		normalizeObjective(Objective{Quality: 0.7, Speed: 0.2, Cost: 0.1}),
		normalizeObjective(Objective{Quality: 0.6, Speed: 0.25, Cost: 0.15}),
		normalizeObjective(Objective{Quality: 0.5, Speed: 0.3, Cost: 0.2}),
	}
	strategies := []string{base.StrategyMode, "deterministic", "adaptive", "arena"}
	schedulers := []string{base.SchedulerMode, "dag_parallel_v1", "serial"}
	maxParallelism := []int{base.MaxParallelism, base.MaxParallelism - 2, base.MaxParallelism + 2, 1, 6, 8, 10, 12}
	modelPreferences := []string{base.ModelPreference, "codex", "gemini", "nim"}
	contextBudgets := []int{base.ContextBudget, base.ContextBudget - 4000, base.ContextBudget + 4000, 24000, 32000, 36000, 42000}

	out := make([]candidateConfig, 0, 96)
	seen := make(map[string]struct{}, 96)
	addCandidate(&out, seen, base)

	for _, strategy := range strategies {
		candidate := base
		candidate.StrategyMode = strategy
		candidate.Rationale = base.Rationale + " + mutate strategy_mode"
		addCandidate(&out, seen, candidate)
	}
	for _, schedulerMode := range schedulers {
		candidate := base
		candidate.SchedulerMode = schedulerMode
		candidate.Rationale = base.Rationale + " + mutate scheduler_mode"
		addCandidate(&out, seen, candidate)
	}
	for _, parallelism := range maxParallelism {
		candidate := base
		candidate.MaxParallelism = parallelism
		candidate.Rationale = base.Rationale + " + mutate max_parallelism"
		addCandidate(&out, seen, candidate)
	}
	for _, candidateObjective := range objectives {
		candidate := base
		candidate.Objective = candidateObjective
		candidate.Rationale = base.Rationale + " + mutate objective"
		addCandidate(&out, seen, candidate)
	}
	for _, provider := range modelPreferences {
		candidate := base
		candidate.ModelPreference = provider
		candidate.FallbackChain = defaultFallbackChain(provider)
		candidate.Rationale = base.Rationale + " + mutate model_preference/fallback_chain"
		addCandidate(&out, seen, candidate)
	}
	for _, budget := range contextBudgets {
		candidate := base
		candidate.ContextBudget = budget
		candidate.Rationale = base.Rationale + " + mutate context_budget"
		addCandidate(&out, seen, candidate)
	}

	return out
}

func normalizeCandidate(candidate candidateConfig) candidateConfig {
	candidate.StrategyMode = strings.ToLower(strings.TrimSpace(candidate.StrategyMode))
	if candidate.StrategyMode == "" {
		candidate.StrategyMode = "adaptive"
	}
	switch candidate.StrategyMode {
	case "deterministic", "adaptive", "arena":
	default:
		candidate.StrategyMode = "adaptive"
	}

	candidate.SchedulerMode = strings.ToLower(strings.TrimSpace(candidate.SchedulerMode))
	if candidate.SchedulerMode == "" {
		candidate.SchedulerMode = "dag_parallel_v1"
	}
	if candidate.SchedulerMode != "dag_parallel_v1" && candidate.SchedulerMode != "serial" {
		candidate.SchedulerMode = "dag_parallel_v1"
	}

	if candidate.MaxParallelism <= 0 {
		candidate.MaxParallelism = 8
	}
	if candidate.MaxParallelism > 32 {
		candidate.MaxParallelism = 32
	}
	if candidate.SchedulerMode == "serial" {
		candidate.MaxParallelism = 1
	}

	candidate.ModelPreference = strings.ToLower(strings.TrimSpace(candidate.ModelPreference))
	if candidate.ModelPreference == "" {
		candidate.ModelPreference = "codex"
	}
	switch candidate.ModelPreference {
	case "codex", "gemini", "nim":
	default:
		candidate.ModelPreference = "codex"
	}
	candidate.FallbackChain = normalizeFallbackChain(candidate.FallbackChain, candidate.ModelPreference)

	if candidate.ContextBudget <= 0 {
		candidate.ContextBudget = 24000
	}
	if candidate.ContextBudget > 200000 {
		candidate.ContextBudget = 200000
	}
	if candidate.ContextBudget < 1000 {
		candidate.ContextBudget = 1000
	}

	candidate.Objective = normalizeObjective(candidate.Objective)
	candidate.Rationale = strings.TrimSpace(candidate.Rationale)
	if candidate.Rationale == "" {
		candidate.Rationale = "bounded mutation candidate"
	}
	return candidate
}

func candidateSignature(candidate candidateConfig) string {
	normalized := normalizeCandidate(candidate)
	return strings.Join([]string{
		normalized.StrategyMode,
		normalized.SchedulerMode,
		fmt.Sprintf("%d", normalized.MaxParallelism),
		normalized.ModelPreference,
		strings.Join(normalized.FallbackChain, ","),
		fmt.Sprintf("%d", normalized.ContextBudget),
		fmt.Sprintf("%.5f-%.5f-%.5f", normalized.Objective.Quality, normalized.Objective.Speed, normalized.Objective.Cost),
	}, "|")
}

func addCandidate(out *[]candidateConfig, seen map[string]struct{}, candidate candidateConfig) {
	normalized := normalizeCandidate(candidate)
	signature := candidateSignature(normalized)
	if _, exists := seen[signature]; exists {
		return
	}
	seen[signature] = struct{}{}
	*out = append(*out, normalized)
}

func defaultFallbackChain(modelPreference string) []string {
	switch strings.ToLower(strings.TrimSpace(modelPreference)) {
	case "codex":
		return []string{"gemini", "nim"}
	case "gemini":
		return []string{"nim"}
	case "nim":
		return []string{}
	default:
		return []string{"gemini", "nim"}
	}
}

func normalizeFallbackChain(chain []string, modelPreference string) []string {
	normalized := make([]string, 0, len(chain))
	seen := make(map[string]struct{}, len(chain))
	for _, provider := range chain {
		provider = strings.ToLower(strings.TrimSpace(provider))
		if provider == "" {
			continue
		}
		switch provider {
		case "codex", "gemini", "nim":
		default:
			continue
		}
		if _, exists := seen[provider]; exists {
			continue
		}
		seen[provider] = struct{}{}
		normalized = append(normalized, provider)
	}
	if len(normalized) == 0 {
		return defaultFallbackChain(modelPreference)
	}
	return normalized
}

func seedFromCurrentRunState(db *store.Store, projectID string, objective Objective) (candidateConfig, bool) {
	if db == nil {
		return candidateConfig{}, false
	}
	runs, err := db.ListRuns(200)
	if err != nil {
		return candidateConfig{}, false
	}
	for _, run := range runs {
		if projectID != "" && !strings.EqualFold(run.ProjectID, projectID) {
			continue
		}
		schedulerMode := string(run.SchedulerMode)
		if schedulerMode == "" {
			schedulerMode = "dag_parallel_v1"
		}
		return candidateConfig{
			StrategyMode:    "adaptive",
			SchedulerMode:   schedulerMode,
			MaxParallelism:  run.MaxParallelism,
			ModelPreference: run.ModelPreference,
			FallbackChain:   append([]string{}, run.FallbackChain...),
			ModelID:         run.ModelID,
			ContextBudget:   run.ContextBudget,
			Objective:       objective,
			Rationale:       "seed current-ui-state from latest run options",
		}, true
	}
	return candidateConfig{}, false
}

func seedFromBestAppliedProfile(db *store.Store, projectID string, objective Objective) (candidateConfig, bool) {
	if db == nil {
		return candidateConfig{}, false
	}
	records, err := db.ListOptimizerRecommendations(store.ListOptimizerRecommendationsOptions{
		ProjectID: projectID,
		Status:    "applied",
		Limit:     100,
	})
	if err != nil || len(records) == 0 {
		return candidateConfig{}, false
	}

	bestIndex := 0
	bestValidationAllPass := false
	for i := range records {
		rec := records[i]
		validation, validationErr := db.GetOptimizerValidationByRecommendation(rec.ID)
		validationAllPass := validationErr == nil && validation.AllPass

		bestRec := records[bestIndex]
		bestCost := bestRec.PredictedGates.PredictedCostPerSuccess
		if bestCost <= 0 {
			bestCost = math.MaxFloat64
		}
		currentCost := rec.PredictedGates.PredictedCostPerSuccess
		if currentCost <= 0 {
			currentCost = math.MaxFloat64
		}

		replace := false
		if validationAllPass != bestValidationAllPass {
			replace = validationAllPass
		} else if rec.PredictedGates.GatePassCount != bestRec.PredictedGates.GatePassCount {
			replace = rec.PredictedGates.GatePassCount > bestRec.PredictedGates.GatePassCount
		} else if rec.PredictedGates.PredictedCompositeScore != bestRec.PredictedGates.PredictedCompositeScore {
			replace = rec.PredictedGates.PredictedCompositeScore > bestRec.PredictedGates.PredictedCompositeScore
		} else if currentCost != bestCost {
			replace = currentCost < bestCost
		} else if rec.UpdatedAt.After(bestRec.UpdatedAt) {
			replace = true
		}
		if replace {
			bestIndex = i
			bestValidationAllPass = validationAllPass
		}
	}

	best := records[bestIndex]
	return candidateConfig{
		StrategyMode:    best.StrategyMode,
		SchedulerMode:   best.SchedulerMode,
		MaxParallelism:  best.MaxParallelism,
		ModelPreference: best.ModelPreference,
		FallbackChain:   append([]string{}, best.FallbackChain...),
		ModelID:         best.ModelID,
		ContextBudget:   best.ContextBudget,
		Objective:       objective,
		Rationale:       "seed best-last-applied-profile",
	}, true
}

func sortScoredCandidates(scoredCandidates []candidateScore) {
	sort.SliceStable(scoredCandidates, func(i, j int) bool {
		left := scoredCandidates[i]
		right := scoredCandidates[j]
		if left.predicted.GatePassCount != right.predicted.GatePassCount {
			return left.predicted.GatePassCount > right.predicted.GatePassCount
		}
		if left.predicted.PredictedCompositeScore != right.predicted.PredictedCompositeScore {
			return left.predicted.PredictedCompositeScore > right.predicted.PredictedCompositeScore
		}

		leftCost := left.predicted.PredictedCostPerSuccess
		if leftCost <= 0 {
			leftCost = math.MaxFloat64
		}
		rightCost := right.predicted.PredictedCostPerSuccess
		if rightCost <= 0 {
			rightCost = math.MaxFloat64
		}
		if leftCost != rightCost {
			return leftCost < rightCost
		}
		return candidateSignature(left.config) < candidateSignature(right.config)
	})
}

func latestScorecardBaseline(cards []runtimeorchestrator.Scorecard) runtimeorchestrator.Scorecard {
	if len(cards) == 0 {
		return runtimeorchestrator.Scorecard{}
	}
	sort.Slice(cards, func(i, j int) bool {
		return cards[i].GeneratedAt.After(cards[j].GeneratedAt)
	})
	return cards[0]
}

func normalizeObjective(raw Objective) Objective {
	q := math.Max(0, raw.Quality)
	speed := math.Max(0, raw.Speed)
	cost := math.Max(0, raw.Cost)
	sum := q + speed + cost
	if sum <= 0 {
		return Objective{Quality: 0.5, Speed: 0.3, Cost: 0.2}
	}
	return Objective{Quality: q / sum, Speed: speed / sum, Cost: cost / sum}
}

func confidenceFromPrediction(pred PredictedGates, next *PredictedGates) string {
	if pred.GatePassCount >= 3 {
		compositeGap := pred.PredictedCompositeScore
		if next != nil {
			compositeGap = pred.PredictedCompositeScore - next.PredictedCompositeScore
		}
		if compositeGap >= highConfidenceMinGap {
			return "high"
		}
		return "medium"
	}
	if pred.GatePassCount == 2 {
		return "medium"
	}
	return "low"
}

func seedFromString(value string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(value))
	return h.Sum32()
}

func (s *Service) publish(runID, eventType string, payload map[string]string) {
	if s.bus == nil {
		return
	}
	_, _ = s.bus.Publish(contracts.Event{
		RunID:   runID,
		Type:    eventType,
		Source:  "optimizer",
		Payload: payload,
	})
}

func errorsIsNoRows(err error) bool {
	if err == nil {
		return false
	}
	if err == sql.ErrNoRows {
		return true
	}
	return strings.Contains(strings.ToLower(err.Error()), strings.ToLower(sql.ErrNoRows.Error()))
}

func round(value float64, decimals int) float64 {
	if decimals < 0 {
		return value
	}
	factor := math.Pow10(decimals)
	return math.Round(value*factor) / factor
}

func clamp(value, minValue, maxValue float64) float64 {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}
