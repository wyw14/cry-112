package door

import (
	"time"

	"github.com/wyw14/cry-112/internal/model"
)

type Service struct {
	states  *StateService
	release *ReleaseService
}

func NewService(ids []string, maximumTemperature float64, now time.Time) *Service {
	states := NewStateService(ids, now)
	return &Service{states: states, release: NewReleaseService(states, maximumTemperature)}
}

func (s *Service) SetDesired(id string, closed bool, now time.Time) (model.DoorState, error) {
	return s.states.SetDesired(id, closed, now)
}

func (s *Service) ApplyPhysical(id string, closed, locked bool, sealPressure float64, now time.Time) (model.DoorState, error) {
	return s.states.ApplyPhysical(id, closed, locked, sealPressure, now)
}

func (s *Service) EvaluateRelease(id string, chamberState model.ChamberState, now time.Time) (ReleaseDecision, error) {
	return s.release.Evaluate(id, chamberState, now)
}

func (s *Service) Unlock(id string, chamberState model.ChamberState, now time.Time) (model.DoorState, error) {
	return s.release.Unlock(id, chamberState, now)
}

func (s *Service) State(id string) (model.DoorState, bool) {
	return s.states.State(id)
}

func (s *Service) List() []model.DoorState {
	return s.states.List()
}

func (s *Service) Snapshot() map[string]model.DoorState {
	return s.states.Snapshot()
}

func (s *Service) Restore(states map[string]model.DoorState) {
	s.states.Restore(states)
}
