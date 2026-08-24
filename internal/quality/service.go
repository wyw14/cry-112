package quality

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/wyw14/cry-112/internal/drying"
	"github.com/wyw14/cry-112/internal/steam"
)

type Service struct {
	mu       sync.RWMutex
	steam    SteamWindow
	moisture LoadMoisture
	releases map[uuid.UUID]ReleaseEvidence
}

func NewService() *Service {
	return &Service{steam: NewSteamWindow(5 * time.Second), moisture: NewLoadMoisture(10 * time.Second), releases: make(map[uuid.UUID]ReleaseEvidence)}
}

func (s *Service) SteamValid(proof steam.QualityProof, now time.Time) (bool, string) {
	return s.steam.Valid(proof, now), s.steam.Reason(proof, now)
}

func (s *Service) LoadDry(proof drying.MoistureProof, now time.Time) bool {
	return s.moisture.Evaluate(proof, now)
}

func (s *Service) RecordRelease(evidence ReleaseEvidence) error {
	if evidence.CycleID == uuid.Nil {
		return fmt.Errorf("cycle identity is required")
	}
	s.mu.Lock()
	s.releases[evidence.CycleID] = evidence
	s.mu.Unlock()
	return nil
}

func (s *Service) Release(cycleID uuid.UUID) (ReleaseEvidence, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	evidence, ok := s.releases[cycleID]
	return evidence, ok
}
