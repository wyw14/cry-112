package chamber

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/wyw14/cry-112/internal/model"
)

type Service struct {
	mu     sync.RWMutex
	states map[string]model.ChamberState
}

func NewService(ids []string, now time.Time) *Service {
	states := make(map[string]model.ChamberState, len(ids))
	for _, id := range ids {
		states[id] = model.ChamberState{ID: id, PressureKPa: 101.3, TemperatureC: 22, JacketTemperatureC: 22, UpdatedAt: now.UTC()}
	}
	return &Service{states: states}
}

func (s *Service) Update(id string, pressure, temperature, jacket, drainBackpressure float64, now time.Time) (model.ChamberState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.states[id]
	if !ok {
		return model.ChamberState{}, fmt.Errorf("chamber %s not found", id)
	}
	if pressure < 0 || temperature < -20 || jacket < -20 || drainBackpressure < 0 {
		return model.ChamberState{}, fmt.Errorf("chamber telemetry is outside physical range")
	}
	state.PressureKPa = pressure
	state.TemperatureC = temperature
	state.JacketTemperatureC = jacket
	state.DrainBackpressure = drainBackpressure
	state.UpdatedAt = now.UTC()
	s.states[id] = state
	return state, nil
}

func (s *Service) State(id string) (model.ChamberState, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, ok := s.states[id]
	return state, ok
}

func (s *Service) List() []model.ChamberState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	states := make([]model.ChamberState, 0, len(s.states))
	for _, state := range s.states {
		states = append(states, state)
	}
	sort.Slice(states, func(i, j int) bool { return states[i].ID < states[j].ID })
	return states
}

func (s *Service) Restore(states map[string]model.ChamberState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, state := range states {
		if _, ok := s.states[id]; ok {
			s.states[id] = state
		}
	}
}

func (s *Service) Snapshot() map[string]model.ChamberState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	copyStates := make(map[string]model.ChamberState, len(s.states))
	for id, state := range s.states {
		copyStates[id] = state
	}
	return copyStates
}
