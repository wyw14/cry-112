package door

import (
	"fmt"
	"sync"
	"time"

	"github.com/wyw14/cry-112/internal/model"
)

type StateService struct {
	mu     sync.RWMutex
	states map[string]model.DoorState
}

func NewStateService(ids []string, now time.Time) *StateService {
	states := make(map[string]model.DoorState, len(ids))
	for _, id := range ids {
		states[id] = model.DoorState{ID: id, DesiredClosed: true, PhysicalClosed: true, Locked: true, SealPressureBar: 3, UpdatedAt: now.UTC()}
	}
	return &StateService{states: states}
}

func (s *StateService) SetDesired(id string, closed bool, now time.Time) (model.DoorState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.states[id]
	if !ok {
		return model.DoorState{}, fmt.Errorf("door %s not found", id)
	}
	state.DesiredClosed = closed
	state.UpdatedAt = now.UTC()
	s.states[id] = state
	return state, nil
}

func (s *StateService) ApplyPhysical(id string, closed, locked bool, sealPressure float64, now time.Time) (model.DoorState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.states[id]
	if !ok {
		return model.DoorState{}, fmt.Errorf("door %s not found", id)
	}
	if sealPressure < 0 {
		return model.DoorState{}, fmt.Errorf("seal pressure cannot be negative")
	}
	state.PhysicalClosed = closed
	state.Locked = locked
	state.SealPressureBar = sealPressure
	state.UpdatedAt = now.UTC()
	s.states[id] = state
	return state, nil
}

func (s *StateService) State(id string) (model.DoorState, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, ok := s.states[id]
	return state, ok
}

func (s *StateService) List() []model.DoorState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	states := make([]model.DoorState, 0, len(s.states))
	for _, state := range s.states {
		states = append(states, state)
	}
	return states
}

func (s *StateService) Snapshot() map[string]model.DoorState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	states := make(map[string]model.DoorState, len(s.states))
	for id, state := range s.states {
		states[id] = state
	}
	return states
}

func (s *StateService) Restore(states map[string]model.DoorState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, state := range states {
		if _, ok := s.states[id]; ok {
			s.states[id] = state
		}
	}
}
