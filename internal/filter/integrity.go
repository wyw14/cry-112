package filter

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

type IntegrityProof struct {
	FilterID     uuid.UUID `json:"filter_id"`
	SessionID    uuid.UUID `json:"session_id"`
	PressureDrop float64   `json:"pressure_drop"`
	LeakRate     float64   `json:"leak_rate"`
	Passed       bool      `json:"passed"`
	ObservedAt   time.Time `json:"observed_at"`
	ExpiresAt    time.Time `json:"expires_at"`
}

func (p IntegrityProof) Valid(now time.Time) bool {
	return p.FilterID != uuid.Nil && p.SessionID != uuid.Nil && p.Passed && !now.After(p.ExpiresAt)
}

type IntegrityService struct {
	mu     sync.RWMutex
	proofs map[uuid.UUID]IntegrityProof
}

func NewIntegrityService() *IntegrityService {
	return &IntegrityService{proofs: make(map[uuid.UUID]IntegrityProof)}
}

func (s *IntegrityService) Record(filterID, sessionID uuid.UUID, pressureDrop, leakRate float64, passed bool, observedAt time.Time) (IntegrityProof, error) {
	if filterID == uuid.Nil || sessionID == uuid.Nil || pressureDrop < 0 || leakRate < 0 {
		return IntegrityProof{}, fmt.Errorf("invalid filter integrity measurement")
	}
	proof := IntegrityProof{FilterID: filterID, SessionID: sessionID, PressureDrop: pressureDrop, LeakRate: leakRate, Passed: passed, ObservedAt: observedAt.UTC(), ExpiresAt: observedAt.Add(24 * time.Hour).UTC()}
	s.mu.Lock()
	s.proofs[filterID] = proof
	s.mu.Unlock()
	return proof, nil
}

func (s *IntegrityService) Permit(filterID uuid.UUID, now time.Time) (IntegrityProof, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	proof, ok := s.proofs[filterID]
	return proof, ok && proof.Valid(now)
}

func (s *IntegrityService) Invalidate(filterID uuid.UUID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.proofs, filterID)
}

func (s *IntegrityService) List() []IntegrityProof {
	s.mu.RLock()
	defer s.mu.RUnlock()
	proofs := make([]IntegrityProof, 0, len(s.proofs))
	for _, proof := range s.proofs {
		proofs = append(proofs, proof)
	}
	return proofs
}
