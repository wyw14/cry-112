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
	if _, ok := s.valves[chamberID]; !ok {
		return Reservation{}, fmt.Errorf("drain valve for chamber %s not found", chamberID)
	}
	// Serialize reserve-and-apply so concurrent chambers cannot interleave a
	// stale allocation onto a peer's valve. The shared budget is rebalanced and
	// every active valve is driven to its post-rebalance share in one step, so
	// the sum of openings can never exceed capacity and no single request can
	// push backpressure past the limit.
	s.mu.Lock()
	defer s.mu.Unlock()
	reservation, snapshot, err := s.allocator.Reserve(chamberID, flow, priority, now)
	if err != nil {
		return Reservation{}, err
	}
	if err := s.applyAllocations(snapshot, now); err != nil {
		// Roll back the reservation and re-sync any peer valves that were
		// throttled before the failure so they track the freed budget. Close the
		// requesting valve too: its reservation is gone, so it must not be left
		// open and uncounted.
		rollback := s.allocator.Release(chamberID)
		if valve, ok := s.valves[chamberID]; ok {
			valve.Close(now)
		}
		_ = s.applyAllocations(rollback, now)
		return Reservation{}, err
	}
	if reservation.Allocated != s.valves[chamberID].State().AllocatedFlow {
		return Reservation{}, fmt.Errorf("drain allocation mismatch for chamber %s", chamberID)
	}
	return reservation, nil
}

// applyAllocations drives every active drain valve to its allocated share of
// the shared capacity budget. It must run after any allocator mutation so the
// physical valve openings track the budget; callers hold s.mu.
func (s *Service) applyAllocations(snapshot AllocationSnapshot, now time.Time) error {
	for _, reservation := range snapshot.Reservations {
		valve, ok := s.valves[reservation.ChamberID]
		if !ok {
			continue
		}
		if _, err := valve.Apply(reservation.Requested, reservation.Allocated, now); err != nil {
			return err
		}
	}
	return nil
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
	s.mu.Lock()
	defer s.mu.Unlock()
	if valve, ok := s.valves[chamberID]; ok {
		valve.Close(now)
	}
	snapshot := s.allocator.Release(chamberID)
	// Re-sync surviving valves to the freed budget so openings track capacity
	// after a chamber finishes draining.
	_ = s.applyAllocations(snapshot, now)
	return snapshot
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
