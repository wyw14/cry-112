package cooling

import (
	"time"

	"github.com/google/uuid"
	"github.com/wyw14/cry-112/internal/filter"
)

type Service struct {
	sequence  *Sequence
	condenser *Condenser
	integrity *filter.IntegrityService
}

func NewService(now time.Time) (*Service, error) {
	integrity := filter.NewIntegrityService()
	sequence, err := NewSequence(integrity)
	if err != nil {
		return nil, err
	}
	return &Service{sequence: sequence, condenser: NewCondenser(32, 8, now), integrity: integrity}, nil
}

func (s *Service) RecordFilter(filterID uuid.UUID, pressureDrop, leakRate float64, passed bool, now time.Time) (filter.IntegrityProof, error) {
	return s.integrity.Record(filterID, uuid.New(), pressureDrop, leakRate, passed, now)
}

func (s *Service) Tick(cycleID uuid.UUID, chamberID string, filterID uuid.UUID, pressureKPa, rate float64, now time.Time) (SequenceState, error) {
	return s.sequence.Tick(cycleID, chamberID, filterID, pressureKPa, rate, now)
}

func (s *Service) ObserveCondenser(temperature, waterFlow float64, now time.Time) (CondenserState, error) {
	return s.condenser.Observe(temperature, waterFlow, now)
}

func (s *Service) Condenser() CondenserState {
	return s.condenser.State()
}

func (s *Service) FilterProofs() []filter.IntegrityProof {
	return s.integrity.List()
}

func (s *Service) InvalidateFilter(filterID uuid.UUID) {
	s.integrity.Invalidate(filterID)
}

func (s *Service) Finish(cycleID uuid.UUID) {
	s.sequence.Finish(cycleID)
}

func (s *Service) SequenceState(cycleID uuid.UUID) (SequenceState, bool) {
	return s.sequence.State(cycleID)
}
