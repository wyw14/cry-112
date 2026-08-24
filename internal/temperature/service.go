package temperature

import (
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/wyw14/cry-112/internal/model"
)

type Service struct {
	mu       sync.RWMutex
	history  *HistoryRegistry
	coldspot *ColdspotEvaluator
	latest   map[uuid.UUID]model.ProbeReading
}

func NewService() *Service {
	history := NewHistoryRegistry(30 * time.Minute)
	return &Service{history: history, coldspot: NewColdspotEvaluator(history), latest: make(map[uuid.UUID]model.ProbeReading)}
}

func (s *Service) Observe(reading model.ProbeReading, placement model.ProbePlacement) error {
	if err := s.coldspot.Observe(reading, placement); err != nil {
		return err
	}
	s.mu.Lock()
	s.latest[reading.ProbeID] = reading
	s.mu.Unlock()
	return nil
}

func (s *Service) Coldspot(placements []model.ProbePlacement, threshold float64, hold time.Duration, now time.Time) ColdspotProof {
	return s.coldspot.Evaluate(placements, threshold, hold, now)
}

func (s *Service) ResetLoad(loadID uuid.UUID) {
	s.coldspot.ResetLoad(loadID)
	s.mu.Lock()
	for id, reading := range s.latest {
		if reading.LoadID == loadID {
			delete(s.latest, id)
		}
	}
	s.mu.Unlock()
}
