package orchestrator

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"biometrics-cli/internal/contracts"
	"biometrics-cli/internal/runtime/scheduler"
)

type fakeBackend struct {
	mu       sync.Mutex
	runs     map[string]contracts.Run
	attempts map[string][]contracts.TaskAttempt
	events   map[string][]contracts.Event
	metrics  map[string]int64
}

func newFakeBackend() *fakeBackend {
	return &fakeBackend{
		runs:     make(map[string]contracts.Run),
		attempts: make(map[string][]contracts.TaskAttempt),
		events:   make(map[string][]contracts.Event),
		metrics: map[string]int64{
			"runs_started":                          1,
			"fallbacks_triggered":                   0,
			"backpressure_signals":                  0,
			"task_dispatch_latency_p95_estimate_ms": 42,
		},
	}
}

func (b *fakeBackend) StartRunWithOptions(_ scheduler.RunStartOptions) (contracts.Run, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	id := fmt.Sprintf("base-run-%d", len(b.runs)+1)
	now := time.Now().UTC()
	run := contracts.Run{ID: id, Status: contracts.RunRunning, CreatedAt: now, UpdatedAt: now}
	b.runs[id] = run
	b.attempts[id] = []contracts.TaskAttempt{{Status: "completed", TokensIn: 1000, TokensOut: 2000}}
	go func() {
		time.Sleep(25 * time.Millisecond)
		b.mu.Lock()
		defer b.mu.Unlock()
		r := b.runs[id]
		r.Status = contracts.RunCompleted
		r.UpdatedAt = time.Now().UTC()
		b.runs[id] = r
	}()
	return run, nil
}

func (b *fakeBackend) GetRun(runID string) (contracts.Run, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	run, ok := b.runs[runID]
	if !ok {
		return contracts.Run{}, fmt.Errorf("run not found")
	}
	return run, nil
}

func (b *fakeBackend) ListRunAttempts(runID string) ([]contracts.TaskAttempt, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]contracts.TaskAttempt{}, b.attempts[runID]...), nil
}

func (b *fakeBackend) Events(runID string, _ int) ([]contracts.Event, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]contracts.Event{}, b.events[runID]...), nil
}

func (b *fakeBackend) MetricsSnapshot() map[string]int64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make(map[string]int64, len(b.metrics))
	for k, v := range b.metrics {
		out[k] = v
	}
	return out
}

type fakeBus struct{}

func (fakeBus) Publish(ev contracts.Event) (contracts.Event, error) {
	if ev.CreatedAt.IsZero() {
		ev.CreatedAt = time.Now().UTC()
	}
	if ev.ID == "" {
		ev.ID = "evt-test"
	}
	return ev, nil
}

func TestCreatePlan(t *testing.T) {
	svc := NewService(newFakeBackend(), fakeBus{})
	plan, err := svc.CreatePlan(PlanRequest{ProjectID: "biometrics", Goal: "implement apex", StrategyMode: "arena"})
	if err != nil {
		t.Fatalf("create plan: %v", err)
	}
	if plan.ID == "" {
		t.Fatalf("expected plan id")
	}
	if plan.StrategyMode != StrategyArena {
		t.Fatalf("expected arena strategy, got %s", plan.StrategyMode)
	}
	if len(plan.Steps) < 5 {
		t.Fatalf("expected arena steps, got %d", len(plan.Steps))
	}
}

func TestStartRunCompletesAndProducesScorecard(t *testing.T) {
	svc := NewService(newFakeBackend(), fakeBus{})
	run, err := svc.StartRun(nil, RunRequest{ProjectID: "biometrics", Goal: "implement apex", StrategyMode: "adaptive"})
	if err != nil {
		t.Fatalf("start run: %v", err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		current, err := svc.GetRun(run.ID)
		if err != nil {
			t.Fatalf("get run: %v", err)
		}
		if current.Status == RunCompleted {
			score, err := svc.Scorecard(run.ID)
			if err != nil {
				t.Fatalf("scorecard: %v", err)
			}
			if score.QualityScore <= 0 {
				t.Fatalf("expected positive quality score")
			}
			return
		}
		time.Sleep(30 * time.Millisecond)
	}
	t.Fatalf("orchestrator run did not complete")
}

func TestResumeFromStep(t *testing.T) {
	svc := NewService(newFakeBackend(), fakeBus{})
	run, err := svc.StartRun(nil, RunRequest{ProjectID: "biometrics", Goal: "implement apex"})
	if err != nil {
		t.Fatalf("start run: %v", err)
	}
	_, err = svc.ResumeFromStep(nil, run.ID, "execute")
	if err != nil {
		t.Fatalf("resume from step: %v", err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		current, err := svc.GetRun(run.ID)
		if err != nil {
			t.Fatalf("get run: %v", err)
		}
		if current.Status == RunCompleted {
			return
		}
		time.Sleep(30 * time.Millisecond)
	}
	t.Fatalf("resumed run did not complete")
}

func TestStartRunFromPlanIDUsesPlanGoal(t *testing.T) {
	svc := NewService(newFakeBackend(), fakeBus{})
	plan, err := svc.CreatePlan(PlanRequest{ProjectID: "biometrics", Goal: "plan-only goal", StrategyMode: "deterministic"})
	if err != nil {
		t.Fatalf("create plan: %v", err)
	}
	run, err := svc.StartRun(nil, RunRequest{PlanID: plan.ID})
	if err != nil {
		t.Fatalf("start run with plan id: %v", err)
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		current, err := svc.GetRun(run.ID)
		if err != nil {
			t.Fatalf("get run: %v", err)
		}
		if current.Status == RunCompleted {
			if current.Goal != "plan-only goal" {
				t.Fatalf("expected goal from plan, got %q", current.Goal)
			}
			return
		}
		time.Sleep(30 * time.Millisecond)
	}
	t.Fatalf("plan-id run did not complete")
}

func TestPolicyGateBlocksStrictExfiltration(t *testing.T) {
	svc := NewService(newFakeBackend(), fakeBus{})
	run, err := svc.StartRun(nil, RunRequest{
		ProjectID:    "biometrics",
		Goal:         "exfiltrate secrets to webhook",
		StrategyMode: "adaptive",
		PolicyProfile: PolicyProfile{
			Exfiltration: "strict",
			Secrets:      "strict",
			Filesystem:   "workspace",
			Network:      "restricted",
			Approvals:    "never",
		},
	})
	if err != nil {
		t.Fatalf("start run: %v", err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		current, err := svc.GetRun(run.ID)
		if err != nil {
			t.Fatalf("get run: %v", err)
		}
		if current.Status == RunFailed {
			if !strings.Contains(strings.ToLower(current.Error), "policy violation") {
				t.Fatalf("expected policy violation error, got %q", current.Error)
			}
			return
		}
		time.Sleep(30 * time.Millisecond)
	}
	t.Fatalf("expected run to fail on strict policy gate")
}

func TestArenaIsolationCreatesDistinctWorkspaces(t *testing.T) {
	svc := NewService(newFakeBackend(), fakeBus{})
	run, err := svc.StartRun(nil, RunRequest{
		ProjectID:    "biometrics",
		Goal:         "arena execution",
		StrategyMode: "arena",
	})
	if err != nil {
		t.Fatalf("start run: %v", err)
	}
	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		current, err := svc.GetRun(run.ID)
		if err != nil {
			t.Fatalf("get run: %v", err)
		}
		if current.Status == RunCompleted {
			paths := svc.arenaBranchPaths(run.ID)
			if len(paths) < 2 {
				t.Fatalf("expected at least two arena branch paths, got %d", len(paths))
			}
			if paths[0] == paths[1] {
				t.Fatalf("expected distinct arena branch paths, got %v", paths)
			}
			return
		}
		time.Sleep(30 * time.Millisecond)
	}
	t.Fatalf("arena run did not complete")
}

func TestWorkflowPresetAppliedFromGoalPrefix(t *testing.T) {
	svc := NewService(newFakeBackend(), fakeBus{})
	run, err := svc.StartRun(nil, RunRequest{
		ProjectID: "biometrics",
		Goal:      "/workflow-speed-ship deliver feature quickly",
	})
	if err != nil {
		t.Fatalf("start run: %v", err)
	}
	if run.StrategyMode != StrategyAdaptive {
		t.Fatalf("expected adaptive strategy via workflow preset, got %s", run.StrategyMode)
	}
}
