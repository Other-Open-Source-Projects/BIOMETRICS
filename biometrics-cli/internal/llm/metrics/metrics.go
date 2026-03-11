package metrics

import "sync/atomic"

type Counters struct {
	selected          atomic.Int64
	fallbackTriggered atomic.Int64
	fallbackExhausted atomic.Int64
}

func (c *Counters) IncSelected() {
	c.selected.Add(1)
}

func (c *Counters) IncFallbackTriggered() {
	c.fallbackTriggered.Add(1)
}

func (c *Counters) IncFallbackExhausted() {
	c.fallbackExhausted.Add(1)
}

func (c *Counters) Snapshot() map[string]int64 {
	return map[string]int64{
		"llm_model_selected":           c.selected.Load(),
		"llm_model_fallback_triggered": c.fallbackTriggered.Load(),
		"llm_model_fallback_exhausted": c.fallbackExhausted.Load(),
	}
}
