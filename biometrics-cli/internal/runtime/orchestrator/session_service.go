package orchestrator

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"biometrics-cli/internal/runtime/background"
	store "biometrics-cli/internal/store/sqlite"
)

const (
	sessionDefaultMaxJobs             = 60
	sessionCooldown                   = 5 * time.Second
	sessionErrorStormWindow           = 3 * time.Minute
	sessionErrorStormFailureThreshold = 3
)

var sessionAgentsInOrder = []string{
	OrchestratorPaneBackend,
	OrchestratorPaneFrontend,
	OrchestratorPaneOrchestrator,
}

var defaultSessionModels = map[string]OrchestratorAgentModel{
	OrchestratorPaneBackend: {
		Provider: "nim",
		ModelID:  "qwen-3.5-397b",
	},
	OrchestratorPaneFrontend: {
		Provider: "gemini",
		ModelID:  "google/gemini-3-flash",
	},
	OrchestratorPaneOrchestrator: {
		Provider: "gemini",
		ModelID:  "gemini-3.1-pro-preview",
	},
}

type sessionStore interface {
	CreateOrchestratorSession(rec store.OrchestratorSessionRecord) (store.OrchestratorSessionRecord, error)
	GetOrchestratorSession(sessionID string) (store.OrchestratorSessionRecord, error)
	ListOrchestratorSessions(projectID string, limit int) ([]store.OrchestratorSessionRecord, error)
	UpdateOrchestratorSessionLifecycle(sessionID, status string, guardrailPaused bool, guardrailReason string) (store.OrchestratorSessionRecord, error)
	IncrementOrchestratorSessionJobsStarted(sessionID string) (store.OrchestratorSessionRecord, error)
	AppendOrchestratorMessage(msg store.OrchestratorMessageRecord) (store.OrchestratorMessageRecord, error)
	ListOrchestratorMessages(sessionID string, afterCursor int64, limit int) ([]store.OrchestratorMessageRecord, error)
	UpsertOrchestratorAgentState(state store.OrchestratorAgentStateRecord) (store.OrchestratorAgentStateRecord, error)
	GetOrchestratorAgentState(sessionID, agentID string) (store.OrchestratorAgentStateRecord, error)
	ListOrchestratorAgentStates(sessionID string) ([]store.OrchestratorAgentStateRecord, error)
	CreateOrchestratorJobLink(link store.OrchestratorJobLinkRecord) (store.OrchestratorJobLinkRecord, error)
	UpdateOrchestratorJobLinkStatus(sessionID, jobID, jobStatus string) error
	CountFailedOrchestratorJobsSince(sessionID string, since time.Time) (int, error)
}

type sessionStoreProvider interface {
	Store() *store.Store
}

type sessionBackgroundGateway interface {
	Start(ctx context.Context, req background.StartRequest) (background.Job, error)
	Get(jobID string) (background.Job, bool)
}

type sessionRuntimeState struct {
	lastTriggeredCursor map[string]int64
	nextEventCursor     int64
}

func (s *Service) initSessionDependencies(backend runBackend) {
	provider, ok := backend.(sessionStoreProvider)
	if !ok {
		return
	}
	s.sessionStore = provider.Store()
}

func (s *Service) SetBackgroundAgents(manager sessionBackgroundGateway) {
	s.sessionMu.Lock()
	defer s.sessionMu.Unlock()
	s.background = manager
}

func (s *Service) CreateSession(req OrchestratorSessionCreateRequest) (OrchestratorSession, error) {
	if s.sessionStore == nil {
		return OrchestratorSession{}, fmt.Errorf("orchestrator session store is not configured")
	}

	projectID := strings.TrimSpace(req.ProjectID)
	if projectID == "" {
		projectID = "biometrics"
	}
	maxJobs := req.MaxJobs
	if maxJobs <= 0 {
		maxJobs = sessionDefaultMaxJobs
	}

	rec, err := s.sessionStore.CreateOrchestratorSession(store.OrchestratorSessionRecord{
		ProjectID: projectID,
		Status:    store.OrchestratorSessionStatusActive,
		MaxJobs:   maxJobs,
	})
	if err != nil {
		return OrchestratorSession{}, err
	}

	for _, agentID := range sessionAgentsInOrder {
		model := defaultModelForSessionAgent(agentID)
		if override, ok := req.AgentModels[agentID]; ok {
			model = normalizeModelOverride(override, model)
		}
		_, err := s.sessionStore.UpsertOrchestratorAgentState(store.OrchestratorAgentStateRecord{
			SessionID:     rec.ID,
			AgentID:       agentID,
			Status:        "idle",
			ModelProvider: model.Provider,
			ModelID:       model.ModelID,
		})
		if err != nil {
			return OrchestratorSession{}, err
		}
	}

	s.sessionMu.Lock()
	s.sessionRuntime[rec.ID] = &sessionRuntimeState{
		lastTriggeredCursor: make(map[string]int64, len(sessionAgentsInOrder)),
	}
	s.sessionMu.Unlock()

	s.publishSessionEvent(rec.ID, "orchestrator.session.created", map[string]string{
		"project_id": rec.ProjectID,
		"status":     rec.Status,
		"max_jobs":   strconv.Itoa(rec.MaxJobs),
	})

	return s.GetSession(rec.ID)
}

func (s *Service) ListSessions(projectID string, limit int) ([]OrchestratorSession, error) {
	if s.sessionStore == nil {
		return nil, fmt.Errorf("orchestrator session store is not configured")
	}
	records, err := s.sessionStore.ListOrchestratorSessions(projectID, limit)
	if err != nil {
		return nil, err
	}
	out := make([]OrchestratorSession, 0, len(records))
	for _, rec := range records {
		_ = s.ensureDefaultAgentStates(rec.ID)
		states, err := s.sessionStore.ListOrchestratorAgentStates(rec.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, orchestratorSessionFromRecord(rec, states))
	}
	return out, nil
}

func (s *Service) GetSession(sessionID string) (OrchestratorSession, error) {
	if s.sessionStore == nil {
		return OrchestratorSession{}, fmt.Errorf("orchestrator session store is not configured")
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return OrchestratorSession{}, fmt.Errorf("session id is required")
	}
	rec, err := s.sessionStore.GetOrchestratorSession(sessionID)
	if err != nil {
		return OrchestratorSession{}, err
	}
	if err := s.ensureDefaultAgentStates(sessionID); err != nil {
		return OrchestratorSession{}, err
	}
	states, err := s.sessionStore.ListOrchestratorAgentStates(sessionID)
	if err != nil {
		return OrchestratorSession{}, err
	}
	return orchestratorSessionFromRecord(rec, states), nil
}

func (s *Service) ListSessionMessages(sessionID string, afterCursor int64, limit int) ([]OrchestratorMessage, error) {
	if s.sessionStore == nil {
		return nil, fmt.Errorf("orchestrator session store is not configured")
	}
	messages, err := s.sessionStore.ListOrchestratorMessages(sessionID, afterCursor, limit)
	if err != nil {
		return nil, err
	}
	out := make([]OrchestratorMessage, 0, len(messages))
	for _, msg := range messages {
		out = append(out, orchestratorMessageFromRecord(msg))
	}
	return out, nil
}

func (s *Service) AppendSessionMessage(sessionID string, req OrchestratorMessageAppendRequest) (OrchestratorMessage, error) {
	if s.sessionStore == nil {
		return OrchestratorMessage{}, fmt.Errorf("orchestrator session store is not configured")
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return OrchestratorMessage{}, fmt.Errorf("session id is required")
	}
	content := strings.TrimSpace(req.Content)
	if content == "" {
		return OrchestratorMessage{}, fmt.Errorf("content is required")
	}

	rec, err := s.sessionStore.GetOrchestratorSession(sessionID)
	if err != nil {
		return OrchestratorMessage{}, err
	}
	if rec.Status == store.OrchestratorSessionStatusKilled {
		return OrchestratorMessage{}, fmt.Errorf("session is killed")
	}

	authorKind := normalizeSessionAuthorKind(req.AuthorKind)
	authorID := normalizeSessionAuthorID(req.AuthorID)
	if authorKind == OrchestratorAuthorKindUser {
		authorID = "user"
	}
	targetPane := normalizeSessionPane(req.TargetPane)

	storedMessage, err := s.sessionStore.AppendOrchestratorMessage(store.OrchestratorMessageRecord{
		SessionID:  sessionID,
		AuthorKind: authorKind,
		AuthorID:   authorID,
		TargetPane: targetPane,
		Content:    content,
	})
	if err != nil {
		return OrchestratorMessage{}, err
	}
	message := orchestratorMessageFromRecord(storedMessage)

	s.publishSessionEvent(sessionID, "orchestrator.message.created", map[string]string{
		"message_cursor": strconv.FormatInt(message.Cursor, 10),
		"author_kind":    message.AuthorKind,
		"author_id":      message.AuthorID,
		"target_pane":    message.TargetPane,
		"content":        message.Content,
	})

	if shouldTriggerAgents(message) {
		go s.triggerAgentsForMessage(message)
	}

	return message, nil
}

func (s *Service) PauseSession(sessionID string, reason string) (OrchestratorSession, error) {
	updated, err := s.updateSessionLifecycle(sessionID, store.OrchestratorSessionStatusPaused, true, reason, "orchestrator.guardrail.paused")
	if err != nil {
		return OrchestratorSession{}, err
	}
	return s.GetSession(updated.ID)
}

func (s *Service) ResumeSession(sessionID string) (OrchestratorSession, error) {
	updated, err := s.updateSessionLifecycle(sessionID, store.OrchestratorSessionStatusActive, false, "", "orchestrator.session.resumed")
	if err != nil {
		return OrchestratorSession{}, err
	}
	for _, agentID := range sessionAgentsInOrder {
		state, stateErr := s.getOrDefaultAgentState(sessionID, agentID)
		if stateErr != nil {
			return OrchestratorSession{}, stateErr
		}
		state.Status = "idle"
		state.LastError = ""
		state.CooldownUntil = nil
		if _, stateErr = s.sessionStore.UpsertOrchestratorAgentState(state); stateErr != nil {
			return OrchestratorSession{}, stateErr
		}
	}
	return s.GetSession(updated.ID)
}

func (s *Service) KillSession(sessionID string) (OrchestratorSession, error) {
	updated, err := s.updateSessionLifecycle(sessionID, store.OrchestratorSessionStatusKilled, true, "kill_switch", "orchestrator.session.killed")
	if err != nil {
		return OrchestratorSession{}, err
	}
	now := time.Now().UTC()
	for _, agentID := range sessionAgentsInOrder {
		state, stateErr := s.getOrDefaultAgentState(sessionID, agentID)
		if stateErr != nil {
			return OrchestratorSession{}, stateErr
		}
		state.Status = "paused"
		state.LastError = "killed by operator"
		state.CooldownUntil = &now
		if _, stateErr = s.sessionStore.UpsertOrchestratorAgentState(state); stateErr != nil {
			return OrchestratorSession{}, stateErr
		}
	}
	return s.GetSession(updated.ID)
}

func (s *Service) SetSessionAgentModel(sessionID, agentID string, req OrchestratorAgentModelOverrideRequest) (OrchestratorAgentState, error) {
	if s.sessionStore == nil {
		return OrchestratorAgentState{}, fmt.Errorf("orchestrator session store is not configured")
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return OrchestratorAgentState{}, fmt.Errorf("session id is required")
	}
	agentID = normalizeSessionAgentID(agentID)
	if _, err := s.sessionStore.GetOrchestratorSession(sessionID); err != nil {
		return OrchestratorAgentState{}, err
	}

	state, err := s.getOrDefaultAgentState(sessionID, agentID)
	if err != nil {
		return OrchestratorAgentState{}, err
	}
	defaultModel := defaultModelForSessionAgent(agentID)
	model := normalizeModelOverride(OrchestratorAgentModel{Provider: req.Provider, ModelID: req.ModelID}, defaultModel)
	state.ModelProvider = model.Provider
	state.ModelID = model.ModelID
	updated, err := s.sessionStore.UpsertOrchestratorAgentState(state)
	if err != nil {
		return OrchestratorAgentState{}, err
	}

	s.publishSessionEvent(sessionID, "orchestrator.agent.model.updated", map[string]string{
		"agent_id": agentID,
		"provider": model.Provider,
		"model_id": model.ModelID,
	})
	return orchestratorAgentStateFromRecord(updated), nil
}

func (s *Service) updateSessionLifecycle(sessionID, status string, guardrailPaused bool, guardrailReason string, eventType string) (store.OrchestratorSessionRecord, error) {
	if s.sessionStore == nil {
		return store.OrchestratorSessionRecord{}, fmt.Errorf("orchestrator session store is not configured")
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return store.OrchestratorSessionRecord{}, fmt.Errorf("session id is required")
	}
	rec, err := s.sessionStore.UpdateOrchestratorSessionLifecycle(sessionID, status, guardrailPaused, guardrailReason)
	if err != nil {
		return store.OrchestratorSessionRecord{}, err
	}
	payload := map[string]string{
		"status":           rec.Status,
		"guardrail_paused": strconv.FormatBool(rec.GuardrailPaused),
		"guardrail_reason": rec.GuardrailReason,
	}
	s.publishSessionEvent(sessionID, eventType, payload)
	return rec, nil
}

func (s *Service) ensureDefaultAgentStates(sessionID string) error {
	if s.sessionStore == nil {
		return fmt.Errorf("orchestrator session store is not configured")
	}
	for _, agentID := range sessionAgentsInOrder {
		_, err := s.sessionStore.GetOrchestratorAgentState(sessionID, agentID)
		if err == nil {
			continue
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		model := defaultModelForSessionAgent(agentID)
		if _, err := s.sessionStore.UpsertOrchestratorAgentState(store.OrchestratorAgentStateRecord{
			SessionID:     sessionID,
			AgentID:       agentID,
			Status:        "idle",
			ModelProvider: model.Provider,
			ModelID:       model.ModelID,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) getOrDefaultAgentState(sessionID, agentID string) (store.OrchestratorAgentStateRecord, error) {
	state, err := s.sessionStore.GetOrchestratorAgentState(sessionID, agentID)
	if err == nil {
		return state, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return store.OrchestratorAgentStateRecord{}, err
	}
	model := defaultModelForSessionAgent(agentID)
	return s.sessionStore.UpsertOrchestratorAgentState(store.OrchestratorAgentStateRecord{
		SessionID:     sessionID,
		AgentID:       agentID,
		Status:        "idle",
		ModelProvider: model.Provider,
		ModelID:       model.ModelID,
	})
}

func (s *Service) triggerAgentsForMessage(msg OrchestratorMessage) {
	if s.sessionStore == nil {
		return
	}
	agents := inferAgentsToTrigger(msg)
	for _, agentID := range agents {
		s.triggerAgent(msg, agentID)
	}
}

func (s *Service) triggerAgent(msg OrchestratorMessage, agentID string) {
	agentID = normalizeSessionAgentID(agentID)

	s.sessionMu.Lock()
	runtimeState, ok := s.sessionRuntime[msg.SessionID]
	if !ok {
		runtimeState = &sessionRuntimeState{lastTriggeredCursor: make(map[string]int64, len(sessionAgentsInOrder))}
		s.sessionRuntime[msg.SessionID] = runtimeState
	}
	if runtimeState.lastTriggeredCursor[agentID] >= msg.Cursor {
		s.sessionMu.Unlock()
		return
	}
	runtimeState.lastTriggeredCursor[agentID] = msg.Cursor
	bg := s.background
	s.sessionMu.Unlock()

	session, err := s.sessionStore.GetOrchestratorSession(msg.SessionID)
	if err != nil {
		return
	}
	if session.Status != store.OrchestratorSessionStatusActive || session.GuardrailPaused {
		return
	}
	if session.JobsStarted >= session.MaxJobs {
		_, _ = s.PauseSession(session.ID, "budget_limit_reached")
		return
	}

	state, err := s.getOrDefaultAgentState(session.ID, agentID)
	if err != nil {
		return
	}
	now := time.Now().UTC()
	if state.CooldownUntil != nil && state.CooldownUntil.After(now) {
		return
	}
	if state.Status == "running" {
		return
	}

	state.Status = "running"
	state.LastError = ""
	state.LastActiveAt = &now
	state.CooldownUntil = nil
	if _, err := s.sessionStore.UpsertOrchestratorAgentState(state); err != nil {
		return
	}

	s.publishSessionEvent(session.ID, "orchestrator.agent.triggered", map[string]string{
		"agent_id":       agentID,
		"trigger_cursor": strconv.FormatInt(msg.Cursor, 10),
		"trigger_kind":   msg.AuthorKind,
	})

	if bg == nil {
		s.appendAgentMessage(session.ID, agentID, fmt.Sprintf("%s acknowledged and is waiting for background workers.", displayAgentName(agentID)))
		s.markAgentIdleWithCooldown(session.ID, agentID, now.Add(sessionCooldown), "")
		return
	}

	prompt := s.buildAgentPrompt(session.ID, agentID, msg)
	job, err := bg.Start(context.Background(), background.StartRequest{
		ProjectID:       session.ProjectID,
		Agent:           backgroundAgentForPane(agentID),
		Prompt:          prompt,
		ModelPreference: state.ModelProvider,
		ModelID:         state.ModelID,
		ContextBudget:   24000,
	})
	if err != nil {
		s.handleAgentJobFailure(session.ID, agentID, "", err)
		return
	}

	_, _ = s.sessionStore.IncrementOrchestratorSessionJobsStarted(session.ID)
	_, _ = s.sessionStore.CreateOrchestratorJobLink(store.OrchestratorJobLinkRecord{
		SessionID:     session.ID,
		MessageCursor: msg.Cursor,
		AgentID:       agentID,
		JobID:         job.ID,
		JobStatus:     job.Status,
	})
	if job.Status == "" {
		_ = s.sessionStore.UpdateOrchestratorJobLinkStatus(session.ID, job.ID, "running")
	}

	s.publishSessionEvent(session.ID, "orchestrator.agent.job.started", map[string]string{
		"agent_id":       agentID,
		"job_id":         job.ID,
		"message_cursor": strconv.FormatInt(msg.Cursor, 10),
	})

	go s.waitForAgentJob(session.ID, agentID, job.ID)
}

func (s *Service) waitForAgentJob(sessionID, agentID, jobID string) {
	s.sessionMu.Lock()
	bg := s.background
	s.sessionMu.Unlock()
	if bg == nil {
		s.handleAgentJobFailure(sessionID, agentID, jobID, fmt.Errorf("background manager unavailable"))
		return
	}

	ticker := time.NewTicker(700 * time.Millisecond)
	defer ticker.Stop()
	timeout := time.NewTimer(4 * time.Minute)
	defer timeout.Stop()

	for {
		select {
		case <-ticker.C:
			job, ok := bg.Get(jobID)
			if !ok {
				s.handleAgentJobFailure(sessionID, agentID, jobID, fmt.Errorf("background job not found"))
				return
			}
			switch job.Status {
			case background.StatusCompleted:
				_ = s.sessionStore.UpdateOrchestratorJobLinkStatus(sessionID, jobID, background.StatusCompleted)
				s.publishSessionEvent(sessionID, "orchestrator.agent.job.completed", map[string]string{
					"agent_id": agentID,
					"job_id":   jobID,
				})
				output := strings.TrimSpace(job.Output)
				if output == "" {
					output = fmt.Sprintf("%s completed work and returned no textual summary.", displayAgentName(agentID))
				}
				s.appendAgentMessage(sessionID, agentID, output)
				s.markAgentIdleWithCooldown(sessionID, agentID, time.Now().UTC().Add(sessionCooldown), "")
				return
			case background.StatusFailed:
				err := fmt.Errorf("%s", strings.TrimSpace(job.Error))
				if strings.TrimSpace(job.Error) == "" {
					err = fmt.Errorf("background job failed")
				}
				s.handleAgentJobFailure(sessionID, agentID, jobID, err)
				return
			case background.StatusCancelled:
				s.handleAgentJobFailure(sessionID, agentID, jobID, fmt.Errorf("background job cancelled"))
				return
			}
		case <-timeout.C:
			s.handleAgentJobFailure(sessionID, agentID, jobID, fmt.Errorf("background job timed out"))
			return
		}
	}
}

func (s *Service) handleAgentJobFailure(sessionID, agentID, jobID string, err error) {
	errMessage := strings.TrimSpace(err.Error())
	if errMessage == "" {
		errMessage = "unknown error"
	}
	if strings.TrimSpace(jobID) != "" {
		_ = s.sessionStore.UpdateOrchestratorJobLinkStatus(sessionID, jobID, "failed")
	}

	s.publishSessionEvent(sessionID, "orchestrator.agent.job.failed", map[string]string{
		"agent_id": agentID,
		"job_id":   jobID,
		"error":    errMessage,
	})
	s.appendAgentMessage(sessionID, agentID, fmt.Sprintf("%s encountered an error: %s", displayAgentName(agentID), errMessage))
	cooldownUntil := time.Now().UTC().Add(sessionCooldown)
	s.markAgentIdleWithCooldown(sessionID, agentID, cooldownUntil, errMessage)

	failures, countErr := s.sessionStore.CountFailedOrchestratorJobsSince(sessionID, time.Now().UTC().Add(-sessionErrorStormWindow))
	if countErr == nil && failures >= sessionErrorStormFailureThreshold {
		_, _ = s.PauseSession(sessionID, "error_storm_detected")
	}
}

func (s *Service) markAgentIdleWithCooldown(sessionID, agentID string, cooldownUntil time.Time, lastError string) {
	state, err := s.getOrDefaultAgentState(sessionID, agentID)
	if err != nil {
		return
	}
	state.Status = "idle"
	if strings.TrimSpace(lastError) != "" {
		state.Status = "error"
		state.LastError = strings.TrimSpace(lastError)
	} else {
		state.LastError = ""
	}
	state.CooldownUntil = &cooldownUntil
	now := time.Now().UTC()
	state.LastActiveAt = &now
	_, _ = s.sessionStore.UpsertOrchestratorAgentState(state)
}

func (s *Service) appendAgentMessage(sessionID, agentID, content string) {
	content = strings.TrimSpace(content)
	if content == "" {
		return
	}
	_, _ = s.AppendSessionMessage(sessionID, OrchestratorMessageAppendRequest{
		AuthorKind: OrchestratorAuthorKindAgent,
		AuthorID:   agentID,
		TargetPane: OrchestratorPaneAll,
		Content:    content,
	})
}

func (s *Service) buildAgentPrompt(sessionID, agentID string, trigger OrchestratorMessage) string {
	history, _ := s.sessionStore.ListOrchestratorMessages(sessionID, 0, 24)
	if len(history) > 8 {
		history = history[len(history)-8:]
	}
	lines := make([]string, 0, len(history)+8)
	lines = append(lines,
		"BioCode Orchestrator Session",
		"Role: "+agentRolePrompt(agentID),
		"",
		"Recent shared messages:",
	)
	for _, item := range history {
		content := strings.TrimSpace(item.Content)
		if content == "" {
			continue
		}
		content = strings.ReplaceAll(content, "\n", " ")
		if len(content) > 280 {
			content = content[:280] + "..."
		}
		lines = append(lines, fmt.Sprintf("- [%s/%s] %s", item.AuthorID, item.TargetPane, content))
	}
	lines = append(lines,
		"",
		"Respond with concrete next action and concise result. If blocked, say blocker and proposed mitigation.",
		"Focus strictly on your role.",
		"Trigger message cursor: "+strconv.FormatInt(trigger.Cursor, 10),
	)
	return strings.Join(lines, "\n")
}

func (s *Service) publishSessionEvent(sessionID, eventType string, payload map[string]string) {
	if payload == nil {
		payload = map[string]string{}
	}
	payload["session_id"] = strings.TrimSpace(sessionID)
	payload["cursor"] = strconv.FormatInt(s.nextSessionEventCursor(sessionID), 10)
	s.publish(sessionID, eventType, payload)
}

func (s *Service) nextSessionEventCursor(sessionID string) int64 {
	sessionID = strings.TrimSpace(sessionID)
	s.sessionMu.Lock()
	defer s.sessionMu.Unlock()
	state, ok := s.sessionRuntime[sessionID]
	if !ok {
		state = &sessionRuntimeState{lastTriggeredCursor: make(map[string]int64, len(sessionAgentsInOrder))}
		s.sessionRuntime[sessionID] = state
	}
	state.nextEventCursor++
	return state.nextEventCursor
}

func inferAgentsToTrigger(msg OrchestratorMessage) []string {
	text := strings.ToLower(strings.TrimSpace(msg.Content))
	agents := make([]string, 0, 3)
	add := func(agentID string) {
		agentID = normalizeSessionAgentID(agentID)
		for _, existing := range agents {
			if existing == agentID {
				return
			}
		}
		agents = append(agents, agentID)
	}

	switch normalizeSessionPane(msg.TargetPane) {
	case OrchestratorPaneBackend:
		add(OrchestratorPaneBackend)
	case OrchestratorPaneFrontend:
		add(OrchestratorPaneFrontend)
	case OrchestratorPaneOrchestrator:
		add(OrchestratorPaneOrchestrator)
	}

	if containsAny(text, "api", "backend", "server", "database", "sql", "migration", "endpoint", "go ", "golang") {
		add(OrchestratorPaneBackend)
	}
	if containsAny(text, "frontend", "ui", "react", "css", "layout", "sidebar", "component", "ux") {
		add(OrchestratorPaneFrontend)
	}
	if containsAny(text, "plan", "coordinate", "handoff", "priorit", "orchestr") {
		add(OrchestratorPaneOrchestrator)
	}

	if len(agents) == 0 {
		add(OrchestratorPaneBackend)
		add(OrchestratorPaneFrontend)
	}

	if msg.AuthorKind == OrchestratorAuthorKindUser {
		add(OrchestratorPaneOrchestrator)
	}
	if msg.AuthorKind == OrchestratorAuthorKindAgent && containsAny(text, "handoff", "@backend", "@frontend", "@orchestrator") {
		if containsAny(text, "@backend") {
			add(OrchestratorPaneBackend)
		}
		if containsAny(text, "@frontend") {
			add(OrchestratorPaneFrontend)
		}
		if containsAny(text, "@orchestrator") {
			add(OrchestratorPaneOrchestrator)
		}
	}

	return agents
}

func shouldTriggerAgents(msg OrchestratorMessage) bool {
	if msg.AuthorKind == OrchestratorAuthorKindUser {
		return true
	}
	if msg.AuthorKind == OrchestratorAuthorKindAgent {
		return containsAny(strings.ToLower(msg.Content), "handoff", "@backend", "@frontend", "@orchestrator")
	}
	return false
}

func orchestratorSessionFromRecord(rec store.OrchestratorSessionRecord, states []store.OrchestratorAgentStateRecord) OrchestratorSession {
	mapped := make([]OrchestratorAgentState, 0, len(states))
	for _, state := range states {
		mapped = append(mapped, orchestratorAgentStateFromRecord(state))
	}
	return OrchestratorSession{
		ID:        rec.ID,
		ProjectID: rec.ProjectID,
		Status:    rec.Status,
		Guardrails: OrchestratorGuardrailState{
			Paused: rec.GuardrailPaused,
			Reason: rec.GuardrailReason,
		},
		MaxJobs:     rec.MaxJobs,
		JobsStarted: rec.JobsStarted,
		Agents:      mapped,
		CreatedAt:   rec.CreatedAt,
		UpdatedAt:   rec.UpdatedAt,
		PausedAt:    rec.PausedAt,
		KilledAt:    rec.KilledAt,
	}
}

func orchestratorMessageFromRecord(rec store.OrchestratorMessageRecord) OrchestratorMessage {
	return OrchestratorMessage{
		ID:         rec.ID,
		Cursor:     rec.Cursor,
		SessionID:  rec.SessionID,
		AuthorKind: rec.AuthorKind,
		AuthorID:   rec.AuthorID,
		TargetPane: rec.TargetPane,
		Content:    rec.Content,
		CreatedAt:  rec.CreatedAt,
	}
}

func orchestratorAgentStateFromRecord(rec store.OrchestratorAgentStateRecord) OrchestratorAgentState {
	return OrchestratorAgentState{
		SessionID: rec.SessionID,
		AgentID:   rec.AgentID,
		Status:    rec.Status,
		Model: OrchestratorAgentModel{
			Provider: rec.ModelProvider,
			ModelID:  rec.ModelID,
		},
		CooldownUntil: rec.CooldownUntil,
		LastError:     rec.LastError,
		LastActiveAt:  rec.LastActiveAt,
		CreatedAt:     rec.CreatedAt,
		UpdatedAt:     rec.UpdatedAt,
	}
}

func defaultModelForSessionAgent(agentID string) OrchestratorAgentModel {
	agentID = normalizeSessionAgentID(agentID)
	if model, ok := defaultSessionModels[agentID]; ok {
		return model
	}
	return defaultSessionModels[OrchestratorPaneOrchestrator]
}

func normalizeModelOverride(raw OrchestratorAgentModel, fallback OrchestratorAgentModel) OrchestratorAgentModel {
	provider := strings.ToLower(strings.TrimSpace(raw.Provider))
	if provider == "" {
		provider = strings.ToLower(strings.TrimSpace(fallback.Provider))
	}
	modelID := strings.TrimSpace(raw.ModelID)
	if modelID == "" {
		modelID = strings.TrimSpace(fallback.ModelID)
	}
	return OrchestratorAgentModel{Provider: provider, ModelID: modelID}
}

func normalizeSessionPane(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "backend", "backend_agent":
		return OrchestratorPaneBackend
	case "frontend", "frontend_agent":
		return OrchestratorPaneFrontend
	case "orchestrator", "main_orchestrator":
		return OrchestratorPaneOrchestrator
	default:
		return OrchestratorPaneAll
	}
}

func normalizeSessionAuthorKind(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case OrchestratorAuthorKindAgent:
		return OrchestratorAuthorKindAgent
	case OrchestratorAuthorKindSystem:
		return OrchestratorAuthorKindSystem
	default:
		return OrchestratorAuthorKindUser
	}
}

func normalizeSessionAuthorID(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "backend", "backend_agent":
		return OrchestratorPaneBackend
	case "frontend", "frontend_agent":
		return OrchestratorPaneFrontend
	case "orchestrator", "main_orchestrator":
		return OrchestratorPaneOrchestrator
	case "system":
		return "system"
	default:
		return "user"
	}
}

func normalizeSessionAgentID(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "backend", "backend_agent":
		return OrchestratorPaneBackend
	case "frontend", "frontend_agent":
		return OrchestratorPaneFrontend
	default:
		return OrchestratorPaneOrchestrator
	}
}

func backgroundAgentForPane(agentID string) string {
	switch normalizeSessionAgentID(agentID) {
	case OrchestratorPaneBackend:
		return "coder"
	case OrchestratorPaneFrontend:
		return "coder"
	default:
		return "planner"
	}
}

func agentRolePrompt(agentID string) string {
	switch normalizeSessionAgentID(agentID) {
	case OrchestratorPaneBackend:
		return "Backend specialist. Focus on API, server runtime, storage, migrations, and reliability."
	case OrchestratorPaneFrontend:
		return "Frontend specialist. Focus on UI architecture, React components, CSS, accessibility, and UX consistency."
	default:
		return "Main orchestrator. Coordinate backend/frontend sequencing, dependencies, and concise handoffs."
	}
}

func displayAgentName(agentID string) string {
	switch normalizeSessionAgentID(agentID) {
	case OrchestratorPaneBackend:
		return "Backend"
	case OrchestratorPaneFrontend:
		return "Frontend"
	default:
		return "Orchestrator"
	}
}
