package background

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"biometrics-cli/internal/contracts"
	"biometrics-cli/internal/llm"
	"biometrics-cli/internal/runtime/bus"
)

const (
	StatusQueued    = "queued"
	StatusRunning   = "running"
	StatusCompleted = "completed"
	StatusFailed    = "failed"
	StatusCancelled = "cancelled"
)

var (
	ErrNotConfigured = errors.New("background agent manager is not configured")
	ErrNotFound      = errors.New("background agent not found")
)

type StartRequest struct {
	ProjectID       string
	Agent           string
	Prompt          string
	ModelPreference string
	FallbackChain   []string
	ModelID         string
	ContextBudget   int
}

type Job struct {
	ID              string     `json:"id"`
	ProjectID       string     `json:"project_id"`
	Agent           string     `json:"agent"`
	Prompt          string     `json:"prompt"`
	Status          string     `json:"status"`
	ModelPreference string     `json:"model_preference,omitempty"`
	FallbackChain   []string   `json:"fallback_chain,omitempty"`
	ModelID         string     `json:"model_id,omitempty"`
	ContextBudget   int        `json:"context_budget,omitempty"`
	Provider        string     `json:"provider,omitempty"`
	Output          string     `json:"output,omitempty"`
	Error           string     `json:"error,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	StartedAt       *time.Time `json:"started_at,omitempty"`
	FinishedAt      *time.Time `json:"finished_at,omitempty"`
}

type Manager struct {
	mu      sync.RWMutex
	gateway llm.Gateway
	bus     *bus.EventBus
	jobs    map[string]*jobHandle
	seq     atomic.Uint64
}

type jobHandle struct {
	job    Job
	cancel context.CancelFunc
}

func NewManager(gateway llm.Gateway, eventBus *bus.EventBus) *Manager {
	return &Manager{
		gateway: gateway,
		bus:     eventBus,
		jobs:    make(map[string]*jobHandle),
	}
}

func (m *Manager) Start(_ context.Context, req StartRequest) (Job, error) {
	if m == nil || m.gateway == nil {
		return Job{}, ErrNotConfigured
	}
	req = normalizeStartRequest(req)

	now := time.Now().UTC()
	id := fmt.Sprintf("bg-%d-%06d", now.Unix(), m.seq.Add(1))
	runCtx, cancel := context.WithCancel(context.Background())
	handle := &jobHandle{
		job: Job{
			ID:              id,
			ProjectID:       req.ProjectID,
			Agent:           req.Agent,
			Prompt:          req.Prompt,
			Status:          StatusQueued,
			ModelPreference: req.ModelPreference,
			FallbackChain:   append([]string{}, req.FallbackChain...),
			ModelID:         req.ModelID,
			ContextBudget:   req.ContextBudget,
			CreatedAt:       now,
		},
		cancel: cancel,
	}

	var snapshot Job
	m.mu.Lock()
	m.jobs[id] = handle
	snapshot = handle.job
	m.mu.Unlock()

	m.publish(handle.job.ID, "background.agent.created", map[string]string{
		"agent":            handle.job.Agent,
		"project_id":       handle.job.ProjectID,
		"model_preference": handle.job.ModelPreference,
		"model_id":         handle.job.ModelID,
	})

	go m.execute(runCtx, handle.job.ID)
	return snapshot, nil
}

func (m *Manager) List() []Job {
	if m == nil {
		return []Job{}
	}
	m.mu.RLock()
	out := make([]Job, 0, len(m.jobs))
	for _, handle := range m.jobs {
		out = append(out, handle.job)
	}
	m.mu.RUnlock()

	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out
}

func (m *Manager) Get(jobID string) (Job, bool) {
	if m == nil {
		return Job{}, false
	}
	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		return Job{}, false
	}

	m.mu.RLock()
	handle, ok := m.jobs[jobID]
	if !ok {
		m.mu.RUnlock()
		return Job{}, false
	}
	job := handle.job
	m.mu.RUnlock()
	return job, true
}

func (m *Manager) Cancel(jobID string) (Job, error) {
	if m == nil {
		return Job{}, ErrNotConfigured
	}
	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		return Job{}, ErrNotFound
	}

	var (
		cancel context.CancelFunc
		job    Job
		ok     bool
	)

	now := time.Now().UTC()
	m.mu.Lock()
	handle, exists := m.jobs[jobID]
	if !exists {
		m.mu.Unlock()
		return Job{}, ErrNotFound
	}
	if isTerminal(handle.job.Status) {
		job = handle.job
		m.mu.Unlock()
		return job, nil
	}

	handle.job.Status = StatusCancelled
	handle.job.Error = "cancelled by operator"
	handle.job.Output = ""
	handle.job.FinishedAt = &now
	cancel = handle.cancel
	job = handle.job
	ok = true
	m.mu.Unlock()

	if ok && cancel != nil {
		cancel()
	}
	m.publish(job.ID, "background.agent.cancelled", map[string]string{
		"agent":      job.Agent,
		"project_id": job.ProjectID,
	})
	return job, nil
}

func (m *Manager) execute(ctx context.Context, jobID string) {
	job, ok := m.markRunning(jobID)
	if !ok {
		return
	}

	response, err := m.gateway.Execute(ctx, llm.Request{
		RunID:           "background:" + job.ID,
		Agent:           job.Agent,
		ProjectID:       job.ProjectID,
		Prompt:          job.Prompt,
		ModelPreference: job.ModelPreference,
		FallbackChain:   append([]string{}, job.FallbackChain...),
		ModelID:         job.ModelID,
		ContextBudget:   job.ContextBudget,
	})

	if err != nil {
		if errors.Is(err, context.Canceled) {
			m.markCancelled(jobID)
			return
		}
		m.markFailed(jobID, err)
		return
	}
	m.markCompleted(jobID, response)
}

func (m *Manager) markRunning(jobID string) (Job, bool) {
	now := time.Now().UTC()
	var job Job
	_, ok := m.update(jobID, func(handle *jobHandle) {
		handle.job.Status = StatusRunning
		handle.job.StartedAt = &now
		job = handle.job
	})
	if ok {
		m.publish(jobID, "background.agent.started", map[string]string{
			"agent":            job.Agent,
			"project_id":       job.ProjectID,
			"model_preference": job.ModelPreference,
			"model_id":         job.ModelID,
		})
	}
	return job, ok
}

func (m *Manager) markCompleted(jobID string, response llm.Response) {
	now := time.Now().UTC()
	updated, ok := m.update(jobID, func(handle *jobHandle) {
		// Preserve explicit cancel decisions when operator stops a job while result returns.
		if handle.job.Status == StatusCancelled {
			return
		}
		handle.job.Status = StatusCompleted
		handle.job.Provider = strings.TrimSpace(response.Provider)
		if strings.TrimSpace(response.ModelID) != "" {
			handle.job.ModelID = strings.TrimSpace(response.ModelID)
		}
		handle.job.Output = strings.TrimSpace(response.Output)
		handle.job.Error = ""
		handle.job.FinishedAt = &now
	})
	if !ok {
		return
	}
	if updated.Status != StatusCompleted {
		return
	}
	m.publish(jobID, "background.agent.completed", map[string]string{
		"agent":      updated.Agent,
		"project_id": updated.ProjectID,
		"provider":   updated.Provider,
		"model_id":   updated.ModelID,
	})
}

func (m *Manager) markFailed(jobID string, err error) {
	now := time.Now().UTC()
	errMsg := strings.TrimSpace(err.Error())
	if errMsg == "" {
		errMsg = "background agent failed"
	}
	updated, ok := m.update(jobID, func(handle *jobHandle) {
		if handle.job.Status == StatusCancelled {
			return
		}
		handle.job.Status = StatusFailed
		handle.job.Error = errMsg
		handle.job.Output = ""
		handle.job.FinishedAt = &now
	})
	if !ok {
		return
	}
	if updated.Status != StatusFailed {
		return
	}
	m.publish(jobID, "background.agent.failed", map[string]string{
		"agent":      updated.Agent,
		"project_id": updated.ProjectID,
		"error":      errMsg,
	})
}

func (m *Manager) markCancelled(jobID string) {
	now := time.Now().UTC()
	updated, ok := m.update(jobID, func(handle *jobHandle) {
		handle.job.Status = StatusCancelled
		handle.job.Error = "cancelled by operator"
		handle.job.Output = ""
		handle.job.FinishedAt = &now
	})
	if !ok {
		return
	}
	m.publish(jobID, "background.agent.cancelled", map[string]string{
		"agent":      updated.Agent,
		"project_id": updated.ProjectID,
	})
}

func (m *Manager) update(jobID string, mutate func(handle *jobHandle)) (Job, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	handle, ok := m.jobs[jobID]
	if !ok {
		return Job{}, false
	}
	if mutate != nil {
		mutate(handle)
	}
	return handle.job, true
}

func (m *Manager) publish(jobID, eventType string, payload map[string]string) {
	if m.bus == nil {
		return
	}
	_, _ = m.bus.Publish(contracts.Event{
		RunID:   jobID,
		Type:    eventType,
		Source:  "background.manager",
		Payload: payload,
	})
}

func normalizeStartRequest(req StartRequest) StartRequest {
	req.ProjectID = strings.TrimSpace(req.ProjectID)
	if req.ProjectID == "" {
		req.ProjectID = "biometrics"
	}
	req.Agent = strings.TrimSpace(req.Agent)
	if req.Agent == "" {
		req.Agent = "coder"
	}
	req.Prompt = strings.TrimSpace(req.Prompt)
	req.ModelPreference = strings.ToLower(strings.TrimSpace(req.ModelPreference))
	if req.ModelPreference == "" {
		req.ModelPreference = "codex"
	}
	req.ModelID = strings.TrimSpace(req.ModelID)
	req.FallbackChain = normalizeProviderIDs(req.FallbackChain)
	if req.ContextBudget <= 0 {
		req.ContextBudget = 24000
	}
	if req.ContextBudget > 200000 {
		req.ContextBudget = 200000
	}
	return req
}

func normalizeProviderIDs(in []string) []string {
	if len(in) == 0 {
		return []string{}
	}
	out := make([]string, 0, len(in))
	seen := make(map[string]struct{}, len(in))
	for _, raw := range in {
		normalized := strings.ToLower(strings.TrimSpace(raw))
		if normalized == "" {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	return out
}

func isTerminal(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case StatusCompleted, StatusFailed, StatusCancelled:
		return true
	default:
		return false
	}
}
