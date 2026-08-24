package vacuum

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/wyw14/cry-112/internal/chamber"
	"github.com/wyw14/cry-112/internal/model"
)

type Service struct {
	mu      sync.RWMutex
	pulses  *PulseService
	retries *RetryCoordinator
	proofs  map[uuid.UUID]AirRemovalProof
}

func NewService() *Service {
	return &Service{pulses: NewPulseService(10 * time.Second), retries: NewRetryCoordinator(3, 32, 8), proofs: make(map[uuid.UUID]AirRemovalProof)}
}

func (s *Service) RecordPulse(cycleID uuid.UUID, sequence int, minimumKPa float64, startedAt, completedAt time.Time, isolatedStart, isolatedEnd float64, recipe model.Recipe) (AirRemovalProof, error) {
	start := chamber.PressurePoint{PressureKPa: isolatedStart, ObservedAt: completedAt.Add(-10 * time.Second)}
	end := chamber.PressurePoint{PressureKPa: isolatedEnd, ObservedAt: completedAt}
	if _, err := s.pulses.Record(cycleID, sequence, minimumKPa, startedAt, completedAt, start, end, recipe.MaximumReboundKPa); err != nil {
		return AirRemovalProof{}, err
	}
	proof := s.pulses.Proof(cycleID, recipe.RequiredVacuumPulses, recipe.VacuumDepthKPa, completedAt)
	s.mu.Lock()
	s.proofs[cycleID] = proof
	s.mu.Unlock()
	return proof, nil
}

func (s *Service) AirRemovalProof(cycleID uuid.UUID) (AirRemovalProof, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	proof, ok := s.proofs[cycleID]
	proof.Pulses = append([]Pulse(nil), proof.Pulses...)
	return proof, ok
}

func (s *Service) CavitationRetry(cycleID uuid.UUID, temperature, waterFlow float64, observedAt, now time.Time) (RetryRecord, error) {
	return s.retries.Request(cycleID, "cavitation", CondenserProof{TemperatureC: temperature, WaterFlow: waterFlow, ObservedAt: observedAt}, now)
}

func (s *Service) RetryHistory(cycleID uuid.UUID) []RetryRecord {
	return s.retries.Records(cycleID)
}

func (s *Service) Reset(cycleID uuid.UUID) error {
	if cycleID == uuid.Nil {
		return fmt.Errorf("cycle identity is required")
	}
	s.pulses.Reset(cycleID)
	s.retries.Reset(cycleID)
	s.mu.Lock()
	delete(s.proofs, cycleID)
	s.mu.Unlock()
	return nil
}
