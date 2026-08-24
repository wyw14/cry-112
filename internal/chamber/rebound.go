package chamber

import (
	"fmt"
	"time"
)

type PressurePoint struct {
	PressureKPa float64   `json:"pressure_kpa"`
	ObservedAt  time.Time `json:"observed_at"`
}

type ReboundProof struct {
	Start       PressurePoint `json:"start"`
	End         PressurePoint `json:"end"`
	RiseKPa     float64       `json:"rise_kpa"`
	RateKPaSec  float64       `json:"rate_kpa_sec"`
	MaximumRise float64       `json:"maximum_rise"`
	Valid       bool          `json:"valid"`
}

type ReboundEvaluator struct {
	minimumIsolation time.Duration
}

func NewReboundEvaluator(minimumIsolation time.Duration) ReboundEvaluator {
	return ReboundEvaluator{minimumIsolation: minimumIsolation}
}

func (e ReboundEvaluator) Evaluate(start, end PressurePoint, maximumRise float64) (ReboundProof, error) {
	duration := end.ObservedAt.Sub(start.ObservedAt)
	if duration < e.minimumIsolation {
		return ReboundProof{}, fmt.Errorf("isolation duration %s is shorter than %s", duration, e.minimumIsolation)
	}
	if start.PressureKPa < 0 || end.PressureKPa < 0 || maximumRise <= 0 {
		return ReboundProof{}, fmt.Errorf("invalid rebound measurement")
	}
	rise := end.PressureKPa - start.PressureKPa
	if rise < 0 {
		rise = 0
	}
	proof := ReboundProof{
		Start:       start,
		End:         end,
		RiseKPa:     rise,
		RateKPaSec:  rise / duration.Seconds(),
		MaximumRise: maximumRise,
	}
	proof.Valid = proof.RiseKPa <= proof.MaximumRise
	return proof, nil
}
