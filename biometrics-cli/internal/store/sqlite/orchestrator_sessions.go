package sqlite

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	OrchestratorSessionStatusActive = "active"
	OrchestratorSessionStatusPaused = "paused"
	OrchestratorSessionStatusKilled = "killed"
)

type OrchestratorSessionRecord struct {
	ID              string     `json:"id"`
	ProjectID       string     `json:"project_id"`
	Status          string     `json:"status"`
	GuardrailPaused bool       `json:"guardrail_paused"`
	GuardrailReason string     `json:"guardrail_reason,omitempty"`
	MaxJobs         int        `json:"max_jobs"`
	JobsStarted     int        `json:"jobs_started"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	PausedAt        *time.Time `json:"paused_at,omitempty"`
	KilledAt        *time.Time `json:"killed_at,omitempty"`
}

type OrchestratorMessageRecord struct {
	ID         string    `json:"id"`
	Cursor     int64     `json:"cursor"`
	SessionID  string    `json:"session_id"`
	AuthorKind string    `json:"author_kind"`
	AuthorID   string    `json:"author_id"`
	TargetPane string    `json:"target_pane"`
	Content    string    `json:"content"`
	CreatedAt  time.Time `json:"created_at"`
}

type OrchestratorAgentStateRecord struct {
	SessionID     string     `json:"session_id"`
	AgentID       string     `json:"agent_id"`
	Status        string     `json:"status"`
	ModelProvider string     `json:"model_provider,omitempty"`
	ModelID       string     `json:"model_id,omitempty"`
	CooldownUntil *time.Time `json:"cooldown_until,omitempty"`
	LastError     string     `json:"last_error,omitempty"`
	LastActiveAt  *time.Time `json:"last_active_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type OrchestratorJobLinkRecord struct {
	ID            int64     `json:"id"`
	SessionID     string    `json:"session_id"`
	MessageCursor int64     `json:"message_cursor,omitempty"`
	AgentID       string    `json:"agent_id"`
	JobID         string    `json:"job_id"`
	JobStatus     string    `json:"job_status"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (s *Store) CreateOrchestratorSession(rec OrchestratorSessionRecord) (OrchestratorSessionRecord, error) {
	rec.ID = strings.TrimSpace(rec.ID)
	if rec.ID == "" {
		rec.ID = "orc-session-" + uuid.NewString()
	}
	rec.ProjectID = strings.TrimSpace(rec.ProjectID)
	if rec.ProjectID == "" {
		rec.ProjectID = "biometrics"
	}
	rec.Status = normalizeOrchestratorSessionStatus(rec.Status)
	rec.GuardrailReason = strings.TrimSpace(rec.GuardrailReason)
	if rec.MaxJobs <= 0 {
		rec.MaxJobs = 60
	}
	if rec.MaxJobs > 10000 {
		rec.MaxJobs = 10000
	}
	if rec.JobsStarted < 0 {
		rec.JobsStarted = 0
	}

	now := time.Now().UTC()
	if rec.CreatedAt.IsZero() {
		rec.CreatedAt = now
	}
	rec.UpdatedAt = now

	var pausedAt interface{} = nil
	if rec.Status == OrchestratorSessionStatusPaused {
		t := now
		rec.PausedAt = &t
		pausedAt = t.Format(time.RFC3339Nano)
	}
	var killedAt interface{} = nil
	if rec.Status == OrchestratorSessionStatusKilled {
		t := now
		rec.KilledAt = &t
		killedAt = t.Format(time.RFC3339Nano)
	}

	_, err := s.db.Exec(
		`INSERT INTO orchestrator_sessions (
			id, project_id, status, guardrail_paused, guardrail_reason, max_jobs, jobs_started,
			created_at, updated_at, paused_at, killed_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		rec.ID,
		rec.ProjectID,
		rec.Status,
		boolToInt(rec.GuardrailPaused),
		rec.GuardrailReason,
		rec.MaxJobs,
		rec.JobsStarted,
		rec.CreatedAt.Format(time.RFC3339Nano),
		rec.UpdatedAt.Format(time.RFC3339Nano),
		pausedAt,
		killedAt,
	)
	if err != nil {
		return OrchestratorSessionRecord{}, fmt.Errorf("insert orchestrator session: %w", err)
	}
	return rec, nil
}

func (s *Store) GetOrchestratorSession(sessionID string) (OrchestratorSessionRecord, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return OrchestratorSessionRecord{}, fmt.Errorf("session id is required")
	}

	row := s.db.QueryRow(
		`SELECT
			id, project_id, status, guardrail_paused, guardrail_reason, max_jobs, jobs_started,
			created_at, updated_at, paused_at, killed_at
		FROM orchestrator_sessions
		WHERE id = ?`,
		sessionID,
	)
	rec, err := scanOrchestratorSession(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return OrchestratorSessionRecord{}, err
		}
		return OrchestratorSessionRecord{}, fmt.Errorf("get orchestrator session: %w", err)
	}
	return rec, nil
}

func (s *Store) ListOrchestratorSessions(projectID string, limit int) ([]OrchestratorSessionRecord, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 200 {
		limit = 200
	}

	projectID = strings.TrimSpace(projectID)
	query := `SELECT
		id, project_id, status, guardrail_paused, guardrail_reason, max_jobs, jobs_started,
		created_at, updated_at, paused_at, killed_at
	FROM orchestrator_sessions`
	args := make([]interface{}, 0, 2)
	if projectID != "" {
		query += ` WHERE project_id = ?`
		args = append(args, projectID)
	}
	query += ` ORDER BY created_at DESC LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list orchestrator sessions: %w", err)
	}
	defer rows.Close()

	out := make([]OrchestratorSessionRecord, 0, limit)
	for rows.Next() {
		rec, err := scanOrchestratorSession(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, nil
}

func (s *Store) UpdateOrchestratorSessionLifecycle(
	sessionID, status string,
	guardrailPaused bool,
	guardrailReason string,
) (OrchestratorSessionRecord, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return OrchestratorSessionRecord{}, fmt.Errorf("session id is required")
	}
	status = normalizeOrchestratorSessionStatus(status)
	guardrailReason = strings.TrimSpace(guardrailReason)
	now := time.Now().UTC().Format(time.RFC3339Nano)

	switch status {
	case OrchestratorSessionStatusPaused:
		_, err := s.db.Exec(
			`UPDATE orchestrator_sessions
			 SET status = ?, guardrail_paused = ?, guardrail_reason = ?, updated_at = ?, paused_at = ?
			 WHERE id = ?`,
			status,
			boolToInt(guardrailPaused),
			guardrailReason,
			now,
			now,
			sessionID,
		)
		if err != nil {
			return OrchestratorSessionRecord{}, fmt.Errorf("update paused orchestrator session: %w", err)
		}
	case OrchestratorSessionStatusKilled:
		_, err := s.db.Exec(
			`UPDATE orchestrator_sessions
			 SET status = ?, guardrail_paused = ?, guardrail_reason = ?, updated_at = ?, killed_at = ?
			 WHERE id = ?`,
			status,
			boolToInt(guardrailPaused),
			guardrailReason,
			now,
			now,
			sessionID,
		)
		if err != nil {
			return OrchestratorSessionRecord{}, fmt.Errorf("update killed orchestrator session: %w", err)
		}
	default:
		_, err := s.db.Exec(
			`UPDATE orchestrator_sessions
			 SET status = ?, guardrail_paused = ?, guardrail_reason = ?, updated_at = ?, paused_at = NULL
			 WHERE id = ?`,
			OrchestratorSessionStatusActive,
			boolToInt(guardrailPaused),
			guardrailReason,
			now,
			sessionID,
		)
		if err != nil {
			return OrchestratorSessionRecord{}, fmt.Errorf("update active orchestrator session: %w", err)
		}
	}

	return s.GetOrchestratorSession(sessionID)
}

func (s *Store) IncrementOrchestratorSessionJobsStarted(sessionID string) (OrchestratorSessionRecord, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return OrchestratorSessionRecord{}, fmt.Errorf("session id is required")
	}
	_, err := s.db.Exec(
		`UPDATE orchestrator_sessions
		 SET jobs_started = jobs_started + 1, updated_at = ?
		 WHERE id = ?`,
		time.Now().UTC().Format(time.RFC3339Nano),
		sessionID,
	)
	if err != nil {
		return OrchestratorSessionRecord{}, fmt.Errorf("increment orchestrator session jobs_started: %w", err)
	}
	return s.GetOrchestratorSession(sessionID)
}

func (s *Store) AppendOrchestratorMessage(msg OrchestratorMessageRecord) (OrchestratorMessageRecord, error) {
	msg.SessionID = strings.TrimSpace(msg.SessionID)
	if msg.SessionID == "" {
		return OrchestratorMessageRecord{}, fmt.Errorf("session_id is required")
	}
	msg.AuthorKind = normalizeOrchestratorAuthorKind(msg.AuthorKind)
	msg.AuthorID = normalizeOrchestratorAuthorID(msg.AuthorID)
	msg.TargetPane = normalizeOrchestratorTargetPane(msg.TargetPane)
	msg.Content = strings.TrimSpace(msg.Content)
	if msg.Content == "" {
		return OrchestratorMessageRecord{}, fmt.Errorf("content is required")
	}
	now := time.Now().UTC()
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = now
	}

	result, err := s.db.Exec(
		`INSERT INTO orchestrator_messages (
			session_id, author_kind, author_id, target_pane, content, created_at
		) VALUES (?, ?, ?, ?, ?, ?)`,
		msg.SessionID,
		msg.AuthorKind,
		msg.AuthorID,
		msg.TargetPane,
		msg.Content,
		msg.CreatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return OrchestratorMessageRecord{}, fmt.Errorf("insert orchestrator message: %w", err)
	}
	cursor, _ := result.LastInsertId()
	msg.Cursor = cursor
	msg.ID = fmt.Sprintf("%d", cursor)
	return msg, nil
}

func (s *Store) ListOrchestratorMessages(sessionID string, afterCursor int64, limit int) ([]OrchestratorMessageRecord, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return []OrchestratorMessageRecord{}, fmt.Errorf("session id is required")
	}
	if limit <= 0 {
		limit = 200
	}
	if limit > 2000 {
		limit = 2000
	}
	if afterCursor < 0 {
		afterCursor = 0
	}

	rows, err := s.db.Query(
		`SELECT cursor, session_id, author_kind, author_id, target_pane, content, created_at
		 FROM orchestrator_messages
		 WHERE session_id = ? AND cursor > ?
		 ORDER BY cursor ASC
		 LIMIT ?`,
		sessionID,
		afterCursor,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list orchestrator messages: %w", err)
	}
	defer rows.Close()

	out := make([]OrchestratorMessageRecord, 0, limit)
	for rows.Next() {
		msg, err := scanOrchestratorMessage(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, msg)
	}
	return out, nil
}

func (s *Store) UpsertOrchestratorAgentState(state OrchestratorAgentStateRecord) (OrchestratorAgentStateRecord, error) {
	state.SessionID = strings.TrimSpace(state.SessionID)
	if state.SessionID == "" {
		return OrchestratorAgentStateRecord{}, fmt.Errorf("session_id is required")
	}
	state.AgentID = normalizeOrchestratorAgentID(state.AgentID)
	state.Status = normalizeOrchestratorAgentStatus(state.Status)
	state.ModelProvider = strings.ToLower(strings.TrimSpace(state.ModelProvider))
	state.ModelID = strings.TrimSpace(state.ModelID)
	state.LastError = strings.TrimSpace(state.LastError)
	now := time.Now().UTC()
	if state.CreatedAt.IsZero() {
		state.CreatedAt = now
	}
	state.UpdatedAt = now

	var cooldownUntil interface{} = nil
	if state.CooldownUntil != nil && !state.CooldownUntil.IsZero() {
		cooldownUntil = state.CooldownUntil.UTC().Format(time.RFC3339Nano)
	}
	var lastActiveAt interface{} = nil
	if state.LastActiveAt != nil && !state.LastActiveAt.IsZero() {
		lastActiveAt = state.LastActiveAt.UTC().Format(time.RFC3339Nano)
	}

	_, err := s.db.Exec(
		`INSERT INTO orchestrator_agent_state (
			session_id, agent_id, status, model_provider, model_id, cooldown_until, last_error, last_active_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(session_id, agent_id) DO UPDATE SET
			status = excluded.status,
			model_provider = excluded.model_provider,
			model_id = excluded.model_id,
			cooldown_until = excluded.cooldown_until,
			last_error = excluded.last_error,
			last_active_at = excluded.last_active_at,
			updated_at = excluded.updated_at`,
		state.SessionID,
		state.AgentID,
		state.Status,
		state.ModelProvider,
		state.ModelID,
		cooldownUntil,
		state.LastError,
		lastActiveAt,
		state.CreatedAt.Format(time.RFC3339Nano),
		state.UpdatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return OrchestratorAgentStateRecord{}, fmt.Errorf("upsert orchestrator agent state: %w", err)
	}
	return s.GetOrchestratorAgentState(state.SessionID, state.AgentID)
}

func (s *Store) GetOrchestratorAgentState(sessionID, agentID string) (OrchestratorAgentStateRecord, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return OrchestratorAgentStateRecord{}, fmt.Errorf("session id is required")
	}
	agentID = normalizeOrchestratorAgentID(agentID)
	row := s.db.QueryRow(
		`SELECT session_id, agent_id, status, model_provider, model_id, cooldown_until, last_error, last_active_at, created_at, updated_at
		 FROM orchestrator_agent_state
		 WHERE session_id = ? AND agent_id = ?`,
		sessionID,
		agentID,
	)
	state, err := scanOrchestratorAgentState(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return OrchestratorAgentStateRecord{}, err
		}
		return OrchestratorAgentStateRecord{}, fmt.Errorf("get orchestrator agent state: %w", err)
	}
	return state, nil
}

func (s *Store) ListOrchestratorAgentStates(sessionID string) ([]OrchestratorAgentStateRecord, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil, fmt.Errorf("session id is required")
	}
	rows, err := s.db.Query(
		`SELECT session_id, agent_id, status, model_provider, model_id, cooldown_until, last_error, last_active_at, created_at, updated_at
		 FROM orchestrator_agent_state
		 WHERE session_id = ?
		 ORDER BY agent_id ASC`,
		sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("list orchestrator agent states: %w", err)
	}
	defer rows.Close()

	out := make([]OrchestratorAgentStateRecord, 0, 3)
	for rows.Next() {
		state, err := scanOrchestratorAgentState(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, state)
	}
	return out, nil
}

func (s *Store) CreateOrchestratorJobLink(link OrchestratorJobLinkRecord) (OrchestratorJobLinkRecord, error) {
	link.SessionID = strings.TrimSpace(link.SessionID)
	if link.SessionID == "" {
		return OrchestratorJobLinkRecord{}, fmt.Errorf("session_id is required")
	}
	link.AgentID = normalizeOrchestratorAgentID(link.AgentID)
	link.JobID = strings.TrimSpace(link.JobID)
	if link.JobID == "" {
		return OrchestratorJobLinkRecord{}, fmt.Errorf("job_id is required")
	}
	link.JobStatus = normalizeOrchestratorJobStatus(link.JobStatus)
	now := time.Now().UTC()
	if link.CreatedAt.IsZero() {
		link.CreatedAt = now
	}
	link.UpdatedAt = now

	result, err := s.db.Exec(
		`INSERT INTO orchestrator_job_links (
			session_id, message_cursor, agent_id, job_id, job_status, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		link.SessionID,
		link.MessageCursor,
		link.AgentID,
		link.JobID,
		link.JobStatus,
		link.CreatedAt.Format(time.RFC3339Nano),
		link.UpdatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return OrchestratorJobLinkRecord{}, fmt.Errorf("insert orchestrator job link: %w", err)
	}
	id, _ := result.LastInsertId()
	link.ID = id
	return link, nil
}

func (s *Store) UpdateOrchestratorJobLinkStatus(sessionID, jobID, jobStatus string) error {
	sessionID = strings.TrimSpace(sessionID)
	jobID = strings.TrimSpace(jobID)
	if sessionID == "" || jobID == "" {
		return fmt.Errorf("session_id and job_id are required")
	}
	jobStatus = normalizeOrchestratorJobStatus(jobStatus)
	_, err := s.db.Exec(
		`UPDATE orchestrator_job_links
		 SET job_status = ?, updated_at = ?
		 WHERE session_id = ? AND job_id = ?`,
		jobStatus,
		time.Now().UTC().Format(time.RFC3339Nano),
		sessionID,
		jobID,
	)
	if err != nil {
		return fmt.Errorf("update orchestrator job link status: %w", err)
	}
	return nil
}

func (s *Store) CountFailedOrchestratorJobsSince(sessionID string, since time.Time) (int, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return 0, fmt.Errorf("session id is required")
	}
	var count int
	err := s.db.QueryRow(
		`SELECT COUNT(1)
		 FROM orchestrator_job_links
		 WHERE session_id = ?
		 AND job_status IN ('failed', 'cancelled')
		 AND updated_at >= ?`,
		sessionID,
		since.UTC().Format(time.RFC3339Nano),
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count failed orchestrator jobs: %w", err)
	}
	return count, nil
}

func scanOrchestratorSession(scanner interface {
	Scan(dest ...interface{}) error
}) (OrchestratorSessionRecord, error) {
	var rec OrchestratorSessionRecord
	var guardrailPaused int
	var createdAt string
	var updatedAt string
	var pausedAt sql.NullString
	var killedAt sql.NullString
	if err := scanner.Scan(
		&rec.ID,
		&rec.ProjectID,
		&rec.Status,
		&guardrailPaused,
		&rec.GuardrailReason,
		&rec.MaxJobs,
		&rec.JobsStarted,
		&createdAt,
		&updatedAt,
		&pausedAt,
		&killedAt,
	); err != nil {
		return OrchestratorSessionRecord{}, err
	}
	rec.Status = normalizeOrchestratorSessionStatus(rec.Status)
	rec.GuardrailPaused = guardrailPaused != 0
	rec.GuardrailReason = strings.TrimSpace(rec.GuardrailReason)
	rec.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	rec.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAt)
	if pausedAt.Valid && strings.TrimSpace(pausedAt.String) != "" {
		t, err := time.Parse(time.RFC3339Nano, pausedAt.String)
		if err == nil {
			rec.PausedAt = &t
		}
	}
	if killedAt.Valid && strings.TrimSpace(killedAt.String) != "" {
		t, err := time.Parse(time.RFC3339Nano, killedAt.String)
		if err == nil {
			rec.KilledAt = &t
		}
	}
	return rec, nil
}

func scanOrchestratorMessage(scanner interface {
	Scan(dest ...interface{}) error
}) (OrchestratorMessageRecord, error) {
	var msg OrchestratorMessageRecord
	var createdAt string
	if err := scanner.Scan(
		&msg.Cursor,
		&msg.SessionID,
		&msg.AuthorKind,
		&msg.AuthorID,
		&msg.TargetPane,
		&msg.Content,
		&createdAt,
	); err != nil {
		return OrchestratorMessageRecord{}, err
	}
	msg.ID = fmt.Sprintf("%d", msg.Cursor)
	msg.AuthorKind = normalizeOrchestratorAuthorKind(msg.AuthorKind)
	msg.AuthorID = normalizeOrchestratorAuthorID(msg.AuthorID)
	msg.TargetPane = normalizeOrchestratorTargetPane(msg.TargetPane)
	msg.Content = strings.TrimSpace(msg.Content)
	msg.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	return msg, nil
}

func scanOrchestratorAgentState(scanner interface {
	Scan(dest ...interface{}) error
}) (OrchestratorAgentStateRecord, error) {
	var state OrchestratorAgentStateRecord
	var cooldownUntil sql.NullString
	var lastActiveAt sql.NullString
	var createdAt string
	var updatedAt string
	if err := scanner.Scan(
		&state.SessionID,
		&state.AgentID,
		&state.Status,
		&state.ModelProvider,
		&state.ModelID,
		&cooldownUntil,
		&state.LastError,
		&lastActiveAt,
		&createdAt,
		&updatedAt,
	); err != nil {
		return OrchestratorAgentStateRecord{}, err
	}
	state.AgentID = normalizeOrchestratorAgentID(state.AgentID)
	state.Status = normalizeOrchestratorAgentStatus(state.Status)
	state.ModelProvider = strings.ToLower(strings.TrimSpace(state.ModelProvider))
	state.ModelID = strings.TrimSpace(state.ModelID)
	state.LastError = strings.TrimSpace(state.LastError)
	state.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	state.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAt)
	if cooldownUntil.Valid && strings.TrimSpace(cooldownUntil.String) != "" {
		t, err := time.Parse(time.RFC3339Nano, cooldownUntil.String)
		if err == nil {
			state.CooldownUntil = &t
		}
	}
	if lastActiveAt.Valid && strings.TrimSpace(lastActiveAt.String) != "" {
		t, err := time.Parse(time.RFC3339Nano, lastActiveAt.String)
		if err == nil {
			state.LastActiveAt = &t
		}
	}
	return state, nil
}

func normalizeOrchestratorSessionStatus(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case OrchestratorSessionStatusPaused:
		return OrchestratorSessionStatusPaused
	case OrchestratorSessionStatusKilled:
		return OrchestratorSessionStatusKilled
	default:
		return OrchestratorSessionStatusActive
	}
}

func normalizeOrchestratorAuthorKind(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "agent":
		return "agent"
	case "system":
		return "system"
	default:
		return "user"
	}
}

func normalizeOrchestratorAuthorID(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "backend":
		return "backend"
	case "frontend":
		return "frontend"
	case "orchestrator":
		return "orchestrator"
	case "system":
		return "system"
	default:
		return "user"
	}
}

func normalizeOrchestratorTargetPane(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "backend":
		return "backend"
	case "frontend":
		return "frontend"
	case "orchestrator":
		return "orchestrator"
	default:
		return "all"
	}
}

func normalizeOrchestratorAgentID(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "backend":
		return "backend"
	case "frontend":
		return "frontend"
	default:
		return "orchestrator"
	}
}

func normalizeOrchestratorAgentStatus(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "running":
		return "running"
	case "paused":
		return "paused"
	case "error":
		return "error"
	default:
		return "idle"
	}
}

func normalizeOrchestratorJobStatus(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "running":
		return "running"
	case "completed":
		return "completed"
	case "failed":
		return "failed"
	case "cancelled":
		return "cancelled"
	default:
		return "queued"
	}
}
