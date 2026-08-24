package drying

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

type State struct {
	CycleID        uuid.UUID     `json:"cycle_id"`
	LoadID         uuid.UUID     `json:"load_id"`
	JacketHumidity float64       `json:"jacket_humidity"`
	JacketReady    bool          `json:"jacket_ready"`
	LoadProof      MoistureProof `json:"load_proof"`
	Complete       bool          `json:"complete"`
	UpdatedAt      time.Time     `json:"updated_at"`
}

type Service struct {
	mu      sync.RWMutex
	tracker *MoistureTracker
	states  map[uuid.UUID]State
}

func NewService() *Service {
	return &Service{tracker: NewMoistureTracker(), states: make(map[uuid.UUID]State)}
}

func (s *Service) ObserveLoad(reading MoistureReading) error {
	return s.tracker.Observe(reading)
}

func (s *Service) ApplyHumidity(cycleID, loadID uuid.UUID, jacketHumidity float64, required []uuid.UUID, maximumJacket, maximumLoad float64, now time.Time) (State, error) {
	if cycleID == uuid.Nil || loadID == uuid.Nil || jacketHumidity < 0 {
		return State{}, fmt.Errorf("invalid drying state input")
	}
	proof := s.tracker.Evaluate(loadID, required, maximumLoad, now)
	state := State{
		CycleID:        cycleID,
		LoadID:         loadID,
		JacketHumidity: jacketHumidity,
		JacketReady:    jacketHumidity <= maximumJacket,
		LoadProof:      proof,
		UpdatedAt:      now.UTC(),
	}
	state.Complete = state.JacketReady && state.LoadProof.Valid()
	s.mu.Lock()
	s.states[cycleID] = state
	s.mu.Unlock()
	return state, nil
}

func (s *Service) State(cycleID uuid.UUID) (State, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, ok := s.states[cycleID]
	state.LoadProof.Readings = append([]MoistureReading(nil), state.LoadProof.Readings...)
	return state, ok
}

func (s *Service) Finish(cycleID, loadID uuid.UUID) {
	s.mu.Lock()
	delete(s.states, cycleID)
	s.mu.Unlock()
	s.tracker.Reset(loadID)
}
