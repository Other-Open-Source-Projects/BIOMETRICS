package durable

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"biometrics-cli/internal/metrics"
)

type Checkpoint struct {
	ID          string                 `json:"id"`
	AgentID     string                 `json:"agent_id"`
	SessionID   string                 `json:"session_id"`
	StepNumber  int                    `json:"step_number"`
	TaskID      string                 `json:"task_id"`
	Input       map[string]interface{} `json:"input"`
	Output      map[string]interface{} `json:"output,omitempty"`
	Status      string                 `json:"status"` // "running", "completed", "failed", "rolled_back"
	Timestamp   time.Time              `json:"timestamp"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	Duration    time.Duration          `json:"duration,omitempty"`
	Error       string                 `json:"error,omitempty"`
	RollbackID  string                 `json:"rollback_id,omitempty"`
}

type Journal struct {
	mu          sync.RWMutex
	checkpoints map[string][]*Checkpoint
	journalDir  string
	maxSteps    int
	persistence PersistenceLayer
}

type PersistenceLayer interface {
	Save(checkpoint *Checkpoint) error
	Load(agentID string) ([]*Checkpoint, error)
	Delete(checkpointID string) error
}

type FilePersistence struct {
	journalDir string
}

func NewFilePersistence(journalDir string) *FilePersistence {
	os.MkdirAll(journalDir, 0755)
	return &FilePersistence{journalDir: journalDir}
}

func (fp *FilePersistence) Save(checkpoint *Checkpoint) error {
	filename := filepath.Join(fp.journalDir, fmt.Sprintf("%s.json", checkpoint.ID))
	data, err := json.MarshalIndent(checkpoint, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal checkpoint: %w", err)
	}
	return os.WriteFile(filename, data, 0644)
}

func (fp *FilePersistence) Load(agentID string) ([]*Checkpoint, error) {
	pattern := filepath.Join(fp.journalDir, fmt.Sprintf("%s_*.json", agentID))
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	checkpoints := make([]*Checkpoint, 0)
	for _, match := range matches {
		data, err := os.ReadFile(match)
		if err != nil {
			continue
		}
		var cp Checkpoint
		if err := json.Unmarshal(data, &cp); err == nil {
			checkpoints = append(checkpoints, &cp)
		}
	}
	return checkpoints, nil
}

func (fp *FilePersistence) Delete(checkpointID string) error {
	filename := filepath.Join(fp.journalDir, fmt.Sprintf("%s.json", checkpointID))
	return os.Remove(filename)
}

func NewJournal(journalDir string, maxSteps int) *Journal {
	os.MkdirAll(journalDir, 0755)
	return &Journal{
		checkpoints: make(map[string][]*Checkpoint),
		journalDir:  journalDir,
		maxSteps:    maxSteps,
		persistence: NewFilePersistence(journalDir),
	}
}

func (j *Journal) CreateCheckpoint(agentID, sessionID, taskID string, input map[string]interface{}) (*Checkpoint, error) {
	j.mu.Lock()
	defer j.mu.Unlock()

	steps := j.checkpoints[agentID]
	stepNumber := len(steps) + 1

	if j.maxSteps > 0 && stepNumber > j.maxSteps {
		return nil, fmt.Errorf("max steps exceeded for agent %s", agentID)
	}

	checkpoint := &Checkpoint{
		ID:         fmt.Sprintf("%s_%d_%d", agentID, stepNumber, time.Now().UnixNano()),
		AgentID:    agentID,
		SessionID:  sessionID,
		StepNumber: stepNumber,
		TaskID:     taskID,
		Input:      input,
		Status:     "running",
		Timestamp:  time.Now(),
	}

	j.checkpoints[agentID] = append(j.checkpoints[agentID], checkpoint)

	if err := j.persistence.Save(checkpoint); err != nil {
		return nil, err
	}

	metrics.DurableCheckpointsCreated.WithLabelValues(agentID).Inc()

	return checkpoint, nil
}

func (j *Journal) CompleteCheckpoint(checkpointID string, output map[string]interface{}) error {
	j.mu.Lock()
	defer j.mu.Unlock()

	for agentID, checkpoints := range j.checkpoints {
		for i, cp := range checkpoints {
			if cp.ID == checkpointID {
				now := time.Now()
				cp.Output = output
				cp.Status = "completed"
				cp.CompletedAt = &now
				cp.Duration = time.Since(cp.Timestamp)

				j.checkpoints[agentID][i] = cp
				j.persistence.Save(cp)

				metrics.DurableCheckpointsCompleted.WithLabelValues(agentID).Inc()
				return nil
			}
		}
	}

	return fmt.Errorf("checkpoint not found: %s", checkpointID)
}

func (j *Journal) FailCheckpoint(checkpointID, errMsg string) error {
	j.mu.Lock()
	defer j.mu.Unlock()

	for agentID, checkpoints := range j.checkpoints {
		for i, cp := range checkpoints {
			if cp.ID == checkpointID {
				cp.Status = "failed"
				cp.Error = errMsg
				cp.CompletedAt = &cp.Timestamp

				j.checkpoints[agentID][i] = cp
				j.persistence.Save(cp)

				metrics.DurableCheckpointsFailed.WithLabelValues(agentID).Inc()
				return nil
			}
		}
	}

	return fmt.Errorf("checkpoint not found: %s", checkpointID)
}

func (j *Journal) GetCompletedSteps(agentID string) []*Checkpoint {
	j.mu.RLock()
	defer j.mu.RUnlock()

	var completed []*Checkpoint
	for _, cp := range j.checkpoints[agentID] {
		if cp.Status == "completed" {
			completed = append(completed, cp)
		}
	}
	return completed
}

func (j *Journal) GetRunningStep(agentID string) *Checkpoint {
	j.mu.RLock()
	defer j.mu.RUnlock()

	for _, cp := range j.checkpoints[agentID] {
		if cp.Status == "running" {
			return cp
		}
	}
	return nil
}

func (j *Journal) Replay(agentID string, executor func(*Checkpoint) error) ([]*Checkpoint, error) {
	j.mu.RLock()
	completed := j.GetCompletedSteps(agentID)
	running := j.GetRunningStep(agentID)
	j.mu.RUnlock()

	replayed := make([]*Checkpoint, 0)

	for _, cp := range completed {
		if err := executor(cp); err != nil {
			return replayed, fmt.Errorf("replay failed at step %d: %w", cp.StepNumber, err)
		}
		replayed = append(replayed, cp)
		metrics.DurableStepsReplayed.WithLabelValues(agentID).Inc()
	}

	if running != nil {
		if err := executor(running); err != nil {
			return replayed, fmt.Errorf("replay failed at running step: %w", err)
		}
		replayed = append(replayed, running)
	}

	return replayed, nil
}

func (j *Journal) GetStats(agentID string) map[string]interface{} {
	j.mu.RLock()
	defer j.mu.RUnlock()

	checkpoints := j.checkpoints[agentID]
	running := 0
	completed := 0
	failed := 0

	for _, cp := range checkpoints {
		switch cp.Status {
		case "running":
			running++
		case "completed":
			completed++
		case "failed":
			failed++
		}
	}

	return map[string]interface{}{
		"total":     len(checkpoints),
		"running":   running,
		"completed": completed,
		"failed":    failed,
		"step_size": j.maxSteps,
	}
}

func (j *Journal) Clear(agentID string) {
	j.mu.Lock()
	defer j.mu.Unlock()

	delete(j.checkpoints, agentID)
}
