package steam

import (
	"sync"
	"time"

	"github.com/wyw14/cry-112/internal/model"
	"github.com/wyw14/cry-112/internal/vacuum"
)

type Service struct {
	mu          sync.RWMutex
	analyzer    *Analyzer
	conditioner Conditioner
	state       model.SteamState
}

func NewService(now time.Time) *Service {
	return &Service{analyzer: NewAnalyzer(5 * time.Minute), conditioner: NewConditioner(120), state: model.SteamState{TemperatureC: 22, UpdatedAt: now.UTC()}}
}

func (s *Service) Observe(temperature, ncg float64, now time.Time) (model.SteamState, error) {
	if err := s.analyzer.Observe(Sample{TemperatureC: temperature, NCGPercent: ncg, ObservedAt: now.UTC()}); err != nil {
		return model.SteamState{}, err
	}
	s.mu.Lock()
	s.state.TemperatureC = temperature
	s.state.NCGPercent = ncg
	s.state.UpdatedAt = now.UTC()
	state := s.state
	s.mu.Unlock()
	return state, nil
}

func (s *Service) Quality(maximumNCG, minimumTemperature float64, now time.Time) QualityProof {
	return s.analyzer.Evaluate(maximumNCG, minimumTemperature, now)
}

func (s *Service) Condition(proof vacuum.AirRemovalProof, jacketTemperature float64, now time.Time) (ConditioningDecision, error) {
	decision, err := s.conditioner.Evaluate(proof, jacketTemperature, now)
	if err != nil {
		return ConditioningDecision{}, err
	}
	s.mu.Lock()
	s.state.SupplyOpen = decision.SupplyOpen
	s.state.UpdatedAt = now.UTC()
	s.mu.Unlock()
	return decision, nil
}

func (s *Service) CloseSupply(now time.Time) model.SteamState {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.SupplyOpen = false
	s.state.UpdatedAt = now.UTC()
	return s.state
}

func (s *Service) State() model.SteamState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

func (s *Service) Restore(state model.SteamState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state = state
}

func (s *Service) ResetAnalyzer() {
	s.analyzer.Reset()
}
