package actor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"biometrics-cli/internal/contracts"
	"biometrics-cli/internal/runtime/supervisor"
)

type Handler func(context.Context, contracts.AgentEnvelope) contracts.AgentResult

type ref struct {
	name    string
	mailbox chan contracts.AgentEnvelope
	handler Handler
}

type System struct {
	mu         sync.RWMutex
	actors     map[string]*ref
	supervisor *supervisor.Supervisor
	started    bool
}

func NewSystem(sup *supervisor.Supervisor) *System {
	return &System{
		actors:     make(map[string]*ref),
		supervisor: sup,
	}
}

func (s *System) Register(name string, mailboxSize int, handler Handler) error {
	if mailboxSize <= 0 {
		mailboxSize = 32
	}
	if handler == nil {
		return fmt.Errorf("handler required for actor %s", name)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.actors[name]; exists {
		return fmt.Errorf("actor %s already registered", name)
	}

	s.actors[name] = &ref{
		name:    name,
		mailbox: make(chan contracts.AgentEnvelope, mailboxSize),
		handler: handler,
	}
	return nil
}

func (s *System) Start(ctx context.Context) {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return
	}
	actors := make([]*ref, 0, len(s.actors))
	for _, actor := range s.actors {
		actors = append(actors, actor)
	}
	s.started = true
	s.mu.Unlock()

	for _, actor := range actors {
		a := actor
		s.supervisor.StartActor(ctx, a.name, func(loopCtx context.Context) {
			for {
				select {
				case <-loopCtx.Done():
					return
				case env := <-a.mailbox:
					result := a.handler(loopCtx, env)
					if env.ResponseCh != nil {
						select {
						case env.ResponseCh <- result:
						default:
						}
					}
				}
			}
		})
	}
}

func (s *System) Send(ctx context.Context, actorName string, env contracts.AgentEnvelope, timeout time.Duration) (contracts.AgentResult, error) {
	s.mu.RLock()
	a, ok := s.actors[actorName]
	s.mu.RUnlock()
	if !ok {
		return contracts.AgentResult{}, fmt.Errorf("unknown actor %s", actorName)
	}
	if timeout <= 0 {
		timeout = 90 * time.Second
	}

	responseCh := make(chan contracts.AgentResult, 1)
	env.ResponseCh = responseCh
	env.DispatchedAt = time.Now().UTC()

	select {
	case <-ctx.Done():
		return contracts.AgentResult{}, ctx.Err()
	case a.mailbox <- env:
	}

	select {
	case <-ctx.Done():
		return contracts.AgentResult{}, ctx.Err()
	case <-time.After(timeout):
		return contracts.AgentResult{}, fmt.Errorf("actor %s timeout", actorName)
	case res := <-responseCh:
		return res, nil
	}
}

func (s *System) Actors() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]string, 0, len(s.actors))
	for name := range s.actors {
		out = append(out, name)
	}
	return out
}
