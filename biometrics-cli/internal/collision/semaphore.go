package collision

import (
	"context"
	"fmt"
	"golang.org/x/sync/semaphore"
)

type ModelPool struct {
	gemini  *semaphore.Weighted
	kimi    *semaphore.Weighted
	minimax *semaphore.Weighted
}

// NewModelPool initialisiert die harten Limits aus Mandat 0.37
func NewModelPool() *ModelPool {
	return &ModelPool{
		gemini:  semaphore.NewWeighted(1),  // MAX 1
		kimi:    semaphore.NewWeighted(1),  // MAX 1
		minimax: semaphore.NewWeighted(10), // MAX 10
	}
}

// Acquire blockiert, bis das Modell frei ist.
func (p *ModelPool) Acquire(ctx context.Context, modelName string) error {
	switch modelName {
	case "google/antigravity-gemini-3.1-pro":
		return p.gemini.Acquire(ctx, 1)
	case "kimi-k2.5":
		return p.kimi.Acquire(ctx, 1)
	case "minimax-m2.5":
		return p.minimax.Acquire(ctx, 1)
	default:
		return fmt.Errorf("unknown model: %s", modelName)
	}
}

// Release gibt den Lock wieder frei. MUSS via defer aufgerufen werden!
func (p *ModelPool) Release(modelName string) {
	switch modelName {
	case "google/antigravity-gemini-3.1-pro":
		p.gemini.Release(1)
	case "kimi-k2.5":
		p.kimi.Release(1)
	case "minimax-m2.5":
		p.minimax.Release(1)
	}
}
