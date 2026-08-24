package cycle

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/wyw14/cry-112/internal/model"
)

type Registry struct {
	mu     sync.RWMutex
	cycles map[uuid.UUID]model.Cycle
}

func NewRegistry() *Registry {
	return &Registry{cycles: make(map[uuid.UUID]model.Cycle)}
}

func (r *Registry) Create(cycle model.Cycle) error {
	if cycle.ID == uuid.Nil {
		return fmt.Errorf("cycle identity is required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.cycles[cycle.ID]; exists {
		return fmt.Errorf("cycle %s already exists", cycle.ID)
	}
	r.cycles[cycle.ID] = cycle.Clone()
	return nil
}

func (r *Registry) Get(id uuid.UUID) (model.Cycle, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	cycle, ok := r.cycles[id]
	return cycle.Clone(), ok
}

func (r *Registry) Save(cycle model.Cycle) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.cycles[cycle.ID]; !exists {
		return fmt.Errorf("cycle %s not found", cycle.ID)
	}
	r.cycles[cycle.ID] = cycle.Clone()
	return nil
}

func (r *Registry) Transition(id uuid.UUID, next model.Phase, now time.Time) (model.Cycle, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	cycle, ok := r.cycles[id]
	if !ok {
		return model.Cycle{}, fmt.Errorf("cycle %s not found", id)
	}
	if err := cycle.Transition(next, now); err != nil {
		return model.Cycle{}, err
	}
	r.cycles[id] = cycle
	return cycle.Clone(), nil
}

func (r *Registry) List() []model.Cycle {
	r.mu.RLock()
	defer r.mu.RUnlock()
	cycles := make([]model.Cycle, 0, len(r.cycles))
	for _, cycle := range r.cycles {
		cycles = append(cycles, cycle.Clone())
	}
	sort.Slice(cycles, func(i, j int) bool { return cycles[i].StartedAt.Before(cycles[j].StartedAt) })
	return cycles
}

func (r *Registry) Snapshot() map[uuid.UUID]model.Cycle {
	r.mu.RLock()
	defer r.mu.RUnlock()
	cycles := make(map[uuid.UUID]model.Cycle, len(r.cycles))
	for id, cycle := range r.cycles {
		cycles[id] = cycle.Clone()
	}
	return cycles
}

func (r *Registry) Restore(cycles map[uuid.UUID]model.Cycle) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cycles = make(map[uuid.UUID]model.Cycle, len(cycles))
	for id, cycle := range cycles {
		r.cycles[id] = cycle.Clone()
	}
}
