package scheduler

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"biometrics-cli/internal/contracts"
	"biometrics-cli/internal/policy"
	"biometrics-cli/internal/runtime/actor"
	"biometrics-cli/internal/runtime/bus"
	"biometrics-cli/internal/runtime/supervisor"
	store "biometrics-cli/internal/store/sqlite"
)

func TestTopologicalNodeOrderDetectsCycle(t *testing.T) {
	defs := []taskNodeDef{
		{key: "a", depends: []string{"b"}},
		{key: "b", depends: []string{"a"}},
	}

	if _, err := topologicalNodeOrder(defs); err == nil {
		t.Fatal("expected cycle detection error, got nil")
	}
}

func TestBuildTaskNodeDefsHighCardinality(t *testing.T) {
	workPackages := make([]contracts.WorkPackage, 0, 50)
	for i := 0; i < 50; i++ {
		workPackages = append(workPackages, contracts.WorkPackage{
			ID:       fmt.Sprintf("wp-%02d", i+1),
			Title:    fmt.Sprintf("package-%02d", i+1),
			Priority: 100 - i,
		})
	}

	defs := buildTaskNodeDefs(contracts.PlannerPlan{
		Version:      1,
		WorkPackages: workPackages,
	})
	if got, want := len(defs), 203; got != want {
		t.Fatalf("unexpected node def count: got %d want %d", got, want)
	}

	ordered, err := topologicalNodeOrder(defs)
	if err != nil {
		t.Fatalf("topological order failed: %v", err)
	}
	if got, want := len(ordered), len(defs); got != want {
		t.Fatalf("unexpected topological order size: got %d want %d", got, want)
	}
}

func TestCriticalPathHasTerminalNode(t *testing.T) {
	tasks := []contracts.Task{
		{ID: "planner", DependsOn: []string{}},
		{ID: "coder-a", DependsOn: []string{"planner"}},
		{ID: "tester-a", DependsOn: []string{"coder-a"}},
		{ID: "integrator", DependsOn: []string{"tester-a"}},
		{ID: "reporter", DependsOn: []string{"integrator"}},
	}

	path := criticalPathIDs(tasks)
	if len(path) == 0 {
		t.Fatal("critical path should not be empty")
	}
	if path[len(path)-1] != "reporter" {
		t.Fatalf("expected terminal node reporter, got %s", path[len(path)-1])
	}
}

func TestHighCardinalityRunCompletesWithoutQueueDepthFallback(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tmp := t.TempDir()
	db, err := store.New(filepath.Join(tmp, "scheduler-test.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer db.Close()

	eventBus := bus.NewEventBus(db)
	sup := supervisor.New(eventBus)
	actors := actor.NewSystem(sup)
	handler := func(agentName string) actor.Handler {
		return func(_ context.Context, env contracts.AgentEnvelope) contracts.AgentResult {
			return contracts.AgentResult{
				RunID:      env.RunID,
				TaskID:     env.TaskID,
				Agent:      agentName,
				Success:    true,
				Summary:    "ok",
				FinishedAt: time.Now().UTC(),
			}
		}
	}
	for _, name := range []string{"planner", "scoper", "coder", "tester", "reviewer", "fixer", "integrator", "reporter"} {
		if err := actors.Register(name, 16, handler(name)); err != nil {
			t.Fatalf("register actor %s: %v", name, err)
		}
	}
	actors.Start(ctx)

	manager := NewRunManager(db, actors, eventBus, policy.Default(), tmp, nil, nil)
	run, err := manager.StartRunWithOptions(RunStartOptions{
		ProjectID:      "biometrics",
		Goal:           buildHighCardinalityGoal(50),
		Mode:           "autonomous",
		SchedulerMode:  contracts.SchedulerModeDAGParallelV1,
		MaxParallelism: 8,
	})
	if err != nil {
		t.Fatalf("start run: %v", err)
	}

	waitForTerminalRun(t, manager, run.ID, 15*time.Second)

	finished, err := manager.GetRun(run.ID)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	if finished.Status != contracts.RunCompleted {
		t.Fatalf("expected completed run, got %s", finished.Status)
	}
	if finished.FallbackTriggered {
		t.Fatalf("unexpected serial fallback for high-cardinality run")
	}

	graph, err := manager.GetRunGraph(run.ID)
	if err != nil {
		t.Fatalf("get graph: %v", err)
	}
	if got, want := len(graph.Nodes), 203; got != want {
		t.Fatalf("unexpected node count: got %d want %d", got, want)
	}
}

func TestSupervisedRunEmitsCheckpointAndCanResumeToCompletion(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tmp := t.TempDir()
	db, err := store.New(filepath.Join(tmp, "scheduler-supervised.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer db.Close()

	eventBus := bus.NewEventBus(db)
	sup := supervisor.New(eventBus)
	actors := actor.NewSystem(sup)
	handler := func(agentName string) actor.Handler {
		return func(_ context.Context, env contracts.AgentEnvelope) contracts.AgentResult {
			return contracts.AgentResult{
				RunID:      env.RunID,
				TaskID:     env.TaskID,
				Agent:      agentName,
				Success:    true,
				Summary:    "ok",
				FinishedAt: time.Now().UTC(),
			}
		}
	}
	for _, name := range []string{"planner", "scoper", "coder", "tester", "reviewer", "fixer", "integrator", "reporter"} {
		if err := actors.Register(name, 16, handler(name)); err != nil {
			t.Fatalf("register actor %s: %v", name, err)
		}
	}
	actors.Start(ctx)

	manager := NewRunManager(db, actors, eventBus, policy.Default(), tmp, nil, nil)
	run, err := manager.StartRunWithOptions(RunStartOptions{
		ProjectID:      "biometrics",
		Goal:           "supervised flow",
		Mode:           "supervised",
		SchedulerMode:  contracts.SchedulerModeDAGParallelV1,
		MaxParallelism: 4,
	})
	if err != nil {
		t.Fatalf("start run: %v", err)
	}

	deadline := time.Now().Add(20 * time.Second)
	sawPaused := false
	for time.Now().Before(deadline) {
		current, err := manager.GetRun(run.ID)
		if err != nil {
			t.Fatalf("get run: %v", err)
		}
		if current.Status == contracts.RunPaused {
			sawPaused = true
			break
		}
		if current.Status == contracts.RunCompleted || current.Status == contracts.RunFailed || current.Status == contracts.RunCancelled {
			break
		}
		time.Sleep(40 * time.Millisecond)
	}
	if !sawPaused {
		t.Fatalf("expected supervised run to hit at least one checkpoint pause")
	}

	for time.Now().Before(deadline) {
		current, err := manager.GetRun(run.ID)
		if err != nil {
			t.Fatalf("get run: %v", err)
		}
		switch current.Status {
		case contracts.RunCompleted:
			events, err := manager.Events(run.ID, 500)
			if err != nil {
				t.Fatalf("events: %v", err)
			}
			foundCheckpoint := false
			for _, event := range events {
				if event.Type == "run.supervision.checkpoint" {
					foundCheckpoint = true
					break
				}
			}
			if !foundCheckpoint {
				t.Fatalf("expected run.supervision.checkpoint event for supervised run")
			}
			return
		case contracts.RunFailed, contracts.RunCancelled:
			t.Fatalf("expected completed supervised run, got %s", current.Status)
		case contracts.RunPaused:
			if err := manager.ResumeRun(run.ID); err != nil {
				t.Fatalf("resume supervised run: %v", err)
			}
		}
		time.Sleep(40 * time.Millisecond)
	}

	t.Fatalf("supervised run did not complete before timeout")
}

func TestMetricsSnapshotIncludesQueueAndEventBusFields(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tmp := t.TempDir()
	db, err := store.New(filepath.Join(tmp, "scheduler-metrics.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer db.Close()

	eventBus := bus.NewEventBus(db)
	sup := supervisor.New(eventBus)
	actors := actor.NewSystem(sup)
	for _, name := range []string{"planner", "scoper", "coder", "tester", "reviewer", "fixer", "integrator", "reporter"} {
		agentName := name
		if err := actors.Register(agentName, 8, func(_ context.Context, env contracts.AgentEnvelope) contracts.AgentResult {
			return contracts.AgentResult{RunID: env.RunID, TaskID: env.TaskID, Agent: agentName, Success: true, Summary: "ok", FinishedAt: time.Now().UTC()}
		}); err != nil {
			t.Fatalf("register actor %s: %v", agentName, err)
		}
	}
	actors.Start(ctx)

	manager := NewRunManager(db, actors, eventBus, policy.Default(), tmp, nil, nil)
	metrics := manager.MetricsSnapshot()

	required := []string{
		"scheduler_ready_queue_depth",
		"scheduler_ready_queue_depth_max",
		"eventbus_dropped_events",
		"eventbus_subscribers",
	}
	for _, key := range required {
		if _, ok := metrics[key]; !ok {
			t.Fatalf("expected metric key %q", key)
		}
	}
}

func TestRunFailureRedactsSecretsInEventsAndAttempts(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tmp := t.TempDir()
	db, err := store.New(filepath.Join(tmp, "scheduler-redaction.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer db.Close()

	eventBus := bus.NewEventBus(db)
	sup := supervisor.New(eventBus)
	actors := actor.NewSystem(sup)
	for _, name := range []string{"planner", "scoper", "coder", "tester", "reviewer", "fixer", "integrator", "reporter"} {
		agentName := name
		handler := func(_ context.Context, env contracts.AgentEnvelope) contracts.AgentResult {
			if agentName == "coder" {
				return contracts.AgentResult{
					RunID:      env.RunID,
					TaskID:     env.TaskID,
					Agent:      agentName,
					Success:    false,
					Summary:    "token=example-secret-value",
					Error:      "api_key=example-private-key",
					FinishedAt: time.Now().UTC(),
				}
			}
			return contracts.AgentResult{
				RunID:      env.RunID,
				TaskID:     env.TaskID,
				Agent:      agentName,
				Success:    true,
				Summary:    "ok",
				FinishedAt: time.Now().UTC(),
			}
		}
		if err := actors.Register(agentName, 8, handler); err != nil {
			t.Fatalf("register actor %s: %v", agentName, err)
		}
	}
	actors.Start(ctx)

	manager := NewRunManager(db, actors, eventBus, policy.Default(), tmp, nil, nil)
	run, err := manager.StartRunWithOptions(RunStartOptions{
		ProjectID:      "biometrics",
		Goal:           "single package",
		Mode:           "autonomous",
		SchedulerMode:  contracts.SchedulerModeDAGParallelV1,
		MaxParallelism: 4,
	})
	if err != nil {
		t.Fatalf("start run: %v", err)
	}

	waitForTerminalRun(t, manager, run.ID, 8*time.Second)

	finished, err := manager.GetRun(run.ID)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	if finished.Status != contracts.RunFailed {
		t.Fatalf("expected failed run, got %s", finished.Status)
	}
	if strings.Contains(finished.Error, "example-private-key") {
		t.Fatalf("run error contains unredacted secret: %s", finished.Error)
	}

	attempts, err := manager.ListRunAttempts(run.ID)
	if err != nil {
		t.Fatalf("list attempts: %v", err)
	}
	foundFailed := false
	for _, attempt := range attempts {
		if attempt.Status != "failed" {
			continue
		}
		foundFailed = true
		if strings.Contains(attempt.Error, "example-private-key") {
			t.Fatalf("attempt error contains unredacted secret: %s", attempt.Error)
		}
		if strings.Contains(attempt.Log, "example-secret-value") {
			t.Fatalf("attempt log contains unredacted secret: %s", attempt.Log)
		}
	}
	if !foundFailed {
		t.Fatal("expected at least one failed attempt")
	}

	events, err := manager.Events(run.ID, 500)
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(events) == 0 {
		t.Fatal("expected run events")
	}
	for _, event := range events {
		for _, value := range event.Payload {
			if strings.Contains(value, "example-private-key") || strings.Contains(value, "example-secret-value") {
				t.Fatalf("event payload contains unredacted value: %+v", event)
			}
		}
	}
}

func TestAgentTimeoutDefaultFromEnvCanFailSlowAgent(t *testing.T) {
	t.Setenv("BIOMETRICS_AGENT_TIMEOUT_DEFAULT_SECONDS", "1")
	t.Setenv("BIOMETRICS_AGENT_TIMEOUT_CODER_SECONDS", "3")
	t.Setenv("BIOMETRICS_AGENT_TIMEOUT_FIXER_SECONDS", "3")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tmp := t.TempDir()
	db, err := store.New(filepath.Join(tmp, "scheduler-timeout-default.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer db.Close()

	eventBus := bus.NewEventBus(db)
	sup := supervisor.New(eventBus)
	actors := actor.NewSystem(sup)

	for _, name := range []string{"planner", "scoper", "coder", "tester", "reviewer", "fixer", "integrator", "reporter"} {
		agentName := name
		handler := func(_ context.Context, env contracts.AgentEnvelope) contracts.AgentResult {
			if agentName == "tester" {
				time.Sleep(1500 * time.Millisecond)
			}
			return contracts.AgentResult{
				RunID:      env.RunID,
				TaskID:     env.TaskID,
				Agent:      agentName,
				Success:    true,
				Summary:    "ok",
				FinishedAt: time.Now().UTC(),
			}
		}
		if err := actors.Register(agentName, 16, handler); err != nil {
			t.Fatalf("register actor %s: %v", agentName, err)
		}
	}
	actors.Start(ctx)

	manager := NewRunManager(db, actors, eventBus, policy.Default(), tmp, nil, nil)
	run, err := manager.StartRunWithOptions(RunStartOptions{
		ProjectID:      "biometrics",
		Goal:           "single package",
		Mode:           "autonomous",
		SchedulerMode:  contracts.SchedulerModeDAGParallelV1,
		MaxParallelism: 4,
	})
	if err != nil {
		t.Fatalf("start run: %v", err)
	}

	waitForTerminalRun(t, manager, run.ID, 12*time.Second)

	finished, err := manager.GetRun(run.ID)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	if finished.Status != contracts.RunFailed {
		t.Fatalf("expected failed run, got %s", finished.Status)
	}
	if !strings.Contains(finished.Error, "actor tester timeout") {
		t.Fatalf("expected tester timeout in run error, got %q", finished.Error)
	}
}

func TestAgentTimeoutDefaultsProvideLongerCoderAndFixerBudgets(t *testing.T) {
	t.Setenv("BIOMETRICS_AGENT_TIMEOUT_DEFAULT_SECONDS", "")
	t.Setenv("BIOMETRICS_AGENT_TIMEOUT_CODER_SECONDS", "")
	t.Setenv("BIOMETRICS_AGENT_TIMEOUT_FIXER_SECONDS", "")

	cfg := loadAgentTimeoutConfig()

	wantDefault := time.Duration(defaultAgentTimeoutSeconds) * time.Second
	if cfg.defaultTimeout != wantDefault {
		t.Fatalf("unexpected default timeout: got %s want %s", cfg.defaultTimeout, wantDefault)
	}

	wantCoder := time.Duration(defaultCoderTimeoutSeconds) * time.Second
	if got := cfg.overrides["coder"]; got != wantCoder {
		t.Fatalf("unexpected coder timeout: got %s want %s", got, wantCoder)
	}

	wantFixer := time.Duration(defaultFixerTimeoutSeconds) * time.Second
	if got := cfg.overrides["fixer"]; got != wantFixer {
		t.Fatalf("unexpected fixer timeout: got %s want %s", got, wantFixer)
	}
}

func TestAgentTimeoutOverrideAllowsCoderLongerThanDefault(t *testing.T) {
	t.Setenv("BIOMETRICS_AGENT_TIMEOUT_DEFAULT_SECONDS", "1")
	t.Setenv("BIOMETRICS_AGENT_TIMEOUT_CODER_SECONDS", "3")
	t.Setenv("BIOMETRICS_AGENT_TIMEOUT_FIXER_SECONDS", "3")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tmp := t.TempDir()
	db, err := store.New(filepath.Join(tmp, "scheduler-timeout-coder.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer db.Close()

	eventBus := bus.NewEventBus(db)
	sup := supervisor.New(eventBus)
	actors := actor.NewSystem(sup)

	for _, name := range []string{"planner", "scoper", "coder", "tester", "reviewer", "fixer", "integrator", "reporter"} {
		agentName := name
		handler := func(_ context.Context, env contracts.AgentEnvelope) contracts.AgentResult {
			if agentName == "coder" {
				time.Sleep(1500 * time.Millisecond)
			}
			return contracts.AgentResult{
				RunID:      env.RunID,
				TaskID:     env.TaskID,
				Agent:      agentName,
				Success:    true,
				Summary:    "ok",
				FinishedAt: time.Now().UTC(),
			}
		}
		if err := actors.Register(agentName, 16, handler); err != nil {
			t.Fatalf("register actor %s: %v", agentName, err)
		}
	}
	actors.Start(ctx)

	manager := NewRunManager(db, actors, eventBus, policy.Default(), tmp, nil, nil)
	run, err := manager.StartRunWithOptions(RunStartOptions{
		ProjectID:      "biometrics",
		Goal:           "single package",
		Mode:           "autonomous",
		SchedulerMode:  contracts.SchedulerModeDAGParallelV1,
		MaxParallelism: 4,
	})
	if err != nil {
		t.Fatalf("start run: %v", err)
	}

	waitForTerminalRun(t, manager, run.ID, 12*time.Second)

	finished, err := manager.GetRun(run.ID)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	if finished.Status != contracts.RunCompleted {
		t.Fatalf("expected completed run with coder override, got %s (error=%q)", finished.Status, finished.Error)
	}

	attempts, err := manager.ListRunAttempts(run.ID)
	if err != nil {
		t.Fatalf("list attempts: %v", err)
	}
	for _, attempt := range attempts {
		if attempt.Agent != "coder" || attempt.Status != "failed" {
			continue
		}
		if strings.Contains(attempt.Error, "actor coder timeout") {
			t.Fatalf("unexpected coder timeout with override: %q", attempt.Error)
		}
	}
}

func buildHighCardinalityGoal(parts int) string {
	if parts <= 0 {
		parts = 1
	}
	segments := make([]string, 0, parts)
	for i := 1; i <= parts; i++ {
		segments = append(segments, fmt.Sprintf("work package %02d", i))
	}
	return strings.Join(segments, ", ")
}

func waitForTerminalRun(t *testing.T, manager *RunManager, runID string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		run, err := manager.GetRun(runID)
		if err == nil && (run.Status == contracts.RunCompleted || run.Status == contracts.RunFailed || run.Status == contracts.RunCancelled) {
			return
		}
		time.Sleep(30 * time.Millisecond)
	}
	t.Fatalf("run %s did not reach terminal status before timeout", runID)
}
