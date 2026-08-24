package steam

import (
	"fmt"
	"time"

	"github.com/wyw14/cry-112/internal/vacuum"
)

type ConditioningDecision struct {
	AirRemoved  bool      `json:"air_removed"`
	SupplyOpen  bool      `json:"supply_open"`
	Reason      string    `json:"reason"`
	EvaluatedAt time.Time `json:"evaluated_at"`
}

type Conditioner struct {
	minimumJacketTemperature float64
}

func NewConditioner(minimumJacketTemperature float64) Conditioner {
	return Conditioner{minimumJacketTemperature: minimumJacketTemperature}
}

func (c Conditioner) Evaluate(proof vacuum.AirRemovalProof, jacketTemperature float64, now time.Time) (ConditioningDecision, error) {
	if jacketTemperature < -20 {
		return ConditioningDecision{}, fmt.Errorf("jacket temperature is outside physical range")
	}
	decision := ConditioningDecision{AirRemoved: proof.Valid(), EvaluatedAt: now.UTC()}
	if !proof.Valid() {
		decision.Reason = "air removal proof is incomplete"
		return decision, nil
	}
	if jacketTemperature < c.minimumJacketTemperature {
		decision.Reason = "jacket is below conditioning temperature"
		return decision, nil
	}
	decision.SupplyOpen = true
	decision.Reason = "air removal and jacket readiness are proven"
	return decision, nil
}
