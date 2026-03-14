package bus

import (
	"sync"
	"sync/atomic"

	"biometrics-cli/internal/contracts"
	store "biometrics-cli/internal/store/sqlite"
)

type EventBus struct {
	mu     sync.RWMutex
	subs   map[int]chan contracts.Event
	nextID int
	store  *store.Store
	redact func(string) string

	droppedEvents atomic.Int64
}

type Option func(*EventBus)

func WithRedactor(redactor func(string) string) Option {
	return func(bus *EventBus) {
		bus.redact = redactor
	}
}

func NewEventBus(s *store.Store, opts ...Option) *EventBus {
	b := &EventBus{
		subs:  make(map[int]chan contracts.Event),
		store: s,
		redact: func(value string) string {
			return value
		},
	}
	for _, opt := range opts {
		if opt != nil {
			opt(b)
		}
	}
	return b
}

func (b *EventBus) Subscribe(buffer int) (int, <-chan contracts.Event) {
	if buffer <= 0 {
		buffer = 64
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	b.nextID++
	id := b.nextID
	ch := make(chan contracts.Event, buffer)
	b.subs[id] = ch
	return id, ch
}

func (b *EventBus) Unsubscribe(id int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if ch, ok := b.subs[id]; ok {
		close(ch)
		delete(b.subs, id)
	}
}

func (b *EventBus) Publish(ev contracts.Event) (contracts.Event, error) {
	ev = b.sanitizeEvent(ev)
	persisted, err := b.store.AppendEvent(ev)
	if err != nil {
		return contracts.Event{}, err
	}

	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, ch := range b.subs {
		select {
		case ch <- persisted:
		default:
			// backpressure safe drop
			b.droppedEvents.Add(1)
		}
	}

	return persisted, nil
}

func (b *EventBus) sanitizeEvent(ev contracts.Event) contracts.Event {
	if b.redact == nil {
		return ev
	}
	ev.Type = b.redact(ev.Type)
	ev.Source = b.redact(ev.Source)
	if len(ev.Payload) == 0 {
		return ev
	}
	cloned := make(map[string]string, len(ev.Payload))
	for key, value := range ev.Payload {
		cloned[key] = b.redact(value)
	}
	ev.Payload = cloned
	return ev
}

func (b *EventBus) Replay(runID string, limit int) ([]contracts.Event, error) {
	return b.store.ListEvents(runID, limit)
}

func (b *EventBus) MetricsSnapshot() map[string]int64 {
	b.mu.RLock()
	subscribers := int64(len(b.subs))
	b.mu.RUnlock()

	return map[string]int64{
		"eventbus_dropped_events": b.droppedEvents.Load(),
		"eventbus_subscribers":    subscribers,
	}
}
