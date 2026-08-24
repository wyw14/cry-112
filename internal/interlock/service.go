package interlock

import (
	"sync"
	"time"

	"github.com/wyw14/cry-112/internal/model"
)

type Service struct {
	mu          sync.RWMutex
	passThrough PassThrough
	last        map[string]PassThroughDecision
}

func NewService() *Service {
	return &Service{passThrough: NewPassThrough(), last: make(map[string]PassThroughDecision)}
}

func (s *Service) PermitRelease(requested, peer model.DoorState, now time.Time) (PassThroughDecision, error) {
	decision, err := s.passThrough.EvaluatePeer(requested, peer, now)
	if err != nil {
		return PassThroughDecision{}, err
	}
	s.mu.Lock()
	s.last[requested.ID] = decision
	s.mu.Unlock()
	return decision, nil
}
