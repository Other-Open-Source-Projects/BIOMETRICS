package session

import (
	"biometrics-cli/internal/metrics"
	"biometrics-cli/internal/paths"
	"biometrics-cli/internal/state"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Session struct {
	ID        string                 `json:"id"`
	Agent     string                 `json:"agent"`
	Model     string                 `json:"model"`
	Status    string                 `json:"status"`
	StartedAt time.Time              `json:"started_at"`
	EndedAt   time.Time              `json:"ended_at,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Logs      []string               `json:"logs,omitempty"`
}

type Manager struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	active   map[string]*Session
}

var SessionManager = &Manager{
	sessions: make(map[string]*Session),
	active:   make(map[string]*Session),
}

func (m *Manager) Create(agent, model string) *Session {
	m.mu.Lock()
	defer m.mu.Unlock()

	session := &Session{
		ID:        fmt.Sprintf("sess-%d", time.Now().UnixNano()),
		Agent:     agent,
		Model:     model,
		Status:    "active",
		StartedAt: time.Now(),
		Data:      make(map[string]interface{}),
		Logs:      make([]string, 0),
	}

	m.sessions[session.ID] = session
	m.active[session.ID] = session

	metrics.SessionsCreatedTotal.Inc()
	state.GlobalState.Log("INFO", fmt.Sprintf("Created session: %s (agent: %s, model: %s)", session.ID, agent, model))

	return session
}

func (m *Manager) Get(id string) (*Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[id]
	return session, exists
}

func (m *Manager) End(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.active[id]
	if !exists {
		return fmt.Errorf("session not found or already ended")
	}

	session.Status = "ended"
	session.EndedAt = time.Now()
	delete(m.active, id)

	metrics.SessionsEndedTotal.Inc()
	state.GlobalState.Log("INFO", fmt.Sprintf("Ended session: %s", id))

	return nil
}

func (m *Manager) AddLog(id, log string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if session, exists := m.sessions[id]; exists {
		session.Logs = append(session.Logs, fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), log))
	}
}

func (m *Manager) GetActive() []*Session {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessions := make([]*Session, 0, len(m.active))
	for _, session := range m.active {
		sessions = append(sessions, session)
	}
	return sessions
}

func (m *Manager) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	activeCount := len(m.active)
	totalCount := len(m.sessions)

	return map[string]interface{}{
		"active_sessions": activeCount,
		"total_sessions":  totalCount,
		"ended_sessions":  totalCount - activeCount,
	}
}

func (m *Manager) Save() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	data, err := json.MarshalIndent(m.sessions, "", "  ")
	if err != nil {
		return err
	}

	path := paths.SisyphusSessionsPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (m *Manager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(paths.SisyphusSessionsPath())
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &m.sessions)
}

func StartSession(agent, model string) *Session {
	return SessionManager.Create(agent, model)
}

func EndSession(id string) error {
	return SessionManager.End(id)
}

func GetSession(id string) (*Session, bool) {
	return SessionManager.Get(id)
}
