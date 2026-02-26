package optimizer

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	runtimeorchestrator "biometrics-cli/internal/runtime/orchestrator"
	store "biometrics-cli/internal/store/sqlite"
)

type mockOrchestrator struct {
	lastRequest runtimeorchestrator.RunRequest
	runs        map[string]runtimeorchestrator.Run
}

func newMockOrchestrator() *mockOrchestrator {
	return &mockOrchestrator{
		runs: make(map[string]runtimeorchestrator.Run),
	}
}

func (m *mockOrchestrator) StartRun(_ context.Context, req runtimeorchestrator.RunRequest) (runtimeorchestrator.Run, error) {
	m.lastRequest = req
	run := runtimeorchestrator.Run{
		ID:                        "orc-run-test",
		PlanID:                    "orc-plan-test",
		ProjectID:                 req.ProjectID,
		Goal:                      req.Goal,
		StrategyMode:              runtimeorchestrator.NormalizeStrategyMode(req.StrategyMode),
		OptimizerRecommendationID: req.OptimizerRecommendationID,
		OptimizerConfidence:       req.OptimizerConfidence,
		Status:                    runtimeorchestrator.RunRunning,
		CreatedAt:                 time.Now().UTC(),
		UpdatedAt:                 time.Now().UTC(),
	}
	m.runs[run.ID] = run
	return run, nil
}

func (m *mockOrchestrator) GetRun(runID string) (runtimeorchestrator.Run, error) {
	if run, ok := m.runs[runID]; ok {
		return run, nil
	}
	return runtimeorchestrator.Run{}, context.Canceled
}

func (m *mockOrchestrator) ListScorecards() []runtimeorchestrator.Scorecard {
	return []runtimeorchestrator.Scorecard{
		{
			RunID:                    "orc-run-scorecard",
			QualityScore:             0.89,
			MedianTimeToGreenSeconds: 900,
			CostPerSuccess:           0.0042,
			CompositeScore:           0.84,
			GeneratedAt:              time.Now().UTC(),
		},
	}
}

func TestBuildCandidatesIncludesSeedsAndBoundaries(t *testing.T) {
	db, err := store.New(filepath.Join(t.TempDir(), "optimizer.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if _, err := db.CreateRun(store.CreateRunOptions{
		ProjectID:       "biometrics",
		Goal:            "seed current ui state",
		Mode:            "autonomous",
		SchedulerMode:   "serial",
		MaxParallelism:  12,
		ModelPreference: "gemini",
		FallbackChain:   []string{"nim"},
		ContextBudget:   28000,
	}); err != nil {
		t.Fatalf("create run seed: %v", err)
	}

	if _, err := db.CreateOptimizerRecommendation(store.OptimizerRecommendationRecord{
		ID:              "opt-rec-applied",
		ProjectID:       "biometrics",
		Goal:            "seed applied profile",
		StrategyMode:    "arena",
		SchedulerMode:   "dag_parallel_v1",
		MaxParallelism:  10,
		ModelPreference: "codex",
		FallbackChain:   []string{"gemini", "nim"},
		ContextBudget:   42000,
		Objective:       store.OptimizerObjectiveRecord{Quality: 0.6, Speed: 0.25, Cost: 0.15},
		Confidence:      "high",
		PredictedGates:  store.OptimizerPredictedGatesRecord{GatePassCount: 4, PredictedCompositeScore: 0.93, PredictedCostPerSuccess: 0.0028},
		Rationale:       "historical winner",
		Status:          "applied",
		AppliedRunID:    "orc-run-123",
	}); err != nil {
		t.Fatalf("create applied recommendation seed: %v", err)
	}

	candidates := buildCandidates(db, "biometrics", normalizeObjective(Objective{Quality: 0.5, Speed: 0.3, Cost: 0.2}))
	if len(candidates) == 0 {
		t.Fatalf("expected non-empty candidate set")
	}

	foundWorkflowSeed := false
	foundCurrentUISeed := false
	foundAppliedSeed := false
	for _, candidate := range candidates {
		if candidate.Rationale == "seed workflow-apex-hardening" {
			foundWorkflowSeed = true
		}
		if candidate.Rationale == "seed current-ui-state from latest run options" {
			foundCurrentUISeed = true
		}
		if candidate.Rationale == "seed best-last-applied-profile" {
			foundAppliedSeed = true
		}
		if candidate.MaxParallelism < 1 || candidate.MaxParallelism > 32 {
			t.Fatalf("candidate max_parallelism out of bounds: %d", candidate.MaxParallelism)
		}
		if candidate.SchedulerMode == "serial" && candidate.MaxParallelism != 1 {
			t.Fatalf("serial scheduler must force max_parallelism=1, got %d", candidate.MaxParallelism)
		}
		if candidate.ContextBudget < 1000 || candidate.ContextBudget > 200000 {
			t.Fatalf("candidate context_budget out of bounds: %d", candidate.ContextBudget)
		}
	}
	if !foundWorkflowSeed {
		t.Fatalf("expected workflow-apex-hardening seed candidate")
	}
	if !foundCurrentUISeed {
		t.Fatalf("expected current-ui-state seed candidate")
	}
	if !foundAppliedSeed {
		t.Fatalf("expected best-last-applied-profile seed candidate")
	}
}

func TestSortScoredCandidatesRanking(t *testing.T) {
	candidates := []candidateScore{
		{
			config: candidateConfig{StrategyMode: "adaptive", SchedulerMode: "dag_parallel_v1", MaxParallelism: 8, ModelPreference: "codex", ContextBudget: 24000, Objective: normalizeObjective(Objective{Quality: 0.5, Speed: 0.3, Cost: 0.2})},
			predicted: PredictedGates{
				GatePassCount:           3,
				PredictedCompositeScore: 0.91,
				PredictedCostPerSuccess: 0.0032,
			},
		},
		{
			config: candidateConfig{StrategyMode: "arena", SchedulerMode: "dag_parallel_v1", MaxParallelism: 10, ModelPreference: "codex", ContextBudget: 36000, Objective: normalizeObjective(Objective{Quality: 0.6, Speed: 0.25, Cost: 0.15})},
			predicted: PredictedGates{
				GatePassCount:           4,
				PredictedCompositeScore: 0.89,
				PredictedCostPerSuccess: 0.0029,
			},
		},
		{
			config: candidateConfig{StrategyMode: "deterministic", SchedulerMode: "dag_parallel_v1", MaxParallelism: 6, ModelPreference: "codex", ContextBudget: 26000, Objective: normalizeObjective(Objective{Quality: 0.55, Speed: 0.25, Cost: 0.2})},
			predicted: PredictedGates{
				GatePassCount:           3,
				PredictedCompositeScore: 0.91,
				PredictedCostPerSuccess: 0.0025,
			},
		},
	}

	sortScoredCandidates(candidates)
	if candidates[0].predicted.GatePassCount != 4 {
		t.Fatalf("expected top candidate with highest gate pass count")
	}
	if candidates[1].predicted.PredictedCostPerSuccess >= candidates[2].predicted.PredictedCostPerSuccess {
		t.Fatalf("expected lower predicted_cost_per_success tie-break for equal gate/composite scores")
	}
}

func TestConfidenceFromPredictionCompositeGap(t *testing.T) {
	best := PredictedGates{GatePassCount: 3, PredictedCompositeScore: 0.91}
	next := PredictedGates{GatePassCount: 3, PredictedCompositeScore: 0.86}
	if got := confidenceFromPrediction(best, &next); got != "high" {
		t.Fatalf("expected high confidence, got %q", got)
	}

	nextClose := PredictedGates{GatePassCount: 3, PredictedCompositeScore: 0.90}
	if got := confidenceFromPrediction(best, &nextClose); got != "medium" {
		t.Fatalf("expected medium confidence for narrow gap, got %q", got)
	}

	medium := PredictedGates{GatePassCount: 2, PredictedCompositeScore: 0.72}
	if got := confidenceFromPrediction(medium, nil); got != "medium" {
		t.Fatalf("expected medium confidence for 2/4 gates, got %q", got)
	}

	low := PredictedGates{GatePassCount: 1, PredictedCompositeScore: 0.66}
	if got := confidenceFromPrediction(low, nil); got != "low" {
		t.Fatalf("expected low confidence, got %q", got)
	}
}

func TestRecommendationTransitionsApplyRejectAndIdempotence(t *testing.T) {
	db, err := store.New(filepath.Join(t.TempDir(), "optimizer.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	rec, err := db.CreateOptimizerRecommendation(store.OptimizerRecommendationRecord{
		ID:              "opt-rec-generated",
		ProjectID:       "biometrics",
		Goal:            "manual apply",
		StrategyMode:    "adaptive",
		SchedulerMode:   "dag_parallel_v1",
		MaxParallelism:  8,
		ModelPreference: "codex",
		FallbackChain:   []string{"gemini", "nim"},
		ContextBudget:   32000,
		Objective:       store.OptimizerObjectiveRecord{Quality: 0.5, Speed: 0.3, Cost: 0.2},
		Confidence:      "medium",
		PredictedGates: store.OptimizerPredictedGatesRecord{
			GatePassCount:           3,
			PredictedCompositeScore: 0.88,
			PredictedCostPerSuccess: 0.0031,
		},
		Rationale: "generated",
		Status:    "generated",
	})
	if err != nil {
		t.Fatalf("create recommendation: %v", err)
	}

	orc := newMockOrchestrator()
	svc := NewService(db, orc, nil, nil)

	applyResult, err := svc.ApplyRecommendation(context.Background(), rec.ID)
	if err != nil {
		t.Fatalf("apply recommendation: %v", err)
	}
	if applyResult.RecommendationID != rec.ID {
		t.Fatalf("expected recommendation id %q, got %q", rec.ID, applyResult.RecommendationID)
	}
	if applyResult.Run.ID == "" {
		t.Fatalf("expected run id on apply")
	}
	if orc.lastRequest.OptimizerRecommendationID != rec.ID {
		t.Fatalf("expected optimizer recommendation id in orchestrator request")
	}

	applyAgain, err := svc.ApplyRecommendation(context.Background(), rec.ID)
	if err != nil {
		t.Fatalf("idempotent apply should succeed: %v", err)
	}
	if applyAgain.Run.ID != applyResult.Run.ID {
		t.Fatalf("expected idempotent apply to return same run id")
	}

	if _, err := svc.RejectRecommendation(rec.ID, "should fail"); err == nil {
		t.Fatalf("expected rejecting applied recommendation to fail")
	}

	rejected, err := db.CreateOptimizerRecommendation(store.OptimizerRecommendationRecord{
		ID:              "opt-rec-reject",
		ProjectID:       "biometrics",
		Goal:            "manual reject",
		StrategyMode:    "deterministic",
		SchedulerMode:   "dag_parallel_v1",
		MaxParallelism:  6,
		ModelPreference: "codex",
		FallbackChain:   []string{"gemini"},
		ContextBudget:   24000,
		Objective:       store.OptimizerObjectiveRecord{Quality: 0.6, Speed: 0.2, Cost: 0.2},
		Confidence:      "low",
		PredictedGates:  store.OptimizerPredictedGatesRecord{GatePassCount: 1, PredictedCompositeScore: 0.70, PredictedCostPerSuccess: 0.0039},
		Rationale:       "generated",
		Status:          "generated",
	})
	if err != nil {
		t.Fatalf("create second recommendation: %v", err)
	}

	rejectResult, err := svc.RejectRecommendation(rejected.ID, "operator override")
	if err != nil {
		t.Fatalf("reject recommendation: %v", err)
	}
	if rejectResult.Status != "rejected" {
		t.Fatalf("expected rejected status, got %q", rejectResult.Status)
	}

	rejectAgain, err := svc.RejectRecommendation(rejected.ID, "idempotent")
	if err != nil {
		t.Fatalf("idempotent reject should succeed: %v", err)
	}
	if rejectAgain.Status != "rejected" {
		t.Fatalf("expected rejected status on second reject call")
	}
}
