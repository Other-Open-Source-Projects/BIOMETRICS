package supervisor

import (
	"context"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"biometrics-cli/internal/runtime/bus"
	store "biometrics-cli/internal/store/sqlite"
)

func TestSupervisorRestartsPanickingActor(t *testing.T) {
	tmp := t.TempDir()
	db, err := store.New(filepath.Join(tmp, "events.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer db.Close()

	eventBus := bus.NewEventBus(db)
	s := New(eventBus)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	subID, ch := eventBus.Subscribe(10)
	defer eventBus.Unsubscribe(subID)

	var runs atomic.Int32
	s.StartActor(ctx, "tester", func(context.Context) {
		current := runs.Add(1)
		if current == 1 {
			panic("boom")
		}
		cancel()
	})

	deadline := time.After(2 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatalf("timeout waiting for restart event")
		case ev := <-ch:
			if ev.Type == "agent.restarted" {
				restartDeadline := time.Now().Add(500 * time.Millisecond)
				for runs.Load() < 2 && time.Now().Before(restartDeadline) {
					time.Sleep(10 * time.Millisecond)
				}
				if runs.Load() < 2 {
					t.Fatalf("expected actor rerun after panic, got runs=%d", runs.Load())
				}
				return
			}
		}
	}
}
