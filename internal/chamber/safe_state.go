package chamber

import (
	"fmt"
	"time"

	"github.com/wyw14/cry-112/internal/model"
)

type SafetyProof struct {
	ChamberID       string    `json:"chamber_id"`
	PressureSafe    bool      `json:"pressure_safe"`
	TemperatureSafe bool      `json:"temperature_safe"`
	SealReleased    bool      `json:"seal_released"`
	ObservedAt      time.Time `json:"observed_at"`
	SourceFresh     bool      `json:"source_fresh"`
}

func (p SafetyProof) Valid() bool {
	return p.PressureSafe && p.TemperatureSafe && p.SealReleased && p.SourceFresh
}

type SafetyEvaluator struct {
	ambientPressureKPa float64
	pressureTolerance  float64
	maximumTemperature float64
	maximumAge         time.Duration
}

func NewSafetyEvaluator(maximumTemperature float64) SafetyEvaluator {
	return SafetyEvaluator{ambientPressureKPa: 101.3, pressureTolerance: 3, maximumTemperature: maximumTemperature, maximumAge: 5 * time.Second}
}

func (e SafetyEvaluator) Evaluate(state model.ChamberState, sealPressureBar float64, now time.Time) (SafetyProof, error) {
	if state.ID == "" {
		return SafetyProof{}, fmt.Errorf("chamber state has no identity")
	}
	if sealPressureBar < 0 {
		return SafetyProof{}, fmt.Errorf("seal pressure cannot be negative")
	}
	pressureDifference := state.PressureKPa - e.ambientPressureKPa
	if pressureDifference < 0 {
		pressureDifference = -pressureDifference
	}
	proof := SafetyProof{
		ChamberID:       state.ID,
		PressureSafe:    pressureDifference <= e.pressureTolerance,
		TemperatureSafe: state.TemperatureC <= e.maximumTemperature,
		SealReleased:    sealPressureBar <= 0.15,
		ObservedAt:      now.UTC(),
		SourceFresh:     now.Sub(state.UpdatedAt) <= e.maximumAge,
	}
	return proof, nil
}
