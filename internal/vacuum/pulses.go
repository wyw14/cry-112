package vacuum

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/wyw14/cry-112/internal/chamber"
)

type Pulse struct {
	ID          uuid.UUID            `json:"id"`
	Sequence    int                  `json:"sequence"`
	MinimumKPa  float64              `json:"minimum_kpa"`
	StartedAt   time.Time            `json:"started_at"`
	CompletedAt time.Time            `json:"completed_at"`
	Rebound     chamber.ReboundProof `json:"rebound"`
}

func (p Pulse) Valid(maximumDepthKPa float64) bool {
	return p.Sequence > 0 && p.MinimumKPa <= maximumDepthKPa && p.Rebound.Valid && p.CompletedAt.After(p.StartedAt)
}

type AirRemovalProof struct {
	CycleID        uuid.UUID `json:"cycle_id"`
	RequiredPulses int       `json:"required_pulses"`
	Pulses         []Pulse   `json:"pulses"`
	DepthValid     bool      `json:"depth_valid"`
	ReboundValid   bool      `json:"rebound_valid"`
	SequenceValid  bool      `json:"sequence_valid"`
	ProvedAt       time.Time `json:"proved_at"`
}

func (p AirRemovalProof) Valid() bool {
	// Air is only proven excluded when all three conditions hold together:
	// the vacuum reached the required depth, the isolation rebound stayed
	// within the recipe limit, and the prescribed pulse sequence completed.
	// A failure of any single condition must keep the cycle out of exposure
	// even if the others pass.
	return p.DepthValid && p.ReboundValid && p.SequenceValid && len(p.Pulses) == p.RequiredPulses
}

type PulseService struct {
	mu        sync.RWMutex
	byCycle   map[uuid.UUID][]Pulse
	evaluator chamber.ReboundEvaluator
}

func NewPulseService(isolation time.Duration) *PulseService {
	return &PulseService{byCycle: make(map[uuid.UUID][]Pulse), evaluator: chamber.NewReboundEvaluator(isolation)}
}

func (s *PulseService) Record(cycleID uuid.UUID, sequence int, minimumKPa float64, startedAt, completedAt time.Time, start, end chamber.PressurePoint, maximumRise float64) (Pulse, error) {
	if cycleID == uuid.Nil || sequence < 1 || minimumKPa < 0 {
		return Pulse{}, fmt.Errorf("invalid pulse identity or measurement")
	}
	rebound, err := s.evaluator.Evaluate(start, end, maximumRise)
	if err != nil {
		return Pulse{}, err
	}
	pulse := Pulse{ID: uuid.New(), Sequence: sequence, MinimumKPa: minimumKPa, StartedAt: startedAt.UTC(), CompletedAt: completedAt.UTC(), Rebound: rebound}
	s.mu.Lock()
	defer s.mu.Unlock()
	existing := s.byCycle[cycleID]
	if sequence != len(existing)+1 {
		return Pulse{}, fmt.Errorf("pulse %d does not follow %d", sequence, len(existing))
	}
	s.byCycle[cycleID] = append(existing, pulse)
	return pulse, nil
}

func (s *PulseService) Proof(cycleID uuid.UUID, required int, maximumDepthKPa float64, now time.Time) AirRemovalProof {
	s.mu.RLock()
	pulses := append([]Pulse(nil), s.byCycle[cycleID]...)
	s.mu.RUnlock()
	proof := AirRemovalProof{CycleID: cycleID, RequiredPulses: required, Pulses: pulses, ProvedAt: now.UTC(), DepthValid: true, ReboundValid: true, SequenceValid: len(pulses) == required}
	for index, pulse := range pulses {
		proof.DepthValid = proof.DepthValid && pulse.MinimumKPa <= maximumDepthKPa
		proof.ReboundValid = proof.ReboundValid && pulse.Rebound.Valid
		proof.SequenceValid = proof.SequenceValid && pulse.Sequence == index+1
	}
	if len(pulses) != required {
		proof.DepthValid = false
		proof.ReboundValid = false
	}
	return proof
}

func (s *PulseService) Reset(cycleID uuid.UUID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.byCycle, cycleID)
}
