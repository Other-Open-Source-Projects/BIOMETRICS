package sqlite

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"biometrics-cli/internal/contracts"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

type Store struct {
	db *sql.DB
	mu sync.RWMutex
}

type storeColumn struct {
	name string
	stmt string
}

type CreateRunOptions struct {
	ProjectID                 string
	Goal                      string
	Mode                      string
	Skills                    []string
	SkillSelectionMode        string
	SchedulerMode             contracts.SchedulerMode
	MaxParallelism            int
	FallbackTriggered         bool
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

type CreateTaskOptions struct {
	Name           string
	Agent          string
	DependsOn      []string
	Priority       int
	MaxAttempts    int
	LifecycleState contracts.TaskLifecycleState
}

type CreateAttemptOptions struct {
	RunID         string
	TaskID        string
	Agent         string
	Status        string
	Log           string
	Error         string
	Provider      string
	ModelID       string
	FallbackIndex int
	LatencyMs     int64
	TokensIn      int
	TokensOut     int
	ProviderTrail []contracts.ProviderAttempt
	StartedAt     time.Time
	FinishedAt    time.Time
}

type OptimizerObjectiveRecord struct {
	Quality float64 `json:"quality"`
	Speed   float64 `json:"speed"`
	Cost    float64 `json:"cost"`
}

type OptimizerPredictedGatesRecord struct {
	QualityPass                 bool    `json:"quality_pass"`
	TimePass                    bool    `json:"time_pass"`
	CostPass                    bool    `json:"cost_pass"`
	RegressionPass              bool    `json:"regression_pass"`
	AllPass                     bool    `json:"all_pass"`
	GatePassCount               int     `json:"gate_pass_count"`
	PredictedQualityScore       float64 `json:"predicted_quality_score"`
	PredictedTimeImprovementPct float64 `json:"predicted_time_improvement_percent"`
	PredictedCostImprovementPct float64 `json:"predicted_cost_improvement_percent"`
	PredictedCostPerSuccess     float64 `json:"predicted_cost_per_success"`
	PredictedRegressionDetected bool    `json:"predicted_regression_detected"`
	PredictedCompositeScore     float64 `json:"predicted_composite_score"`
}

type OptimizerRecommendationRecord struct {
	ID                   string                        `json:"id"`
	ProjectID            string                        `json:"project_id"`
	Goal                 string                        `json:"goal"`
	StrategyMode         string                        `json:"strategy_mode"`
	SchedulerMode        string                        `json:"scheduler_mode"`
	MaxParallelism       int                           `json:"max_parallelism"`
	ModelPreference      string                        `json:"model_preference"`
	FallbackChain        []string                      `json:"fallback_chain,omitempty"`
	ModelID              string                        `json:"model_id,omitempty"`
	ContextBudget        int                           `json:"context_budget"`
	Objective            OptimizerObjectiveRecord      `json:"objective"`
	Confidence           string                        `json:"confidence"`
	PredictedGates       OptimizerPredictedGatesRecord `json:"predicted_gates"`
	Rationale            string                        `json:"rationale"`
	SourceScorecardRunID string                        `json:"source_scorecard_run_id,omitempty"`
	Status               string                        `json:"status"`
	AppliedRunID         string                        `json:"applied_run_id,omitempty"`
	RejectedReason       string                        `json:"rejected_reason,omitempty"`
	CreatedAt            time.Time                     `json:"created_at"`
	UpdatedAt            time.Time                     `json:"updated_at"`
}

type ListOptimizerRecommendationsOptions struct {
	ProjectID string
	Status    string
	Limit     int
}

type OptimizerValidationRecord struct {
	ID               string    `json:"id"`
	RecommendationID string    `json:"recommendation_id"`
	EvalRunID        string    `json:"eval_run_id,omitempty"`
	Status           string    `json:"status"`
	QualityPass      bool      `json:"quality_pass"`
	TimePass         bool      `json:"time_pass"`
	CostPass         bool      `json:"cost_pass"`
	RegressionPass   bool      `json:"regression_pass"`
	AllPass          bool      `json:"all_pass"`
	Summary          string    `json:"summary,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func New(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}

	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	pragmas := []string{
		"PRAGMA journal_mode=WAL;",
		"PRAGMA synchronous=NORMAL;",
		"PRAGMA foreign_keys=ON;",
	}
	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			return nil, fmt.Errorf("pragma %q: %w", pragma, err)
		}
	}

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) Ping() error {
	if s.db == nil {
		return fmt.Errorf("store database is not initialized")
	}
	return s.db.Ping()
}

func (s *Store) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS runs (
				id TEXT PRIMARY KEY,
				project_id TEXT NOT NULL,
				goal TEXT NOT NULL,
				mode TEXT NOT NULL,
				skills TEXT NOT NULL DEFAULT '[]',
				skill_selection_mode TEXT NOT NULL DEFAULT 'auto',
				status TEXT NOT NULL,
				error TEXT NOT NULL DEFAULT '',
				scheduler_mode TEXT NOT NULL DEFAULT 'dag_parallel_v1',
				max_parallelism INTEGER NOT NULL DEFAULT 8,
				fallback_triggered INTEGER NOT NULL DEFAULT 0,
				model_preference TEXT NOT NULL DEFAULT 'codex',
				fallback_chain TEXT NOT NULL DEFAULT '[]',
				model_id TEXT NOT NULL DEFAULT '',
					context_budget INTEGER NOT NULL DEFAULT 24000,
					blueprint_profile TEXT NOT NULL DEFAULT '',
					blueprint_modules TEXT NOT NULL DEFAULT '[]',
					bootstrap INTEGER NOT NULL DEFAULT 0,
					optimizer_recommendation_id TEXT NOT NULL DEFAULT '',
					optimizer_confidence TEXT NOT NULL DEFAULT '',
					created_at TEXT NOT NULL,
					updated_at TEXT NOT NULL
			);`,
		`CREATE TABLE IF NOT EXISTS tasks (
			id TEXT PRIMARY KEY,
			run_id TEXT NOT NULL,
			name TEXT NOT NULL,
			agent TEXT NOT NULL,
			status TEXT NOT NULL,
			lifecycle_state TEXT NOT NULL DEFAULT 'blocked',
			depends_on TEXT NOT NULL,
			priority INTEGER NOT NULL DEFAULT 0,
			max_attempts INTEGER NOT NULL DEFAULT 3,
			attempt INTEGER NOT NULL DEFAULT 0,
			last_error TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			started_at TEXT,
			finished_at TEXT,
			completed_at TEXT,
			FOREIGN KEY(run_id) REFERENCES runs(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS attempts (
				id TEXT PRIMARY KEY,
				run_id TEXT NOT NULL,
				task_id TEXT NOT NULL,
				agent TEXT NOT NULL,
				status TEXT NOT NULL,
				log TEXT NOT NULL DEFAULT '',
				error TEXT NOT NULL DEFAULT '',
				provider TEXT NOT NULL DEFAULT '',
				model_id TEXT NOT NULL DEFAULT '',
				fallback_index INTEGER NOT NULL DEFAULT 0,
				latency_ms INTEGER NOT NULL DEFAULT 0,
				tokens_in INTEGER NOT NULL DEFAULT 0,
				tokens_out INTEGER NOT NULL DEFAULT 0,
				provider_trail TEXT NOT NULL DEFAULT '[]',
				started_at TEXT NOT NULL,
				finished_at TEXT NOT NULL,
				FOREIGN KEY(run_id) REFERENCES runs(id) ON DELETE CASCADE,
				FOREIGN KEY(task_id) REFERENCES tasks(id) ON DELETE CASCADE
			);`,
		`CREATE TABLE IF NOT EXISTS events (
			id TEXT PRIMARY KEY,
			run_id TEXT NOT NULL DEFAULT '',
			type TEXT NOT NULL,
			source TEXT NOT NULL,
			payload TEXT NOT NULL DEFAULT '{}',
			created_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS run_graphs (
			run_id TEXT PRIMARY KEY,
			graph_json TEXT NOT NULL,
			created_at TEXT NOT NULL,
			FOREIGN KEY(run_id) REFERENCES runs(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS optimizer_recommendations (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL,
			goal TEXT NOT NULL,
			strategy_mode TEXT NOT NULL,
			scheduler_mode TEXT NOT NULL,
			max_parallelism INTEGER NOT NULL DEFAULT 8,
			model_preference TEXT NOT NULL DEFAULT 'codex',
			fallback_chain TEXT NOT NULL DEFAULT '[]',
			model_id TEXT NOT NULL DEFAULT '',
			context_budget INTEGER NOT NULL DEFAULT 24000,
			objective_json TEXT NOT NULL,
			confidence TEXT NOT NULL DEFAULT 'low',
			predicted_gates_json TEXT NOT NULL,
			rationale TEXT NOT NULL DEFAULT '',
			source_scorecard_run_id TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'generated',
			applied_run_id TEXT NOT NULL DEFAULT '',
			rejected_reason TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS optimizer_validations (
			id TEXT PRIMARY KEY,
			recommendation_id TEXT NOT NULL,
			eval_run_id TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'pending',
			quality_pass INTEGER NOT NULL DEFAULT 0,
			time_pass INTEGER NOT NULL DEFAULT 0,
			cost_pass INTEGER NOT NULL DEFAULT 0,
			regression_pass INTEGER NOT NULL DEFAULT 0,
			all_pass INTEGER NOT NULL DEFAULT 0,
			summary TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			FOREIGN KEY(recommendation_id) REFERENCES optimizer_recommendations(id) ON DELETE CASCADE
		);`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_run_id ON tasks(run_id);`,
		`CREATE INDEX IF NOT EXISTS idx_events_run_created ON events(run_id, created_at);`,
		`CREATE INDEX IF NOT EXISTS idx_attempts_run_started ON attempts(run_id, started_at);`,
		`CREATE INDEX IF NOT EXISTS idx_optimizer_recommendations_project_created ON optimizer_recommendations(project_id, created_at);`,
		`CREATE INDEX IF NOT EXISTS idx_optimizer_recommendations_status_created ON optimizer_recommendations(status, created_at);`,
		`CREATE INDEX IF NOT EXISTS idx_optimizer_validations_recommendation ON optimizer_validations(recommendation_id, created_at);`,
	}

	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}

	if err := s.ensureRunColumns(); err != nil {
		return err
	}
	if err := s.ensureTaskColumns(); err != nil {
		return err
	}
	if err := s.ensureAttemptColumns(); err != nil {
		return err
	}

	return nil
}

func (s *Store) ensureRunColumns() error {
	required := []storeColumn{
		{name: "skills", stmt: `ALTER TABLE runs ADD COLUMN skills TEXT NOT NULL DEFAULT '[]'`},
		{name: "skill_selection_mode", stmt: `ALTER TABLE runs ADD COLUMN skill_selection_mode TEXT NOT NULL DEFAULT 'auto'`},
		{name: "scheduler_mode", stmt: `ALTER TABLE runs ADD COLUMN scheduler_mode TEXT NOT NULL DEFAULT 'dag_parallel_v1'`},
		{name: "max_parallelism", stmt: `ALTER TABLE runs ADD COLUMN max_parallelism INTEGER NOT NULL DEFAULT 8`},
		{name: "fallback_triggered", stmt: `ALTER TABLE runs ADD COLUMN fallback_triggered INTEGER NOT NULL DEFAULT 0`},
		{name: "model_preference", stmt: `ALTER TABLE runs ADD COLUMN model_preference TEXT NOT NULL DEFAULT 'codex'`},
		{name: "fallback_chain", stmt: `ALTER TABLE runs ADD COLUMN fallback_chain TEXT NOT NULL DEFAULT '[]'`},
		{name: "model_id", stmt: `ALTER TABLE runs ADD COLUMN model_id TEXT NOT NULL DEFAULT ''`},
		{name: "context_budget", stmt: `ALTER TABLE runs ADD COLUMN context_budget INTEGER NOT NULL DEFAULT 24000`},
		{name: "blueprint_profile", stmt: `ALTER TABLE runs ADD COLUMN blueprint_profile TEXT NOT NULL DEFAULT ''`},
		{name: "blueprint_modules", stmt: `ALTER TABLE runs ADD COLUMN blueprint_modules TEXT NOT NULL DEFAULT '[]'`},
		{name: "bootstrap", stmt: `ALTER TABLE runs ADD COLUMN bootstrap INTEGER NOT NULL DEFAULT 0`},
		{name: "optimizer_recommendation_id", stmt: `ALTER TABLE runs ADD COLUMN optimizer_recommendation_id TEXT NOT NULL DEFAULT ''`},
		{name: "optimizer_confidence", stmt: `ALTER TABLE runs ADD COLUMN optimizer_confidence TEXT NOT NULL DEFAULT ''`},
	}

	return s.ensureColumns("runs", required)
}

func (s *Store) ensureTaskColumns() error {
	required := []storeColumn{
		{name: "lifecycle_state", stmt: `ALTER TABLE tasks ADD COLUMN lifecycle_state TEXT NOT NULL DEFAULT 'blocked'`},
		{name: "priority", stmt: `ALTER TABLE tasks ADD COLUMN priority INTEGER NOT NULL DEFAULT 0`},
		{name: "max_attempts", stmt: `ALTER TABLE tasks ADD COLUMN max_attempts INTEGER NOT NULL DEFAULT 3`},
		{name: "started_at", stmt: `ALTER TABLE tasks ADD COLUMN started_at TEXT`},
		{name: "finished_at", stmt: `ALTER TABLE tasks ADD COLUMN finished_at TEXT`},
	}

	return s.ensureColumns("tasks", required)
}

func (s *Store) ensureAttemptColumns() error {
	required := []storeColumn{
		{name: "provider", stmt: `ALTER TABLE attempts ADD COLUMN provider TEXT NOT NULL DEFAULT ''`},
		{name: "model_id", stmt: `ALTER TABLE attempts ADD COLUMN model_id TEXT NOT NULL DEFAULT ''`},
		{name: "fallback_index", stmt: `ALTER TABLE attempts ADD COLUMN fallback_index INTEGER NOT NULL DEFAULT 0`},
		{name: "latency_ms", stmt: `ALTER TABLE attempts ADD COLUMN latency_ms INTEGER NOT NULL DEFAULT 0`},
		{name: "tokens_in", stmt: `ALTER TABLE attempts ADD COLUMN tokens_in INTEGER NOT NULL DEFAULT 0`},
		{name: "tokens_out", stmt: `ALTER TABLE attempts ADD COLUMN tokens_out INTEGER NOT NULL DEFAULT 0`},
		{name: "provider_trail", stmt: `ALTER TABLE attempts ADD COLUMN provider_trail TEXT NOT NULL DEFAULT '[]'`},
	}

	return s.ensureColumns("attempts", required)
}

func (s *Store) ensureColumns(table string, required []storeColumn) error {
	rows, err := s.db.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		return fmt.Errorf("inspect %s columns: %w", table, err)
	}
	defer rows.Close()

	existing := make(map[string]struct{})
	for rows.Next() {
		var cid int
		var name string
		var colType string
		var notNull int
		var defaultVal sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &colType, &notNull, &defaultVal, &pk); err != nil {
			return fmt.Errorf("scan %s columns: %w", table, err)
		}
		existing[name] = struct{}{}
	}

	for _, col := range required {
		if _, ok := existing[col.name]; ok {
			continue
		}
		if _, err := s.db.Exec(col.stmt); err != nil {
			return fmt.Errorf("add %s column %s: %w", table, col.name, err)
		}
	}

	return nil
}

func (s *Store) CreateRun(opts CreateRunOptions) (contracts.Run, error) {
	if opts.SchedulerMode == "" {
		opts.SchedulerMode = contracts.SchedulerModeDAGParallelV1
	}
	if opts.MaxParallelism <= 0 {
		opts.MaxParallelism = 8
	}
	if opts.MaxParallelism > 32 {
		opts.MaxParallelism = 32
	}
	if strings.TrimSpace(opts.SkillSelectionMode) == "" {
		opts.SkillSelectionMode = "auto"
	}
	if strings.TrimSpace(opts.ModelPreference) == "" {
		opts.ModelPreference = "codex"
	}
	if opts.ContextBudget <= 0 {
		opts.ContextBudget = 24000
	}
	if opts.ContextBudget > 200000 {
		opts.ContextBudget = 200000
	}

	modulesRaw, err := json.Marshal(opts.BlueprintModules)
	if err != nil {
		return contracts.Run{}, fmt.Errorf("marshal blueprint modules: %w", err)
	}
	skillsRaw, err := json.Marshal(opts.Skills)
	if err != nil {
		return contracts.Run{}, fmt.Errorf("marshal skills: %w", err)
	}
	fallbackRaw, err := json.Marshal(opts.FallbackChain)
	if err != nil {
		return contracts.Run{}, fmt.Errorf("marshal fallback chain: %w", err)
	}

	now := time.Now().UTC()
	run := contracts.Run{
		ID:                        uuid.NewString(),
		ProjectID:                 opts.ProjectID,
		Goal:                      opts.Goal,
		Mode:                      opts.Mode,
		Skills:                    append([]string{}, opts.Skills...),
		SkillSelectionMode:        opts.SkillSelectionMode,
		Status:                    contracts.RunQueued,
		SchedulerMode:             opts.SchedulerMode,
		MaxParallelism:            opts.MaxParallelism,
		FallbackTriggered:         opts.FallbackTriggered,
		ModelPreference:           opts.ModelPreference,
		FallbackChain:             opts.FallbackChain,
		ModelID:                   strings.TrimSpace(opts.ModelID),
		ContextBudget:             opts.ContextBudget,
		BlueprintProfile:          opts.BlueprintProfile,
		BlueprintModules:          opts.BlueprintModules,
		Bootstrap:                 opts.Bootstrap,
		OptimizerRecommendationID: strings.TrimSpace(opts.OptimizerRecommendationID),
		OptimizerConfidence:       strings.ToLower(strings.TrimSpace(opts.OptimizerConfidence)),
		CreatedAt:                 now,
		UpdatedAt:                 now,
	}

	bootstrapInt := 0
	if run.Bootstrap {
		bootstrapInt = 1
	}
	fallbackInt := 0
	if run.FallbackTriggered {
		fallbackInt = 1
	}

	_, err = s.db.Exec(
		`INSERT INTO runs (id, project_id, goal, mode, skills, skill_selection_mode, status, error, scheduler_mode, max_parallelism, fallback_triggered, model_preference, fallback_chain, model_id, context_budget, blueprint_profile, blueprint_modules, bootstrap, optimizer_recommendation_id, optimizer_confidence, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		run.ID,
		run.ProjectID,
		run.Goal,
		run.Mode,
		string(skillsRaw),
		run.SkillSelectionMode,
		run.Status,
		run.Error,
		run.SchedulerMode,
		run.MaxParallelism,
		fallbackInt,
		run.ModelPreference,
		string(fallbackRaw),
		run.ModelID,
		run.ContextBudget,
		run.BlueprintProfile,
		string(modulesRaw),
		bootstrapInt,
		run.OptimizerRecommendationID,
		run.OptimizerConfidence,
		run.CreatedAt.Format(time.RFC3339Nano),
		run.UpdatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return contracts.Run{}, fmt.Errorf("insert run: %w", err)
	}

	return run, nil
}

func (s *Store) GetRun(runID string) (contracts.Run, error) {
	row := s.db.QueryRow(
		`SELECT id, project_id, goal, mode, skills, skill_selection_mode, status, error, scheduler_mode, max_parallelism, fallback_triggered, model_preference, fallback_chain, model_id, context_budget, blueprint_profile, blueprint_modules, bootstrap, optimizer_recommendation_id, optimizer_confidence, created_at, updated_at
		 FROM runs WHERE id = ?`,
		runID,
	)
	run, err := scanRun(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return contracts.Run{}, err
		}
		return contracts.Run{}, fmt.Errorf("scan run: %w", err)
	}

	return run, nil
}

func (s *Store) UpdateRunStatus(runID string, status contracts.RunStatus, errorMessage string) error {
	_, err := s.db.Exec(
		`UPDATE runs SET status = ?, error = ?, updated_at = ? WHERE id = ?`,
		status,
		errorMessage,
		time.Now().UTC().Format(time.RFC3339Nano),
		runID,
	)
	if err != nil {
		return fmt.Errorf("update run status: %w", err)
	}
	return nil
}

func (s *Store) SetRunFallbackTriggered(runID string, triggered bool) error {
	value := 0
	if triggered {
		value = 1
	}

	_, err := s.db.Exec(
		`UPDATE runs SET fallback_triggered = ?, updated_at = ? WHERE id = ?`,
		value,
		time.Now().UTC().Format(time.RFC3339Nano),
		runID,
	)
	if err != nil {
		return fmt.Errorf("set run fallback: %w", err)
	}
	return nil
}

func (s *Store) ListRuns(limit int) ([]contracts.Run, error) {
	if limit <= 0 {
		limit = 20
	}

	rows, err := s.db.Query(
		`SELECT id, project_id, goal, mode, skills, skill_selection_mode, status, error, scheduler_mode, max_parallelism, fallback_triggered, model_preference, fallback_chain, model_id, context_budget, blueprint_profile, blueprint_modules, bootstrap, optimizer_recommendation_id, optimizer_confidence, created_at, updated_at
		 FROM runs
		 ORDER BY created_at DESC
		 LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list runs: %w", err)
	}
	defer rows.Close()

	out := make([]contracts.Run, 0, limit)
	for rows.Next() {
		run, err := scanRun(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, run)
	}
	return out, nil
}

func (s *Store) CreateTask(runID, name, agent string, dependsOn []string) (contracts.Task, error) {
	lifecycle := contracts.TaskLifecycleBlocked
	if len(dependsOn) == 0 {
		lifecycle = contracts.TaskLifecycleReady
	}
	return s.CreateTaskWithOptions(runID, CreateTaskOptions{
		Name:           name,
		Agent:          agent,
		DependsOn:      dependsOn,
		Priority:       0,
		MaxAttempts:    3,
		LifecycleState: lifecycle,
	})
}

func (s *Store) CreateTaskWithOptions(runID string, opts CreateTaskOptions) (contracts.Task, error) {
	now := time.Now().UTC()
	depsRaw, err := json.Marshal(opts.DependsOn)
	if err != nil {
		return contracts.Task{}, fmt.Errorf("marshal depends_on: %w", err)
	}

	if opts.MaxAttempts <= 0 {
		opts.MaxAttempts = 3
	}
	if opts.LifecycleState == "" {
		opts.LifecycleState = contracts.TaskLifecycleBlocked
		if len(opts.DependsOn) == 0 {
			opts.LifecycleState = contracts.TaskLifecycleReady
		}
	}

	task := contracts.Task{
		ID:             uuid.NewString(),
		RunID:          runID,
		Name:           opts.Name,
		Agent:          opts.Agent,
		Status:         contracts.TaskPending,
		LifecycleState: opts.LifecycleState,
		DependsOn:      opts.DependsOn,
		Priority:       opts.Priority,
		MaxAttempts:    opts.MaxAttempts,
		Attempt:        0,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	_, err = s.db.Exec(
		`INSERT INTO tasks (id, run_id, name, agent, status, lifecycle_state, depends_on, priority, max_attempts, attempt, last_error, created_at, updated_at, started_at, finished_at, completed_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NULL, NULL, NULL)`,
		task.ID,
		task.RunID,
		task.Name,
		task.Agent,
		task.Status,
		task.LifecycleState,
		string(depsRaw),
		task.Priority,
		task.MaxAttempts,
		task.Attempt,
		task.LastError,
		task.CreatedAt.Format(time.RFC3339Nano),
		task.UpdatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return contracts.Task{}, fmt.Errorf("insert task: %w", err)
	}

	return task, nil
}

func (s *Store) GetTask(taskID string) (contracts.Task, error) {
	row := s.db.QueryRow(`SELECT id, run_id, name, agent, status, lifecycle_state, depends_on, priority, max_attempts, attempt, last_error, created_at, updated_at, started_at, finished_at, completed_at FROM tasks WHERE id = ?`, taskID)
	return scanTask(row)
}

func (s *Store) ListTasksByRun(runID string) ([]contracts.Task, error) {
	rows, err := s.db.Query(
		`SELECT id, run_id, name, agent, status, lifecycle_state, depends_on, priority, max_attempts, attempt, last_error, created_at, updated_at, started_at, finished_at, completed_at
		 FROM tasks WHERE run_id = ? ORDER BY created_at ASC`,
		runID,
	)
	if err != nil {
		return nil, fmt.Errorf("list tasks by run: %w", err)
	}
	defer rows.Close()

	var out []contracts.Task
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, task)
	}
	return out, nil
}

func (s *Store) UpdateTaskLifecycle(taskID string, lifecycle contracts.TaskLifecycleState) error {
	if lifecycle == "" {
		return nil
	}
	_, err := s.db.Exec(
		`UPDATE tasks SET lifecycle_state = ?, updated_at = ? WHERE id = ?`,
		lifecycle,
		time.Now().UTC().Format(time.RFC3339Nano),
		taskID,
	)
	if err != nil {
		return fmt.Errorf("update task lifecycle: %w", err)
	}
	return nil
}

func (s *Store) UpdateTaskStatus(taskID string, status contracts.TaskStatus, errorMessage string) error {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	lifecycle := lifecycleFromStatus(status)

	startedAt := sql.NullString{}
	finishedAt := sql.NullString{}
	completedAt := sql.NullString{}

	if status == contracts.TaskRunning {
		startedAt = sql.NullString{String: now, Valid: true}
	}

	if status == contracts.TaskCompleted || status == contracts.TaskFailed || status == contracts.TaskSkipped || status == contracts.TaskCancelled {
		finishedAt = sql.NullString{String: now, Valid: true}
	}
	if status == contracts.TaskCompleted {
		completedAt = sql.NullString{String: now, Valid: true}
	}

	_, err := s.db.Exec(
		`UPDATE tasks
		 SET status = ?, lifecycle_state = ?, last_error = ?, updated_at = ?,
		     started_at = CASE WHEN ? != '' THEN ? ELSE started_at END,
		     finished_at = CASE WHEN ? != '' THEN ? ELSE finished_at END,
		     completed_at = CASE WHEN ? != '' THEN ? ELSE completed_at END
		 WHERE id = ?`,
		status,
		lifecycle,
		errorMessage,
		now,
		startedAt.String,
		startedAt.String,
		finishedAt.String,
		finishedAt.String,
		completedAt.String,
		completedAt.String,
		taskID,
	)
	if err != nil {
		return fmt.Errorf("update task status: %w", err)
	}
	return nil
}

func (s *Store) IncrementTaskAttempt(taskID string) (int, error) {
	_, err := s.db.Exec(`UPDATE tasks SET attempt = attempt + 1, updated_at = ? WHERE id = ?`, time.Now().UTC().Format(time.RFC3339Nano), taskID)
	if err != nil {
		return 0, fmt.Errorf("increment task attempt: %w", err)
	}

	task, err := s.GetTask(taskID)
	if err != nil {
		return 0, err
	}
	return task.Attempt, nil
}

func (s *Store) CreateAttempt(opts CreateAttemptOptions) (contracts.TaskAttempt, error) {
	if opts.StartedAt.IsZero() {
		opts.StartedAt = time.Now().UTC()
	}
	if opts.FinishedAt.IsZero() {
		opts.FinishedAt = opts.StartedAt
	}
	providerTrailRaw, err := json.Marshal(opts.ProviderTrail)
	if err != nil {
		return contracts.TaskAttempt{}, fmt.Errorf("marshal provider trail: %w", err)
	}

	attempt := contracts.TaskAttempt{
		ID:            uuid.NewString(),
		RunID:         opts.RunID,
		TaskID:        opts.TaskID,
		Agent:         opts.Agent,
		Status:        opts.Status,
		Log:           opts.Log,
		Error:         opts.Error,
		Provider:      opts.Provider,
		ModelID:       opts.ModelID,
		FallbackIndex: opts.FallbackIndex,
		LatencyMs:     opts.LatencyMs,
		TokensIn:      opts.TokensIn,
		TokensOut:     opts.TokensOut,
		ProviderTrail: append([]contracts.ProviderAttempt{}, opts.ProviderTrail...),
		StartedAt:     opts.StartedAt.UTC(),
		FinishedAt:    opts.FinishedAt.UTC(),
	}

	_, err = s.db.Exec(
		`INSERT INTO attempts (id, run_id, task_id, agent, status, log, error, provider, model_id, fallback_index, latency_ms, tokens_in, tokens_out, provider_trail, started_at, finished_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		attempt.ID,
		attempt.RunID,
		attempt.TaskID,
		attempt.Agent,
		attempt.Status,
		attempt.Log,
		attempt.Error,
		attempt.Provider,
		attempt.ModelID,
		attempt.FallbackIndex,
		attempt.LatencyMs,
		attempt.TokensIn,
		attempt.TokensOut,
		string(providerTrailRaw),
		attempt.StartedAt.Format(time.RFC3339Nano),
		attempt.FinishedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return contracts.TaskAttempt{}, fmt.Errorf("insert attempt: %w", err)
	}

	return attempt, nil
}

func (s *Store) ListAttemptsByRun(runID string) ([]contracts.TaskAttempt, error) {
	rows, err := s.db.Query(
		`SELECT id, run_id, task_id, agent, status, log, error, provider, model_id, fallback_index, latency_ms, tokens_in, tokens_out, provider_trail, started_at, finished_at
		 FROM attempts WHERE run_id = ? ORDER BY started_at ASC`,
		runID,
	)
	if err != nil {
		return nil, fmt.Errorf("list attempts by run: %w", err)
	}
	defer rows.Close()

	out := make([]contracts.TaskAttempt, 0, 16)
	for rows.Next() {
		var attempt contracts.TaskAttempt
		var startedAt string
		var finishedAt string
		var providerTrailRaw string
		if err := rows.Scan(
			&attempt.ID,
			&attempt.RunID,
			&attempt.TaskID,
			&attempt.Agent,
			&attempt.Status,
			&attempt.Log,
			&attempt.Error,
			&attempt.Provider,
			&attempt.ModelID,
			&attempt.FallbackIndex,
			&attempt.LatencyMs,
			&attempt.TokensIn,
			&attempt.TokensOut,
			&providerTrailRaw,
			&startedAt,
			&finishedAt,
		); err != nil {
			return nil, fmt.Errorf("scan attempt: %w", err)
		}
		if providerTrailRaw != "" {
			_ = json.Unmarshal([]byte(providerTrailRaw), &attempt.ProviderTrail)
		}
		attempt.StartedAt, _ = time.Parse(time.RFC3339Nano, startedAt)
		attempt.FinishedAt, _ = time.Parse(time.RFC3339Nano, finishedAt)
		out = append(out, attempt)
	}
	return out, nil
}

func (s *Store) AppendEvent(ev contracts.Event) (contracts.Event, error) {
	if ev.ID == "" {
		ev.ID = uuid.NewString()
	}
	if ev.CreatedAt.IsZero() {
		ev.CreatedAt = time.Now().UTC()
	}
	payloadRaw, err := json.Marshal(ev.Payload)
	if err != nil {
		return contracts.Event{}, fmt.Errorf("marshal event payload: %w", err)
	}

	_, err = s.db.Exec(
		`INSERT INTO events (id, run_id, type, source, payload, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		ev.ID,
		ev.RunID,
		ev.Type,
		ev.Source,
		string(payloadRaw),
		ev.CreatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return contracts.Event{}, fmt.Errorf("insert event: %w", err)
	}

	return ev, nil
}

func (s *Store) ListEvents(runID string, limit int) ([]contracts.Event, error) {
	if limit <= 0 {
		limit = 200
	}

	query := `SELECT id, run_id, type, source, payload, created_at FROM (
		SELECT id, run_id, type, source, payload, created_at
		FROM events`
	args := make([]interface{}, 0, 2)
	if runID != "" {
		query += ` WHERE run_id = ?`
		args = append(args, runID)
	}
	query += ` ORDER BY created_at DESC, id DESC LIMIT ?
	) ordered_events
	ORDER BY created_at ASC, id ASC`
	args = append(args, limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	defer rows.Close()

	var out []contracts.Event
	for rows.Next() {
		var ev contracts.Event
		var payloadRaw string
		var createdAt string
		if err := rows.Scan(&ev.ID, &ev.RunID, &ev.Type, &ev.Source, &payloadRaw, &createdAt); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		if payloadRaw != "" {
			_ = json.Unmarshal([]byte(payloadRaw), &ev.Payload)
		}
		ev.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
		out = append(out, ev)
	}

	return out, nil
}

func (s *Store) CreateOptimizerRecommendation(rec OptimizerRecommendationRecord) (OptimizerRecommendationRecord, error) {
	if strings.TrimSpace(rec.ID) == "" {
		rec.ID = "opt-rec-" + uuid.NewString()
	}
	rec.ProjectID = strings.TrimSpace(rec.ProjectID)
	if rec.ProjectID == "" {
		rec.ProjectID = "biometrics"
	}
	rec.Goal = strings.TrimSpace(rec.Goal)
	if rec.Goal == "" {
		return OptimizerRecommendationRecord{}, fmt.Errorf("goal is required")
	}
	rec.StrategyMode = strings.ToLower(strings.TrimSpace(rec.StrategyMode))
	if rec.StrategyMode == "" {
		rec.StrategyMode = "adaptive"
	}
	rec.SchedulerMode = strings.ToLower(strings.TrimSpace(rec.SchedulerMode))
	if rec.SchedulerMode == "" {
		rec.SchedulerMode = string(contracts.SchedulerModeDAGParallelV1)
	}
	if rec.MaxParallelism <= 0 {
		rec.MaxParallelism = 8
	}
	if rec.MaxParallelism > 32 {
		rec.MaxParallelism = 32
	}
	rec.ModelPreference = strings.ToLower(strings.TrimSpace(rec.ModelPreference))
	if rec.ModelPreference == "" {
		rec.ModelPreference = "codex"
	}
	rec.ModelID = strings.TrimSpace(rec.ModelID)
	if rec.ContextBudget <= 0 {
		rec.ContextBudget = 24000
	}
	if rec.ContextBudget > 200000 {
		rec.ContextBudget = 200000
	}
	rec.Confidence = strings.ToLower(strings.TrimSpace(rec.Confidence))
	if rec.Confidence == "" {
		rec.Confidence = "low"
	}
	rec.Status = normalizeOptimizerRecommendationStatus(rec.Status)
	rec.Rationale = strings.TrimSpace(rec.Rationale)
	rec.SourceScorecardRunID = strings.TrimSpace(rec.SourceScorecardRunID)
	rec.AppliedRunID = strings.TrimSpace(rec.AppliedRunID)
	rec.RejectedReason = strings.TrimSpace(rec.RejectedReason)

	now := time.Now().UTC()
	if rec.CreatedAt.IsZero() {
		rec.CreatedAt = now
	}
	rec.UpdatedAt = now

	fallbackRaw, err := json.Marshal(rec.FallbackChain)
	if err != nil {
		return OptimizerRecommendationRecord{}, fmt.Errorf("marshal optimizer fallback chain: %w", err)
	}
	objectiveRaw, err := json.Marshal(rec.Objective)
	if err != nil {
		return OptimizerRecommendationRecord{}, fmt.Errorf("marshal optimizer objective: %w", err)
	}
	predictedRaw, err := json.Marshal(rec.PredictedGates)
	if err != nil {
		return OptimizerRecommendationRecord{}, fmt.Errorf("marshal optimizer predicted gates: %w", err)
	}

	_, err = s.db.Exec(
		`INSERT INTO optimizer_recommendations (
			id, project_id, goal, strategy_mode, scheduler_mode, max_parallelism,
			model_preference, fallback_chain, model_id, context_budget, objective_json,
			confidence, predicted_gates_json, rationale, source_scorecard_run_id, status,
			applied_run_id, rejected_reason, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		rec.ID,
		rec.ProjectID,
		rec.Goal,
		rec.StrategyMode,
		rec.SchedulerMode,
		rec.MaxParallelism,
		rec.ModelPreference,
		string(fallbackRaw),
		rec.ModelID,
		rec.ContextBudget,
		string(objectiveRaw),
		rec.Confidence,
		string(predictedRaw),
		rec.Rationale,
		rec.SourceScorecardRunID,
		rec.Status,
		rec.AppliedRunID,
		rec.RejectedReason,
		rec.CreatedAt.Format(time.RFC3339Nano),
		rec.UpdatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return OptimizerRecommendationRecord{}, fmt.Errorf("insert optimizer recommendation: %w", err)
	}
	return rec, nil
}

func (s *Store) ListOptimizerRecommendations(opts ListOptimizerRecommendationsOptions) ([]OptimizerRecommendationRecord, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 200 {
		limit = 200
	}

	conditions := make([]string, 0, 2)
	args := make([]interface{}, 0, 3)
	if projectID := strings.TrimSpace(opts.ProjectID); projectID != "" {
		conditions = append(conditions, "project_id = ?")
		args = append(args, projectID)
	}
	if status := strings.TrimSpace(opts.Status); status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, strings.ToLower(status))
	}

	query := `SELECT
		id, project_id, goal, strategy_mode, scheduler_mode, max_parallelism,
		model_preference, fallback_chain, model_id, context_budget, objective_json,
		confidence, predicted_gates_json, rationale, source_scorecard_run_id, status,
		applied_run_id, rejected_reason, created_at, updated_at
		FROM optimizer_recommendations`
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY created_at DESC LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list optimizer recommendations: %w", err)
	}
	defer rows.Close()

	out := make([]OptimizerRecommendationRecord, 0, limit)
	for rows.Next() {
		rec, err := scanOptimizerRecommendation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, nil
}

func (s *Store) GetOptimizerRecommendation(id string) (OptimizerRecommendationRecord, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return OptimizerRecommendationRecord{}, fmt.Errorf("recommendation id is required")
	}

	row := s.db.QueryRow(
		`SELECT
			id, project_id, goal, strategy_mode, scheduler_mode, max_parallelism,
			model_preference, fallback_chain, model_id, context_budget, objective_json,
			confidence, predicted_gates_json, rationale, source_scorecard_run_id, status,
			applied_run_id, rejected_reason, created_at, updated_at
		FROM optimizer_recommendations WHERE id = ?`,
		id,
	)
	rec, err := scanOptimizerRecommendation(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return OptimizerRecommendationRecord{}, err
		}
		return OptimizerRecommendationRecord{}, fmt.Errorf("get optimizer recommendation: %w", err)
	}
	return rec, nil
}

func (s *Store) UpdateOptimizerRecommendationStatus(id, status, appliedRunID, rejectedReason string) (OptimizerRecommendationRecord, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return OptimizerRecommendationRecord{}, fmt.Errorf("recommendation id is required")
	}
	status = normalizeOptimizerRecommendationStatus(status)
	appliedRunID = strings.TrimSpace(appliedRunID)
	rejectedReason = strings.TrimSpace(rejectedReason)

	_, err := s.db.Exec(
		`UPDATE optimizer_recommendations
		 SET status = ?, applied_run_id = ?, rejected_reason = ?, updated_at = ?
		 WHERE id = ?`,
		status,
		appliedRunID,
		rejectedReason,
		time.Now().UTC().Format(time.RFC3339Nano),
		id,
	)
	if err != nil {
		return OptimizerRecommendationRecord{}, fmt.Errorf("update optimizer recommendation status: %w", err)
	}
	return s.GetOptimizerRecommendation(id)
}

func (s *Store) UpsertOptimizerValidation(rec OptimizerValidationRecord) (OptimizerValidationRecord, error) {
	if strings.TrimSpace(rec.RecommendationID) == "" {
		return OptimizerValidationRecord{}, fmt.Errorf("recommendation_id is required")
	}
	if strings.TrimSpace(rec.ID) == "" {
		rec.ID = "opt-val-" + uuid.NewString()
	}
	rec.Status = strings.ToLower(strings.TrimSpace(rec.Status))
	if rec.Status == "" {
		rec.Status = "pending"
	}
	rec.EvalRunID = strings.TrimSpace(rec.EvalRunID)
	rec.Summary = strings.TrimSpace(rec.Summary)

	now := time.Now().UTC()
	if rec.CreatedAt.IsZero() {
		rec.CreatedAt = now
	}
	rec.UpdatedAt = now

	_, err := s.db.Exec(
		`INSERT INTO optimizer_validations (
			id, recommendation_id, eval_run_id, status,
			quality_pass, time_pass, cost_pass, regression_pass, all_pass,
			summary, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			eval_run_id = excluded.eval_run_id,
			status = excluded.status,
			quality_pass = excluded.quality_pass,
			time_pass = excluded.time_pass,
			cost_pass = excluded.cost_pass,
			regression_pass = excluded.regression_pass,
			all_pass = excluded.all_pass,
			summary = excluded.summary,
			updated_at = excluded.updated_at`,
		rec.ID,
		rec.RecommendationID,
		rec.EvalRunID,
		rec.Status,
		boolToInt(rec.QualityPass),
		boolToInt(rec.TimePass),
		boolToInt(rec.CostPass),
		boolToInt(rec.RegressionPass),
		boolToInt(rec.AllPass),
		rec.Summary,
		rec.CreatedAt.Format(time.RFC3339Nano),
		rec.UpdatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return OptimizerValidationRecord{}, fmt.Errorf("upsert optimizer validation: %w", err)
	}
	return rec, nil
}

func (s *Store) GetOptimizerValidationByRecommendation(recommendationID string) (OptimizerValidationRecord, error) {
	recommendationID = strings.TrimSpace(recommendationID)
	if recommendationID == "" {
		return OptimizerValidationRecord{}, fmt.Errorf("recommendation id is required")
	}

	row := s.db.QueryRow(
		`SELECT
			id, recommendation_id, eval_run_id, status,
			quality_pass, time_pass, cost_pass, regression_pass, all_pass,
			summary, created_at, updated_at
		FROM optimizer_validations
		WHERE recommendation_id = ?
		ORDER BY created_at DESC
		LIMIT 1`,
		recommendationID,
	)
	return scanOptimizerValidation(row)
}

func (s *Store) SaveRunGraph(runID string, graph contracts.TaskGraph) error {
	if graph.RunID == "" {
		graph.RunID = runID
	}
	if graph.CreatedAt.IsZero() {
		graph.CreatedAt = time.Now().UTC()
	}

	raw, err := json.Marshal(graph)
	if err != nil {
		return fmt.Errorf("marshal run graph: %w", err)
	}

	_, err = s.db.Exec(
		`INSERT INTO run_graphs (run_id, graph_json, created_at)
		 VALUES (?, ?, ?)
		 ON CONFLICT(run_id) DO UPDATE SET graph_json = excluded.graph_json, created_at = excluded.created_at`,
		runID,
		string(raw),
		graph.CreatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("save run graph: %w", err)
	}
	return nil
}

func (s *Store) GetRunGraph(runID string) (contracts.TaskGraph, error) {
	row := s.db.QueryRow(`SELECT graph_json FROM run_graphs WHERE run_id = ?`, runID)
	var raw string
	if err := row.Scan(&raw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return contracts.TaskGraph{}, err
		}
		return contracts.TaskGraph{}, fmt.Errorf("scan run graph: %w", err)
	}

	var graph contracts.TaskGraph
	if err := json.Unmarshal([]byte(raw), &graph); err != nil {
		return contracts.TaskGraph{}, fmt.Errorf("unmarshal run graph: %w", err)
	}
	if graph.RunID == "" {
		graph.RunID = runID
	}
	return graph, nil
}

func lifecycleFromStatus(status contracts.TaskStatus) contracts.TaskLifecycleState {
	switch status {
	case contracts.TaskRunning:
		return contracts.TaskLifecycleRunning
	case contracts.TaskCompleted:
		return contracts.TaskLifecycleCompleted
	case contracts.TaskFailed:
		return contracts.TaskLifecycleFailed
	case contracts.TaskSkipped:
		return contracts.TaskLifecycleSkipped
	case contracts.TaskCancelled:
		return contracts.TaskLifecycleCancelled
	default:
		return contracts.TaskLifecycleBlocked
	}
}

func scanRun(scanner interface {
	Scan(dest ...interface{}) error
}) (contracts.Run, error) {
	var run contracts.Run
	var skillsRaw string
	var skillSelectionMode string
	var fallbackRaw string
	var modulesRaw string
	var fallbackInt int
	var bootstrapInt int
	var optimizerRecommendationID string
	var optimizerConfidence string
	var createdAt string
	var updatedAt string

	if err := scanner.Scan(
		&run.ID,
		&run.ProjectID,
		&run.Goal,
		&run.Mode,
		&skillsRaw,
		&skillSelectionMode,
		&run.Status,
		&run.Error,
		&run.SchedulerMode,
		&run.MaxParallelism,
		&fallbackInt,
		&run.ModelPreference,
		&fallbackRaw,
		&run.ModelID,
		&run.ContextBudget,
		&run.BlueprintProfile,
		&modulesRaw,
		&bootstrapInt,
		&optimizerRecommendationID,
		&optimizerConfidence,
		&createdAt,
		&updatedAt,
	); err != nil {
		return contracts.Run{}, fmt.Errorf("scan run: %w", err)
	}

	if modulesRaw != "" {
		_ = json.Unmarshal([]byte(modulesRaw), &run.BlueprintModules)
	}
	if skillsRaw != "" {
		_ = json.Unmarshal([]byte(skillsRaw), &run.Skills)
	}
	if fallbackRaw != "" {
		_ = json.Unmarshal([]byte(fallbackRaw), &run.FallbackChain)
	}
	run.SkillSelectionMode = skillSelectionMode
	run.FallbackTriggered = fallbackInt != 0
	run.Bootstrap = bootstrapInt != 0
	run.OptimizerRecommendationID = strings.TrimSpace(optimizerRecommendationID)
	run.OptimizerConfidence = strings.ToLower(strings.TrimSpace(optimizerConfidence))

	run.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	run.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAt)

	if run.BlueprintModules == nil {
		run.BlueprintModules = []string{}
	}
	if run.Skills == nil {
		run.Skills = []string{}
	}
	if strings.TrimSpace(run.SkillSelectionMode) == "" {
		run.SkillSelectionMode = "auto"
	}
	if run.SchedulerMode == "" {
		run.SchedulerMode = contracts.SchedulerModeDAGParallelV1
	}
	if run.MaxParallelism <= 0 {
		run.MaxParallelism = 8
	}
	if run.ContextBudget <= 0 {
		run.ContextBudget = 24000
	}
	if run.ModelPreference == "" {
		run.ModelPreference = "codex"
	}
	if run.FallbackChain == nil {
		run.FallbackChain = []string{}
	}

	return run, nil
}

func scanTask(scanner interface {
	Scan(dest ...interface{}) error
}) (contracts.Task, error) {
	var task contracts.Task
	var depsRaw string
	var createdAt string
	var updatedAt string
	var startedAt sql.NullString
	var finishedAt sql.NullString
	var completedAt sql.NullString
	if err := scanner.Scan(
		&task.ID,
		&task.RunID,
		&task.Name,
		&task.Agent,
		&task.Status,
		&task.LifecycleState,
		&depsRaw,
		&task.Priority,
		&task.MaxAttempts,
		&task.Attempt,
		&task.LastError,
		&createdAt,
		&updatedAt,
		&startedAt,
		&finishedAt,
		&completedAt,
	); err != nil {
		return contracts.Task{}, fmt.Errorf("scan task: %w", err)
	}
	if depsRaw != "" {
		_ = json.Unmarshal([]byte(depsRaw), &task.DependsOn)
	}
	task.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	task.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAt)

	if startedAt.Valid {
		t, err := time.Parse(time.RFC3339Nano, startedAt.String)
		if err == nil {
			task.StartedAt = &t
		}
	}
	if finishedAt.Valid {
		t, err := time.Parse(time.RFC3339Nano, finishedAt.String)
		if err == nil {
			task.FinishedAt = &t
		}
	}
	if completedAt.Valid {
		t, err := time.Parse(time.RFC3339Nano, completedAt.String)
		if err == nil {
			task.CompletedAt = &t
		}
	}

	if task.LifecycleState == "" {
		task.LifecycleState = lifecycleFromStatus(task.Status)
	}
	if task.MaxAttempts <= 0 {
		task.MaxAttempts = 3
	}

	return task, nil
}

func scanOptimizerRecommendation(scanner interface {
	Scan(dest ...interface{}) error
}) (OptimizerRecommendationRecord, error) {
	var rec OptimizerRecommendationRecord
	var fallbackRaw string
	var objectiveRaw string
	var predictedRaw string
	var createdAt string
	var updatedAt string
	if err := scanner.Scan(
		&rec.ID,
		&rec.ProjectID,
		&rec.Goal,
		&rec.StrategyMode,
		&rec.SchedulerMode,
		&rec.MaxParallelism,
		&rec.ModelPreference,
		&fallbackRaw,
		&rec.ModelID,
		&rec.ContextBudget,
		&objectiveRaw,
		&rec.Confidence,
		&predictedRaw,
		&rec.Rationale,
		&rec.SourceScorecardRunID,
		&rec.Status,
		&rec.AppliedRunID,
		&rec.RejectedReason,
		&createdAt,
		&updatedAt,
	); err != nil {
		return OptimizerRecommendationRecord{}, fmt.Errorf("scan optimizer recommendation: %w", err)
	}
	if fallbackRaw != "" {
		_ = json.Unmarshal([]byte(fallbackRaw), &rec.FallbackChain)
	}
	if objectiveRaw != "" {
		_ = json.Unmarshal([]byte(objectiveRaw), &rec.Objective)
	}
	if predictedRaw != "" {
		_ = json.Unmarshal([]byte(predictedRaw), &rec.PredictedGates)
	}
	rec.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	rec.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAt)
	if rec.FallbackChain == nil {
		rec.FallbackChain = []string{}
	}
	return rec, nil
}

func scanOptimizerValidation(scanner interface {
	Scan(dest ...interface{}) error
}) (OptimizerValidationRecord, error) {
	var out OptimizerValidationRecord
	var qualityInt int
	var timeInt int
	var costInt int
	var regressionInt int
	var allInt int
	var createdAt string
	var updatedAt string
	if err := scanner.Scan(
		&out.ID,
		&out.RecommendationID,
		&out.EvalRunID,
		&out.Status,
		&qualityInt,
		&timeInt,
		&costInt,
		&regressionInt,
		&allInt,
		&out.Summary,
		&createdAt,
		&updatedAt,
	); err != nil {
		return OptimizerValidationRecord{}, err
	}
	out.QualityPass = qualityInt != 0
	out.TimePass = timeInt != 0
	out.CostPass = costInt != 0
	out.RegressionPass = regressionInt != 0
	out.AllPass = allInt != 0
	out.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	out.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAt)
	return out, nil
}

func normalizeOptimizerRecommendationStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "applied":
		return "applied"
	case "rejected":
		return "rejected"
	default:
		return "generated"
	}
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
