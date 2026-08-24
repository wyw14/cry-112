package door

import (
	"fmt"
	"time"

	"github.com/wyw14/cry-112/internal/chamber"
	"github.com/wyw14/cry-112/internal/model"
)

type ReleaseDecision struct {
	DoorID      string              `json:"door_id"`
	Unlock      bool                `json:"unlock"`
	Proof       chamber.SafetyProof `json:"proof"`
	Reason      string              `json:"reason"`
	EvaluatedAt time.Time           `json:"evaluated_at"`
}

type ReleaseService struct {
	states    *StateService
	evaluator chamber.SafetyEvaluator
}

func NewReleaseService(states *StateService, maximumTemperature float64) *ReleaseService {
	return &ReleaseService{states: states, evaluator: chamber.NewSafetyEvaluator(maximumTemperature)}
}

func (s *ReleaseService) Evaluate(doorID string, chamberState model.ChamberState, now time.Time) (ReleaseDecision, error) {
	doorState, ok := s.states.State(doorID)
	if !ok {
		return ReleaseDecision{}, fmt.Errorf("door %s not found", doorID)
	}
	proof, err := s.evaluator.Evaluate(chamberState, doorState.SealPressureBar, now)
	if err != nil {
		return ReleaseDecision{}, err
	}
	decision := ReleaseDecision{DoorID: doorID, Proof: proof, EvaluatedAt: now.UTC()}
	if !proof.PressureSafe || !proof.SealReleased || !proof.TemperatureSafe || !proof.SourceFresh {
		decision.Reason = "chamber, seal, temperature or freshness proof is incomplete"
		return decision, nil
	}
	decision.Unlock = true
	decision.Reason = "all physical release proofs are valid"
	return decision, nil
}

func (s *ReleaseService) Unlock(doorID string, chamberState model.ChamberState, now time.Time) (model.DoorState, error) {
	decision, err := s.Evaluate(doorID, chamberState, now)
	if err != nil {
		return model.DoorState{}, err
	}
	if !decision.Unlock {
		return model.DoorState{}, fmt.Errorf("door release denied: %s", decision.Reason)
	}
	state, _ := s.states.State(doorID)
	return s.states.ApplyPhysical(doorID, state.PhysicalClosed, false, state.SealPressureBar, now)
}
