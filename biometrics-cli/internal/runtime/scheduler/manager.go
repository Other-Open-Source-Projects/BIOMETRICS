package scheduler

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"biometrics-cli/internal/blueprints"
	contextindexer "biometrics-cli/internal/context/indexer"
	"biometrics-cli/internal/contracts"
	opencodeexec "biometrics-cli/internal/executor/opencode"
	"biometrics-cli/internal/onboarding"
	"biometrics-cli/internal/planning"
	"biometrics-cli/internal/policy"
	promptcompiler "biometrics-cli/internal/prompt/compiler"
	"biometrics-cli/internal/runtime/actor"
	"biometrics-cli/internal/runtime/bus"
	"biometrics-cli/internal/skillkit"
	store "biometrics-cli/internal/store/sqlite"
)

type RunStartOptions struct {
	ProjectID                 string
	Goal                      string
	Mode                      string
	Skills                    []string
	SkillSelectionMode        string
	SchedulerMode             contracts.SchedulerMode
	MaxParallelism            int
	ModelPreference           string
	FallbackChain             []string
	ModelID                   string
	ContextBudget             int
	BlueprintProfile          string
	BlueprintModules          []string
	Bootstrap                 bool
	OptimizerRecommendationID string
	OptimizerConfidence       string
}

type CodexAuthBroker interface {
	Status(ctx context.Context) (contracts.CodexAuthStatus, error)
	Login(ctx context.Context) (contracts.CodexAuthStatus, error)
	Logout(ctx context.Context) (contracts.CodexAuthStatus, error)
}

type ModelCatalogProvider interface {
	ModelCatalog(ctx context.Context) contracts.ModelCatalog
	MetricsSnapshot() map[string]int64
}

type RunManager struct {
	store             *store.Store
	actors            *actor.System
	bus               *bus.EventBus
	policy            *policy.Engine
	workspace         string
	blueprintRegistry *blueprints.Registry
	blueprintApplier  *blueprints.Applier
	skills            *skillkit.Manager
	codexBroker       CodexAuthBroker
	modelProvider     ModelCatalogProvider
	agentTimeouts     agentTimeoutConfig

	mu       sync.RWMutex
	controls map[string]*runControl
	metrics  runMetrics
}

const (
	defaultAgentTimeoutSeconds = 120
	// Coder/Fixer may traverse multiple provider attempts before returning.
	// Keep defaults above single-provider execution timeout to avoid false timeouts under fallback.
	defaultCoderTimeoutSeconds = 600
	defaultFixerTimeoutSeconds = 600
)

type agentTimeoutConfig struct {
	defaultTimeout time.Duration
	overrides      map[string]time.Duration
}

type runMetrics struct {
	runsStarted         atomic.Int64
	runsCompleted       atomic.Int64
	runsFailed          atomic.Int64
	tasksStarted        atomic.Int64
	tasksCompleted      atomic.Int64
	tasksFailed         atomic.Int64
	retriesScheduled    atomic.Int64
	fallbacksTriggered  atomic.Int64
	backpressureSignals atomic.Int64
	dispatchCount       atomic.Int64
	dispatchSumMs       atomic.Int64
	dispatchMaxMs       atomic.Int64
	dispatchLe25Ms      atomic.Int64
	dispatchLe50Ms      atomic.Int64
	dispatchLe100Ms     atomic.Int64
	dispatchLe250Ms     atomic.Int64
	dispatchLe500Ms     atomic.Int64
	dispatchLe1000Ms    atomic.Int64
	dispatchLe2000Ms    atomic.Int64
	dispatchGt2000Ms    atomic.Int64
	readyQueueDepth     atomic.Int64
	readyQueueDepthMax  atomic.Int64
}

type runControl struct {
	ctx    context.Context
	cancel context.CancelFunc

	mu     sync.Mutex
	paused bool
	cond   *sync.Cond
}

type taskNodeDef struct {
	key      string
	name     string
	agent    string
	depends  []string
	priority int
}

type taskOutcome struct {
	Task    contracts.Task
	Result  contracts.AgentResult
	Success bool
	Err     error
}

func NewRunManager(
	s *store.Store,
	actors *actor.System,
	eventBus *bus.EventBus,
	pol *policy.Engine,
	workspace string,
	blueprintRegistry *blueprints.Registry,
	blueprintApplier *blueprints.Applier,
) *RunManager {
	timeoutCfg := loadAgentTimeoutConfig()
	return &RunManager{
		store:             s,
		actors:            actors,
		bus:               eventBus,
		policy:            pol,
		workspace:         workspace,
		blueprintRegistry: blueprintRegistry,
		blueprintApplier:  blueprintApplier,
		agentTimeouts:     timeoutCfg,
		controls:          make(map[string]*runControl),
	}
}

func (m *RunManager) SetCodexAuthBroker(broker CodexAuthBroker) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.codexBroker = broker
}

func (m *RunManager) SetModelCatalogProvider(provider ModelCatalogProvider) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.modelProvider = provider
}

func (m *RunManager) SetSkillManager(skills *skillkit.Manager) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.skills = skills
}

func (m *RunManager) Store() *store.Store {
	return m.store
}

func (m *RunManager) CodexAuthStatus(ctx context.Context) (contracts.CodexAuthStatus, error) {
	m.mu.RLock()
	broker := m.codexBroker
	m.mu.RUnlock()

	if broker == nil {
		return contracts.CodexAuthStatus{
			Ready:       false,
			LoggedIn:    false,
			LastError:   "codex auth broker not configured",
			LastChecked: time.Now().UTC(),
		}, nil
	}
	return broker.Status(ctx)
}

func (m *RunManager) CodexAuthLogin(ctx context.Context) (contracts.CodexAuthStatus, error) {
	m.mu.RLock()
	broker := m.codexBroker
	m.mu.RUnlock()

	if broker == nil {
		return contracts.CodexAuthStatus{
			Ready:       false,
			LoggedIn:    false,
			LastError:   "codex auth broker not configured",
			LastChecked: time.Now().UTC(),
		}, fmt.Errorf("codex auth broker not configured")
	}
	status, err := broker.Login(ctx)
	return status, err
}

func (m *RunManager) CodexAuthLogout(ctx context.Context) (contracts.CodexAuthStatus, error) {
	m.mu.RLock()
	broker := m.codexBroker
	m.mu.RUnlock()

	if broker == nil {
		return contracts.CodexAuthStatus{
			Ready:       false,
			LoggedIn:    false,
			LastError:   "codex auth broker not configured",
			LastChecked: time.Now().UTC(),
		}, fmt.Errorf("codex auth broker not configured")
	}
	status, err := broker.Logout(ctx)
	return status, err
}

func (m *RunManager) ModelsCatalog(ctx context.Context) contracts.ModelCatalog {
	m.mu.RLock()
	provider := m.modelProvider
	m.mu.RUnlock()

	if provider == nil {
		return contracts.ModelCatalog{
			DefaultPrimary: "codex",
			DefaultChain:   []string{"gemini", "nim"},
			Providers:      []contracts.ModelProvider{},
		}
	}
	return provider.ModelCatalog(ctx)
}

func (m *RunManager) ListSkills() ([]skillkit.SkillMetadata, error) {
	m.mu.RLock()
	skills := m.skills
	m.mu.RUnlock()
	if skills == nil {
		return []skillkit.SkillMetadata{}, nil
	}
	return skills.List()
}

func (m *RunManager) GetSkill(name string) (skillkit.SkillMetadata, error) {
	m.mu.RLock()
	skills := m.skills
	m.mu.RUnlock()
	if skills == nil {
		return skillkit.SkillMetadata{}, fmt.Errorf("skills system is not configured")
	}
	return skills.Get(name)
}

func (m *RunManager) ReloadSkills() (skillkit.LoadOutcome, error) {
	m.mu.RLock()
	skills := m.skills
	m.mu.RUnlock()
	if skills == nil {
		return skillkit.LoadOutcome{
			Skills:        []skillkit.SkillMetadata{},
			Errors:        []skillkit.SkillError{},
			DisabledPaths: map[string]struct{}{},
			LoadedAt:      time.Now().UTC(),
		}, nil
	}
	return skills.Reload()
}

func (m *RunManager) ListInstallableSkills(ctx context.Context, experimental bool) (skillkit.OperationResult, error) {
	m.mu.RLock()
	skills := m.skills
	m.mu.RUnlock()
	if skills == nil {
		return skillkit.OperationResult{Status: "failed", Message: "skills system is not configured"}, fmt.Errorf("skills system is not configured")
	}
	return skills.ListInstallable(ctx, experimental)
}

func (m *RunManager) InstallSkill(ctx context.Context, req skillkit.InstallRequest) (skillkit.OperationResult, error) {
	m.mu.RLock()
	skills := m.skills
	m.mu.RUnlock()
	if skills == nil {
		return skillkit.OperationResult{Status: "failed", Message: "skills system is not configured"}, fmt.Errorf("skills system is not configured")
	}

	_, _ = m.bus.Publish(contracts.Event{
		Type:   "skill.install.started",
		Source: "scheduler",
		Payload: map[string]string{
			"name": req.Name,
			"repo": req.Repo,
		},
	})
	result, err := skills.Install(ctx, req)
	if err != nil {
		_, _ = m.bus.Publish(contracts.Event{
			Type:   "skill.install.failed",
			Source: "scheduler",
			Payload: map[string]string{
				"error": m.redact(err.Error()),
				"name":  req.Name,
			},
		})
		return result, err
	}
	_, _ = m.bus.Publish(contracts.Event{
		Type:   "skill.install.succeeded",
		Source: "scheduler",
		Payload: map[string]string{
			"name":    req.Name,
			"message": result.Message,
		},
	})
	return result, nil
}

func (m *RunManager) CreateSkill(ctx context.Context, req skillkit.CreateRequest) (skillkit.OperationResult, error) {
	m.mu.RLock()
	skills := m.skills
	m.mu.RUnlock()
	if skills == nil {
		return skillkit.OperationResult{Status: "failed", Message: "skills system is not configured"}, fmt.Errorf("skills system is not configured")
	}

	_, _ = m.bus.Publish(contracts.Event{
		Type:   "skill.create.started",
		Source: "scheduler",
		Payload: map[string]string{
			"name": req.Name,
			"path": req.Path,
		},
	})
	result, err := skills.Create(ctx, req)
	if err != nil {
		_, _ = m.bus.Publish(contracts.Event{
			Type:   "skill.create.failed",
			Source: "scheduler",
			Payload: map[string]string{
				"name":  req.Name,
				"error": m.redact(err.Error()),
			},
		})
		return result, err
	}
	_, _ = m.bus.Publish(contracts.Event{
		Type:   "skill.create.succeeded",
		Source: "scheduler",
		Payload: map[string]string{
			"name":    req.Name,
			"message": result.Message,
		},
	})
	return result, nil
}

func (m *RunManager) EnableSkill(reference string) (skillkit.OperationResult, error) {
	m.mu.RLock()
	skills := m.skills
	m.mu.RUnlock()
	if skills == nil {
		return skillkit.OperationResult{Status: "failed", Message: "skills system is not configured"}, fmt.Errorf("skills system is not configured")
	}
	return skills.Enable(reference)
}

func (m *RunManager) DisableSkill(reference string) (skillkit.OperationResult, error) {
	m.mu.RLock()
	skills := m.skills
	m.mu.RUnlock()
	if skills == nil {
		return skillkit.OperationResult{Status: "failed", Message: "skills system is not configured"}, fmt.Errorf("skills system is not configured")
	}
	return skills.Disable(reference)
}

func (m *RunManager) StartRun(projectID, goal, mode string) (contracts.Run, error) {
	return m.StartRunWithOptions(RunStartOptions{
		ProjectID: projectID,
		Goal:      goal,
		Mode:      mode,
	})
}

func (m *RunManager) StartRunWithOptions(opts RunStartOptions) (contracts.Run, error) {
	m.mu.RLock()
	skillsManager := m.skills
	m.mu.RUnlock()

	if strings.TrimSpace(opts.ProjectID) == "" {
		opts.ProjectID = "biometrics"
	}
	if strings.TrimSpace(opts.Goal) == "" {
		return contracts.Run{}, fmt.Errorf("goal is required")
	}
	if strings.TrimSpace(opts.Mode) == "" {
		opts.Mode = string(contracts.RunModeAutonomous)
	}
	if !contracts.IsValidRunMode(opts.Mode) {
		return contracts.Run{}, fmt.Errorf("invalid mode: %s", opts.Mode)
	}
	opts.Mode = string(contracts.NormalizeRunMode(opts.Mode))

	selectionMode := skillkit.NormalizeSelectionMode(opts.SkillSelectionMode)

	if opts.SchedulerMode == "" {
		opts.SchedulerMode = contracts.SchedulerModeDAGParallelV1
	}
	if opts.SchedulerMode != contracts.SchedulerModeDAGParallelV1 && opts.SchedulerMode != contracts.SchedulerModeSerial {
		return contracts.Run{}, fmt.Errorf("invalid scheduler_mode: %s", opts.SchedulerMode)
	}
	if opts.MaxParallelism <= 0 {
		opts.MaxParallelism = 8
	}
	if opts.MaxParallelism > 32 {
		opts.MaxParallelism = 32
	}
	if opts.SchedulerMode == contracts.SchedulerModeSerial {
		opts.MaxParallelism = 1
	}

	opts.ModelPreference = strings.ToLower(strings.TrimSpace(opts.ModelPreference))
	if opts.ModelPreference == "" {
		opts.ModelPreference = "codex"
	}
	opts.FallbackChain = normalizeProviderIDs(opts.FallbackChain)
	if len(opts.FallbackChain) == 0 {
		switch opts.ModelPreference {
		case "codex":
			opts.FallbackChain = []string{"gemini", "nim"}
		case "gemini":
			opts.FallbackChain = []string{"nim"}
		default:
			opts.FallbackChain = []string{}
		}
	}
	opts.ModelID = strings.TrimSpace(opts.ModelID)
	if opts.ContextBudget <= 0 {
		opts.ContextBudget = 24000
	}
	if opts.ContextBudget > 200000 {
		opts.ContextBudget = 200000
	}

	selectedSkills := []skillkit.SkillMetadata{}
	blockedSkills := []skillkit.BlockedSkill{}
	skillsLoaded := 0
	skillsErrors := 0
	if skillsManager != nil {
		selection, err := skillsManager.Select(opts.Goal, opts.Skills, selectionMode)
		if err != nil {
			return contracts.Run{}, err
		}
		selectedSkills = append([]skillkit.SkillMetadata{}, selection.Selected...)
		blockedSkills = append([]skillkit.BlockedSkill{}, selection.Blocked...)
		loaded, errs, _ := skillsManager.Stats()
		skillsLoaded = loaded
		skillsErrors = errs
	}
	selectedSkillNames := make([]string, 0, len(selectedSkills))
	for _, skill := range selectedSkills {
		selectedSkillNames = append(selectedSkillNames, skill.Name)
	}

	blueprintProfile, blueprintModules, err := m.prepareBlueprintSelection(
		opts.BlueprintProfile,
		opts.BlueprintModules,
		opts.Bootstrap || strings.TrimSpace(opts.BlueprintProfile) != "" || len(opts.BlueprintModules) > 0,
	)
	if err != nil {
		return contracts.Run{}, err
	}

	run, err := m.store.CreateRun(store.CreateRunOptions{
		ProjectID:                 opts.ProjectID,
		Goal:                      opts.Goal,
		Mode:                      opts.Mode,
		Skills:                    selectedSkillNames,
		SkillSelectionMode:        string(selectionMode),
		SchedulerMode:             opts.SchedulerMode,
		MaxParallelism:            opts.MaxParallelism,
		ModelPreference:           opts.ModelPreference,
		FallbackChain:             opts.FallbackChain,
		ModelID:                   opts.ModelID,
		ContextBudget:             opts.ContextBudget,
		BlueprintProfile:          blueprintProfile,
		BlueprintModules:          blueprintModules,
		Bootstrap:                 opts.Bootstrap,
		OptimizerRecommendationID: opts.OptimizerRecommendationID,
		OptimizerConfidence:       opts.OptimizerConfidence,
	})
	if err != nil {
		return contracts.Run{}, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	ctrl := &runControl{ctx: ctx, cancel: cancel}
	ctrl.cond = sync.NewCond(&ctrl.mu)

	m.mu.Lock()
	m.controls[run.ID] = ctrl
	m.mu.Unlock()

	payload := map[string]string{
		"project_id":           run.ProjectID,
		"goal":                 run.Goal,
		"mode":                 run.Mode,
		"skill_selection_mode": run.SkillSelectionMode,
		"scheduler_mode":       string(run.SchedulerMode),
		"max_parallelism":      fmt.Sprintf("%d", run.MaxParallelism),
		"model_preference":     run.ModelPreference,
		"context_budget":       fmt.Sprintf("%d", run.ContextBudget),
	}
	if len(run.Skills) > 0 {
		payload["skills"] = strings.Join(run.Skills, ",")
	}
	if len(run.FallbackChain) > 0 {
		payload["fallback_chain"] = strings.Join(run.FallbackChain, ",")
	}
	if run.ModelID != "" {
		payload["model_id"] = run.ModelID
	}
	if run.BlueprintProfile != "" {
		payload["blueprint_profile"] = run.BlueprintProfile
	}
	if len(run.BlueprintModules) > 0 {
		payload["blueprint_modules"] = strings.Join(run.BlueprintModules, ",")
	}
	if run.Bootstrap {
		payload["bootstrap"] = "true"
	}
	if run.OptimizerRecommendationID != "" {
		payload["optimizer_recommendation_id"] = run.OptimizerRecommendationID
	}
	if run.OptimizerConfidence != "" {
		payload["optimizer_confidence"] = run.OptimizerConfidence
	}
	_, _ = m.bus.Publish(contracts.Event{RunID: run.ID, Type: "run.created", Source: "scheduler", Payload: payload})

	if skillsManager != nil {
		_, _ = m.bus.Publish(contracts.Event{
			RunID:  run.ID,
			Type:   "skills.loaded",
			Source: "scheduler",
			Payload: map[string]string{
				"loaded": fmt.Sprintf("%d", skillsLoaded),
				"errors": fmt.Sprintf("%d", skillsErrors),
				"mode":   run.SkillSelectionMode,
			},
		})
		for _, skill := range selectedSkills {
			_, _ = m.bus.Publish(contracts.Event{
				RunID:  run.ID,
				Type:   "skill.selected",
				Source: "scheduler",
				Payload: map[string]string{
					"name":  skill.Name,
					"scope": string(skill.Scope),
					"path":  skill.PathToSkillMD,
				},
			})
		}
		for _, blocked := range blockedSkills {
			_, _ = m.bus.Publish(contracts.Event{
				RunID:  run.ID,
				Type:   "skill.invocation.blocked",
				Source: "scheduler",
				Payload: map[string]string{
					"name":   blocked.Name,
					"reason": blocked.Reason,
				},
			})
		}
	}

	if run.BlueprintProfile != "" {
		selectedPayload := map[string]string{"profile": run.BlueprintProfile}
		if len(run.BlueprintModules) > 0 {
			selectedPayload["modules"] = strings.Join(run.BlueprintModules, ",")
		}
		_, _ = m.bus.Publish(contracts.Event{RunID: run.ID, Type: "blueprint.selected", Source: "scheduler", Payload: selectedPayload})
	}

	plan := planning.BuildPlan(run.Goal)
	createdTasks, graph, err := m.materializeRunGraph(run.ID, plan)
	if err != nil {
		_ = m.store.UpdateRunStatus(run.ID, contracts.RunFailed, err.Error())
		m.cleanupRun(run.ID)
		return contracts.Run{}, err
	}
	if err := m.store.SaveRunGraph(run.ID, graph); err != nil {
		_ = m.store.UpdateRunStatus(run.ID, contracts.RunFailed, err.Error())
		m.cleanupRun(run.ID)
		return contracts.Run{}, err
	}

	_, _ = m.bus.Publish(contracts.Event{
		RunID:  run.ID,
		Type:   "run.graph.materialized",
		Source: "scheduler",
		Payload: map[string]string{
			"nodes":         fmt.Sprintf("%d", len(graph.Nodes)),
			"edges":         fmt.Sprintf("%d", len(graph.Edges)),
			"work_packages": fmt.Sprintf("%d", len(plan.WorkPackages)),
		},
	})

	go m.executeRun(run, ctrl, createdTasks)
	return run, nil
}

func (m *RunManager) materializeRunGraph(runID string, plan contracts.PlannerPlan) ([]contracts.Task, contracts.TaskGraph, error) {
	nodeDefs := buildTaskNodeDefs(plan)
	orderedKeys, err := topologicalNodeOrder(nodeDefs)
	if err != nil {
		return nil, contracts.TaskGraph{}, err
	}

	defsByKey := make(map[string]taskNodeDef, len(nodeDefs))
	for _, def := range nodeDefs {
		defsByKey[def.key] = def
	}

	idByKey := make(map[string]string, len(nodeDefs))
	tasks := make([]contracts.Task, 0, len(nodeDefs))
	graph := contracts.TaskGraph{
		RunID:     runID,
		Nodes:     make([]contracts.TaskGraphNode, 0, len(nodeDefs)),
		Edges:     make([]contracts.TaskGraphEdge, 0, len(nodeDefs)*2),
		CreatedAt: time.Now().UTC(),
	}

	for _, key := range orderedKeys {
		def := defsByKey[key]
		dependsOnIDs := make([]string, 0, len(def.depends))
		for _, depKey := range def.depends {
			depID, ok := idByKey[depKey]
			if !ok {
				return nil, contracts.TaskGraph{}, fmt.Errorf("unknown dependency key %s", depKey)
			}
			dependsOnIDs = append(dependsOnIDs, depID)
		}

		lifecycle := contracts.TaskLifecycleBlocked
		if len(dependsOnIDs) == 0 {
			lifecycle = contracts.TaskLifecycleReady
		}

		task, err := m.store.CreateTaskWithOptions(runID, store.CreateTaskOptions{
			Name:           def.name,
			Agent:          def.agent,
			DependsOn:      dependsOnIDs,
			Priority:       def.priority,
			MaxAttempts:    3,
			LifecycleState: lifecycle,
		})
		if err != nil {
			return nil, contracts.TaskGraph{}, fmt.Errorf("create task %s: %w", def.name, err)
		}

		idByKey[key] = task.ID
		tasks = append(tasks, task)
		graph.Nodes = append(graph.Nodes, contracts.TaskGraphNode{
			ID:             task.ID,
			Name:           task.Name,
			Agent:          task.Agent,
			DependsOn:      append([]string{}, task.DependsOn...),
			Priority:       task.Priority,
			Status:         task.Status,
			LifecycleState: task.LifecycleState,
		})

		for _, depID := range dependsOnIDs {
			graph.Edges = append(graph.Edges, contracts.TaskGraphEdge{From: depID, To: task.ID})
		}
	}

	graph.CriticalPath = criticalPathIDs(tasks)
	return tasks, graph, nil
}

func buildTaskNodeDefs(plan contracts.PlannerPlan) []taskNodeDef {
	defs := make([]taskNodeDef, 0, 3+len(plan.WorkPackages)*4)
	defs = append(defs, taskNodeDef{key: "planner", name: "planner", agent: "planner", depends: []string{}, priority: 1000})

	reviewerKeys := make([]string, 0, len(plan.WorkPackages))
	for i, wp := range plan.WorkPackages {
		idx := i + 1
		scoperKey := fmt.Sprintf("scoper.%s", wp.ID)
		coderKey := fmt.Sprintf("coder.%s", wp.ID)
		testerKey := fmt.Sprintf("tester.%s", wp.ID)
		reviewerKey := fmt.Sprintf("reviewer.%s", wp.ID)
		reviewerKeys = append(reviewerKeys, reviewerKey)

		defs = append(defs,
			taskNodeDef{key: scoperKey, name: fmt.Sprintf("scoper:%s", wp.ID), agent: "scoper", depends: []string{"planner"}, priority: 900 - idx},
			taskNodeDef{key: coderKey, name: fmt.Sprintf("coder:%s", wp.ID), agent: "coder", depends: []string{scoperKey}, priority: 800 - idx},
			taskNodeDef{key: testerKey, name: fmt.Sprintf("tester:%s", wp.ID), agent: "tester", depends: []string{coderKey}, priority: 700 - idx},
			taskNodeDef{key: reviewerKey, name: fmt.Sprintf("reviewer:%s", wp.ID), agent: "reviewer", depends: []string{testerKey}, priority: 600 - idx},
		)
	}

	defs = append(defs,
		taskNodeDef{key: "integrator", name: "integrator", agent: "integrator", depends: reviewerKeys, priority: 100},
		taskNodeDef{key: "reporter", name: "reporter", agent: "reporter", depends: []string{"integrator"}, priority: 50},
	)

	return defs
}

func topologicalNodeOrder(defs []taskNodeDef) ([]string, error) {
	if len(defs) == 0 {
		return nil, fmt.Errorf("no task definitions")
	}

	remaining := make(map[string]int, len(defs))
	dependents := make(map[string][]string, len(defs))
	exists := make(map[string]struct{}, len(defs))
	for _, def := range defs {
		exists[def.key] = struct{}{}
	}
	for _, def := range defs {
		remaining[def.key] = len(def.depends)
		for _, dep := range def.depends {
			if _, ok := exists[dep]; !ok {
				return nil, fmt.Errorf("dependency %s for %s does not exist", dep, def.key)
			}
			dependents[dep] = append(dependents[dep], def.key)
		}
	}

	ready := make([]string, 0, len(defs))
	for key, count := range remaining {
		if count == 0 {
			ready = append(ready, key)
		}
	}
	sort.Strings(ready)

	ordered := make([]string, 0, len(defs))
	for len(ready) > 0 {
		current := ready[0]
		ready = ready[1:]
		ordered = append(ordered, current)
		for _, dep := range dependents[current] {
			remaining[dep]--
			if remaining[dep] == 0 {
				ready = append(ready, dep)
			}
		}
		sort.Strings(ready)
	}

	if len(ordered) != len(defs) {
		return nil, fmt.Errorf("task graph cycle detected")
	}

	return ordered, nil
}

func criticalPathIDs(tasks []contracts.Task) []string {
	if len(tasks) == 0 {
		return []string{}
	}

	byID := make(map[string]contracts.Task, len(tasks))
	dependents := make(map[string][]string, len(tasks))
	indegree := make(map[string]int, len(tasks))
	for _, t := range tasks {
		byID[t.ID] = t
		indegree[t.ID] = len(t.DependsOn)
		for _, dep := range t.DependsOn {
			dependents[dep] = append(dependents[dep], t.ID)
		}
	}

	ready := make([]string, 0, len(tasks))
	for id, degree := range indegree {
		if degree == 0 {
			ready = append(ready, id)
		}
	}
	sort.Strings(ready)

	distance := make(map[string]int, len(tasks))
	parent := make(map[string]string, len(tasks))
	for len(ready) > 0 {
		id := ready[0]
		ready = ready[1:]
		for _, child := range dependents[id] {
			if distance[id]+1 > distance[child] {
				distance[child] = distance[id] + 1
				parent[child] = id
			}
			indegree[child]--
			if indegree[child] == 0 {
				ready = append(ready, child)
			}
		}
	}

	terminal := ""
	best := -1
	for id, dist := range distance {
		if dist > best {
			best = dist
			terminal = id
		}
	}
	if terminal == "" {
		for _, t := range tasks {
			if len(t.DependsOn) == 0 {
				return []string{t.ID}
			}
		}
		return []string{}
	}

	path := make([]string, 0, best+1)
	for current := terminal; current != ""; current = parent[current] {
		path = append(path, current)
	}
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	return path
}

func (m *RunManager) prepareBlueprintSelection(profileID string, moduleIDs []string, requireProfile bool) (string, []string, error) {
	profileID = strings.TrimSpace(profileID)
	moduleIDs = normalizeModuleIDs(moduleIDs)

	if !requireProfile {
		return "", []string{}, nil
	}
	if m.blueprintRegistry == nil {
		return "", nil, fmt.Errorf("blueprint catalog is not configured")
	}

	var profile blueprints.ProfileSpec
	var err error
	if profileID == "" {
		profile, err = m.blueprintRegistry.DefaultProfile()
	} else {
		profile, err = m.blueprintRegistry.GetProfile(profileID)
	}
	if err != nil {
		return "", nil, err
	}

	if len(moduleIDs) == 0 {
		return profile.ID, []string{}, nil
	}

	allowed := make(map[string]struct{}, len(profile.Modules))
	for _, module := range profile.Modules {
		allowed[module.ID] = struct{}{}
	}

	for _, moduleID := range moduleIDs {
		if _, ok := allowed[moduleID]; !ok {
			return "", nil, fmt.Errorf("unknown blueprint module %q for profile %q", moduleID, profile.ID)
		}
	}

	return profile.ID, moduleIDs, nil
}

func (m *RunManager) executeRun(run contracts.Run, ctrl *runControl, tasks []contracts.Task) {
	_ = m.store.UpdateRunStatus(run.ID, contracts.RunRunning, "")
	_, _ = m.bus.Publish(contracts.Event{RunID: run.ID, Type: "run.started", Source: "scheduler"})
	m.metrics.runsStarted.Add(1)

	outputs := make(map[string]string)
	var outputsMu sync.RWMutex

	if run.BlueprintProfile != "" {
		outputs["blueprint_profile"] = run.BlueprintProfile
	}
	if len(run.BlueprintModules) > 0 {
		outputs["blueprint_modules"] = strings.Join(run.BlueprintModules, ",")
	}

	if run.Bootstrap {
		if err := m.applyBlueprintForRun(run); err != nil {
			m.failRun(run.ID, fmt.Sprintf("blueprint bootstrap failed: %v", err))
			return
		}
	}

	if len(tasks) == 0 {
		loaded, err := m.store.ListTasksByRun(run.ID)
		if err != nil {
			m.failRun(run.ID, fmt.Sprintf("list tasks: %v", err))
			return
		}
		tasks = loaded
	}
	if len(tasks) == 0 {
		m.failRun(run.ID, "run has no tasks")
		return
	}

	tasksByID := make(map[string]contracts.Task, len(tasks))
	remainingDeps := make(map[string]int, len(tasks))
	dependents := make(map[string][]string, len(tasks))
	for _, task := range tasks {
		tasksByID[task.ID] = task
		remainingDeps[task.ID] = len(task.DependsOn)
		for _, dep := range task.DependsOn {
			dependents[dep] = append(dependents[dep], task.ID)
		}
	}

	readyQueue := make([]string, 0, len(tasks))
	for _, task := range tasks {
		if remainingDeps[task.ID] == 0 {
			readyQueue = append(readyQueue, task.ID)
		} else {
			_ = m.store.UpdateTaskLifecycle(task.ID, contracts.TaskLifecycleBlocked)
			_, _ = m.bus.Publish(contracts.Event{
				RunID:  run.ID,
				Type:   "task.blocked",
				Source: "scheduler",
				Payload: map[string]string{
					"task_id":      task.ID,
					"waiting_deps": fmt.Sprintf("%d", remainingDeps[task.ID]),
				},
			})
		}
	}

	currentParallelism := run.MaxParallelism
	if currentParallelism <= 0 {
		currentParallelism = 8
	}
	if run.SchedulerMode == contracts.SchedulerModeSerial {
		currentParallelism = 1
	}

	backpressureThreshold := currentParallelism * 8
	if backpressureThreshold < 16 {
		backpressureThreshold = 16
	}
	const backpressureSignalCooldown = 10 * time.Second
	lastBackpressureSignal := time.Time{}

	resultCh := make(chan taskOutcome, len(tasks)+32)
	running := make(map[string]struct{}, len(tasks))
	completed := 0
	total := len(tasks)
	supervised := contracts.NormalizeRunMode(run.Mode) == contracts.RunModeSupervised
	supervisionSeen := map[string]struct{}{}
	reviewerTotal := 0
	reviewerCompleted := 0
	if supervised {
		for _, task := range tasks {
			if strings.HasPrefix(task.Name, "reviewer:") {
				reviewerTotal++
			}
		}
	}

	for completed < total {
		if m.isCancelled(ctrl) {
			m.cancelRunInternal(run.ID)
			return
		}
		m.waitIfPaused(ctrl)
		m.observeReadyQueueDepth(len(readyQueue))

		if len(readyQueue) > backpressureThreshold {
			now := time.Now()
			if lastBackpressureSignal.IsZero() || now.Sub(lastBackpressureSignal) >= backpressureSignalCooldown {
				m.emitBackpressure(run.ID, len(readyQueue), currentParallelism)
				lastBackpressureSignal = now
			}
		}

		for len(readyQueue) > 0 && len(running) < currentParallelism {
			taskID := readyQueue[0]
			task := tasksByID[taskID]

			if supervised && task.Agent == "integrator" {
				if m.pauseAtSupervisionCheckpoint(run.ID, ctrl, supervisionSeen, "before-integrator", "waiting for approval before integrator") {
					if m.isCancelled(ctrl) {
						m.cancelRunInternal(run.ID)
						return
					}
				}
			}

			dispatchStart := time.Now()
			readyQueue = readyQueue[1:]
			_ = m.store.UpdateTaskLifecycle(task.ID, contracts.TaskLifecycleReady)
			_, _ = m.bus.Publish(contracts.Event{
				RunID:  run.ID,
				Type:   "task.ready",
				Source: "scheduler",
				Payload: map[string]string{
					"task_id": task.ID,
					"task":    task.Name,
					"agent":   task.Agent,
				},
			})
			m.observeDispatchLatency(time.Since(dispatchStart))

			running[taskID] = struct{}{}
			go func(current contracts.Task) {
				outcome := m.executeTaskWithRetries(ctrl.ctx, run, current, &outputsMu, outputs)
				resultCh <- outcome
			}(task)
		}

		if len(running) == 0 {
			if len(readyQueue) == 0 {
				if run.SchedulerMode == contracts.SchedulerModeDAGParallelV1 && !run.FallbackTriggered {
					currentParallelism = m.triggerSerialFallback(&run, "dag_invariant_no_runnable_tasks")
					continue
				}
				m.failRun(run.ID, "dag invariant violation: no runnable tasks")
				return
			}
		}

		outcome := <-resultCh
		delete(running, outcome.Task.ID)

		if outcome.Err != nil {
			m.failRun(run.ID, outcome.Err.Error())
			return
		}
		if !outcome.Success {
			m.failRun(run.ID, outcome.Result.Error)
			return
		}

		outputsMu.Lock()
		outputs[outcome.Task.Name+"_output"] = outcome.Result.Summary
		outputsMu.Unlock()

		if supervised {
			if outcome.Task.Name == "planner" {
				if m.pauseAtSupervisionCheckpoint(run.ID, ctrl, supervisionSeen, "after-planner", "planner completed") {
					if m.isCancelled(ctrl) {
						m.cancelRunInternal(run.ID)
						return
					}
				}
			}

			if strings.HasPrefix(outcome.Task.Name, "reviewer:") {
				reviewerCompleted++
				if reviewerTotal > 0 && reviewerCompleted == reviewerTotal {
					if m.pauseAtSupervisionCheckpoint(run.ID, ctrl, supervisionSeen, "after-work-package-block", "all reviewer tasks completed") {
						if m.isCancelled(ctrl) {
							m.cancelRunInternal(run.ID)
							return
						}
					}
				}
			}
		}

		completed++

		for _, depID := range dependents[outcome.Task.ID] {
			remainingDeps[depID]--
			if remainingDeps[depID] == 0 {
				readyQueue = append(readyQueue, depID)
			}
		}
	}

	_ = m.store.UpdateRunStatus(run.ID, contracts.RunCompleted, "")
	_, _ = m.bus.Publish(contracts.Event{RunID: run.ID, Type: "run.completed", Source: "scheduler"})
	m.metrics.runsCompleted.Add(1)
	m.cleanupRun(run.ID)
}

func (m *RunManager) pauseAtSupervisionCheckpoint(runID string, ctrl *runControl, seen map[string]struct{}, checkpoint, reason string) bool {
	if _, ok := seen[checkpoint]; ok {
		return false
	}
	seen[checkpoint] = struct{}{}

	_, _ = m.bus.Publish(contracts.Event{
		RunID:  runID,
		Type:   "run.supervision.checkpoint",
		Source: "scheduler",
		Payload: map[string]string{
			"checkpoint": checkpoint,
			"reason":     reason,
		},
	})

	if err := m.PauseRun(runID); err != nil {
		return false
	}

	m.waitIfPaused(ctrl)
	return true
}

func (m *RunManager) executeTaskWithRetries(
	ctx context.Context,
	run contracts.Run,
	task contracts.Task,
	outputsMu *sync.RWMutex,
	outputs map[string]string,
) taskOutcome {
	maxAttempts := task.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 3
	}

	var last contracts.AgentResult
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if attempt > 1 {
			_ = m.store.UpdateTaskLifecycle(task.ID, contracts.TaskLifecycleRetrying)
		}

		res, err := m.executeTaskOnce(ctx, run, task, outputsMu, outputs)
		if err != nil {
			return taskOutcome{Task: task, Err: err, Success: false}
		}
		last = res
		if res.Success {
			return taskOutcome{Task: task, Result: res, Success: true}
		}

		if (task.Agent == "tester" || task.Agent == "reviewer") && attempt < maxAttempts {
			backoff := retryBackoff(attempt)
			m.metrics.retriesScheduled.Add(1)
			_, _ = m.bus.Publish(contracts.Event{
				RunID:  run.ID,
				Type:   "task.retry.scheduled",
				Source: "scheduler",
				Payload: map[string]string{
					"task_id":         task.ID,
					"task":            task.Name,
					"attempt":         fmt.Sprintf("%d", attempt+1),
					"backoff_seconds": fmt.Sprintf("%d", int(backoff.Seconds())),
				},
			})

			fixTask, err := m.store.CreateTaskWithOptions(run.ID, store.CreateTaskOptions{
				Name:           fmt.Sprintf("fixer:%s:a%d", task.Name, attempt),
				Agent:          "fixer",
				DependsOn:      []string{task.ID},
				Priority:       task.Priority,
				MaxAttempts:    1,
				LifecycleState: contracts.TaskLifecycleReady,
			})
			if err != nil {
				return taskOutcome{Task: task, Err: fmt.Errorf("create fixer task: %w", err), Success: false}
			}
			fixResult, fixErr := m.executeTaskOnce(ctx, run, fixTask, outputsMu, outputs)
			if fixErr != nil {
				return taskOutcome{Task: task, Err: fixErr, Success: false}
			}
			if !fixResult.Success {
				return taskOutcome{Task: task, Result: fixResult, Success: false}
			}

			select {
			case <-ctx.Done():
				return taskOutcome{Task: task, Err: ctx.Err(), Success: false}
			case <-time.After(backoff):
			}
			continue
		}

		return taskOutcome{Task: task, Result: res, Success: false}
	}

	return taskOutcome{Task: task, Result: last, Success: last.Success}
}

func retryBackoff(attempt int) time.Duration {
	switch attempt {
	case 1:
		return 2 * time.Second
	case 2:
		return 5 * time.Second
	default:
		return 15 * time.Second
	}
}

func (m *RunManager) executeTaskOnce(
	ctx context.Context,
	run contracts.Run,
	task contracts.Task,
	outputsMu *sync.RWMutex,
	outputs map[string]string,
) (contracts.AgentResult, error) {
	attemptNumber, err := m.store.IncrementTaskAttempt(task.ID)
	if err != nil {
		return contracts.AgentResult{}, err
	}

	if err := m.store.UpdateTaskStatus(task.ID, contracts.TaskRunning, ""); err != nil {
		return contracts.AgentResult{}, err
	}
	m.metrics.tasksStarted.Add(1)

	_, _ = m.bus.Publish(contracts.Event{
		RunID:  run.ID,
		Type:   "task.started",
		Source: "scheduler",
		Payload: map[string]string{
			"task_id": task.ID,
			"task":    task.Name,
			"agent":   task.Agent,
			"attempt": fmt.Sprintf("%d", attemptNumber),
		},
	})

	outputsMu.RLock()
	input := cloneMap(outputs)
	outputsMu.RUnlock()

	contextSources := contextindexer.Build(run, task, input)
	basePrompt := fmt.Sprintf("Project: %s\nGoal: %s\nTask: %s", run.ProjectID, run.Goal, task.Name)
	instructionBundle := basePrompt

	m.mu.RLock()
	skillsManager := m.skills
	m.mu.RUnlock()
	if skillsManager != nil {
		if projectDocs, err := skillsManager.ReadProjectDocs(); err == nil && strings.TrimSpace(projectDocs) != "" {
			instructionBundle = skillkit.MergeUserInstructionsWithProjectDocs(instructionBundle, projectDocs)
		}
		if len(run.Skills) > 0 {
			if selected, err := skillsManager.ResolveByNames(run.Skills); err == nil && len(selected) > 0 {
				rendered := skillsManager.Render(selected)
				if strings.TrimSpace(rendered) != "" {
					instructionBundle = strings.TrimSpace(instructionBundle) + "\n\n" + strings.TrimSpace(rendered)
				}
			}
		}
	}

	compiledPrompt := promptcompiler.Compile(instructionBundle, contextSources, run.ContextBudget)
	input["compiled_prompt_bytes"] = fmt.Sprintf("%d", compiledPrompt.UsedBytes)
	input["compiled_prompt_sources"] = strings.Join(compiledPrompt.SelectedSources, ",")

	_, _ = m.bus.Publish(contracts.Event{
		RunID:  run.ID,
		Type:   "context.compiled",
		Source: "scheduler",
		Payload: map[string]string{
			"task_id":          task.ID,
			"task":             task.Name,
			"context_budget":   fmt.Sprintf("%d", compiledPrompt.Budget),
			"used_bytes":       fmt.Sprintf("%d", compiledPrompt.UsedBytes),
			"selected_sources": strings.Join(compiledPrompt.SelectedSources, ","),
		},
	})

	started := time.Now().UTC()
	result, sendErr := m.actors.Send(ctx, task.Agent, contracts.AgentEnvelope{
		RunID:              run.ID,
		TaskID:             task.ID,
		TaskName:           task.Name,
		ProjectID:          run.ProjectID,
		Goal:               run.Goal,
		Prompt:             compiledPrompt.Prompt,
		Attempt:            attemptNumber,
		Skills:             append([]string{}, run.Skills...),
		SkillSelectionMode: run.SkillSelectionMode,
		ModelPreference:    run.ModelPreference,
		FallbackChain:      append([]string{}, run.FallbackChain...),
		ModelID:            run.ModelID,
		ContextBudget:      run.ContextBudget,
		BlueprintProfile:   run.BlueprintProfile,
		BlueprintModules:   append([]string{}, run.BlueprintModules...),
		Bootstrap:          run.Bootstrap,
		Input:              input,
	}, m.agentTimeoutFor(task.Agent))
	finished := time.Now().UTC()

	if sendErr != nil {
		redactedError := m.redact(sendErr.Error())
		_ = m.store.UpdateTaskStatus(task.ID, contracts.TaskFailed, redactedError)
		_, _ = m.store.CreateAttempt(store.CreateAttemptOptions{
			RunID:      run.ID,
			TaskID:     task.ID,
			Agent:      task.Agent,
			Status:     "failed",
			Log:        "",
			Error:      redactedError,
			Provider:   "",
			ModelID:    "",
			StartedAt:  started,
			FinishedAt: finished,
		})
		m.metrics.tasksFailed.Add(1)
		_, _ = m.bus.Publish(contracts.Event{
			RunID:  run.ID,
			Type:   "task.failed",
			Source: task.Agent,
			Payload: map[string]string{
				"task_id": task.ID,
				"error":   redactedError,
			},
		})
		return contracts.AgentResult{}, fmt.Errorf("%s", redactedError)
	}

	redactedSummary := m.redact(result.Summary)
	redactedError := m.redact(result.Error)
	result.Summary = redactedSummary
	result.Error = redactedError

	status := "completed"
	taskState := contracts.TaskCompleted
	errMessage := ""
	if !result.Success {
		status = "failed"
		taskState = contracts.TaskFailed
		errMessage = result.Error
	}

	if err := m.store.UpdateTaskStatus(task.ID, taskState, errMessage); err != nil {
		return contracts.AgentResult{}, err
	}
	_, _ = m.store.CreateAttempt(store.CreateAttemptOptions{
		RunID:         run.ID,
		TaskID:        task.ID,
		Agent:         task.Agent,
		Status:        status,
		Log:           result.Summary,
		Error:         result.Error,
		Provider:      result.Provider,
		ModelID:       result.ModelID,
		FallbackIndex: providerFallbackIndex(result.ProviderTrail),
		LatencyMs:     result.LatencyMs,
		TokensIn:      result.TokensIn,
		TokensOut:     result.TokensOut,
		ProviderTrail: append([]contracts.ProviderAttempt{}, result.ProviderTrail...),
		StartedAt:     started,
		FinishedAt:    finished,
	})

	eventType := "task.completed"
	if !result.Success {
		eventType = "task.failed"
		m.metrics.tasksFailed.Add(1)
	} else {
		m.metrics.tasksCompleted.Add(1)
	}

	_, _ = m.bus.Publish(contracts.Event{
		RunID:  run.ID,
		Type:   eventType,
		Source: task.Agent,
		Payload: map[string]string{
			"task_id":  task.ID,
			"task":     task.Name,
			"agent":    task.Agent,
			"summary":  result.Summary,
			"error":    result.Error,
			"provider": result.Provider,
			"model_id": result.ModelID,
		},
	})

	if len(result.Artifacts) > 0 {
		for _, artifact := range result.Artifacts {
			_, _ = m.bus.Publish(contracts.Event{
				RunID:  run.ID,
				Type:   "diff.produced",
				Source: task.Agent,
				Payload: map[string]string{
					"task_id":       task.ID,
					"artifact_path": m.redact(artifact.Path),
					"artifact_type": artifact.Type,
				},
			})
		}
	}

	return result, nil
}

func (m *RunManager) emitBackpressure(runID string, queueSize, parallelism int) {
	m.metrics.backpressureSignals.Add(1)
	_, _ = m.bus.Publish(contracts.Event{
		RunID:  runID,
		Type:   "run.backpressure",
		Source: "scheduler",
		Payload: map[string]string{
			"queue_size":      fmt.Sprintf("%d", queueSize),
			"max_parallelism": fmt.Sprintf("%d", parallelism),
		},
	})
}

func (m *RunManager) observeDispatchLatency(latency time.Duration) {
	ms := latency.Milliseconds()
	if ms < 0 {
		ms = 0
	}

	m.metrics.dispatchCount.Add(1)
	m.metrics.dispatchSumMs.Add(ms)

	for {
		currentMax := m.metrics.dispatchMaxMs.Load()
		if ms <= currentMax {
			break
		}
		if m.metrics.dispatchMaxMs.CompareAndSwap(currentMax, ms) {
			break
		}
	}

	switch {
	case ms <= 25:
		m.metrics.dispatchLe25Ms.Add(1)
	case ms <= 50:
		m.metrics.dispatchLe50Ms.Add(1)
	case ms <= 100:
		m.metrics.dispatchLe100Ms.Add(1)
	case ms <= 250:
		m.metrics.dispatchLe250Ms.Add(1)
	case ms <= 500:
		m.metrics.dispatchLe500Ms.Add(1)
	case ms <= 1000:
		m.metrics.dispatchLe1000Ms.Add(1)
	case ms <= 2000:
		m.metrics.dispatchLe2000Ms.Add(1)
	default:
		m.metrics.dispatchGt2000Ms.Add(1)
	}
}

func (m *RunManager) observeReadyQueueDepth(depth int) {
	if depth < 0 {
		depth = 0
	}

	value := int64(depth)
	m.metrics.readyQueueDepth.Store(value)
	for {
		currentMax := m.metrics.readyQueueDepthMax.Load()
		if value <= currentMax {
			break
		}
		if m.metrics.readyQueueDepthMax.CompareAndSwap(currentMax, value) {
			break
		}
	}
}

func dispatchP95EstimateMs(
	count int64,
	le25 int64,
	le50 int64,
	le100 int64,
	le250 int64,
	le500 int64,
	le1000 int64,
	le2000 int64,
	max int64,
) int64 {
	if count <= 0 {
		return 0
	}
	threshold := (count*95 + 99) / 100
	cumulative := int64(0)
	buckets := []struct {
		limit int64
		count int64
	}{
		{limit: 25, count: le25},
		{limit: 50, count: le50},
		{limit: 100, count: le100},
		{limit: 250, count: le250},
		{limit: 500, count: le500},
		{limit: 1000, count: le1000},
		{limit: 2000, count: le2000},
	}
	for _, bucket := range buckets {
		cumulative += bucket.count
		if cumulative >= threshold {
			return bucket.limit
		}
	}
	return max
}

func (m *RunManager) triggerSerialFallback(run *contracts.Run, reason string) int {
	if run.FallbackTriggered {
		return 1
	}
	run.FallbackTriggered = true
	_ = m.store.SetRunFallbackTriggered(run.ID, true)
	m.metrics.fallbacksTriggered.Add(1)
	_, _ = m.bus.Publish(contracts.Event{
		RunID:  run.ID,
		Type:   "run.fallback.serial",
		Source: "scheduler",
		Payload: map[string]string{
			"reason": reason,
		},
	})
	return 1
}

func (m *RunManager) applyBlueprintForRun(run contracts.Run) error {
	if m.blueprintApplier == nil {
		return fmt.Errorf("blueprint applier is not configured")
	}
	if strings.TrimSpace(run.BlueprintProfile) == "" {
		return fmt.Errorf("blueprint profile is required when bootstrap is enabled")
	}

	_, _ = m.bus.Publish(contracts.Event{
		RunID:  run.ID,
		Type:   "blueprint.bootstrap.started",
		Source: "scheduler",
		Payload: map[string]string{
			"profile": run.BlueprintProfile,
		},
	})

	result, err := m.blueprintApplier.Apply(blueprints.ApplyOptions{
		ProjectID: run.ProjectID,
		ProfileID: run.BlueprintProfile,
		ModuleIDs: run.BlueprintModules,
	})
	if err != nil {
		return err
	}

	for _, moduleID := range result.AppliedModules {
		_, _ = m.bus.Publish(contracts.Event{
			RunID:  run.ID,
			Type:   "blueprint.module.applied",
			Source: "scheduler",
			Payload: map[string]string{
				"profile": run.BlueprintProfile,
				"module":  moduleID,
			},
		})
	}

	for _, moduleID := range result.SkippedModules {
		_, _ = m.bus.Publish(contracts.Event{
			RunID:  run.ID,
			Type:   "blueprint.module.skipped",
			Source: "scheduler",
			Payload: map[string]string{
				"profile": run.BlueprintProfile,
				"module":  moduleID,
			},
		})
	}

	payload := map[string]string{
		"profile":      run.BlueprintProfile,
		"project_path": result.ProjectPath,
	}
	if len(result.ChangedFiles) > 0 {
		payload["changed_files"] = strings.Join(result.ChangedFiles, ",")
	}

	_, _ = m.bus.Publish(contracts.Event{RunID: run.ID, Type: "blueprint.bootstrap.completed", Source: "scheduler", Payload: payload})
	return nil
}

func (m *RunManager) PauseRun(runID string) error {
	m.mu.RLock()
	ctrl, ok := m.controls[runID]
	m.mu.RUnlock()
	if !ok {
		run, err := m.store.GetRun(runID)
		if err != nil {
			return fmt.Errorf("run %s not found", runID)
		}
		switch run.Status {
		case contracts.RunPaused, contracts.RunCancelled, contracts.RunCompleted, contracts.RunFailed:
			return nil
		default:
			return fmt.Errorf("run %s not active", runID)
		}
	}

	ctrl.mu.Lock()
	ctrl.paused = true
	ctrl.mu.Unlock()

	if err := m.store.UpdateRunStatus(runID, contracts.RunPaused, ""); err != nil {
		return err
	}
	_, _ = m.bus.Publish(contracts.Event{RunID: runID, Type: "run.paused", Source: "scheduler"})
	return nil
}

func (m *RunManager) ResumeRun(runID string) error {
	m.mu.RLock()
	ctrl, ok := m.controls[runID]
	m.mu.RUnlock()
	if !ok {
		run, err := m.store.GetRun(runID)
		if err != nil {
			return fmt.Errorf("run %s not found", runID)
		}
		switch run.Status {
		case contracts.RunRunning, contracts.RunPaused, contracts.RunCancelled, contracts.RunCompleted, contracts.RunFailed:
			return nil
		default:
			return fmt.Errorf("run %s not active", runID)
		}
	}

	ctrl.mu.Lock()
	ctrl.paused = false
	ctrl.mu.Unlock()
	ctrl.cond.Broadcast()

	if err := m.store.UpdateRunStatus(runID, contracts.RunRunning, ""); err != nil {
		return err
	}
	_, _ = m.bus.Publish(contracts.Event{RunID: runID, Type: "run.resumed", Source: "scheduler"})
	return nil
}

func (m *RunManager) CancelRun(runID string) error {
	m.mu.RLock()
	ctrl, ok := m.controls[runID]
	m.mu.RUnlock()
	if !ok {
		run, err := m.store.GetRun(runID)
		if err != nil {
			return fmt.Errorf("run %s not found", runID)
		}
		switch run.Status {
		case contracts.RunCancelled, contracts.RunCompleted, contracts.RunFailed:
			return nil
		default:
			return fmt.Errorf("run %s not active", runID)
		}
	}
	ctrl.cancel()
	m.cancelRunInternal(runID)
	return nil
}

func (m *RunManager) GetRun(runID string) (contracts.Run, error) {
	return m.store.GetRun(runID)
}

func (m *RunManager) ListRunTasks(runID string) ([]contracts.Task, error) {
	return m.store.ListTasksByRun(runID)
}

func (m *RunManager) ListRunAttempts(runID string) ([]contracts.TaskAttempt, error) {
	return m.store.ListAttemptsByRun(runID)
}

func (m *RunManager) GetRunGraph(runID string) (contracts.TaskGraph, error) {
	graph, err := m.store.GetRunGraph(runID)
	if err != nil {
		return contracts.TaskGraph{}, err
	}
	return graph, nil
}

func (m *RunManager) ListRecentRuns(limit int) ([]contracts.Run, error) {
	return m.store.ListRuns(limit)
}

func (m *RunManager) ListBlueprintProfiles() ([]blueprints.ProfileSummary, error) {
	if m.blueprintRegistry == nil {
		return nil, fmt.Errorf("blueprint catalog is not configured")
	}
	return m.blueprintRegistry.ListProfiles(), nil
}

func (m *RunManager) GetBlueprintProfile(profileID string) (blueprints.ProfileSummary, error) {
	if m.blueprintRegistry == nil {
		return blueprints.ProfileSummary{}, fmt.Errorf("blueprint catalog is not configured")
	}
	profile, err := m.blueprintRegistry.GetProfile(profileID)
	if err != nil {
		return blueprints.ProfileSummary{}, err
	}
	out := blueprints.ProfileSummary{
		ID:          profile.ID,
		Name:        profile.Name,
		Version:     profile.Version,
		Description: profile.Description,
		Modules:     make([]blueprints.ModuleSummary, 0, len(profile.Modules)),
	}
	for _, module := range profile.Modules {
		out.Modules = append(out.Modules, blueprints.ModuleSummary{ID: module.ID, Name: module.Name, Description: module.Description})
	}
	return out, nil
}

func (m *RunManager) BootstrapProject(projectID, profileID string, moduleIDs []string) (blueprints.ApplyResult, error) {
	if m.blueprintApplier == nil {
		return blueprints.ApplyResult{}, fmt.Errorf("blueprint applier is not configured")
	}
	resolvedProfile, resolvedModules, err := m.prepareBlueprintSelection(profileID, moduleIDs, true)
	if err != nil {
		return blueprints.ApplyResult{}, err
	}

	return m.blueprintApplier.Apply(blueprints.ApplyOptions{
		ProjectID: projectID,
		ProfileID: resolvedProfile,
		ModuleIDs: resolvedModules,
	})
}

func (m *RunManager) Readiness() map[string]interface{} {
	ready := true
	dbStatus := "ok"
	if err := m.store.Ping(); err != nil {
		ready = false
		dbStatus = err.Error()
	}

	actorCount := len(m.actors.Actors())
	if actorCount == 0 {
		ready = false
	}

	codexReady := false
	codexStatus := contracts.CodexAuthStatus{}
	if status, err := m.CodexAuthStatus(context.Background()); err == nil {
		codexStatus = status
		codexReady = status.Ready && status.LoggedIn
	}

	modelCatalog := m.ModelsCatalog(context.Background())
	providerSummary := map[string]interface{}{
		"default_primary": modelCatalog.DefaultPrimary,
		"default_chain":   append([]string{}, modelCatalog.DefaultChain...),
		"providers_total": len(modelCatalog.Providers),
	}
	available := 0
	for _, provider := range modelCatalog.Providers {
		if provider.Available {
			available++
		}
	}
	providerSummary["providers_available"] = available

	readiness := map[string]interface{}{
		"ready":              ready,
		"db":                 dbStatus,
		"actors":             actorCount,
		"scheduler":          "ok",
		"opencode_available": opencodeexec.IsAvailable(),
		"codex_auth_ready":   codexReady,
		"provider_status":    providerSummary,
	}
	skillsLoaded := 0
	skillsErrors := 0
	skillsReady := false
	m.mu.RLock()
	skillsManager := m.skills
	m.mu.RUnlock()
	if skillsManager != nil {
		skillsLoaded, skillsErrors, skillsReady = skillsManager.Stats()
	}
	readiness["skills_loaded"] = skillsLoaded
	readiness["skills_errors"] = skillsErrors
	readiness["skills_system_ready"] = skillsReady
	if codexStatus.LastChecked.Unix() > 0 {
		readiness["codex_auth_status"] = codexStatus
	}
	if status := onboarding.ReadLastStatus(m.workspace); status != "" {
		readiness["onboard_last_status"] = status
	}
	return readiness
}

func (m *RunManager) MetricsSnapshot() map[string]int64 {
	dispatchCount := m.metrics.dispatchCount.Load()
	dispatchSumMs := m.metrics.dispatchSumMs.Load()
	dispatchMaxMs := m.metrics.dispatchMaxMs.Load()
	dispatchLe25Ms := m.metrics.dispatchLe25Ms.Load()
	dispatchLe50Ms := m.metrics.dispatchLe50Ms.Load()
	dispatchLe100Ms := m.metrics.dispatchLe100Ms.Load()
	dispatchLe250Ms := m.metrics.dispatchLe250Ms.Load()
	dispatchLe500Ms := m.metrics.dispatchLe500Ms.Load()
	dispatchLe1000Ms := m.metrics.dispatchLe1000Ms.Load()
	dispatchLe2000Ms := m.metrics.dispatchLe2000Ms.Load()
	dispatchGt2000Ms := m.metrics.dispatchGt2000Ms.Load()
	dispatchP95 := dispatchP95EstimateMs(
		dispatchCount,
		dispatchLe25Ms,
		dispatchLe50Ms,
		dispatchLe100Ms,
		dispatchLe250Ms,
		dispatchLe500Ms,
		dispatchLe1000Ms,
		dispatchLe2000Ms,
		dispatchMaxMs,
	)

	snapshot := map[string]int64{
		"runs_started":                            m.metrics.runsStarted.Load(),
		"runs_completed":                          m.metrics.runsCompleted.Load(),
		"runs_failed":                             m.metrics.runsFailed.Load(),
		"tasks_started":                           m.metrics.tasksStarted.Load(),
		"tasks_completed":                         m.metrics.tasksCompleted.Load(),
		"tasks_failed":                            m.metrics.tasksFailed.Load(),
		"retries_scheduled":                       m.metrics.retriesScheduled.Load(),
		"fallbacks_triggered":                     m.metrics.fallbacksTriggered.Load(),
		"backpressure_signals":                    m.metrics.backpressureSignals.Load(),
		"scheduler_ready_queue_depth":             m.metrics.readyQueueDepth.Load(),
		"scheduler_ready_queue_depth_max":         m.metrics.readyQueueDepthMax.Load(),
		"task_dispatch_latency_count":             dispatchCount,
		"task_dispatch_latency_sum_ms":            dispatchSumMs,
		"task_dispatch_latency_max_ms":            dispatchMaxMs,
		"task_dispatch_latency_p95_estimate_ms":   dispatchP95,
		"task_dispatch_latency_bucket_le_25_ms":   dispatchLe25Ms,
		"task_dispatch_latency_bucket_le_50_ms":   dispatchLe50Ms,
		"task_dispatch_latency_bucket_le_100_ms":  dispatchLe100Ms,
		"task_dispatch_latency_bucket_le_250_ms":  dispatchLe250Ms,
		"task_dispatch_latency_bucket_le_500_ms":  dispatchLe500Ms,
		"task_dispatch_latency_bucket_le_1000_ms": dispatchLe1000Ms,
		"task_dispatch_latency_bucket_le_2000_ms": dispatchLe2000Ms,
		"task_dispatch_latency_bucket_gt_2000_ms": dispatchGt2000Ms,
	}

	if m.bus != nil {
		for key, value := range m.bus.MetricsSnapshot() {
			snapshot[key] = value
		}
	}
	m.mu.RLock()
	modelProvider := m.modelProvider
	m.mu.RUnlock()
	if modelProvider != nil {
		for key, value := range modelProvider.MetricsSnapshot() {
			snapshot[key] = value
		}
	}

	return snapshot
}

func (m *RunManager) ListProjects() ([]map[string]string, error) {
	entries, err := os.ReadDir(m.workspace)
	if err != nil {
		return nil, err
	}

	projects := make([]map[string]string, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if name == "archive" || name == "node_modules" || name == "bin" || name == "logs" || name == "outputs" || name == "inputs" {
			continue
		}

		projectPath := filepath.Join(m.workspace, name)
		if !looksLikeProject(projectPath) {
			continue
		}
		projects = append(projects, map[string]string{"id": name, "name": name})
	}

	sort.Slice(projects, func(i, j int) bool {
		return projects[i]["id"] < projects[j]["id"]
	})

	if len(projects) == 0 {
		projects = append(projects, map[string]string{"id": "biometrics", "name": "biometrics"})
	}

	return projects, nil
}

func (m *RunManager) ReadFile(path string) ([]byte, error) {
	rootAbs, err := filepath.Abs(m.workspace)
	if err != nil {
		return nil, err
	}
	cleanTarget := filepath.Clean(strings.TrimSpace(path))
	if cleanTarget == "." || cleanTarget == "" {
		return nil, fmt.Errorf("path is required")
	}
	if filepath.IsAbs(cleanTarget) {
		return nil, fmt.Errorf("absolute paths blocked")
	}
	validated := filepath.Join(rootAbs, cleanTarget)
	if rel, err := filepath.Rel(rootAbs, validated); err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return nil, fmt.Errorf("path traversal blocked")
	}

	rootResolved, err := filepath.EvalSymlinks(rootAbs)
	if err == nil {
		resolved, err := filepath.EvalSymlinks(validated)
		if err == nil {
			if rel, err := filepath.Rel(rootResolved, resolved); err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				return nil, fmt.Errorf("path traversal blocked")
			}
		}
	}

	return os.ReadFile(validated)
}

func (m *RunManager) ListDir(path string) ([]map[string]interface{}, error) {
	rootAbs, err := filepath.Abs(m.workspace)
	if err != nil {
		return nil, err
	}
	cleanTarget := filepath.Clean(strings.TrimSpace(path))
	if cleanTarget == "" {
		cleanTarget = "."
	}
	if filepath.IsAbs(cleanTarget) {
		return nil, fmt.Errorf("absolute paths blocked")
	}
	validated := filepath.Join(rootAbs, cleanTarget)
	if rel, err := filepath.Rel(rootAbs, validated); err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return nil, fmt.Errorf("path traversal blocked")
	}

	rootResolved, err := filepath.EvalSymlinks(rootAbs)
	if err == nil {
		resolved, err := filepath.EvalSymlinks(validated)
		if err == nil {
			if rel, err := filepath.Rel(rootResolved, resolved); err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				return nil, fmt.Errorf("path traversal blocked")
			}
		}
	}

	entries, err := os.ReadDir(validated)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]interface{}, 0, len(entries))
	for _, entry := range entries {
		info, _ := entry.Info()
		rel, _ := filepath.Rel(m.workspace, filepath.Join(validated, entry.Name()))
		out = append(out, map[string]interface{}{
			"name":  entry.Name(),
			"path":  rel,
			"isDir": entry.IsDir(),
			"size":  info.Size(),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		iDir := out[i]["isDir"].(bool)
		jDir := out[j]["isDir"].(bool)
		if iDir != jDir {
			return iDir
		}
		return out[i]["name"].(string) < out[j]["name"].(string)
	})
	return out, nil
}

func (m *RunManager) Events(runID string, limit int) ([]contracts.Event, error) {
	return m.bus.Replay(runID, limit)
}

func (m *RunManager) waitIfPaused(ctrl *runControl) {
	ctrl.mu.Lock()
	defer ctrl.mu.Unlock()
	for ctrl.paused {
		ctrl.cond.Wait()
	}
}

func (m *RunManager) isCancelled(ctrl *runControl) bool {
	select {
	case <-ctrl.ctx.Done():
		return true
	default:
		return false
	}
}

func (m *RunManager) cancelRunInternal(runID string) {
	_ = m.store.UpdateRunStatus(runID, contracts.RunCancelled, "run cancelled")
	_, _ = m.bus.Publish(contracts.Event{RunID: runID, Type: "run.cancelled", Source: "scheduler"})
	m.cleanupRun(runID)
}

func (m *RunManager) failRun(runID, message string) {
	redacted := m.redact(message)
	_ = m.store.UpdateRunStatus(runID, contracts.RunFailed, redacted)
	_, _ = m.bus.Publish(contracts.Event{
		RunID:  runID,
		Type:   "run.failed",
		Source: "scheduler",
		Payload: map[string]string{
			"error": redacted,
		},
	})
	m.metrics.runsFailed.Add(1)
	m.cleanupRun(runID)
}

func (m *RunManager) cleanupRun(runID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if ctrl, ok := m.controls[runID]; ok {
		ctrl.cancel()
		delete(m.controls, runID)
	}
}

func cloneMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func (m *RunManager) redact(value string) string {
	if m.policy == nil {
		return value
	}
	return m.policy.Redact(value)
}

func normalizeModuleIDs(ids []string) []string {
	if len(ids) == 0 {
		return []string{}
	}
	seen := make(map[string]struct{}, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		trimmed := strings.TrimSpace(id)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	sort.Strings(out)
	return out
}

func normalizeProviderIDs(ids []string) []string {
	if len(ids) == 0 {
		return []string{}
	}
	seen := make(map[string]struct{}, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		normalized := strings.ToLower(strings.TrimSpace(id))
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	return out
}

func providerFallbackIndex(trail []contracts.ProviderAttempt) int {
	if len(trail) == 0 {
		return 0
	}
	return trail[len(trail)-1].FallbackIndex
}

func loadAgentTimeoutConfig() agentTimeoutConfig {
	defaultTimeout := timeoutFromEnvSeconds(
		"BIOMETRICS_AGENT_TIMEOUT_DEFAULT_SECONDS",
		defaultAgentTimeoutSeconds,
	)
	cfg := agentTimeoutConfig{
		defaultTimeout: defaultTimeout,
		overrides:      make(map[string]time.Duration, 2),
	}

	coderTimeout := timeoutFromEnvSeconds("BIOMETRICS_AGENT_TIMEOUT_CODER_SECONDS", defaultCoderTimeoutSeconds)
	if coderTimeout > 0 {
		cfg.overrides["coder"] = coderTimeout
	}

	fixerTimeout := timeoutFromEnvSeconds("BIOMETRICS_AGENT_TIMEOUT_FIXER_SECONDS", defaultFixerTimeoutSeconds)
	if fixerTimeout > 0 {
		cfg.overrides["fixer"] = fixerTimeout
	}

	return cfg
}

func timeoutFromEnvSeconds(key string, fallbackSeconds int) time.Duration {
	trimmed := strings.TrimSpace(os.Getenv(key))
	if trimmed == "" {
		return time.Duration(fallbackSeconds) * time.Second
	}
	parsed, err := strconv.Atoi(trimmed)
	if err != nil || parsed <= 0 {
		return time.Duration(fallbackSeconds) * time.Second
	}
	return time.Duration(parsed) * time.Second
}

func (m *RunManager) agentTimeoutFor(agentName string) time.Duration {
	normalized := strings.ToLower(strings.TrimSpace(agentName))
	if timeout, ok := m.agentTimeouts.overrides[normalized]; ok && timeout > 0 {
		return timeout
	}
	if m.agentTimeouts.defaultTimeout > 0 {
		return m.agentTimeouts.defaultTimeout
	}
	return time.Duration(defaultAgentTimeoutSeconds) * time.Second
}

func looksLikeProject(path string) bool {
	markers := []string{"go.mod", "package.json", ".git"}
	for _, marker := range markers {
		if _, err := os.Stat(filepath.Join(path, marker)); err == nil {
			return true
		}
	}
	return false
}
