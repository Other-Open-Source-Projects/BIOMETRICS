package sqlite

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"biometrics-cli/internal/contracts"
	_ "github.com/mattn/go-sqlite3"
)

func TestCreateRunPersistsBlueprintFields(t *testing.T) {
	s, err := New(filepath.Join(t.TempDir(), "store.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	created, err := s.CreateRun(CreateRunOptions{
		ProjectID:        "biometrics",
		Goal:             "test blueprint persistence",
		Mode:             "autonomous",
		BlueprintProfile: "universal-2026",
		BlueprintModules: []string{"engine", "webapp"},
		Bootstrap:        true,
	})
	if err != nil {
		t.Fatalf("create run: %v", err)
	}

	fetched, err := s.GetRun(created.ID)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}

	if fetched.BlueprintProfile != "universal-2026" {
		t.Fatalf("unexpected blueprint profile: %s", fetched.BlueprintProfile)
	}
	if len(fetched.BlueprintModules) != 2 {
		t.Fatalf("unexpected module count: %d", len(fetched.BlueprintModules))
	}
	if !fetched.Bootstrap {
		t.Fatalf("expected bootstrap to be true")
	}
}

func TestCreateRunDefaultsBlueprintFields(t *testing.T) {
	s, err := New(filepath.Join(t.TempDir(), "store.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	created, err := s.CreateRun(CreateRunOptions{
		ProjectID: "biometrics",
		Goal:      "test defaults",
		Mode:      "autonomous",
	})
	if err != nil {
		t.Fatalf("create run: %v", err)
	}

	fetched, err := s.GetRun(created.ID)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}

	if fetched.BlueprintProfile != "" {
		t.Fatalf("expected empty blueprint profile, got %q", fetched.BlueprintProfile)
	}
	if len(fetched.BlueprintModules) != 0 {
		t.Fatalf("expected no blueprint modules, got %v", fetched.BlueprintModules)
	}
	if fetched.Bootstrap {
		t.Fatalf("expected bootstrap to be false")
	}
}

func TestCreateRunPersistsSchedulerFields(t *testing.T) {
	s, err := New(filepath.Join(t.TempDir(), "store.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	created, err := s.CreateRun(CreateRunOptions{
		ProjectID:         "biometrics",
		Goal:              "scheduler fields",
		Mode:              "autonomous",
		SchedulerMode:     contracts.SchedulerModeSerial,
		MaxParallelism:    1,
		FallbackTriggered: true,
	})
	if err != nil {
		t.Fatalf("create run: %v", err)
	}

	fetched, err := s.GetRun(created.ID)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}

	if fetched.SchedulerMode != contracts.SchedulerModeSerial {
		t.Fatalf("unexpected scheduler mode: %q", fetched.SchedulerMode)
	}
	if fetched.MaxParallelism != 1 {
		t.Fatalf("unexpected max_parallelism: %d", fetched.MaxParallelism)
	}
	if !fetched.FallbackTriggered {
		t.Fatalf("expected fallback_triggered true")
	}
}

func TestSaveAndLoadRunGraph(t *testing.T) {
	s, err := New(filepath.Join(t.TempDir(), "store.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	run, err := s.CreateRun(CreateRunOptions{
		ProjectID: "biometrics",
		Goal:      "graph roundtrip",
		Mode:      "autonomous",
	})
	if err != nil {
		t.Fatalf("create run: %v", err)
	}

	graph := contracts.TaskGraph{
		RunID: run.ID,
		Nodes: []contracts.TaskGraphNode{
			{ID: "task-a", Name: "planner", Agent: "planner", Priority: 100, Status: contracts.TaskPending, LifecycleState: contracts.TaskLifecycleReady},
			{ID: "task-b", Name: "scoper", Agent: "scoper", DependsOn: []string{"task-a"}, Priority: 90, Status: contracts.TaskPending, LifecycleState: contracts.TaskLifecycleBlocked},
		},
		Edges:        []contracts.TaskGraphEdge{{From: "task-a", To: "task-b"}},
		CriticalPath: []string{"task-a", "task-b"},
		CreatedAt:    time.Now().UTC(),
	}

	if err := s.SaveRunGraph(run.ID, graph); err != nil {
		t.Fatalf("save run graph: %v", err)
	}

	loaded, err := s.GetRunGraph(run.ID)
	if err != nil {
		t.Fatalf("get run graph: %v", err)
	}
	if loaded.RunID != run.ID {
		t.Fatalf("unexpected run id: %s", loaded.RunID)
	}
	if len(loaded.Nodes) != 2 {
		t.Fatalf("unexpected node count: %d", len(loaded.Nodes))
	}
	if len(loaded.Edges) != 1 {
		t.Fatalf("unexpected edge count: %d", len(loaded.Edges))
	}
}

func TestListEventsReturnsLatestLimitInAscendingOrder(t *testing.T) {
	s, err := New(filepath.Join(t.TempDir(), "store.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	base := time.Now().UTC().Add(-10 * time.Second)
	for i := 0; i < 5; i++ {
		_, err := s.AppendEvent(contracts.Event{
			RunID:     "run-a",
			Type:      "task.started",
			Source:    "test",
			Payload:   map[string]string{"sequence": string(rune('0' + i))},
			CreatedAt: base.Add(time.Duration(i) * time.Second),
		})
		if err != nil {
			t.Fatalf("append event %d: %v", i, err)
		}
	}
	_, err = s.AppendEvent(contracts.Event{
		RunID:     "run-b",
		Type:      "task.started",
		Source:    "test",
		Payload:   map[string]string{"sequence": "b"},
		CreatedAt: base.Add(12 * time.Second),
	})
	if err != nil {
		t.Fatalf("append run-b event: %v", err)
	}

	events, err := s.ListEvents("run-a", 2)
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if !events[0].CreatedAt.Before(events[1].CreatedAt) {
		t.Fatalf("expected ascending order, got %s then %s", events[0].CreatedAt, events[1].CreatedAt)
	}
	if events[0].CreatedAt != base.Add(3*time.Second) || events[1].CreatedAt != base.Add(4*time.Second) {
		t.Fatalf("expected latest events for run-a, got %s and %s", events[0].CreatedAt, events[1].CreatedAt)
	}
}

func TestMigrateAddsBlueprintColumnsToLegacyRunsTable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.db")
	legacyDB, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("open legacy db: %v", err)
	}
	_, err = legacyDB.Exec(`CREATE TABLE runs (
		id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL,
		goal TEXT NOT NULL,
		mode TEXT NOT NULL,
		status TEXT NOT NULL,
		error TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	);`)
	if err != nil {
		t.Fatalf("create legacy runs table: %v", err)
	}
	if err := legacyDB.Close(); err != nil {
		t.Fatalf("close legacy db: %v", err)
	}

	s, err := New(path)
	if err != nil {
		t.Fatalf("new store with legacy db: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	run, err := s.CreateRun(CreateRunOptions{
		ProjectID:        "biometrics",
		Goal:             "legacy migration",
		Mode:             "autonomous",
		BlueprintProfile: "universal-2026",
		BlueprintModules: []string{"engine"},
		Bootstrap:        true,
	})
	if err != nil {
		t.Fatalf("create run after migration: %v", err)
	}

	fetched, err := s.GetRun(run.ID)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	if fetched.BlueprintProfile != "universal-2026" || !fetched.Bootstrap {
		t.Fatalf("unexpected migrated run blueprint fields: %+v", fetched)
	}
}

func TestCreateListAndUpdateOptimizerRecommendation(t *testing.T) {
	s, err := New(filepath.Join(t.TempDir(), "store.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	created, err := s.CreateOptimizerRecommendation(OptimizerRecommendationRecord{
		ProjectID:      "biometrics",
		Goal:           "improve apex gates",
		StrategyMode:   "adaptive",
		SchedulerMode:  "dag_parallel_v1",
		MaxParallelism: 8,
		ModelPreference: "codex",
		FallbackChain:  []string{"gemini", "nim"},
		ContextBudget:  36000,
		Objective:      OptimizerObjectiveRecord{Quality: 0.6, Speed: 0.25, Cost: 0.15},
		Confidence:     "high",
		PredictedGates: OptimizerPredictedGatesRecord{
			QualityPass:                 true,
			TimePass:                    true,
			CostPass:                    true,
			RegressionPass:              true,
			AllPass:                     true,
			GatePassCount:               4,
			PredictedQualityScore:       0.93,
			PredictedTimeImprovementPct: 28.1,
			PredictedCostImprovementPct: 23.7,
			PredictedCompositeScore:     0.91,
		},
		Rationale: "high-confidence adaptive profile",
		Status:    "generated",
	})
	if err != nil {
		t.Fatalf("create optimizer recommendation: %v", err)
	}
	if created.ID == "" {
		t.Fatalf("expected recommendation id")
	}

	listed, err := s.ListOptimizerRecommendations(ListOptimizerRecommendationsOptions{
		ProjectID: "biometrics",
		Status:    "generated",
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("list optimizer recommendations: %v", err)
	}
	if len(listed) != 1 {
		t.Fatalf("expected 1 recommendation, got %d", len(listed))
	}
	if listed[0].PredictedGates.GatePassCount != 4 {
		t.Fatalf("unexpected gate count: %d", listed[0].PredictedGates.GatePassCount)
	}

	updated, err := s.UpdateOptimizerRecommendationStatus(created.ID, "applied", "orc-run-123", "")
	if err != nil {
		t.Fatalf("update optimizer recommendation status: %v", err)
	}
	if updated.Status != "applied" {
		t.Fatalf("expected status applied, got %q", updated.Status)
	}
	if updated.AppliedRunID != "orc-run-123" {
		t.Fatalf("expected applied run id, got %q", updated.AppliedRunID)
	}
}

func TestUpsertAndGetOptimizerValidation(t *testing.T) {
	s, err := New(filepath.Join(t.TempDir(), "store.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	recommendation, err := s.CreateOptimizerRecommendation(OptimizerRecommendationRecord{
		ProjectID:      "biometrics",
		Goal:           "validate recommendation",
		StrategyMode:   "adaptive",
		SchedulerMode:  "dag_parallel_v1",
		MaxParallelism: 8,
		ModelPreference: "codex",
		ContextBudget:  32000,
		Objective:      OptimizerObjectiveRecord{Quality: 0.5, Speed: 0.3, Cost: 0.2},
		Confidence:     "medium",
		PredictedGates: OptimizerPredictedGatesRecord{},
		Status:         "generated",
	})
	if err != nil {
		t.Fatalf("create recommendation: %v", err)
	}

	validation, err := s.UpsertOptimizerValidation(OptimizerValidationRecord{
		ID:               "opt-val-test",
		RecommendationID: recommendation.ID,
		EvalRunID:        "eval-run-1",
		Status:           "completed",
		QualityPass:      true,
		TimePass:         true,
		CostPass:         false,
		RegressionPass:   true,
		AllPass:          false,
		Summary:          "partial gate pass",
	})
	if err != nil {
		t.Fatalf("upsert validation: %v", err)
	}
	if validation.ID == "" {
		t.Fatalf("expected validation id")
	}

	got, err := s.GetOptimizerValidationByRecommendation(recommendation.ID)
	if err != nil {
		t.Fatalf("get validation by recommendation: %v", err)
	}
	if got.EvalRunID != "eval-run-1" {
		t.Fatalf("unexpected eval_run_id: %q", got.EvalRunID)
	}
	if got.CostPass {
		t.Fatalf("expected cost pass false")
	}
}
