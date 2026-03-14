package orchestrator

import (
	"sort"
	"strings"
	"sync"
	"time"
)

type MemoryScope string

const (
	MemoryScopeGlobal    MemoryScope = "global"
	MemoryScopeWorkspace MemoryScope = "workspace"
	MemoryScopeRun       MemoryScope = "run"
)

type MemoryRecord struct {
	Scope      MemoryScope `json:"scope"`
	Workspace  string      `json:"workspace,omitempty"`
	RunID      string      `json:"run_id,omitempty"`
	Key        string      `json:"key"`
	Value      string      `json:"value"`
	Source     string      `json:"source"`
	Provenance string      `json:"provenance,omitempty"`
	CreatedAt  time.Time   `json:"created_at"`
	ExpiresAt  *time.Time  `json:"expires_at,omitempty"`
}

func (r MemoryRecord) expired(now time.Time) bool {
	return r.ExpiresAt != nil && !r.ExpiresAt.After(now)
}

type MemoryStore struct {
	mu      sync.RWMutex
	records []MemoryRecord
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{records: make([]MemoryRecord, 0, 64)}
}

func (s *MemoryStore) Put(record MemoryRecord) {
	if strings.TrimSpace(record.Key) == "" {
		return
	}
	now := time.Now().UTC()
	if record.CreatedAt.IsZero() {
		record.CreatedAt = now
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.pruneLocked(now)

	updated := false
	for i := range s.records {
		if s.records[i].Scope == record.Scope &&
			s.records[i].Workspace == record.Workspace &&
			s.records[i].RunID == record.RunID &&
			s.records[i].Key == record.Key {
			s.records[i] = record
			updated = true
			break
		}
	}
	if !updated {
		s.records = append(s.records, record)
	}
}

func (s *MemoryStore) Get(scope MemoryScope, workspace, runID, key string) (MemoryRecord, bool) {
	now := time.Now().UTC()
	s.mu.RLock()
	defer s.mu.RUnlock()
	for i := len(s.records) - 1; i >= 0; i-- {
		record := s.records[i]
		if record.expired(now) {
			continue
		}
		if record.Scope == scope && record.Workspace == workspace && record.RunID == runID && record.Key == key {
			return record, true
		}
	}
	return MemoryRecord{}, false
}

func (s *MemoryStore) List(scope MemoryScope, workspace, runID string) []MemoryRecord {
	now := time.Now().UTC()
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]MemoryRecord, 0, len(s.records))
	for _, record := range s.records {
		if record.expired(now) {
			continue
		}
		if scope != "" && record.Scope != scope {
			continue
		}
		if workspace != "" && record.Workspace != workspace {
			continue
		}
		if runID != "" && record.RunID != runID {
			continue
		}
		out = append(out, record)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.Before(out[j].CreatedAt)
	})
	return out
}

func (s *MemoryStore) PruneExpired() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pruneLocked(time.Now().UTC())
}

func (s *MemoryStore) pruneLocked(now time.Time) {
	if len(s.records) == 0 {
		return
	}
	filtered := s.records[:0]
	for _, record := range s.records {
		if record.expired(now) {
			continue
		}
		filtered = append(filtered, record)
	}
	s.records = filtered
}
