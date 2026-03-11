package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"biometrics-cli/internal/blueprints"
	"biometrics-cli/internal/contracts"
	"biometrics-cli/internal/llm"
	"biometrics-cli/internal/optimizer"
	"biometrics-cli/internal/policy"
	"biometrics-cli/internal/runtime/actor"
	"biometrics-cli/internal/runtime/background"
	"biometrics-cli/internal/runtime/bus"
	runtimeorchestrator "biometrics-cli/internal/runtime/orchestrator"
	"biometrics-cli/internal/runtime/scheduler"
	"biometrics-cli/internal/runtime/supervisor"
	"biometrics-cli/internal/skillkit"
	store "biometrics-cli/internal/store/sqlite"
)

type fakeLLMGateway struct {
	execute func(context.Context, llm.Request) (llm.Response, error)
}

func (f fakeLLMGateway) Execute(ctx context.Context, req llm.Request) (llm.Response, error) {
	if f.execute == nil {
		return llm.Response{}, fmt.Errorf("execute not implemented")
	}
	return f.execute(ctx, req)
}

func (f fakeLLMGateway) ModelCatalog(_ context.Context) contracts.ModelCatalog {
	return contracts.ModelCatalog{
		DefaultPrimary: "codex",
		DefaultChain:   []string{"gemini", "nim"},
		Providers: []contracts.ModelProvider{
			{ID: "codex", Name: "OpenAI Codex", Status: "ready", Default: true, Available: true},
			{ID: "gemini", Name: "Google Gemini", Status: "ready", Default: false, Available: true},
			{ID: "nim", Name: "NVIDIA NIM", Status: "ready", Default: false, Available: true},
		},
	}
}

func (f fakeLLMGateway) MetricsSnapshot() map[string]int64 {
	return map[string]int64{}
}

func setupTestServer(t *testing.T) (*Server, *scheduler.RunManager, context.CancelFunc, string) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	tmp := t.TempDir()
	mustWriteFile(t, filepath.Join(tmp, "biometrics", "go.mod"), "module biometrics-test\n")
	writeBlueprintFixtures(t, tmp)
	writeSkillFixtures(t, tmp)

	db, err := store.New(filepath.Join(tmp, "test.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

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
		if err := actors.Register(name, 8, handler(name)); err != nil {
			t.Fatalf("register actor %s: %v", name, err)
		}
	}
	actors.Start(ctx)

	registry, err := blueprints.NewRegistry(tmp, "")
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	applier := blueprints.NewApplier(tmp, registry)

	manager := scheduler.NewRunManager(db, actors, eventBus, policy.Default(), tmp, registry, applier)
	skillManager, err := skillkit.NewManager(skillkit.ManagerOptions{
		Workspace: tmp,
		CWD:       tmp,
	})
	if err != nil {
		t.Fatalf("skill manager: %v", err)
	}
	manager.SetSkillManager(skillManager)
	server := NewServer(manager, eventBus)
	return server, manager, cancel, tmp
}

func writeSkillFixtures(t *testing.T, workspace string) {
	t.Helper()
	mustWriteFile(t, filepath.Join(workspace, ".codex", "skills", "release-ops", "SKILL.md"), `---
name: release-ops
description: Release operational checks and reproducible gate execution. Use when release gates, soak evidence, or cutover checks are requested.
---

# release-ops
run release checks.
`)
}

func TestCreateRunAndFetchTasks(t *testing.T) {
	server, manager, _, _ := setupTestServer(t)

	body := map[string]string{
		"project_id": "biometrics",
		"goal":       "build a feature",
		"mode":       "autonomous",
	}
	raw, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}

	var run contracts.Run
	if err := json.NewDecoder(w.Body).Decode(&run); err != nil {
		t.Fatalf("decode run: %v", err)
	}
	if run.ID == "" {
		t.Fatal("run ID must not be empty")
	}

	taskReq := httptest.NewRequest(http.MethodGet, "/api/v1/runs/"+run.ID+"/tasks", nil)
	taskW := httptest.NewRecorder()
	server.Handler().ServeHTTP(taskW, taskReq)
	if taskW.Code != http.StatusOK {
		t.Fatalf("expected 200 for tasks, got %d", taskW.Code)
	}

	var tasks []contracts.Task
	if err := json.NewDecoder(taskW.Body).Decode(&tasks); err != nil {
		t.Fatalf("decode tasks: %v", err)
	}
	if len(tasks) < 7 {
		t.Fatalf("expected at least 7 tasks, got %d", len(tasks))
	}

	waitForRunTerminalState(t, manager, run.ID)
}

func TestCreateRunWithSchedulerOptions(t *testing.T) {
	server, _, _, _ := setupTestServer(t)

	raw := []byte(`{"project_id":"biometrics","goal":"serial scheduling","mode":"autonomous","scheduler_mode":"serial","max_parallelism":12}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}

	var run contracts.Run
	if err := json.NewDecoder(w.Body).Decode(&run); err != nil {
		t.Fatalf("decode run: %v", err)
	}
	if run.SchedulerMode != contracts.SchedulerModeSerial {
		t.Fatalf("expected scheduler mode serial, got %q", run.SchedulerMode)
	}
	if run.MaxParallelism != 1 {
		t.Fatalf("expected max_parallelism 1 in serial mode, got %d", run.MaxParallelism)
	}
}

func TestCreateRunWithModelRoutingOptions(t *testing.T) {
	server, _, _, _ := setupTestServer(t)

	raw := []byte(`{"project_id":"biometrics","goal":"routing options","mode":"autonomous","model_preference":"codex","fallback_chain":["gemini","nim"],"model_id":"codex-pro","context_budget":18000}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}

	var run contracts.Run
	if err := json.NewDecoder(w.Body).Decode(&run); err != nil {
		t.Fatalf("decode run: %v", err)
	}
	if run.ModelPreference != "codex" {
		t.Fatalf("expected model_preference codex, got %q", run.ModelPreference)
	}
	if run.ModelID != "codex-pro" {
		t.Fatalf("expected model_id codex-pro, got %q", run.ModelID)
	}
	if run.ContextBudget != 18000 {
		t.Fatalf("expected context_budget 18000, got %d", run.ContextBudget)
	}
	if len(run.FallbackChain) != 2 || run.FallbackChain[0] != "gemini" || run.FallbackChain[1] != "nim" {
		t.Fatalf("unexpected fallback_chain: %#v", run.FallbackChain)
	}
}

func TestCreateRunWithSkillsSelectionOptions(t *testing.T) {
	server, _, _, _ := setupTestServer(t)

	raw := []byte(`{"project_id":"biometrics","goal":"Use $release-ops to prepare gate reports","mode":"autonomous","skills":["release-ops"],"skill_selection_mode":"explicit"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", w.Code, w.Body.String())
	}

	var run contracts.Run
	if err := json.NewDecoder(w.Body).Decode(&run); err != nil {
		t.Fatalf("decode run: %v", err)
	}
	if run.SkillSelectionMode != "explicit" {
		t.Fatalf("expected skill_selection_mode explicit, got %q", run.SkillSelectionMode)
	}
	if len(run.Skills) != 1 || run.Skills[0] != "release-ops" {
		t.Fatalf("expected selected skill release-ops, got %#v", run.Skills)
	}
}

func TestSkillsEndpointsListGetAndToggle(t *testing.T) {
	server, manager, _, workspace := setupTestServer(t)

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/skills", nil)
	listW := httptest.NewRecorder()
	server.Handler().ServeHTTP(listW, listReq)
	if listW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", listW.Code)
	}
	var skills []map[string]interface{}
	if err := json.NewDecoder(listW.Body).Decode(&skills); err != nil {
		t.Fatalf("decode skills list: %v", err)
	}
	if len(skills) == 0 {
		t.Fatalf("expected at least one skill in list")
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/skills/release-ops", nil)
	getW := httptest.NewRecorder()
	server.Handler().ServeHTTP(getW, getReq)
	if getW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", getW.Code, getW.Body.String())
	}

	disableBody := bytes.NewBufferString(`{"name":"release-ops"}`)
	disableReq := httptest.NewRequest(http.MethodPost, "/api/v1/skills/disable", disableBody)
	disableReq.Header.Set("Content-Type", "application/json")
	disableW := httptest.NewRecorder()
	server.Handler().ServeHTTP(disableW, disableReq)
	if disableW.Code != http.StatusOK {
		t.Fatalf("expected 200 disable, got %d body=%s", disableW.Code, disableW.Body.String())
	}

	disabledSkill, err := manager.GetSkill("release-ops")
	if err != nil {
		t.Fatalf("get disabled skill: %v", err)
	}
	if disabledSkill.Enabled {
		t.Fatalf("expected skill to be disabled")
	}

	enableBody := bytes.NewBufferString(`{"name":"release-ops"}`)
	enableReq := httptest.NewRequest(http.MethodPost, "/api/v1/skills/enable", enableBody)
	enableReq.Header.Set("Content-Type", "application/json")
	enableW := httptest.NewRecorder()
	server.Handler().ServeHTTP(enableW, enableReq)
	if enableW.Code != http.StatusOK {
		t.Fatalf("expected 200 enable, got %d body=%s", enableW.Code, enableW.Body.String())
	}

	enabledSkill, err := manager.GetSkill("release-ops")
	if err != nil {
		t.Fatalf("get enabled skill: %v", err)
	}
	if !enabledSkill.Enabled {
		t.Fatalf("expected skill to be enabled")
	}

	reloadReq := httptest.NewRequest(http.MethodPost, "/api/v1/skills/reload", nil)
	reloadW := httptest.NewRecorder()
	server.Handler().ServeHTTP(reloadW, reloadReq)
	if reloadW.Code != http.StatusOK {
		t.Fatalf("expected 200 reload, got %d body=%s", reloadW.Code, reloadW.Body.String())
	}

	configPath := filepath.Join(workspace, ".codex", "config.toml")
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("expected skill config to be persisted at %s: %v", configPath, err)
	}
}

func TestModelsEndpoint(t *testing.T) {
	server, _, _, _ := setupTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/models", nil)
	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var catalog contracts.ModelCatalog
	if err := json.NewDecoder(w.Body).Decode(&catalog); err != nil {
		t.Fatalf("decode model catalog: %v", err)
	}
	if catalog.DefaultPrimary == "" {
		t.Fatalf("expected default primary provider")
	}
}

func TestBackgroundAgentsEndpointRequiresManager(t *testing.T) {
	server, _, _, _ := setupTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents/background", bytes.NewBufferString(`{"prompt":"draft release notes"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when background manager missing, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestBackgroundAgentsLifecycleEndpoints(t *testing.T) {
	server, _, _, _ := setupTestServer(t)
	server.SetBackgroundAgents(background.NewManager(fakeLLMGateway{
		execute: func(_ context.Context, req llm.Request) (llm.Response, error) {
			return llm.Response{
				Output:    "background task complete",
				Provider:  req.ModelPreference,
				ModelID:   req.ModelID,
				CreatedAt: time.Now().UTC(),
			}, nil
		},
	}, nil))

	createReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/agents/background",
		bytes.NewBufferString(`{"project_id":"biometrics","agent":"coder","prompt":"prepare patch notes","model_preference":"nim","model_id":"nvidia-nim/qwen-3.5-397b"}`),
	)
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	server.Handler().ServeHTTP(createW, createReq)
	if createW.Code != http.StatusCreated {
		t.Fatalf("expected 201 create, got %d body=%s", createW.Code, createW.Body.String())
	}

	var created background.Job
	if err := json.NewDecoder(createW.Body).Decode(&created); err != nil {
		t.Fatalf("decode created background job: %v", err)
	}
	if created.ID == "" {
		t.Fatalf("expected background job id")
	}

	var current background.Job
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		statusReq := httptest.NewRequest(http.MethodGet, "/api/v1/agents/background/"+created.ID, nil)
		statusW := httptest.NewRecorder()
		server.Handler().ServeHTTP(statusW, statusReq)
		if statusW.Code != http.StatusOK {
			t.Fatalf("expected 200 status, got %d body=%s", statusW.Code, statusW.Body.String())
		}
		if err := json.NewDecoder(statusW.Body).Decode(&current); err != nil {
			t.Fatalf("decode background status: %v", err)
		}
		if current.Status == background.StatusCompleted {
			break
		}
		time.Sleep(40 * time.Millisecond)
	}
	if current.Status != background.StatusCompleted {
		t.Fatalf("expected completed background job, got %q", current.Status)
	}
	if current.Output == "" {
		t.Fatalf("expected output to be set on completed background job")
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/agents/background", nil)
	listW := httptest.NewRecorder()
	server.Handler().ServeHTTP(listW, listReq)
	if listW.Code != http.StatusOK {
		t.Fatalf("expected 200 list, got %d body=%s", listW.Code, listW.Body.String())
	}
	var listed []background.Job
	if err := json.NewDecoder(listW.Body).Decode(&listed); err != nil {
		t.Fatalf("decode background list: %v", err)
	}
	if len(listed) == 0 {
		t.Fatalf("expected at least one background job in list")
	}
}

func TestCodexAuthEndpointsWithoutBroker(t *testing.T) {
	server, _, _, _ := setupTestServer(t)

	statusReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/codex/status", nil)
	statusW := httptest.NewRecorder()
	server.Handler().ServeHTTP(statusW, statusReq)
	if statusW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", statusW.Code)
	}
	var status contracts.CodexAuthStatus
	if err := json.NewDecoder(statusW.Body).Decode(&status); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	if status.Ready {
		t.Fatalf("expected codex auth ready=false without configured broker")
	}

	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/codex/login", nil)
	loginW := httptest.NewRecorder()
	server.Handler().ServeHTTP(loginW, loginReq)
	if loginW.Code != http.StatusBadGateway {
		t.Fatalf("expected 502 when broker missing, got %d", loginW.Code)
	}
}

func TestOptimizerEndpointsDisabledByFlag(t *testing.T) {
	t.Setenv("BIOMETRICS_OPTIMIZER_ENABLED", "false")
	server, _, _, _ := setupTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orchestrator/optimizer/recommendations", nil)
	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 with optimizer disabled, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestOptimizerRecommendationLifecycleEndpoints(t *testing.T) {
	server, _, _, _ := setupTestServer(t)

	generateReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/orchestrator/optimizer/recommendations",
		bytes.NewBufferString(`{"project_id":"biometrics","goal":"stabilize apex gate pass rate"}`),
	)
	generateReq.Header.Set("Content-Type", "application/json")
	generateW := httptest.NewRecorder()
	server.Handler().ServeHTTP(generateW, generateReq)
	if generateW.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", generateW.Code, generateW.Body.String())
	}

	var recommendation optimizer.Recommendation
	if err := json.NewDecoder(generateW.Body).Decode(&recommendation); err != nil {
		t.Fatalf("decode recommendation: %v", err)
	}
	if recommendation.ID == "" {
		t.Fatalf("expected recommendation id")
	}
	if recommendation.Status != "generated" {
		t.Fatalf("expected generated status, got %q", recommendation.Status)
	}

	listReq := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/orchestrator/optimizer/recommendations?project_id=biometrics&status=generated&limit=5",
		nil,
	)
	listW := httptest.NewRecorder()
	server.Handler().ServeHTTP(listW, listReq)
	if listW.Code != http.StatusOK {
		t.Fatalf("expected 200 list, got %d body=%s", listW.Code, listW.Body.String())
	}
	var list []optimizer.Recommendation
	if err := json.NewDecoder(listW.Body).Decode(&list); err != nil {
		t.Fatalf("decode recommendation list: %v", err)
	}
	if len(list) == 0 {
		t.Fatalf("expected at least one recommendation")
	}

	applyReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/orchestrator/optimizer/recommendations/"+recommendation.ID+"/apply",
		nil,
	)
	applyW := httptest.NewRecorder()
	server.Handler().ServeHTTP(applyW, applyReq)
	if applyW.Code != http.StatusOK {
		t.Fatalf("expected 200 apply, got %d body=%s", applyW.Code, applyW.Body.String())
	}
	var applyResult optimizer.ApplyResult
	if err := json.NewDecoder(applyW.Body).Decode(&applyResult); err != nil {
		t.Fatalf("decode apply result: %v", err)
	}
	if applyResult.RecommendationID == "" {
		t.Fatalf("expected recommendation_id in apply response")
	}
	if applyResult.Run.ID == "" {
		t.Fatalf("expected orchestrator run id after apply")
	}
	if applyResult.RecommendationID != recommendation.ID {
		t.Fatalf("expected recommendation_id %q, got %q", recommendation.ID, applyResult.RecommendationID)
	}
	if applyResult.Recommendation.ID != "" && applyResult.Recommendation.Status != "applied" {
		t.Fatalf("expected recommendation status applied when recommendation payload is present, got %q", applyResult.Recommendation.Status)
	}

	applyAgainReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/orchestrator/optimizer/recommendations/"+recommendation.ID+"/apply",
		nil,
	)
	applyAgainW := httptest.NewRecorder()
	server.Handler().ServeHTTP(applyAgainW, applyAgainReq)
	if applyAgainW.Code != http.StatusOK {
		t.Fatalf("expected 200 on idempotent apply, got %d body=%s", applyAgainW.Code, applyAgainW.Body.String())
	}
	var applyAgain optimizer.ApplyResult
	if err := json.NewDecoder(applyAgainW.Body).Decode(&applyAgain); err != nil {
		t.Fatalf("decode second apply result: %v", err)
	}
	if applyAgain.RecommendationID != recommendation.ID {
		t.Fatalf("expected idempotent recommendation_id %q, got %q", recommendation.ID, applyAgain.RecommendationID)
	}
	if applyAgain.Run.ID != applyResult.Run.ID {
		t.Fatalf("expected idempotent run id %q, got %q", applyResult.Run.ID, applyAgain.Run.ID)
	}

	anotherReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/orchestrator/optimizer/recommendations",
		bytes.NewBufferString(`{"project_id":"biometrics","goal":"cost gate hardening"}`),
	)
	anotherReq.Header.Set("Content-Type", "application/json")
	anotherW := httptest.NewRecorder()
	server.Handler().ServeHTTP(anotherW, anotherReq)
	if anotherW.Code != http.StatusCreated {
		t.Fatalf("expected 201 second recommendation, got %d body=%s", anotherW.Code, anotherW.Body.String())
	}

	var second optimizer.Recommendation
	if err := json.NewDecoder(anotherW.Body).Decode(&second); err != nil {
		t.Fatalf("decode second recommendation: %v", err)
	}
	rejectReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/orchestrator/optimizer/recommendations/"+second.ID+"/reject",
		bytes.NewBufferString(`{"reason":"manual override for today"}`),
	)
	rejectReq.Header.Set("Content-Type", "application/json")
	rejectW := httptest.NewRecorder()
	server.Handler().ServeHTTP(rejectW, rejectReq)
	if rejectW.Code != http.StatusOK {
		t.Fatalf("expected 200 reject, got %d body=%s", rejectW.Code, rejectW.Body.String())
	}
	var rejected optimizer.Recommendation
	if err := json.NewDecoder(rejectW.Body).Decode(&rejected); err != nil {
		t.Fatalf("decode rejected recommendation: %v", err)
	}
	if rejected.Status != "rejected" {
		t.Fatalf("expected rejected status, got %q", rejected.Status)
	}
	if !strings.Contains(strings.ToLower(rejected.RejectedReason), "manual override") {
		t.Fatalf("expected rejected reason to be set, got %q", rejected.RejectedReason)
	}

	rejectAgainReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/orchestrator/optimizer/recommendations/"+second.ID+"/reject",
		bytes.NewBufferString(`{"reason":"retry reject"}`),
	)
	rejectAgainReq.Header.Set("Content-Type", "application/json")
	rejectAgainW := httptest.NewRecorder()
	server.Handler().ServeHTTP(rejectAgainW, rejectAgainReq)
	if rejectAgainW.Code != http.StatusOK {
		t.Fatalf("expected 200 on idempotent reject, got %d body=%s", rejectAgainW.Code, rejectAgainW.Body.String())
	}
	var rejectedAgain optimizer.Recommendation
	if err := json.NewDecoder(rejectAgainW.Body).Decode(&rejectedAgain); err != nil {
		t.Fatalf("decode idempotent reject: %v", err)
	}
	if rejectedAgain.Status != "rejected" {
		t.Fatalf("expected rejected status on idempotent reject, got %q", rejectedAgain.Status)
	}
}

func TestOrchestratorSessionLifecycleEndpoints(t *testing.T) {
	server, _, _, _ := setupTestServer(t)

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/orchestrator/sessions", bytes.NewBufferString(`{"project_id":"biometrics"}`))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	server.Handler().ServeHTTP(createW, createReq)
	if createW.Code != http.StatusCreated {
		t.Fatalf("expected 201 create, got %d body=%s", createW.Code, createW.Body.String())
	}

	var created runtimeorchestrator.OrchestratorSession
	if err := json.NewDecoder(createW.Body).Decode(&created); err != nil {
		t.Fatalf("decode created session: %v", err)
	}
	if created.ID == "" {
		t.Fatalf("expected non-empty session id")
	}
	if len(created.Agents) != 3 {
		t.Fatalf("expected 3 default agents, got %d", len(created.Agents))
	}
	if created.Status != runtimeorchestrator.OrchestratorSessionStatusActive {
		t.Fatalf("expected active status, got %q", created.Status)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/orchestrator/sessions?project_id=biometrics", nil)
	listW := httptest.NewRecorder()
	server.Handler().ServeHTTP(listW, listReq)
	if listW.Code != http.StatusOK {
		t.Fatalf("expected 200 list, got %d body=%s", listW.Code, listW.Body.String())
	}
	var sessions []runtimeorchestrator.OrchestratorSession
	if err := json.NewDecoder(listW.Body).Decode(&sessions); err != nil {
		t.Fatalf("decode sessions list: %v", err)
	}
	if len(sessions) == 0 {
		t.Fatalf("expected at least one orchestrator session")
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/orchestrator/sessions/"+created.ID, nil)
	getW := httptest.NewRecorder()
	server.Handler().ServeHTTP(getW, getReq)
	if getW.Code != http.StatusOK {
		t.Fatalf("expected 200 get, got %d body=%s", getW.Code, getW.Body.String())
	}

	appendReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/orchestrator/sessions/"+created.ID+"/messages",
		bytes.NewBufferString(`{"author_kind":"user","target_pane":"backend","content":"Please add API validation middleware."}`),
	)
	appendReq.Header.Set("Content-Type", "application/json")
	appendW := httptest.NewRecorder()
	server.Handler().ServeHTTP(appendW, appendReq)
	if appendW.Code != http.StatusCreated {
		t.Fatalf("expected 201 append, got %d body=%s", appendW.Code, appendW.Body.String())
	}

	var appended runtimeorchestrator.OrchestratorMessage
	if err := json.NewDecoder(appendW.Body).Decode(&appended); err != nil {
		t.Fatalf("decode appended message: %v", err)
	}
	if appended.Cursor <= 0 {
		t.Fatalf("expected positive message cursor, got %d", appended.Cursor)
	}
	if appended.AuthorKind != runtimeorchestrator.OrchestratorAuthorKindUser {
		t.Fatalf("expected user author kind, got %q", appended.AuthorKind)
	}

	messagesReq := httptest.NewRequest(http.MethodGet, "/api/v1/orchestrator/sessions/"+created.ID+"/messages?cursor=0&limit=100", nil)
	messagesW := httptest.NewRecorder()
	server.Handler().ServeHTTP(messagesW, messagesReq)
	if messagesW.Code != http.StatusOK {
		t.Fatalf("expected 200 messages, got %d body=%s", messagesW.Code, messagesW.Body.String())
	}
	var messages []runtimeorchestrator.OrchestratorMessage
	if err := json.NewDecoder(messagesW.Body).Decode(&messages); err != nil {
		t.Fatalf("decode messages: %v", err)
	}
	foundUserMessage := false
	for _, message := range messages {
		if message.AuthorKind == runtimeorchestrator.OrchestratorAuthorKindUser && strings.Contains(message.Content, "API validation middleware") {
			foundUserMessage = true
			break
		}
	}
	if !foundUserMessage {
		t.Fatalf("expected to find appended user message in session history")
	}

	modelOverrideReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/orchestrator/sessions/"+created.ID+"/agents/backend/model",
		bytes.NewBufferString(`{"provider":"nim","model_id":"qwen-3.5-397b"}`),
	)
	modelOverrideReq.Header.Set("Content-Type", "application/json")
	modelOverrideW := httptest.NewRecorder()
	server.Handler().ServeHTTP(modelOverrideW, modelOverrideReq)
	if modelOverrideW.Code != http.StatusOK {
		t.Fatalf("expected 200 model override, got %d body=%s", modelOverrideW.Code, modelOverrideW.Body.String())
	}
	var backendState runtimeorchestrator.OrchestratorAgentState
	if err := json.NewDecoder(modelOverrideW.Body).Decode(&backendState); err != nil {
		t.Fatalf("decode backend state: %v", err)
	}
	if backendState.AgentID != "backend" {
		t.Fatalf("expected backend agent, got %q", backendState.AgentID)
	}
	if backendState.Model.Provider != "nim" {
		t.Fatalf("expected model provider nim, got %q", backendState.Model.Provider)
	}

	pauseReq := httptest.NewRequest(http.MethodPost, "/api/v1/orchestrator/sessions/"+created.ID+"/pause", bytes.NewBufferString(`{"reason":"manual guardrail"}`))
	pauseReq.Header.Set("Content-Type", "application/json")
	pauseW := httptest.NewRecorder()
	server.Handler().ServeHTTP(pauseW, pauseReq)
	if pauseW.Code != http.StatusOK {
		t.Fatalf("expected 200 pause, got %d body=%s", pauseW.Code, pauseW.Body.String())
	}
	var paused runtimeorchestrator.OrchestratorSession
	if err := json.NewDecoder(pauseW.Body).Decode(&paused); err != nil {
		t.Fatalf("decode paused session: %v", err)
	}
	if paused.Status != runtimeorchestrator.OrchestratorSessionStatusPaused {
		t.Fatalf("expected paused status, got %q", paused.Status)
	}
	if !paused.Guardrails.Paused {
		t.Fatalf("expected guardrail paused=true")
	}

	resumeReq := httptest.NewRequest(http.MethodPost, "/api/v1/orchestrator/sessions/"+created.ID+"/resume", nil)
	resumeW := httptest.NewRecorder()
	server.Handler().ServeHTTP(resumeW, resumeReq)
	if resumeW.Code != http.StatusOK {
		t.Fatalf("expected 200 resume, got %d body=%s", resumeW.Code, resumeW.Body.String())
	}
	var resumed runtimeorchestrator.OrchestratorSession
	if err := json.NewDecoder(resumeW.Body).Decode(&resumed); err != nil {
		t.Fatalf("decode resumed session: %v", err)
	}
	if resumed.Status != runtimeorchestrator.OrchestratorSessionStatusActive {
		t.Fatalf("expected active status after resume, got %q", resumed.Status)
	}

	killReq := httptest.NewRequest(http.MethodPost, "/api/v1/orchestrator/sessions/"+created.ID+"/kill", nil)
	killW := httptest.NewRecorder()
	server.Handler().ServeHTTP(killW, killReq)
	if killW.Code != http.StatusOK {
		t.Fatalf("expected 200 kill, got %d body=%s", killW.Code, killW.Body.String())
	}
	var killed runtimeorchestrator.OrchestratorSession
	if err := json.NewDecoder(killW.Body).Decode(&killed); err != nil {
		t.Fatalf("decode killed session: %v", err)
	}
	if killed.Status != runtimeorchestrator.OrchestratorSessionStatusKilled {
		t.Fatalf("expected killed status, got %q", killed.Status)
	}
}

func TestParseOrchestratorEventCursor(t *testing.T) {
	tests := []struct {
		name    string
		payload map[string]string
		want    int64
	}{
		{
			name:    "empty payload",
			payload: nil,
			want:    0,
		},
		{
			name: "cursor field",
			payload: map[string]string{
				"cursor": "17",
			},
			want: 17,
		},
		{
			name: "message cursor fallback",
			payload: map[string]string{
				"message_cursor": "9",
			},
			want: 9,
		},
		{
			name: "invalid cursor uses message cursor fallback",
			payload: map[string]string{
				"cursor":         "abc",
				"message_cursor": "11",
			},
			want: 11,
		},
		{
			name: "both values invalid",
			payload: map[string]string{
				"cursor":         "abc",
				"message_cursor": "def",
			},
			want: 0,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if got := parseOrchestratorEventCursor(tc.payload); got != tc.want {
				t.Fatalf("expected %d, got %d", tc.want, got)
			}
		})
	}
}

func TestParseBoundedIntQuery(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		fallback int
		min      int
		max      int
		want     int
	}{
		{name: "valid value", raw: "12", fallback: 20, min: 1, max: 200, want: 12},
		{name: "invalid uses fallback", raw: "oops", fallback: 20, min: 1, max: 200, want: 20},
		{name: "below min clamped", raw: "-5", fallback: 20, min: 1, max: 200, want: 1},
		{name: "above max clamped", raw: "999", fallback: 20, min: 1, max: 200, want: 200},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if got := parseBoundedIntQuery(tc.raw, tc.fallback, tc.min, tc.max); got != tc.want {
				t.Fatalf("expected %d, got %d", tc.want, got)
			}
		})
	}
}

func TestParseNonNegativeInt64Query(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want int64
	}{
		{name: "positive", raw: "42", want: 42},
		{name: "zero", raw: "0", want: 0},
		{name: "negative clamped", raw: "-7", want: 0},
		{name: "invalid clamped", raw: "abc", want: 0},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if got := parseNonNegativeInt64Query(tc.raw); got != tc.want {
				t.Fatalf("expected %d, got %d", tc.want, got)
			}
		})
	}
}

func TestOrchestratorSessionStreamReturnsNotFoundForUnknownSession(t *testing.T) {
	server, _, _, _ := setupTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orchestrator/sessions/unknown-session/stream", nil)
	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown session stream, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestCreateRunRejectsInvalidSchedulerMode(t *testing.T) {
	server, _, _, _ := setupTestServer(t)

	raw := []byte(`{"project_id":"biometrics","goal":"invalid scheduler","scheduler_mode":"invalid_mode"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCreateRunRejectsInvalidMode(t *testing.T) {
	server, _, _, _ := setupTestServer(t)

	raw := []byte(`{"project_id":"biometrics","goal":"invalid mode","mode":"manual"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	if !strings.Contains(strings.ToLower(w.Body.String()), "invalid mode") {
		t.Fatalf("expected invalid mode error, got %s", w.Body.String())
	}
}

func TestSupervisedRunCheckpointPauseAndResume(t *testing.T) {
	server, manager, _, _ := setupTestServer(t)

	raw := []byte(`{"project_id":"biometrics","goal":"supervised checkpoint test","mode":"supervised","scheduler_mode":"dag_parallel_v1","max_parallelism":4}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}

	var run contracts.Run
	if err := json.NewDecoder(w.Body).Decode(&run); err != nil {
		t.Fatalf("decode run: %v", err)
	}
	if run.Mode != string(contracts.RunModeSupervised) {
		t.Fatalf("expected supervised mode, got %q", run.Mode)
	}

	waitForRunStatus(t, manager, run.ID, contracts.RunPaused, 4*time.Second)

	events, err := manager.Events(run.ID, 500)
	if err != nil {
		t.Fatalf("events: %v", err)
	}
	if indexOfEvent(events, "run.supervision.checkpoint") < 0 {
		t.Fatalf("expected run.supervision.checkpoint event, got %d events", len(events))
	}

	resumeReq := httptest.NewRequest(http.MethodPost, "/api/v1/runs/"+run.ID+"/resume", nil)
	resumeW := httptest.NewRecorder()
	server.Handler().ServeHTTP(resumeW, resumeReq)
	if resumeW.Code != http.StatusOK {
		t.Fatalf("expected 200 resume, got %d body=%s", resumeW.Code, resumeW.Body.String())
	}

	deadline := time.Now().Add(12 * time.Second)
	for time.Now().Before(deadline) {
		current, err := manager.GetRun(run.ID)
		if err != nil {
			t.Fatalf("get run: %v", err)
		}
		switch current.Status {
		case contracts.RunRunning, contracts.RunCompleted:
			_ = manager.CancelRun(run.ID)
			waitForRunTerminalState(t, manager, run.ID)
			return
		case contracts.RunFailed, contracts.RunCancelled:
			t.Fatalf("expected supervised run to stay resumable, got %s", current.Status)
		}
		time.Sleep(40 * time.Millisecond)
	}

	t.Fatalf("supervised run did not leave paused state before deadline")
}

func TestEventOrderingStartedBeforeCompleted(t *testing.T) {
	server, manager, _, _ := setupTestServer(t)

	raw := []byte(`{"project_id":"biometrics","goal":"event ordering check","mode":"autonomous"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}

	var run contracts.Run
	if err := json.NewDecoder(w.Body).Decode(&run); err != nil {
		t.Fatalf("decode run: %v", err)
	}

	waitForRunTerminalState(t, manager, run.ID)

	events, err := manager.Events(run.ID, 300)
	if err != nil {
		t.Fatalf("events: %v", err)
	}

	startedIndex := -1
	completedIndex := -1
	for idx, ev := range events {
		if ev.Type == "task.started" && startedIndex == -1 {
			startedIndex = idx
		}
		if ev.Type == "task.completed" && completedIndex == -1 {
			completedIndex = idx
		}
	}

	if startedIndex == -1 || completedIndex == -1 {
		t.Fatalf("expected both task.started and task.completed, got %d events", len(events))
	}
	if startedIndex > completedIndex {
		t.Fatalf("event ordering invalid: started=%d completed=%d", startedIndex, completedIndex)
	}
}

func TestEventReplayReturnsLatestLimit(t *testing.T) {
	server, manager, _, _ := setupTestServer(t)

	raw := []byte(`{"project_id":"biometrics","goal":"event replay latest limit","mode":"autonomous"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}

	var run contracts.Run
	if err := json.NewDecoder(w.Body).Decode(&run); err != nil {
		t.Fatalf("decode run: %v", err)
	}
	waitForRunTerminalState(t, manager, run.ID)

	allEvents, err := manager.Events(run.ID, 500)
	if err != nil {
		t.Fatalf("all events: %v", err)
	}
	if len(allEvents) < 2 {
		t.Fatalf("expected at least 2 events, got %d", len(allEvents))
	}
	latestTwo, err := manager.Events(run.ID, 2)
	if err != nil {
		t.Fatalf("latest events: %v", err)
	}
	if len(latestTwo) != 2 {
		t.Fatalf("expected 2 latest events, got %d", len(latestTwo))
	}

	refreshed, err := manager.Events(run.ID, 500)
	if err != nil {
		t.Fatalf("refresh events: %v", err)
	}
	if len(refreshed) < 2 {
		t.Fatalf("expected at least 2 refreshed events, got %d", len(refreshed))
	}
	expectedA := refreshed[len(refreshed)-2]
	expectedB := refreshed[len(refreshed)-1]
	if latestTwo[0].ID != expectedA.ID || latestTwo[1].ID != expectedB.ID {
		t.Fatalf("expected replay latest IDs [%s %s], got [%s %s]", expectedA.ID, expectedB.ID, latestTwo[0].ID, latestTwo[1].ID)
	}
}

func TestRunGraphAndAttemptsEndpoints(t *testing.T) {
	server, manager, _, _ := setupTestServer(t)

	raw := []byte(`{"project_id":"biometrics","goal":"graph endpoint check","mode":"autonomous","scheduler_mode":"dag_parallel_v1","max_parallelism":6}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}

	var run contracts.Run
	if err := json.NewDecoder(w.Body).Decode(&run); err != nil {
		t.Fatalf("decode run: %v", err)
	}

	waitForRunTerminalState(t, manager, run.ID)

	graphReq := httptest.NewRequest(http.MethodGet, "/api/v1/runs/"+run.ID+"/graph", nil)
	graphW := httptest.NewRecorder()
	server.Handler().ServeHTTP(graphW, graphReq)
	if graphW.Code != http.StatusOK {
		t.Fatalf("expected 200 for graph, got %d", graphW.Code)
	}

	var graph contracts.TaskGraph
	if err := json.NewDecoder(graphW.Body).Decode(&graph); err != nil {
		t.Fatalf("decode graph: %v", err)
	}
	if graph.RunID != run.ID {
		t.Fatalf("unexpected graph run id: %s", graph.RunID)
	}
	if len(graph.Nodes) < 7 {
		t.Fatalf("expected >= 7 graph nodes, got %d", len(graph.Nodes))
	}

	attemptReq := httptest.NewRequest(http.MethodGet, "/api/v1/runs/"+run.ID+"/attempts", nil)
	attemptW := httptest.NewRecorder()
	server.Handler().ServeHTTP(attemptW, attemptReq)
	if attemptW.Code != http.StatusOK {
		t.Fatalf("expected 200 for attempts, got %d", attemptW.Code)
	}

	var attempts []contracts.TaskAttempt
	if err := json.NewDecoder(attemptW.Body).Decode(&attempts); err != nil {
		t.Fatalf("decode attempts: %v", err)
	}
	if len(attempts) < 7 {
		t.Fatalf("expected >= 7 attempts, got %d", len(attempts))
	}
}

func TestHealthReadyAndMetricsEndpoints(t *testing.T) {
	server, manager, _, _ := setupTestServer(t)

	raw := []byte(`{"project_id":"biometrics","goal":"metrics endpoint check","mode":"autonomous","scheduler_mode":"dag_parallel_v1","max_parallelism":4}`)
	runReq := httptest.NewRequest(http.MethodPost, "/api/v1/runs", bytes.NewReader(raw))
	runReq.Header.Set("Content-Type", "application/json")
	runW := httptest.NewRecorder()
	server.Handler().ServeHTTP(runW, runReq)
	if runW.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", runW.Code)
	}
	var run contracts.Run
	if err := json.NewDecoder(runW.Body).Decode(&run); err != nil {
		t.Fatalf("decode run: %v", err)
	}
	waitForRunTerminalState(t, manager, run.ID)

	readyReq := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	readyW := httptest.NewRecorder()
	server.Handler().ServeHTTP(readyW, readyReq)
	if readyW.Code != http.StatusOK {
		t.Fatalf("expected 200 for /health/ready, got %d", readyW.Code)
	}

	var ready map[string]interface{}
	if err := json.NewDecoder(readyW.Body).Decode(&ready); err != nil {
		t.Fatalf("decode readiness: %v", err)
	}
	if ready["ready"] != true {
		t.Fatalf("expected readiness true, got %#v", ready["ready"])
	}
	if _, ok := ready["opencode_available"].(bool); !ok {
		t.Fatalf("expected opencode_available boolean in readiness payload, got %#v", ready["opencode_available"])
	}
	if raw, ok := ready["onboard_last_status"]; ok {
		if _, cast := raw.(string); !cast {
			t.Fatalf("expected onboard_last_status to be string when present, got %#v", raw)
		}
	}

	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsW := httptest.NewRecorder()
	server.Handler().ServeHTTP(metricsW, metricsReq)
	if metricsW.Code != http.StatusOK {
		t.Fatalf("expected 200 for /metrics, got %d", metricsW.Code)
	}
	if got := metricsW.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/plain") {
		t.Fatalf("unexpected metrics content type: %q", got)
	}
	body := metricsW.Body.String()
	if !bytes.Contains([]byte(body), []byte("biometrics_runs_started")) {
		t.Fatalf("expected metrics body to include runs_started counter, got: %s", body)
	}
	if !bytes.Contains([]byte(body), []byte("biometrics_task_dispatch_latency_count")) {
		t.Fatalf("expected metrics body to include dispatch latency count, got: %s", body)
	}
	if !bytes.Contains([]byte(body), []byte("biometrics_task_dispatch_latency_p95_estimate_ms")) {
		t.Fatalf("expected metrics body to include dispatch latency p95 estimate, got: %s", body)
	}
	if !bytes.Contains([]byte(body), []byte("biometrics_eventbus_dropped_events")) {
		t.Fatalf("expected metrics body to include eventbus dropped events, got: %s", body)
	}
	if !bytes.Contains([]byte(body), []byte("biometrics_eventbus_subscribers")) {
		t.Fatalf("expected metrics body to include eventbus subscriber count, got: %s", body)
	}
	if !bytes.Contains([]byte(body), []byte("biometrics_scheduler_ready_queue_depth")) {
		t.Fatalf("expected metrics body to include scheduler ready queue depth, got: %s", body)
	}
}

func TestSSEWriterIncludesIDTypedAndMessageFrames(t *testing.T) {
	w := httptest.NewRecorder()
	ev := contracts.Event{
		ID:      "evt-1",
		RunID:   "run-1",
		Type:    "task.ready",
		Source:  "scheduler",
		Payload: map[string]string{"task_id": "t1"},
	}
	writeSSE(w, ev)

	body := w.Body.String()
	if strings.Count(body, "id: evt-1") != 2 {
		t.Fatalf("expected SSE id frame to be emitted twice (typed+message), got body: %s", body)
	}
	if !strings.Contains(body, "event: task.ready") {
		t.Fatalf("expected typed SSE frame in body: %s", body)
	}
	if !strings.Contains(body, "event: message") {
		t.Fatalf("expected compatibility message frame in body: %s", body)
	}
}

func TestSSEEndpointReplaysLatestLimitWithEventIDs(t *testing.T) {
	server, manager, _, _ := setupTestServer(t)

	raw := []byte(`{"project_id":"biometrics","goal":"sse replay latest limit","mode":"autonomous"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}

	var run contracts.Run
	if err := json.NewDecoder(w.Body).Decode(&run); err != nil {
		t.Fatalf("decode run: %v", err)
	}
	waitForRunTerminalState(t, manager, run.ID)

	latest, err := manager.Events(run.ID, 2)
	if err != nil {
		t.Fatalf("events: %v", err)
	}
	if len(latest) != 2 {
		t.Fatalf("expected 2 latest events, got %d", len(latest))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	streamReq := httptest.NewRequest(http.MethodGet, "/api/v1/events?run_id="+run.ID+"&limit=2", nil).WithContext(ctx)
	streamW := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		server.Handler().ServeHTTP(streamW, streamReq)
		close(done)
	}()

	time.Sleep(120 * time.Millisecond)
	cancel()
	<-done

	body := streamW.Body.String()
	for _, ev := range latest {
		if !strings.Contains(body, "id: "+ev.ID) {
			t.Fatalf("expected replayed SSE payload to include event id %s, body=%s", ev.ID, body)
		}
	}
	if strings.Count(body, "event: message") < 2 {
		t.Fatalf("expected at least two message compatibility frames, body=%s", body)
	}
}

func TestSSEEndpointStreamsLiveTypedAndMessageFrames(t *testing.T) {
	server, _, _, _ := setupTestServer(t)

	liveRunID := "live-stream-run"
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	streamReq := httptest.NewRequest(http.MethodGet, "/api/v1/events?run_id="+liveRunID+"&limit=1", nil).WithContext(ctx)
	streamW := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		server.Handler().ServeHTTP(streamW, streamReq)
		close(done)
	}()

	time.Sleep(120 * time.Millisecond)
	ev, err := server.bus.Publish(contracts.Event{
		RunID:  liveRunID,
		Type:   "task.ready",
		Source: "scheduler",
		Payload: map[string]string{
			"task_id": "live-task",
		},
	})
	if err != nil {
		t.Fatalf("publish event: %v", err)
	}

	time.Sleep(120 * time.Millisecond)
	cancel()
	<-done

	body := streamW.Body.String()
	if !strings.Contains(body, "id: "+ev.ID) {
		t.Fatalf("expected live SSE payload to include event id, body=%s", body)
	}
	if !strings.Contains(body, "event: task.ready") {
		t.Fatalf("expected live typed event frame, body=%s", body)
	}
	if !strings.Contains(body, "event: message") {
		t.Fatalf("expected live compatibility message frame, body=%s", body)
	}
}

func TestOrchestratorSessionStreamReplaysEventsAfterCursor(t *testing.T) {
	server, _, _, _ := setupTestServer(t)

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/orchestrator/sessions", bytes.NewBufferString(`{"project_id":"biometrics"}`))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	server.Handler().ServeHTTP(createW, createReq)
	if createW.Code != http.StatusCreated {
		t.Fatalf("expected 201 create, got %d body=%s", createW.Code, createW.Body.String())
	}

	var session runtimeorchestrator.OrchestratorSession
	if err := json.NewDecoder(createW.Body).Decode(&session); err != nil {
		t.Fatalf("decode session: %v", err)
	}
	if strings.TrimSpace(session.ID) == "" {
		t.Fatalf("expected session id")
	}

	older, err := server.bus.Publish(contracts.Event{
		RunID:  session.ID,
		Type:   "orchestrator.agent.triggered",
		Source: "test",
		Payload: map[string]string{
			"cursor": "2",
		},
	})
	if err != nil {
		t.Fatalf("publish older event: %v", err)
	}
	newer, err := server.bus.Publish(contracts.Event{
		RunID:  session.ID,
		Type:   "orchestrator.agent.job.started",
		Source: "test",
		Payload: map[string]string{
			"cursor": "7",
		},
	})
	if err != nil {
		t.Fatalf("publish newer event: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	streamReq := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/orchestrator/sessions/"+session.ID+"/stream?cursor=3&limit=20",
		nil,
	).WithContext(ctx)
	streamW := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		server.Handler().ServeHTTP(streamW, streamReq)
		close(done)
	}()

	time.Sleep(120 * time.Millisecond)
	cancel()
	<-done

	body := streamW.Body.String()
	if strings.Contains(body, "id: "+older.ID) {
		t.Fatalf("expected cursor filter to skip older event id=%s body=%s", older.ID, body)
	}
	if !strings.Contains(body, "id: "+newer.ID) {
		t.Fatalf("expected cursor filter to include newer event id=%s body=%s", newer.ID, body)
	}
	if !strings.Contains(body, "event: orchestrator.agent.job.started") {
		t.Fatalf("expected typed orchestrator event frame, body=%s", body)
	}
}

func TestOrchestratorSessionStreamCursorFallbackUsesMessageCursor(t *testing.T) {
	server, _, _, _ := setupTestServer(t)

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/orchestrator/sessions", bytes.NewBufferString(`{"project_id":"biometrics"}`))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	server.Handler().ServeHTTP(createW, createReq)
	if createW.Code != http.StatusCreated {
		t.Fatalf("expected 201 create, got %d body=%s", createW.Code, createW.Body.String())
	}

	var session runtimeorchestrator.OrchestratorSession
	if err := json.NewDecoder(createW.Body).Decode(&session); err != nil {
		t.Fatalf("decode session: %v", err)
	}

	published, err := server.bus.Publish(contracts.Event{
		RunID:  session.ID,
		Type:   "orchestrator.message.created",
		Source: "test",
		Payload: map[string]string{
			"cursor":         "invalid",
			"message_cursor": "11",
		},
	})
	if err != nil {
		t.Fatalf("publish message event: %v", err)
	}

	// Cursor below message_cursor should still replay this event.
	{
		ctx, cancel := context.WithCancel(context.Background())
		streamReq := httptest.NewRequest(
			http.MethodGet,
			"/api/v1/orchestrator/sessions/"+session.ID+"/stream?cursor=10&limit=20",
			nil,
		).WithContext(ctx)
		streamW := httptest.NewRecorder()
		done := make(chan struct{})
		go func() {
			server.Handler().ServeHTTP(streamW, streamReq)
			close(done)
		}()

		time.Sleep(120 * time.Millisecond)
		cancel()
		<-done

		if body := streamW.Body.String(); !strings.Contains(body, "id: "+published.ID) {
			t.Fatalf("expected fallback cursor replay for event id=%s body=%s", published.ID, body)
		}
	}

	// Cursor equal to message_cursor should skip this event.
	{
		ctx, cancel := context.WithCancel(context.Background())
		streamReq := httptest.NewRequest(
			http.MethodGet,
			"/api/v1/orchestrator/sessions/"+session.ID+"/stream?cursor=11&limit=20",
			nil,
		).WithContext(ctx)
		streamW := httptest.NewRecorder()
		done := make(chan struct{})
		go func() {
			server.Handler().ServeHTTP(streamW, streamReq)
			close(done)
		}()

		time.Sleep(120 * time.Millisecond)
		cancel()
		<-done

		if body := streamW.Body.String(); strings.Contains(body, "id: "+published.ID) {
			t.Fatalf("expected event to be filtered at cursor boundary id=%s body=%s", published.ID, body)
		}
	}
}

func TestDAGParallelHighCardinalityRunCompletesWithoutFallback(t *testing.T) {
	server, manager, _, _ := setupTestServer(t)

	goal := buildHighCardinalityGoal(50)
	payload := fmt.Sprintf(`{"project_id":"biometrics","goal":%q,"mode":"autonomous","scheduler_mode":"dag_parallel_v1","max_parallelism":8}`, goal)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}

	var run contracts.Run
	if err := json.NewDecoder(w.Body).Decode(&run); err != nil {
		t.Fatalf("decode run: %v", err)
	}

	waitForRunTerminalState(t, manager, run.ID)

	finished, err := manager.GetRun(run.ID)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	if finished.Status != contracts.RunCompleted {
		t.Fatalf("expected completed run, got status=%s error=%s", finished.Status, finished.Error)
	}
	if finished.FallbackTriggered {
		t.Fatalf("unexpected serial fallback for high-cardinality DAG run")
	}

	graphReq := httptest.NewRequest(http.MethodGet, "/api/v1/runs/"+run.ID+"/graph", nil)
	graphW := httptest.NewRecorder()
	server.Handler().ServeHTTP(graphW, graphReq)
	if graphW.Code != http.StatusOK {
		t.Fatalf("expected 200 for graph endpoint, got %d", graphW.Code)
	}

	var graph contracts.TaskGraph
	if err := json.NewDecoder(graphW.Body).Decode(&graph); err != nil {
		t.Fatalf("decode graph: %v", err)
	}
	if got, want := len(graph.Nodes), 203; got != want {
		t.Fatalf("unexpected graph node count: got %d want %d", got, want)
	}

	events, err := manager.Events(run.ID, 1000)
	if err != nil {
		t.Fatalf("list run events: %v", err)
	}
	if indexOfEvent(events, "run.fallback.serial") >= 0 {
		t.Fatalf("did not expect run.fallback.serial event for high-cardinality run")
	}
}

func TestAPIContractDriftRunTaskEventSchemas(t *testing.T) {
	server, manager, _, _ := setupTestServer(t)

	raw := []byte(`{"project_id":"biometrics","goal":"contract drift check","mode":"autonomous","scheduler_mode":"dag_parallel_v1","max_parallelism":4}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}

	var created contracts.Run
	if err := json.NewDecoder(w.Body).Decode(&created); err != nil {
		t.Fatalf("decode run: %v", err)
	}
	waitForRunTerminalState(t, manager, created.ID)

	runSchema := loadContractSchema(t, "run.schema.json")
	taskSchema := loadContractSchema(t, "task.schema.json")
	eventSchema := loadContractSchema(t, "event.schema.json")
	attemptSchema := loadContractSchema(t, "attempt.schema.json")
	graphSchema := loadContractSchema(t, "graph.schema.json")
	errorSchema := loadContractSchema(t, "error.schema.json")

	runReq := httptest.NewRequest(http.MethodGet, "/api/v1/runs/"+created.ID, nil)
	runW := httptest.NewRecorder()
	server.Handler().ServeHTTP(runW, runReq)
	if runW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", runW.Code)
	}
	var runObj map[string]interface{}
	if err := json.NewDecoder(runW.Body).Decode(&runObj); err != nil {
		t.Fatalf("decode run object: %v", err)
	}
	validateObjectAgainstSchema(t, runObj, runSchema, "run")

	taskReq := httptest.NewRequest(http.MethodGet, "/api/v1/runs/"+created.ID+"/tasks", nil)
	taskW := httptest.NewRecorder()
	server.Handler().ServeHTTP(taskW, taskReq)
	if taskW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", taskW.Code)
	}
	var tasks []map[string]interface{}
	if err := json.NewDecoder(taskW.Body).Decode(&tasks); err != nil {
		t.Fatalf("decode task objects: %v", err)
	}
	if len(tasks) == 0 {
		t.Fatal("expected tasks in run")
	}
	for i, task := range tasks {
		validateObjectAgainstSchema(t, task, taskSchema, fmt.Sprintf("task[%d]", i))
	}

	events, err := manager.Events(created.ID, 500)
	if err != nil {
		t.Fatalf("list run events: %v", err)
	}
	if len(events) == 0 {
		t.Fatal("expected events in run")
	}
	for i, event := range events {
		rawEvent, marshalErr := json.Marshal(event)
		if marshalErr != nil {
			t.Fatalf("marshal event[%d]: %v", i, marshalErr)
		}
		var eventObj map[string]interface{}
		if unmarshalErr := json.Unmarshal(rawEvent, &eventObj); unmarshalErr != nil {
			t.Fatalf("unmarshal event[%d]: %v", i, unmarshalErr)
		}
		validateObjectAgainstSchema(t, eventObj, eventSchema, fmt.Sprintf("event[%d]", i))
	}

	graphReq := httptest.NewRequest(http.MethodGet, "/api/v1/runs/"+created.ID+"/graph", nil)
	graphW := httptest.NewRecorder()
	server.Handler().ServeHTTP(graphW, graphReq)
	if graphW.Code != http.StatusOK {
		t.Fatalf("expected 200 for graph, got %d", graphW.Code)
	}
	var graphObj map[string]interface{}
	if err := json.NewDecoder(graphW.Body).Decode(&graphObj); err != nil {
		t.Fatalf("decode graph object: %v", err)
	}
	validateObjectAgainstSchema(t, graphObj, graphSchema, "graph")

	attemptReq := httptest.NewRequest(http.MethodGet, "/api/v1/runs/"+created.ID+"/attempts", nil)
	attemptW := httptest.NewRecorder()
	server.Handler().ServeHTTP(attemptW, attemptReq)
	if attemptW.Code != http.StatusOK {
		t.Fatalf("expected 200 for attempts, got %d", attemptW.Code)
	}
	var attempts []map[string]interface{}
	if err := json.NewDecoder(attemptW.Body).Decode(&attempts); err != nil {
		t.Fatalf("decode attempt objects: %v", err)
	}
	if len(attempts) == 0 {
		t.Fatal("expected attempts in run")
	}
	for i, attempt := range attempts {
		validateObjectAgainstSchema(t, attempt, attemptSchema, fmt.Sprintf("attempt[%d]", i))
	}

	errorReq := httptest.NewRequest(http.MethodGet, "/api/v1/runs/non-existent-run", nil)
	errorW := httptest.NewRecorder()
	server.Handler().ServeHTTP(errorW, errorReq)
	if errorW.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing run, got %d", errorW.Code)
	}
	var errorObj map[string]interface{}
	if err := json.NewDecoder(errorW.Body).Decode(&errorObj); err != nil {
		t.Fatalf("decode error object: %v", err)
	}
	validateObjectAgainstSchema(t, errorObj, errorSchema, "error")
}

func TestRunControlEndpointsAreIdempotentAfterTerminalState(t *testing.T) {
	server, manager, _, _ := setupTestServer(t)

	raw := []byte(`{"project_id":"biometrics","goal":"idempotence check","mode":"autonomous"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}

	var run contracts.Run
	if err := json.NewDecoder(w.Body).Decode(&run); err != nil {
		t.Fatalf("decode run: %v", err)
	}
	waitForRunTerminalState(t, manager, run.ID)

	for _, action := range []string{"pause", "pause", "resume", "resume", "cancel", "cancel"} {
		actReq := httptest.NewRequest(http.MethodPost, "/api/v1/runs/"+run.ID+"/"+action, nil)
		actW := httptest.NewRecorder()
		server.Handler().ServeHTTP(actW, actReq)
		if actW.Code != http.StatusOK {
			t.Fatalf("expected 200 for action %s, got %d (%s)", action, actW.Code, actW.Body.String())
		}
	}
}

func TestFSPathTraversalIsBlocked(t *testing.T) {
	server, _, _, _ := setupTestServer(t)

	endpoints := []string{
		"/api/v1/fs/tree?path=../",
		"/api/v1/fs/file?path=../secrets.txt",
	}
	for _, endpoint := range endpoints {
		req := httptest.NewRequest(http.MethodGet, endpoint, nil)
		w := httptest.NewRecorder()
		server.Handler().ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for endpoint %s, got %d", endpoint, w.Code)
		}
	}
}

func TestFSPrefixBypassTraversalIsBlocked(t *testing.T) {
	server, _, _, workspace := setupTestServer(t)

	workspaceBase := filepath.Base(workspace)
	siblingDir := workspace + "-sibling"
	if err := os.MkdirAll(siblingDir, 0o755); err != nil {
		t.Fatalf("mkdir sibling dir: %v", err)
	}
	siblingFile := filepath.Join(siblingDir, "leak.txt")
	if err := os.WriteFile(siblingFile, []byte("secret"), 0o644); err != nil {
		t.Fatalf("write sibling file: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/fs/file?path=../"+workspaceBase+"-sibling/leak.txt", nil)
	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for prefix bypass attempt, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestFSSymlinkEscapeIsBlocked(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink behavior differs on windows CI")
	}

	server, _, _, workspace := setupTestServer(t)

	outsideDir := filepath.Join(t.TempDir(), "outside")
	if err := os.MkdirAll(outsideDir, 0o755); err != nil {
		t.Fatalf("mkdir outside dir: %v", err)
	}
	outsideFile := filepath.Join(outsideDir, "secret.txt")
	if err := os.WriteFile(outsideFile, []byte("secret"), 0o644); err != nil {
		t.Fatalf("write outside file: %v", err)
	}

	linkPath := filepath.Join(workspace, "biometrics", "outside-link.txt")
	if err := os.Symlink(outsideFile, linkPath); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/fs/file?path=biometrics/outside-link.txt", nil)
	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for symlink escape, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestBlueprintCatalogEndpoints(t *testing.T) {
	server, _, _, _ := setupTestServer(t)

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/blueprints", nil)
	listW := httptest.NewRecorder()
	server.Handler().ServeHTTP(listW, listReq)
	if listW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", listW.Code)
	}

	var profiles []map[string]interface{}
	if err := json.NewDecoder(listW.Body).Decode(&profiles); err != nil {
		t.Fatalf("decode profiles: %v", err)
	}
	if len(profiles) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(profiles))
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/blueprints/universal-2026", nil)
	getW := httptest.NewRecorder()
	server.Handler().ServeHTTP(getW, getReq)
	if getW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", getW.Code)
	}

	var profile map[string]interface{}
	if err := json.NewDecoder(getW.Body).Decode(&profile); err != nil {
		t.Fatalf("decode profile: %v", err)
	}
	if profile["id"] != "universal-2026" {
		t.Fatalf("unexpected profile id: %v", profile["id"])
	}
}

func TestRunWithBlueprintBootstrapEvents(t *testing.T) {
	server, manager, _, _ := setupTestServer(t)

	raw := []byte(`{"project_id":"biometrics","goal":"bootstrap docs","mode":"autonomous","blueprint_profile":"universal-2026","blueprint_modules":["engine"],"bootstrap":true}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}

	var run contracts.Run
	if err := json.NewDecoder(w.Body).Decode(&run); err != nil {
		t.Fatalf("decode run: %v", err)
	}

	waitForRunTerminalState(t, manager, run.ID)

	events, err := manager.Events(run.ID, 400)
	if err != nil {
		t.Fatalf("events: %v", err)
	}

	selected := indexOfEvent(events, "blueprint.selected")
	started := indexOfEvent(events, "blueprint.bootstrap.started")
	applied := indexOfEvent(events, "blueprint.module.applied")
	completed := indexOfEvent(events, "blueprint.bootstrap.completed")
	if selected == -1 || started == -1 || applied == -1 || completed == -1 {
		t.Fatalf("missing blueprint events: selected=%d started=%d applied=%d completed=%d", selected, started, applied, completed)
	}
	if !(selected < started && started < applied && applied < completed) {
		t.Fatalf("blueprint event order invalid: selected=%d started=%d applied=%d completed=%d", selected, started, applied, completed)
	}
}

func TestProjectBootstrapEndpoint(t *testing.T) {
	server, _, _, workspace := setupTestServer(t)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/projects/biometrics/bootstrap",
		bytes.NewReader([]byte(`{"blueprint_profile":"universal-2026","blueprint_modules":["website"]}`)),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("decode bootstrap result: %v", err)
	}
	if result["profile_id"] != "universal-2026" {
		t.Fatalf("unexpected profile id: %v", result["profile_id"])
	}

	if _, err := os.Stat(filepath.Join(workspace, "biometrics", "BLUEPRINT.md")); err != nil {
		t.Fatalf("expected BLUEPRINT.md to exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(workspace, "biometrics", "AGENTS.md")); err != nil {
		t.Fatalf("expected AGENTS.md to exist: %v", err)
	}
}

func TestOrchestratorEndpointsPlanRunScorecard(t *testing.T) {
	server, _, _, _ := setupTestServer(t)

	planReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/orchestrator/plans",
		bytes.NewReader([]byte(`{"project_id":"biometrics","goal":"deliver apex orchestration","strategy_mode":"arena","objective":{"quality":0.6,"speed":0.25,"cost":0.15}}`)),
	)
	planReq.Header.Set("Content-Type", "application/json")
	planW := httptest.NewRecorder()
	server.Handler().ServeHTTP(planW, planReq)
	if planW.Code != http.StatusCreated {
		t.Fatalf("expected 201 plan, got %d body=%s", planW.Code, planW.Body.String())
	}

	var plan map[string]interface{}
	if err := json.NewDecoder(planW.Body).Decode(&plan); err != nil {
		t.Fatalf("decode plan: %v", err)
	}
	planID, _ := plan["id"].(string)
	if planID == "" {
		t.Fatalf("expected plan id")
	}

	runReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/orchestrator/runs",
		bytes.NewReader([]byte(fmt.Sprintf(`{"plan_id":%q,"project_id":"biometrics","goal":"deliver apex orchestration"}`, planID))),
	)
	runReq.Header.Set("Content-Type", "application/json")
	runW := httptest.NewRecorder()
	server.Handler().ServeHTTP(runW, runReq)
	if runW.Code != http.StatusCreated {
		t.Fatalf("expected 201 run, got %d body=%s", runW.Code, runW.Body.String())
	}

	var run map[string]interface{}
	if err := json.NewDecoder(runW.Body).Decode(&run); err != nil {
		t.Fatalf("decode run: %v", err)
	}
	runID, _ := run["id"].(string)
	if runID == "" {
		t.Fatalf("expected orchestrator run id")
	}

	deadline := time.Now().Add(6 * time.Second)
	for time.Now().Before(deadline) {
		runGetReq := httptest.NewRequest(http.MethodGet, "/api/v1/orchestrator/runs/"+runID, nil)
		runGetW := httptest.NewRecorder()
		server.Handler().ServeHTTP(runGetW, runGetReq)
		if runGetW.Code != http.StatusOK {
			t.Fatalf("expected 200 run get, got %d body=%s", runGetW.Code, runGetW.Body.String())
		}

		scoreReq := httptest.NewRequest(http.MethodGet, "/api/v1/orchestrator/runs/"+runID+"/scorecard", nil)
		scoreW := httptest.NewRecorder()
		server.Handler().ServeHTTP(scoreW, scoreReq)
		if scoreW.Code == http.StatusOK {
			var score map[string]interface{}
			if err := json.NewDecoder(scoreW.Body).Decode(&score); err != nil {
				t.Fatalf("decode scorecard: %v", err)
			}
			if quality, ok := score["quality_score"].(float64); !ok || quality <= 0 {
				t.Fatalf("expected positive quality_score, got %#v", score["quality_score"])
			}
			return
		}
		time.Sleep(40 * time.Millisecond)
	}
	t.Fatalf("scorecard endpoint did not become ready")
}

func TestOrchestratorResumeFromStepEndpoint(t *testing.T) {
	server, _, _, _ := setupTestServer(t)

	runReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/orchestrator/runs",
		bytes.NewReader([]byte(`{"project_id":"biometrics","goal":"resume test","strategy_mode":"adaptive"}`)),
	)
	runReq.Header.Set("Content-Type", "application/json")
	runW := httptest.NewRecorder()
	server.Handler().ServeHTTP(runW, runReq)
	if runW.Code != http.StatusCreated {
		t.Fatalf("expected 201 run, got %d body=%s", runW.Code, runW.Body.String())
	}

	var run map[string]interface{}
	if err := json.NewDecoder(runW.Body).Decode(&run); err != nil {
		t.Fatalf("decode run: %v", err)
	}
	runID, _ := run["id"].(string)
	if runID == "" {
		t.Fatalf("expected orchestrator run id")
	}

	resumeReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/orchestrator/runs/"+runID+"/resume-from-step",
		bytes.NewReader([]byte(`{"step_id":"execute"}`)),
	)
	resumeReq.Header.Set("Content-Type", "application/json")
	resumeW := httptest.NewRecorder()
	server.Handler().ServeHTTP(resumeW, resumeReq)
	if resumeW.Code != http.StatusOK {
		t.Fatalf("expected 200 resume, got %d body=%s", resumeW.Code, resumeW.Body.String())
	}
}

func TestEvalEndpointsRunStatusAndLeaderboard(t *testing.T) {
	server, _, _, _ := setupTestServer(t)

	createReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/evals/run",
		bytes.NewReader([]byte(`{"name":"apex-suite","candidate_strategy_mode":"adaptive","baseline_strategy_mode":"deterministic","sample_size":120,"dataset_id":"apex-suite-v1","seed":42,"tasks_limit":500,"competitor_baselines":["codex","cursor"]}`)),
	)
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	server.Handler().ServeHTTP(createW, createReq)
	if createW.Code != http.StatusCreated {
		t.Fatalf("expected 201 eval run, got %d body=%s", createW.Code, createW.Body.String())
	}

	var run map[string]interface{}
	if err := json.NewDecoder(createW.Body).Decode(&run); err != nil {
		t.Fatalf("decode eval run: %v", err)
	}
	runID, _ := run["id"].(string)
	if runID == "" {
		t.Fatalf("expected eval run id")
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		statusReq := httptest.NewRequest(http.MethodGet, "/api/v1/evals/runs/"+runID, nil)
		statusW := httptest.NewRecorder()
		server.Handler().ServeHTTP(statusW, statusReq)
		if statusW.Code != http.StatusOK {
			t.Fatalf("expected 200 eval status, got %d body=%s", statusW.Code, statusW.Body.String())
		}
		var current map[string]interface{}
		if err := json.NewDecoder(statusW.Body).Decode(&current); err != nil {
			t.Fatalf("decode eval status: %v", err)
		}
		if status, _ := current["status"].(string); status == "completed" {
			if datasetID, _ := current["dataset_id"].(string); datasetID == "" {
				t.Fatalf("expected dataset_id in eval run response")
			}
			paths, _ := current["evidence_paths"].([]interface{})
			if len(paths) < 2 {
				t.Fatalf("expected evidence_paths in eval run response")
			}
			comparison, _ := current["comparison"].(map[string]interface{})
			if len(comparison) == 0 {
				t.Fatalf("expected comparison map in eval run response")
			}
			break
		}
		time.Sleep(30 * time.Millisecond)
	}

	boardReq := httptest.NewRequest(http.MethodGet, "/api/v1/evals/leaderboard", nil)
	boardW := httptest.NewRecorder()
	server.Handler().ServeHTTP(boardW, boardReq)
	if boardW.Code != http.StatusOK {
		t.Fatalf("expected 200 leaderboard, got %d body=%s", boardW.Code, boardW.Body.String())
	}
	var board []map[string]interface{}
	if err := json.NewDecoder(boardW.Body).Decode(&board); err != nil {
		t.Fatalf("decode leaderboard: %v", err)
	}
	if len(board) == 0 {
		t.Fatalf("expected at least one leaderboard entry")
	}
}

func waitForRunTerminalState(t *testing.T, manager *scheduler.RunManager, runID string) {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		r, err := manager.GetRun(runID)
		if err == nil && (r.Status == contracts.RunCompleted || r.Status == contracts.RunFailed || r.Status == contracts.RunCancelled) {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("run %s did not finish before deadline", runID)
}

func waitForRunStatus(t *testing.T, manager *scheduler.RunManager, runID string, status contracts.RunStatus, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		run, err := manager.GetRun(runID)
		if err == nil && run.Status == status {
			return
		}
		if err == nil && (run.Status == contracts.RunCompleted || run.Status == contracts.RunFailed || run.Status == contracts.RunCancelled) && run.Status != status {
			t.Fatalf("run %s reached terminal status %s before %s", runID, run.Status, status)
		}
		time.Sleep(40 * time.Millisecond)
	}
	t.Fatalf("run %s did not reach status %s before deadline", runID, status)
}

func indexOfEvent(events []contracts.Event, eventType string) int {
	for i, ev := range events {
		if ev.Type == eventType {
			return i
		}
	}
	return -1
}

type contractSchema struct {
	Required             []string                   `json:"required"`
	Properties           map[string]json.RawMessage `json:"properties"`
	AdditionalProperties interface{}                `json:"additionalProperties"`
}

type propertySchema struct {
	Enum []string `json:"enum"`
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

func loadContractSchema(t *testing.T, name string) contractSchema {
	t.Helper()
	path := locateRepoPath(t, filepath.Join("docs", "specs", "contracts", name))
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read schema %s: %v", path, err)
	}
	var schema contractSchema
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatalf("parse schema %s: %v", path, err)
	}
	return schema
}

func locateRepoPath(t *testing.T, rel string) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}

	for i := 0; i < 8; i++ {
		candidate := filepath.Join(dir, rel)
		if _, statErr := os.Stat(candidate); statErr == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	t.Fatalf("could not locate %s from test working directory", rel)
	return ""
}

func validateObjectAgainstSchema(t *testing.T, object map[string]interface{}, schema contractSchema, objectLabel string) {
	t.Helper()

	for _, key := range schema.Required {
		if _, ok := object[key]; !ok {
			t.Fatalf("%s missing required key %q", objectLabel, key)
		}
	}

	allowAdditional := true
	if flag, ok := schema.AdditionalProperties.(bool); ok {
		allowAdditional = flag
	}

	for key, value := range object {
		propDef, ok := schema.Properties[key]
		if !ok {
			if !allowAdditional {
				t.Fatalf("%s has unknown key %q but schema disallows additionalProperties", objectLabel, key)
			}
			continue
		}

		var prop propertySchema
		if err := json.Unmarshal(propDef, &prop); err != nil {
			continue
		}
		if len(prop.Enum) == 0 {
			continue
		}

		stringValue, ok := value.(string)
		if !ok {
			continue
		}
		found := false
		for _, allowed := range prop.Enum {
			if stringValue == allowed {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("%s value %q for key %q is not in schema enum %v", objectLabel, stringValue, key, prop.Enum)
		}
	}
}

func writeBlueprintFixtures(t *testing.T, workspace string) {
	t.Helper()
	mustWriteFile(t, filepath.Join(workspace, "templates/blueprints/core/BLUEPRINT.md"), `# BLUEPRINT
## 1. Strategy
## 2. Architecture
## 3. API and Integrations
## 4. Security and Compliance
## 5. CI and Deployment
## 6. Testing Strategy
## 7. Operations
## 8. Maintenance
`)
	mustWriteFile(t, filepath.Join(workspace, "templates/blueprints/core/AGENTS.md"), `# AGENTS
## Scope
## Runtime Rules
## Planning Rules
## Execution Rules
## Quality Gates
## Security Rules
## Documentation Rules
`)
	mustWriteFile(t, filepath.Join(workspace, "templates/blueprints/modules/engine.md"), "## Module: Engine Backend\n")
	mustWriteFile(t, filepath.Join(workspace, "templates/blueprints/modules/webapp.md"), "## Module: Web App SaaS\n")
	mustWriteFile(t, filepath.Join(workspace, "templates/blueprints/modules/website.md"), "## Module: Website\n")
	mustWriteFile(t, filepath.Join(workspace, "templates/blueprints/modules/ecommerce.md"), "## Module: Ecommerce\n")
	mustWriteFile(t, filepath.Join(workspace, "templates/blueprints/catalog.json"), `{
  "version": "1.0.0",
  "source": {
    "repo": "https://github.com/Delqhi/CODE-BLUEPRINTS",
    "commit": "2d562d0f6e8c519574d7ca3b57a153ad0b446596"
  },
  "profiles": [
    {
      "id": "universal-2026",
      "name": "Universal 2026",
      "version": "2026.02.1",
      "description": "Curated default blueprint profile for BIOMETRICS V3.",
      "core": {
        "blueprint_template": "templates/blueprints/core/BLUEPRINT.md",
        "agents_template": "templates/blueprints/core/AGENTS.md"
      },
      "modules": [
        {
          "id": "engine",
          "name": "Engine",
          "description": "Backend and runtime module",
          "template": "templates/blueprints/modules/engine.md"
        },
        {
          "id": "webapp",
          "name": "WebApp",
          "description": "SaaS web application module",
          "template": "templates/blueprints/modules/webapp.md"
        },
        {
          "id": "website",
          "name": "Website",
          "description": "Marketing and content site module",
          "template": "templates/blueprints/modules/website.md"
        },
        {
          "id": "ecommerce",
          "name": "Ecommerce",
          "description": "Commerce workflow module",
          "template": "templates/blueprints/modules/ecommerce.md"
        }
      ]
    }
  ]
}`)
}

func mustWriteFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
