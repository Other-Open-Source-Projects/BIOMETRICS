package tracker

import (
	"fmt"
	"sync"
	"time"
)

type ModelPool struct {
	mu          sync.RWMutex
	models      map[string]*Model
	maxPerModel int
	waitQueue   map[string][]chan *Model
}

type Model struct {
	Name        string
	Available   int
	Max         int
	InUse       int
	WaitTime    time.Duration
	LastUsed    time.Time
	TotalUsed   int
	TotalFailed int
}

func NewModelPool() *ModelPool {
	return &ModelPool{
		models:      make(map[string]*Model),
		maxPerModel: 3,
		waitQueue:   make(map[string][]chan *Model),
	}
}

func (mp *ModelPool) RegisterModel(name string, max int) {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	mp.models[name] = &Model{
		Name:        name,
		Available:   max,
		Max:         max,
		InUse:       0,
		WaitTime:    0,
		LastUsed:    time.Now(),
		TotalUsed:   0,
		TotalFailed: 0,
	}
}

func (mp *ModelPool) AcquireModel(name string) (*Model, error) {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	model, exists := mp.models[name]
	if !exists {
		return nil, fmt.Errorf("model %s not registered", name)
	}

	if model.Available > 0 {
		model.Available--
		model.InUse++
		model.LastUsed = time.Now()
		model.TotalUsed++
		return model, nil
	}

	return nil, fmt.Errorf("model %s at capacity", name)
}

func (mp *ModelPool) ReleaseModel(name string) {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	model, exists := mp.models[name]
	if !exists {
		return
	}

	if model.InUse > 0 {
		model.InUse--
	}
	if model.Available < model.Max {
		model.Available++
	}
}

func (mp *ModelPool) GetStats() map[string]interface{} {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	stats := make(map[string]interface{})
	for name, model := range mp.models {
		stats[name] = map[string]interface{}{
			"available":  model.Available,
			"in_use":     model.InUse,
			"max":        model.Max,
			"total_used": model.TotalUsed,
		}
	}
	return stats
}

type ModelTracker struct {
	mu     sync.Mutex
	models map[string]bool
}

func NewModelTracker() *ModelTracker {
	mt := &ModelTracker{models: make(map[string]bool)}
	mt.initDefaultModels()
	return mt
}

func (mt *ModelTracker) initDefaultModels() {
	defaultModels := []string{
		"google/antigravity-gemini-3.1-pro",
		"opencode/kimi-k2.5-free",
		"opencode/minimax-m2.5-free",
		"google/antigravity-gemini-3-flash",
	}

	for _, model := range defaultModels {
		mt.models[model] = false
	}
}

func (mt *ModelTracker) Acquire(model string) error {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	if mt.models[model] {
		return fmt.Errorf("model %s is already in use", model)
	}
	mt.models[model] = true
	return nil
}

func (mt *ModelTracker) Release(model string) {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	delete(mt.models, model)
}
