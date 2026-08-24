package cooling

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/wyw14/cry-112/internal/chamber"
	"github.com/wyw14/cry-112/internal/filter"
)

type SequenceState struct {
	CycleID     uuid.UUID              `json:"cycle_id"`
	ChamberID   string                 `json:"chamber_id"`
	FilterID    uuid.UUID              `json:"filter_id"`
	CoolingRate float64                `json:"cooling_rate"`
	Makeup      chamber.MakeupDecision `json:"makeup"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

type Sequence struct {
	mu         sync.RWMutex
	protection chamber.VacuumProtection
	integrity  *filter.IntegrityService
	states     map[uuid.UUID]SequenceState
}

func NewSequence(integrity *filter.IntegrityService) (*Sequence, error) {
	protection, err := chamber.NewVacuumProtection(75, 90)
	if err != nil {
		return nil, err
	}
	return &Sequence{protection: protection, integrity: integrity, states: make(map[uuid.UUID]SequenceState)}, nil
}

func (s *Sequence) Tick(cycleID uuid.UUID, chamberID string, filterID uuid.UUID, pressureKPa, requestedRate float64, now time.Time) (SequenceState, error) {
	if cycleID == uuid.Nil || chamberID == "" || filterID == uuid.Nil || pressureKPa < 0 || requestedRate < 0 {
		return SequenceState{}, fmt.Errorf("invalid cooling sequence input")
	}
	_, permitted := s.integrity.Permit(filterID, now)
	decision := s.protection.Evaluate(pressureKPa, permitted, now)
	rate := requestedRate
	if decision.SlowCooling {
		rate *= 0.25
	}
	state := SequenceState{CycleID: cycleID, ChamberID: chamberID, FilterID: filterID, CoolingRate: rate, Makeup: decision, UpdatedAt: now.UTC()}
	s.mu.Lock()
	s.states[cycleID] = state
	s.mu.Unlock()
	return state, nil
}

func (s *Sequence) State(cycleID uuid.UUID) (SequenceState, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, ok := s.states[cycleID]
	return state, ok
}

func (s *Sequence) Finish(cycleID uuid.UUID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.states, cycleID)
}
