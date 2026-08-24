package condensate

import (
	"fmt"
	"sync"
	"time"

	"github.com/wyw14/cry-112/internal/chamber"
)

type Service struct {
	mu        sync.RWMutex
	allocator *Allocator
	evaluator DrainEvaluator
	valves    map[string]*chamber.DrainValve
	proofs    map[string]DrainProof
}

func NewService(chamberIDs []string, capacity, valveMaximum float64, now time.Time) (*Service, error) {
	allocator, err := NewAllocator(capacity)
	if err != nil {
		return nil, err
	}
	service := &Service{allocator: allocator, evaluator: NewDrainEvaluator(), valves: make(map[string]*chamber.DrainValve), proofs: make(map[string]DrainProof)}
	for _, id := range chamberIDs {
		service.valves[id] = chamber.NewDrainValve(id, valveMaximum, now)
	}
	return service, nil
}

func (s *Service) Request(chamberID string, flow float64, priority int, now time.Time) (Reservation, error) {
	valve, ok := s.valves[chamberID]
	if !ok {
		return Reservation{}, fmt.Errorf("drain valve for chamber %s not found", chamberID)
	}
	reservation, _, err := s.allocator.Reserve(chamberID, flow, priority, now)
	if err != nil {
		return Reservation{}, err
	}
	if _, err := valve.Apply(flow, flow, now); err != nil {
		s.allocator.Release(chamberID)
		return Reservation{}, err
	}
	return reservation, nil
}

func (s *Service) Observe(chamberID string, reading DrainReading, maximumBackpressure float64, now time.Time) (DrainProof, error) {
	valve, ok := s.valves[chamberID]
	if !ok {
		return DrainProof{}, fmt.Errorf("drain valve for chamber %s not found", chamberID)
	}
	proof, err := s.evaluator.Evaluate(reading, valve.State().AllocatedFlow, maximumBackpressure, now)
	if err != nil {
		return DrainProof{}, err
	}
	s.mu.Lock()
	s.proofs[chamberID] = proof
	s.mu.Unlock()
	return proof, nil
}

func (s *Service) Finish(chamberID string, now time.Time) AllocationSnapshot {
	if valve, ok := s.valves[chamberID]; ok {
		valve.Close(now)
	}
	return s.allocator.Release(chamberID)
}

func (s *Service) Snapshot() AllocationSnapshot {
	return s.allocator.Snapshot()
}

func (s *Service) Proof(chamberID string) (DrainProof, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	proof, ok := s.proofs[chamberID]
	return proof, ok
}

func (s *Service) ValveStates() []chamber.DrainValveState {
	states := make([]chamber.DrainValveState, 0, len(s.valves))
	for _, valve := range s.valves {
		states = append(states, valve.State())
	}
	return states
}
