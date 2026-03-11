package orchestrator

import (
	"os"
	"path/filepath"
	"testing"

	store "biometrics-cli/internal/store/sqlite"
)

type fakeBackendWithStore struct {
	*fakeBackend
	store *store.Store
}

func (b *fakeBackendWithStore) Store() *store.Store {
	return b.store
}

func setupSessionService(t *testing.T) (*Service, *store.Store) {
	t.Helper()
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "orchestrator-session-test.db")
	db, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
		_ = os.Remove(dbPath)
	})

	backend := &fakeBackendWithStore{fakeBackend: newFakeBackend(), store: db}
	return NewService(backend, fakeBus{}), db
}

func TestSessionCreateListAndGet(t *testing.T) {
	svc, _ := setupSessionService(t)

	created, err := svc.CreateSession(OrchestratorSessionCreateRequest{ProjectID: "biometrics"})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if created.ID == "" {
		t.Fatalf("expected session id")
	}
	if created.Status != OrchestratorSessionStatusActive {
		t.Fatalf("expected active status, got %q", created.Status)
	}
	if len(created.Agents) != 3 {
		t.Fatalf("expected 3 default agents, got %d", len(created.Agents))
	}

	listed, err := svc.ListSessions("biometrics", 20)
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(listed) == 0 {
		t.Fatalf("expected created session in list")
	}

	fetched, err := svc.GetSession(created.ID)
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if fetched.ID != created.ID {
		t.Fatalf("expected fetched session %q, got %q", created.ID, fetched.ID)
	}
}

func TestSessionMessageAndModelOverride(t *testing.T) {
	svc, _ := setupSessionService(t)
	session, err := svc.CreateSession(OrchestratorSessionCreateRequest{ProjectID: "biometrics"})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	message, err := svc.AppendSessionMessage(session.ID, OrchestratorMessageAppendRequest{
		AuthorKind: OrchestratorAuthorKindUser,
		TargetPane: OrchestratorPaneBackend,
		Content:    "Please harden API validation.",
	})
	if err != nil {
		t.Fatalf("append message: %v", err)
	}
	if message.Cursor <= 0 {
		t.Fatalf("expected cursor > 0, got %d", message.Cursor)
	}

	messages, err := svc.ListSessionMessages(session.ID, 0, 100)
	if err != nil {
		t.Fatalf("list messages: %v", err)
	}
	if len(messages) == 0 {
		t.Fatalf("expected at least one message")
	}

	state, err := svc.SetSessionAgentModel(session.ID, "backend", OrchestratorAgentModelOverrideRequest{
		Provider: "nim",
		ModelID:  "qwen-3.5-397b",
	})
	if err != nil {
		t.Fatalf("set model override: %v", err)
	}
	if state.AgentID != "backend" {
		t.Fatalf("expected backend state, got %q", state.AgentID)
	}
	if state.Model.Provider != "nim" {
		t.Fatalf("expected nim provider, got %q", state.Model.Provider)
	}
}

func TestSessionPauseResumeKill(t *testing.T) {
	svc, _ := setupSessionService(t)
	session, err := svc.CreateSession(OrchestratorSessionCreateRequest{ProjectID: "biometrics"})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	paused, err := svc.PauseSession(session.ID, "manual guardrail")
	if err != nil {
		t.Fatalf("pause session: %v", err)
	}
	if paused.Status != OrchestratorSessionStatusPaused {
		t.Fatalf("expected paused status, got %q", paused.Status)
	}
	if !paused.Guardrails.Paused {
		t.Fatalf("expected guardrail pause state")
	}

	resumed, err := svc.ResumeSession(session.ID)
	if err != nil {
		t.Fatalf("resume session: %v", err)
	}
	if resumed.Status != OrchestratorSessionStatusActive {
		t.Fatalf("expected active status, got %q", resumed.Status)
	}

	killed, err := svc.KillSession(session.ID)
	if err != nil {
		t.Fatalf("kill session: %v", err)
	}
	if killed.Status != OrchestratorSessionStatusKilled {
		t.Fatalf("expected killed status, got %q", killed.Status)
	}
}
