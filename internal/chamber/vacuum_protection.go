package chamber

import (
	"fmt"
	"time"
)

type MakeupDecision struct {
	OpenValve     bool      `json:"open_valve"`
	SlowCooling   bool      `json:"slow_cooling"`
	Reason        string    `json:"reason"`
	PressureKPa   float64   `json:"pressure_kpa"`
	SterilePermit bool      `json:"sterile_permit"`
	EvaluatedAt   time.Time `json:"evaluated_at"`
}

type VacuumProtection struct {
	openThresholdKPa float64
	slowThresholdKPa float64
}

func NewVacuumProtection(openThresholdKPa, slowThresholdKPa float64) (VacuumProtection, error) {
	if openThresholdKPa <= 0 || slowThresholdKPa <= openThresholdKPa {
		return VacuumProtection{}, fmt.Errorf("vacuum protection thresholds are invalid")
	}
	return VacuumProtection{openThresholdKPa: openThresholdKPa, slowThresholdKPa: slowThresholdKPa}, nil
}

func (p VacuumProtection) Evaluate(pressureKPa float64, sterilePermit bool, now time.Time) MakeupDecision {
	decision := MakeupDecision{PressureKPa: pressureKPa, SterilePermit: sterilePermit, EvaluatedAt: now.UTC()}
	if pressureKPa >= p.slowThresholdKPa {
		decision.Reason = "pressure remains above intervention threshold"
		return decision
	}
	if pressureKPa < p.openThresholdKPa {
		decision.OpenValve = true
		decision.Reason = "sterile makeup air permitted"
		return decision
	}
	decision.SlowCooling = true
	if sterilePermit {
		decision.Reason = "pressure requires controlled cooling before valve opening"
	} else {
		decision.Reason = "sterile path is not proven"
	}
	return decision
}
