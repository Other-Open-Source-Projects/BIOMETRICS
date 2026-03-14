package orchestrator

import (
	"testing"
	"time"
)

func TestMemoryStoreTTLAndProvenance(t *testing.T) {
	store := NewMemoryStore()
	now := time.Now().UTC()
	expiresSoon := now.Add(40 * time.Millisecond)

	store.Put(MemoryRecord{
		Scope:      MemoryScopeRun,
		Workspace:  "/tmp/workspace",
		RunID:      "run-1",
		Key:        "agent_context.coder",
		Value:      "coder context",
		Source:     "orchestrator.plan",
		Provenance: "subagent-profile",
		CreatedAt:  now,
		ExpiresAt:  &expiresSoon,
	})

	record, ok := store.Get(MemoryScopeRun, "/tmp/workspace", "run-1", "agent_context.coder")
	if !ok {
		t.Fatalf("expected memory record")
	}
	if record.Provenance != "subagent-profile" {
		t.Fatalf("unexpected provenance: %s", record.Provenance)
	}

	time.Sleep(60 * time.Millisecond)
	store.PruneExpired()
	if _, ok := store.Get(MemoryScopeRun, "/tmp/workspace", "run-1", "agent_context.coder"); ok {
		t.Fatalf("expected expired memory record to be pruned")
	}
}
