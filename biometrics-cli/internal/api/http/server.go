package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"biometrics-cli/internal/contracts"
	"biometrics-cli/internal/evals"
	"biometrics-cli/internal/optimizer"
	"biometrics-cli/internal/runtime/background"
	"biometrics-cli/internal/runtime/bus"
	runtimeorchestrator "biometrics-cli/internal/runtime/orchestrator"
	"biometrics-cli/internal/runtime/scheduler"
	"biometrics-cli/internal/skillkit"
	"github.com/gorilla/websocket"
)

type Server struct {
	manager            *scheduler.RunManager
	backgroundAgents   *background.Manager
	bus                *bus.EventBus
	mux                *http.ServeMux
	orchestrator       *runtimeorchestrator.Service
	orchestratorV1     bool
	evals              *evals.Service
	optimizer          *optimizer.Service
	optimizerEnabled   bool
	optimizerAutoApply bool
}

const (
	defaultOrchestratorListLimit    = 20
	maxOrchestratorListLimit        = 200
	defaultOrchestratorMessageLimit = 200
	maxOrchestratorMessageLimit     = 2000
	defaultOrchestratorStreamLimit  = 200
	maxOrchestratorStreamLimit      = 2000
)

func NewServer(manager *scheduler.RunManager, eventBus *bus.EventBus) *Server {
	optimizerEnabled := boolEnvOrDefault("BIOMETRICS_OPTIMIZER_ENABLED", true)
	optimizerAutoApply := boolEnvOrDefault("BIOMETRICS_OPTIMIZER_AUTO_APPLY", false)
	return NewServerWithFlags(manager, eventBus, optimizerEnabled, optimizerAutoApply)
}

func NewServerWithFlags(manager *scheduler.RunManager, eventBus *bus.EventBus, optimizerEnabled, optimizerAutoApply bool) *Server {
	orchestratorService := runtimeorchestrator.NewService(manager, eventBus)
	evalService := evals.NewService(orchestratorService, eventBus)
	orchestratorV1 := boolEnvOrDefault("BIOCODE_ORCHESTRATOR_V1", true)
	var optimizerService *optimizer.Service
	if optimizerEnabled {
		optimizerService = optimizer.NewService(manager.Store(), orchestratorService, evalService, eventBus)
	}
	s := &Server{
		manager:            manager,
		bus:                eventBus,
		mux:                http.NewServeMux(),
		orchestrator:       orchestratorService,
		orchestratorV1:     orchestratorV1,
		evals:              evalService,
		optimizer:          optimizerService,
		optimizerEnabled:   optimizerEnabled,
		optimizerAutoApply: optimizerAutoApply,
	}
	s.routes()
	return s
}

func (s *Server) SetBackgroundAgents(manager *background.Manager) {
	s.backgroundAgents = manager
	s.orchestrator.SetBackgroundAgents(manager)
}

func (s *Server) Handler() http.Handler {
	return withCORS(s.mux)
}

func (s *Server) routes() {
	s.mux.HandleFunc("/health", s.handleHealth)
	s.mux.HandleFunc("/health/ready", s.handleHealthReady)
	s.mux.HandleFunc("/metrics", s.handleMetrics)
	s.mux.HandleFunc("/api/v1/runs", s.handleRuns)
	s.mux.HandleFunc("/api/v1/runs/", s.handleRunByID)
	s.mux.HandleFunc("/api/v1/projects", s.handleProjects)
	s.mux.HandleFunc("/api/v1/projects/", s.handleProjectByID)
	s.mux.HandleFunc("/api/v1/blueprints", s.handleBlueprints)
	s.mux.HandleFunc("/api/v1/blueprints/", s.handleBlueprintByID)
	s.mux.HandleFunc("/api/v1/models", s.handleModels)
	s.mux.HandleFunc("/api/v1/agents/background", s.handleBackgroundAgents)
	s.mux.HandleFunc("/api/v1/agents/background/", s.handleBackgroundAgentByID)
	s.mux.HandleFunc("/api/v1/auth/codex/status", s.handleCodexAuthStatus)
	s.mux.HandleFunc("/api/v1/auth/codex/login", s.handleCodexAuthLogin)
	s.mux.HandleFunc("/api/v1/auth/codex/logout", s.handleCodexAuthLogout)
	s.mux.HandleFunc("/api/v1/skills", s.handleSkills)
	s.mux.HandleFunc("/api/v1/skills/", s.handleSkillByID)
	s.mux.HandleFunc("/api/v1/skills/reload", s.handleSkillsReload)
	s.mux.HandleFunc("/api/v1/skills/install", s.handleSkillsInstall)
	s.mux.HandleFunc("/api/v1/skills/create", s.handleSkillsCreate)
	s.mux.HandleFunc("/api/v1/skills/enable", s.handleSkillsEnable)
	s.mux.HandleFunc("/api/v1/skills/disable", s.handleSkillsDisable)
	s.mux.HandleFunc("/api/v1/orchestrator/capabilities", s.handleOrchestratorCapabilities)
	s.mux.HandleFunc("/api/v1/orchestrator/plans", s.handleOrchestratorPlans)
	s.mux.HandleFunc("/api/v1/orchestrator/runs", s.handleOrchestratorRuns)
	s.mux.HandleFunc("/api/v1/orchestrator/runs/", s.handleOrchestratorRunByID)
	s.mux.HandleFunc("/api/v1/orchestrator/sessions", s.handleOrchestratorSessions)
	s.mux.HandleFunc("/api/v1/orchestrator/sessions/", s.handleOrchestratorSessionByID)
	s.mux.HandleFunc("/api/v1/orchestrator/optimizer/recommendations", s.handleOptimizerRecommendations)
	s.mux.HandleFunc("/api/v1/orchestrator/optimizer/recommendations/", s.handleOptimizerRecommendationByID)
	s.mux.HandleFunc("/api/v1/evals/run", s.handleEvalsRun)
	s.mux.HandleFunc("/api/v1/evals/runs/", s.handleEvalsRunByID)
	s.mux.HandleFunc("/api/v1/evals/leaderboard", s.handleEvalsLeaderboard)
	s.mux.HandleFunc("/api/v1/fs/tree", s.handleFSTree)
	s.mux.HandleFunc("/api/v1/fs/file", s.handleFSFile)
	s.mux.HandleFunc("/api/v1/events", s.handleEvents)
	s.mux.HandleFunc("/api/v1/ws", s.handleWS)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"service": "biometrics-controlplane-v3",
	})
}

func (s *Server) handleHealthReady(w http.ResponseWriter, _ *http.Request) {
	state := s.manager.Readiness()
	if ready, _ := state["ready"].(bool); !ready {
		writeJSON(w, http.StatusServiceUnavailable, state)
		return
	}
	writeJSON(w, http.StatusOK, state)
}

func (s *Server) handleMetrics(w http.ResponseWriter, _ *http.Request) {
	metrics := s.manager.MetricsSnapshot()
	keys := make([]string, 0, len(metrics))
	for key := range metrics {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	for _, key := range keys {
		fmt.Fprintf(w, "biometrics_%s %d\n", key, metrics[key])
	}
}

func (s *Server) handleRuns(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var req struct {
			ProjectID          string                  `json:"project_id"`
			Goal               string                  `json:"goal"`
			Mode               string                  `json:"mode"`
			Skills             []string                `json:"skills,omitempty"`
			SkillSelectionMode string                  `json:"skill_selection_mode,omitempty"`
			SchedulerMode      contracts.SchedulerMode `json:"scheduler_mode,omitempty"`
			MaxParallelism     int                     `json:"max_parallelism,omitempty"`
			ModelPreference    string                  `json:"model_preference,omitempty"`
			FallbackChain      []string                `json:"fallback_chain,omitempty"`
			ModelID            string                  `json:"model_id,omitempty"`
			ContextBudget      int                     `json:"context_budget,omitempty"`
			BlueprintProfile   string                  `json:"blueprint_profile,omitempty"`
			BlueprintModules   []string                `json:"blueprint_modules,omitempty"`
			Bootstrap          bool                    `json:"bootstrap,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		if strings.TrimSpace(req.ProjectID) == "" {
			req.ProjectID = "biometrics"
		}
		if strings.TrimSpace(req.Goal) == "" {
			writeError(w, http.StatusBadRequest, "goal is required")
			return
		}

		run, err := s.manager.StartRunWithOptions(scheduler.RunStartOptions{
			ProjectID:          req.ProjectID,
			Goal:               req.Goal,
			Mode:               req.Mode,
			Skills:             req.Skills,
			SkillSelectionMode: req.SkillSelectionMode,
			SchedulerMode:      req.SchedulerMode,
			MaxParallelism:     req.MaxParallelism,
			ModelPreference:    req.ModelPreference,
			FallbackChain:      req.FallbackChain,
			ModelID:            req.ModelID,
			ContextBudget:      req.ContextBudget,
			BlueprintProfile:   req.BlueprintProfile,
			BlueprintModules:   req.BlueprintModules,
			Bootstrap:          req.Bootstrap,
		})
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, run)
	case http.MethodGet:
		limit := 20
		if raw := r.URL.Query().Get("limit"); raw != "" {
			if parsed, err := strconv.Atoi(raw); err == nil {
				limit = parsed
			}
		}
		runs, err := s.manager.ListRecentRuns(limit)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, runs)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleRunByID(w http.ResponseWriter, r *http.Request) {
	trimmed := strings.TrimPrefix(r.URL.Path, "/api/v1/runs/")
	parts := strings.Split(strings.Trim(trimmed, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		writeError(w, http.StatusBadRequest, "run id missing")
		return
	}

	runID := parts[0]
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}

	switch {
	case r.Method == http.MethodGet && action == "":
		run, err := s.manager.GetRun(runID)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				writeError(w, http.StatusGone, "run not available")
				return
			}
			writeError(w, http.StatusNotFound, "run not found")
			return
		}
		writeJSON(w, http.StatusOK, run)
	case r.Method == http.MethodGet && action == "tasks":
		tasks, err := s.manager.ListRunTasks(runID)
		if err != nil {
			writeError(w, http.StatusNotFound, "run not found")
			return
		}
		writeJSON(w, http.StatusOK, tasks)
	case r.Method == http.MethodGet && action == "attempts":
		attempts, err := s.manager.ListRunAttempts(runID)
		if err != nil {
			writeError(w, http.StatusNotFound, "run not found")
			return
		}
		writeJSON(w, http.StatusOK, attempts)
	case r.Method == http.MethodGet && action == "graph":
		graph, err := s.manager.GetRunGraph(runID)
		if err != nil {
			writeError(w, http.StatusNotFound, "run graph not found")
			return
		}
		writeJSON(w, http.StatusOK, graph)
	case r.Method == http.MethodPost && action == "pause":
		if err := s.manager.PauseRun(runID); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		status := "paused"
		if run, err := s.manager.GetRun(runID); err == nil {
			status = string(run.Status)
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": status, "run_id": runID})
	case r.Method == http.MethodPost && action == "resume":
		if err := s.manager.ResumeRun(runID); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		status := "running"
		if run, err := s.manager.GetRun(runID); err == nil {
			status = string(run.Status)
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": status, "run_id": runID})
	case r.Method == http.MethodPost && action == "cancel":
		if err := s.manager.CancelRun(runID); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		status := "cancelled"
		if run, err := s.manager.GetRun(runID); err == nil {
			status = string(run.Status)
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": status, "run_id": runID})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleProjects(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	projects, err := s.manager.ListProjects()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, projects)
}

func (s *Server) handleProjectByID(w http.ResponseWriter, r *http.Request) {
	trimmed := strings.TrimPrefix(r.URL.Path, "/api/v1/projects/")
	parts := strings.Split(strings.Trim(trimmed, "/"), "/")
	if len(parts) == 0 || strings.TrimSpace(parts[0]) == "" {
		writeError(w, http.StatusBadRequest, "project id missing")
		return
	}

	projectID := parts[0]
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}

	if r.Method != http.MethodPost || action != "bootstrap" {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		BlueprintProfile string   `json:"blueprint_profile,omitempty"`
		BlueprintModules []string `json:"blueprint_modules,omitempty"`
	}
	if err := decodeOptionalJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	result, err := s.manager.BootstrapProject(projectID, req.BlueprintProfile, req.BlueprintModules)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleBlueprints(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	profiles, err := s.manager.ListBlueprintProfiles()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, profiles)
}

func (s *Server) handleBlueprintByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	profileID := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/blueprints/"), "/")
	if strings.TrimSpace(profileID) == "" {
		writeError(w, http.StatusBadRequest, "blueprint profile missing")
		return
	}

	profile, err := s.manager.GetBlueprintProfile(profileID)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, profile)
}

func (s *Server) handleModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	catalog := s.manager.ModelsCatalog(r.Context())
	writeJSON(w, http.StatusOK, catalog)
}

func (s *Server) handleBackgroundAgents(w http.ResponseWriter, r *http.Request) {
	if s.backgroundAgents == nil {
		writeError(w, http.StatusServiceUnavailable, "background agents are not configured")
		return
	}

	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, s.backgroundAgents.List())
	case http.MethodPost:
		var req struct {
			ProjectID       string   `json:"project_id,omitempty"`
			Agent           string   `json:"agent,omitempty"`
			Prompt          string   `json:"prompt"`
			ModelPreference string   `json:"model_preference,omitempty"`
			FallbackChain   []string `json:"fallback_chain,omitempty"`
			ModelID         string   `json:"model_id,omitempty"`
			ContextBudget   int      `json:"context_budget,omitempty"`
		}
		if err := decodeOptionalJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		if strings.TrimSpace(req.Prompt) == "" {
			writeError(w, http.StatusBadRequest, "prompt is required")
			return
		}

		job, err := s.backgroundAgents.Start(r.Context(), background.StartRequest{
			ProjectID:       req.ProjectID,
			Agent:           req.Agent,
			Prompt:          req.Prompt,
			ModelPreference: req.ModelPreference,
			FallbackChain:   req.FallbackChain,
			ModelID:         req.ModelID,
			ContextBudget:   req.ContextBudget,
		})
		if err != nil {
			if errors.Is(err, background.ErrNotConfigured) {
				writeError(w, http.StatusServiceUnavailable, err.Error())
				return
			}
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, job)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleBackgroundAgentByID(w http.ResponseWriter, r *http.Request) {
	if s.backgroundAgents == nil {
		writeError(w, http.StatusServiceUnavailable, "background agents are not configured")
		return
	}

	trimmed := strings.TrimPrefix(r.URL.Path, "/api/v1/agents/background/")
	parts := strings.Split(strings.Trim(trimmed, "/"), "/")
	if len(parts) == 0 || strings.TrimSpace(parts[0]) == "" {
		writeError(w, http.StatusBadRequest, "background job id missing")
		return
	}

	jobID := parts[0]
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}

	switch {
	case r.Method == http.MethodGet && action == "":
		job, ok := s.backgroundAgents.Get(jobID)
		if !ok {
			writeError(w, http.StatusNotFound, "background job not found")
			return
		}
		writeJSON(w, http.StatusOK, job)
	case r.Method == http.MethodPost && action == "cancel":
		job, err := s.backgroundAgents.Cancel(jobID)
		if err != nil {
			if errors.Is(err, background.ErrNotFound) {
				writeError(w, http.StatusNotFound, "background job not found")
				return
			}
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, job)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleCodexAuthStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	status, err := s.manager.CodexAuthStatus(r.Context())
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (s *Server) handleCodexAuthLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	_, _ = s.bus.Publish(contracts.Event{
		Type:   "auth.codex.login.started",
		Source: "api.http",
	})
	status, err := s.manager.CodexAuthLogin(r.Context())
	if err != nil {
		_, _ = s.bus.Publish(contracts.Event{
			Type:   "auth.codex.login.failed",
			Source: "api.http",
			Payload: map[string]string{
				"error": err.Error(),
			},
		})
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	_, _ = s.bus.Publish(contracts.Event{
		Type:   "auth.codex.login.succeeded",
		Source: "api.http",
		Payload: map[string]string{
			"ready":     strconv.FormatBool(status.Ready),
			"logged_in": strconv.FormatBool(status.LoggedIn),
			"user":      status.User,
		},
	})
	writeJSON(w, http.StatusOK, status)
}

func (s *Server) handleCodexAuthLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	status, err := s.manager.CodexAuthLogout(r.Context())
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (s *Server) handleSkills(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	skills, err := s.manager.ListSkills()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, skills)
}

func (s *Server) handleSkillByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	name := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/skills/"), "/")
	if name == "" {
		writeError(w, http.StatusBadRequest, "skill name missing")
		return
	}
	skill, err := s.manager.GetSkill(name)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, skill)
}

func (s *Server) handleSkillsReload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	outcome, err := s.manager.ReloadSkills()
	if err != nil {
		writeJSON(w, http.StatusBadRequest, outcome)
		return
	}
	writeJSON(w, http.StatusOK, outcome)
}

func (s *Server) handleSkillsInstall(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req skillkit.InstallRequest
	if err := decodeOptionalJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	result, err := s.manager.InstallSkill(r.Context(), req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, result)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleSkillsCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req skillkit.CreateRequest
	if err := decodeOptionalJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	result, err := s.manager.CreateSkill(r.Context(), req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, result)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleSkillsEnable(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		Name string `json:"name,omitempty"`
		Path string `json:"path,omitempty"`
	}
	if err := decodeOptionalJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	reference := strings.TrimSpace(req.Path)
	if reference == "" {
		reference = strings.TrimSpace(req.Name)
	}
	if reference == "" {
		writeError(w, http.StatusBadRequest, "name or path is required")
		return
	}
	result, err := s.manager.EnableSkill(reference)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, result)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleSkillsDisable(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		Name string `json:"name,omitempty"`
		Path string `json:"path,omitempty"`
	}
	if err := decodeOptionalJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	reference := strings.TrimSpace(req.Path)
	if reference == "" {
		reference = strings.TrimSpace(req.Name)
	}
	if reference == "" {
		writeError(w, http.StatusBadRequest, "name or path is required")
		return
	}
	result, err := s.manager.DisableSkill(reference)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, result)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleOrchestratorCapabilities(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, s.orchestrator.Capabilities())
}

func (s *Server) handleOrchestratorPlans(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req runtimeorchestrator.PlanRequest
	if err := decodeOptionalJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	plan, err := s.orchestrator.CreatePlan(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, plan)
}

func (s *Server) handleOrchestratorRuns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req runtimeorchestrator.RunRequest
	if err := decodeOptionalJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	run, err := s.orchestrator.StartRun(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, run)
}

func (s *Server) handleOrchestratorRunByID(w http.ResponseWriter, r *http.Request) {
	trimmed := strings.TrimPrefix(r.URL.Path, "/api/v1/orchestrator/runs/")
	parts := strings.Split(strings.Trim(trimmed, "/"), "/")
	if len(parts) == 0 || strings.TrimSpace(parts[0]) == "" {
		writeError(w, http.StatusBadRequest, "run id missing")
		return
	}

	runID := parts[0]
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}

	switch {
	case r.Method == http.MethodGet && action == "":
		run, err := s.orchestrator.GetRun(runID)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, run)
	case r.Method == http.MethodGet && action == "scorecard":
		scorecard, err := s.orchestrator.Scorecard(runID)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, scorecard)
	case r.Method == http.MethodPost && action == "resume-from-step":
		var req struct {
			StepID string `json:"step_id"`
		}
		if err := decodeOptionalJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		run, err := s.orchestrator.ResumeFromStep(r.Context(), runID, req.StepID)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, run)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleOrchestratorSessions(w http.ResponseWriter, r *http.Request) {
	if !s.orchestratorV1 {
		writeError(w, http.StatusNotFound, "orchestrator sessions are disabled")
		return
	}

	switch r.Method {
	case http.MethodPost:
		var req runtimeorchestrator.OrchestratorSessionCreateRequest
		if err := decodeOptionalJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		session, err := s.orchestrator.CreateSession(req)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, session)
	case http.MethodGet:
		projectID := strings.TrimSpace(r.URL.Query().Get("project_id"))
		limit := parseBoundedIntQuery(r.URL.Query().Get("limit"), defaultOrchestratorListLimit, 1, maxOrchestratorListLimit)
		sessions, err := s.orchestrator.ListSessions(projectID, limit)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, sessions)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleOrchestratorSessionByID(w http.ResponseWriter, r *http.Request) {
	if !s.orchestratorV1 {
		writeError(w, http.StatusNotFound, "orchestrator sessions are disabled")
		return
	}

	trimmed := strings.TrimPrefix(r.URL.Path, "/api/v1/orchestrator/sessions/")
	parts := strings.Split(strings.Trim(trimmed, "/"), "/")
	if len(parts) == 0 || strings.TrimSpace(parts[0]) == "" {
		writeError(w, http.StatusBadRequest, "session id missing")
		return
	}
	sessionID := strings.TrimSpace(parts[0])
	action := ""
	if len(parts) > 1 {
		action = strings.TrimSpace(parts[1])
	}

	switch {
	case r.Method == http.MethodGet && action == "":
		session, err := s.orchestrator.GetSession(sessionID)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, session)
	case r.Method == http.MethodGet && action == "messages":
		cursor := parseNonNegativeInt64Query(r.URL.Query().Get("cursor"))
		limit := parseBoundedIntQuery(r.URL.Query().Get("limit"), defaultOrchestratorMessageLimit, 1, maxOrchestratorMessageLimit)
		messages, err := s.orchestrator.ListSessionMessages(sessionID, cursor, limit)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, messages)
	case r.Method == http.MethodPost && action == "messages":
		var req runtimeorchestrator.OrchestratorMessageAppendRequest
		if err := decodeOptionalJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		message, err := s.orchestrator.AppendSessionMessage(sessionID, req)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, message)
	case r.Method == http.MethodGet && action == "stream":
		if _, err := s.orchestrator.GetSession(sessionID); err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		cursor := parseNonNegativeInt64Query(r.URL.Query().Get("cursor"))
		limit := parseBoundedIntQuery(r.URL.Query().Get("limit"), defaultOrchestratorStreamLimit, 1, maxOrchestratorStreamLimit)
		s.streamOrchestratorSessionEvents(w, r, sessionID, cursor, limit)
	case r.Method == http.MethodPost && action == "pause":
		var req struct {
			Reason string `json:"reason,omitempty"`
		}
		if err := decodeOptionalJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		session, err := s.orchestrator.PauseSession(sessionID, req.Reason)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, session)
	case r.Method == http.MethodPost && action == "resume":
		session, err := s.orchestrator.ResumeSession(sessionID)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, session)
	case r.Method == http.MethodPost && action == "kill":
		session, err := s.orchestrator.KillSession(sessionID)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, session)
	case r.Method == http.MethodPost && action == "agents" && len(parts) >= 4 && strings.EqualFold(parts[3], "model"):
		agentID := parts[2]
		var req runtimeorchestrator.OrchestratorAgentModelOverrideRequest
		if err := decodeOptionalJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		state, err := s.orchestrator.SetSessionAgentModel(sessionID, agentID, req)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, state)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) streamOrchestratorSessionEvents(
	w http.ResponseWriter,
	r *http.Request,
	sessionID string,
	cursor int64,
	limit int,
) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "stream unsupported")
		return
	}

	history, err := s.bus.Replay(sessionID, limit*4)
	if err == nil {
		for _, ev := range history {
			if ev.RunID != sessionID {
				continue
			}
			if !strings.HasPrefix(ev.Type, "orchestrator.") {
				continue
			}
			if shouldSkipOrchestratorEventAtCursor(ev.Payload, cursor) {
				continue
			}
			writeSSE(w, ev)
		}
		flusher.Flush()
	}

	subID, ch := s.bus.Subscribe(128)
	defer s.bus.Unsubscribe(subID)
	pingTicker := time.NewTicker(15 * time.Second)
	defer pingTicker.Stop()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case <-pingTicker.C:
			fmt.Fprintf(w, ": ping\n\n")
			flusher.Flush()
		case ev, ok := <-ch:
			if !ok {
				return
			}
			if ev.RunID != sessionID {
				continue
			}
			if !strings.HasPrefix(ev.Type, "orchestrator.") {
				continue
			}
			if shouldSkipOrchestratorEventAtCursor(ev.Payload, cursor) {
				continue
			}
			writeSSE(w, ev)
			flusher.Flush()
		}
	}
}

func parseBoundedIntQuery(raw string, fallback, min, max int) int {
	value := fallback
	if parsed, err := strconv.Atoi(strings.TrimSpace(raw)); err == nil {
		value = parsed
	}
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func parseNonNegativeInt64Query(raw string) int64 {
	parsed, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || parsed < 0 {
		return 0
	}
	return parsed
}

func parseOrchestratorEventCursor(payload map[string]string) int64 {
	if len(payload) == 0 {
		return 0
	}
	if raw := strings.TrimSpace(payload["cursor"]); raw != "" {
		if parsed, err := strconv.ParseInt(raw, 10, 64); err == nil {
			return parsed
		}
	}
	if raw := strings.TrimSpace(payload["message_cursor"]); raw != "" {
		if parsed, err := strconv.ParseInt(raw, 10, 64); err == nil {
			return parsed
		}
	}
	return 0
}

func shouldSkipOrchestratorEventAtCursor(payload map[string]string, streamCursor int64) bool {
	if streamCursor <= 0 || len(payload) == 0 {
		return false
	}

	eventCursor, hasEventCursor := parsePositiveInt64Payload(payload, "cursor")
	messageCursor, hasMessageCursor := parsePositiveInt64Payload(payload, "message_cursor")

	switch {
	case hasEventCursor && hasMessageCursor:
		return eventCursor <= streamCursor && messageCursor <= streamCursor
	case hasEventCursor:
		return eventCursor <= streamCursor
	case hasMessageCursor:
		return messageCursor <= streamCursor
	default:
		return false
	}
}

func parsePositiveInt64Payload(payload map[string]string, key string) (int64, bool) {
	raw := strings.TrimSpace(payload[key])
	if raw == "" {
		return 0, false
	}
	parsed, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || parsed <= 0 {
		return 0, false
	}
	return parsed, true
}

func (s *Server) handleOptimizerRecommendations(w http.ResponseWriter, r *http.Request) {
	if s.optimizer == nil {
		writeError(w, http.StatusServiceUnavailable, "optimizer is not configured")
		return
	}

	switch r.Method {
	case http.MethodPost:
		var req optimizer.GenerateRequest
		if err := decodeOptionalJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		rec, err := s.optimizer.GenerateRecommendation(r.Context(), req)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, rec)
	case http.MethodGet:
		limit := 20
		if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
			if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
				limit = parsed
			}
		}
		recs, err := s.optimizer.ListRecommendations(optimizer.ListOptions{
			ProjectID: strings.TrimSpace(r.URL.Query().Get("project_id")),
			Status:    strings.TrimSpace(r.URL.Query().Get("status")),
			Limit:     limit,
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, recs)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleOptimizerRecommendationByID(w http.ResponseWriter, r *http.Request) {
	if s.optimizer == nil {
		writeError(w, http.StatusServiceUnavailable, "optimizer is not configured")
		return
	}

	trimmed := strings.TrimPrefix(r.URL.Path, "/api/v1/orchestrator/optimizer/recommendations/")
	parts := strings.Split(strings.Trim(trimmed, "/"), "/")
	if len(parts) == 0 || strings.TrimSpace(parts[0]) == "" {
		writeError(w, http.StatusBadRequest, "recommendation id missing")
		return
	}

	recommendationID := parts[0]
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}

	switch {
	case r.Method == http.MethodGet && action == "":
		rec, err := s.optimizer.GetRecommendation(recommendationID)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, rec)
	case r.Method == http.MethodPost && action == "apply":
		result, err := s.optimizer.ApplyRecommendation(r.Context(), recommendationID)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, result)
	case r.Method == http.MethodPost && action == "reject":
		var req struct {
			Reason string `json:"reason,omitempty"`
		}
		if err := decodeOptionalJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		rec, err := s.optimizer.RejectRecommendation(recommendationID, req.Reason)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, rec)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleEvalsRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req evals.RunRequest
	if err := decodeOptionalJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	run, err := s.evals.StartRun(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, run)
}

func (s *Server) handleEvalsRunByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	runID := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/evals/runs/"), "/")
	if runID == "" {
		writeError(w, http.StatusBadRequest, "eval run id missing")
		return
	}
	run, err := s.evals.GetRun(runID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, run)
}

func (s *Server) handleEvalsLeaderboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, s.evals.Leaderboard())
}

func (s *Server) handleFSTree(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		path = "."
	}
	entries, err := s.manager.ListDir(path)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, entries)
}

func (s *Server) handleFSFile(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if strings.TrimSpace(path) == "" {
		writeError(w, http.StatusBadRequest, "path is required")
		return
	}
	data, err := s.manager.ReadFile(path)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write(data)
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	runID := r.URL.Query().Get("run_id")
	limit := 200
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = parsed
		}
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "stream unsupported")
		return
	}

	history, err := s.manager.Events(runID, limit)
	if err == nil {
		for _, ev := range history {
			if runID != "" && ev.RunID != runID {
				continue
			}
			writeSSE(w, ev)
		}
		flusher.Flush()
	}

	subID, ch := s.bus.Subscribe(128)
	defer s.bus.Unsubscribe(subID)

	pingTicker := time.NewTicker(15 * time.Second)
	defer pingTicker.Stop()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case <-pingTicker.C:
			fmt.Fprintf(w, ": ping\n\n")
			flusher.Flush()
		case ev, ok := <-ch:
			if !ok {
				return
			}
			if runID != "" && ev.RunID != runID {
				continue
			}
			writeSSE(w, ev)
			flusher.Flush()
		}
	}
}

var upgrader = websocket.Upgrader{CheckOrigin: func(_ *http.Request) bool { return true }}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	runID := r.URL.Query().Get("run_id")
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	history, _ := s.manager.Events(runID, 200)
	for _, ev := range history {
		if runID != "" && ev.RunID != runID {
			continue
		}
		if err := conn.WriteJSON(ev); err != nil {
			return
		}
	}

	subID, ch := s.bus.Subscribe(128)
	defer s.bus.Unsubscribe(subID)

	for {
		select {
		case <-r.Context().Done():
			return
		case ev, ok := <-ch:
			if !ok {
				return
			}
			if runID != "" && ev.RunID != runID {
				continue
			}
			if err := conn.WriteJSON(ev); err != nil {
				return
			}
		}
	}
}

func decodeOptionalJSON(r *http.Request, target interface{}) error {
	if r.Body == nil {
		return nil
	}
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(target); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return err
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func writeSSE(w http.ResponseWriter, event contracts.Event) {
	data, _ := json.Marshal(event)
	if event.ID != "" {
		fmt.Fprintf(w, "id: %s\n", event.ID)
	}
	fmt.Fprintf(w, "event: %s\n", event.Type)
	fmt.Fprintf(w, "data: %s\n\n", string(data))

	// Compatibility frame for message-based consumers using `onmessage`.
	if event.ID != "" {
		fmt.Fprintf(w, "id: %s\n", event.ID)
	}
	fmt.Fprintf(w, "event: message\n")
	fmt.Fprintf(w, "data: %s\n\n", string(data))
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func boolEnvOrDefault(name string, fallback bool) bool {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	switch strings.ToLower(raw) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}
