package evals

import (
	"os"
	"testing"
	"time"

	runtimeorchestrator "biometrics-cli/internal/runtime/orchestrator"
)

type fakeScorecards struct{}

func (fakeScorecards) ListScorecards() []runtimeorchestrator.Scorecard {
	return []runtimeorchestrator.Scorecard{{
		QualityScore:             0.92,
		MedianTimeToGreenSeconds: 800,
		CostPerSuccess:           0.0027,
	}}
}

func TestEvalRunAndLeaderboard(t *testing.T) {
	t.Setenv("BIOMETRICS_WORKSPACE", t.TempDir())

	svc := NewService(fakeScorecards{}, nil)
	run, err := svc.StartRun(nil, RunRequest{CandidateStrategy: "adaptive", BaselineStrategy: "deterministic", SampleSize: 100})
	if err != nil {
		t.Fatalf("start eval run: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		current, err := svc.GetRun(run.ID)
		if err != nil {
			t.Fatalf("get eval run: %v", err)
		}
		if current.Status == RunCompleted {
			if current.RegressionDetected {
				t.Fatalf("expected no regression for adaptive vs deterministic")
			}
			board := svc.Leaderboard()
			if len(board) == 0 {
				t.Fatalf("expected leaderboard entry")
			}
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("eval run did not complete")
}

func TestEvalRunProducesEvidenceAndComparison(t *testing.T) {
	workspace := t.TempDir()
	t.Setenv("BIOMETRICS_WORKSPACE", workspace)

	svc := NewService(fakeScorecards{}, nil)
	run, err := svc.StartRun(nil, RunRequest{
		CandidateStrategy:   "adaptive",
		BaselineStrategy:    "deterministic",
		SampleSize:          500,
		DatasetID:           "apex-suite-v1",
		Seed:                42,
		TasksLimit:          500,
		CompetitorBaselines: []string{"codex", "cursor"},
	})
	if err != nil {
		t.Fatalf("start eval run: %v", err)
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		current, err := svc.GetRun(run.ID)
		if err != nil {
			t.Fatalf("get eval run: %v", err)
		}
		if current.Status != RunCompleted {
			time.Sleep(20 * time.Millisecond)
			continue
		}

		if current.DatasetID != "apex-suite-v1" {
			t.Fatalf("unexpected dataset id: %q", current.DatasetID)
		}
		if len(current.EvidencePaths) < 2 {
			t.Fatalf("expected evidence paths, got %v", current.EvidencePaths)
		}
		for _, path := range current.EvidencePaths {
			if _, err := os.Stat(path); err != nil {
				t.Fatalf("expected evidence file %q: %v", path, err)
			}
		}
		if _, ok := current.Comparison[current.BaselineStrategy]; !ok {
			t.Fatalf("expected baseline comparison entry")
		}
		if _, ok := current.Comparison["codex"]; !ok {
			t.Fatalf("expected competitor comparison entry for codex")
		}
		return
	}
	t.Fatalf("eval run did not complete")
}

func TestEvalRegressionGateDetectsQualityDrop(t *testing.T) {
	t.Setenv("BIOMETRICS_WORKSPACE", t.TempDir())

	svc := NewService(fakeScorecards{}, nil)
	run, err := svc.StartRun(nil, RunRequest{
		CandidateStrategy: "deterministic",
		BaselineStrategy:  "codex",
		SampleSize:        500,
		TasksLimit:        500,
		DatasetID:         "apex-suite-v1",
		Seed:              11,
	})
	if err != nil {
		t.Fatalf("start eval run: %v", err)
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		current, err := svc.GetRun(run.ID)
		if err != nil {
			t.Fatalf("get eval run: %v", err)
		}
		if current.Status != RunCompleted {
			time.Sleep(20 * time.Millisecond)
			continue
		}
		baselineCmp, ok := current.Comparison[current.BaselineStrategy]
		if !ok {
			t.Fatalf("expected baseline comparison entry")
		}
		if baselineCmp.QualityDelta >= -0.03 {
			t.Fatalf("expected quality delta below -0.03, got %.4f", baselineCmp.QualityDelta)
		}
		if !current.RegressionDetected {
			t.Fatalf("expected regression to be detected")
		}
		return
	}
	t.Fatalf("eval run did not complete")
}

func TestAdaptiveQualityDoesNotRegressMoreThanThreePercentVsDeterministic(t *testing.T) {
	t.Setenv("BIOMETRICS_WORKSPACE", t.TempDir())

	svc := NewService(fakeScorecards{}, nil)
	run, err := svc.StartRun(nil, RunRequest{
		CandidateStrategy: "adaptive",
		BaselineStrategy:  "deterministic",
		SampleSize:        500,
		TasksLimit:        500,
		DatasetID:         "apex-suite-v1",
		Seed:              20260226,
	})
	if err != nil {
		t.Fatalf("start eval run: %v", err)
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		current, err := svc.GetRun(run.ID)
		if err != nil {
			t.Fatalf("get eval run: %v", err)
		}
		if current.Status != RunCompleted {
			time.Sleep(20 * time.Millisecond)
			continue
		}
		baselineCmp, ok := current.Comparison[current.BaselineStrategy]
		if !ok {
			t.Fatalf("expected baseline comparison entry")
		}
		if baselineCmp.QualityDelta < -0.03 {
			t.Fatalf("quality regression exceeds 3%% gate: %.4f", baselineCmp.QualityDelta)
		}
		return
	}
	t.Fatalf("eval run did not complete")
}
