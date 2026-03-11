package supervisor

import (
	"context"
	"fmt"
	"time"

	"biometrics-cli/internal/contracts"
	"biometrics-cli/internal/runtime/bus"
)

type Supervisor struct {
	bus         *bus.EventBus
	baseBackoff time.Duration
	maxBackoff  time.Duration
}

func New(eventBus *bus.EventBus) *Supervisor {
	return &Supervisor{
		bus:         eventBus,
		baseBackoff: 250 * time.Millisecond,
		maxBackoff:  5 * time.Second,
	}
}

func (s *Supervisor) StartActor(ctx context.Context, actorName string, run func(context.Context)) {
	go func() {
		backoff := s.baseBackoff
		for {
			if ctx.Err() != nil {
				return
			}

			panicked := false
			func() {
				defer func() {
					if rec := recover(); rec != nil {
						panicked = true
						_, _ = s.bus.Publish(contracts.Event{
							Type:   "agent.restarted",
							Source: "supervisor",
							Payload: map[string]string{
								"agent":  actorName,
								"reason": fmt.Sprint(rec),
							},
						})
					}
				}()

				run(ctx)
			}()

			if !panicked {
				return
			}

			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
			}

			backoff *= 2
			if backoff > s.maxBackoff {
				backoff = s.maxBackoff
			}
		}
	}()
}
