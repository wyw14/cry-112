package condensate

import (
	"fmt"
	"math"
	"time"
)

type DrainReading struct {
	Flow            float64   `json:"flow"`
	BackpressureBar float64   `json:"backpressure_bar"`
	ReturnDetected  bool      `json:"return_detected"`
	ObservedAt      time.Time `json:"observed_at"`
}

type DrainProof struct {
	Reading        DrainReading `json:"reading"`
	FlowWithinPlan bool         `json:"flow_within_plan"`
	PressureSafe   bool         `json:"pressure_safe"`
	NoReturn       bool         `json:"no_return"`
	EvaluatedAt    time.Time    `json:"evaluated_at"`
}

func (p DrainProof) Valid() bool {
	return p.FlowWithinPlan && p.PressureSafe && p.NoReturn
}

type DrainEvaluator struct {
	flowTolerance float64
}

func NewDrainEvaluator() DrainEvaluator {
	return DrainEvaluator{flowTolerance: 0.05}
}

func (e DrainEvaluator) Evaluate(reading DrainReading, allocated, maximumBackpressure float64, now time.Time) (DrainProof, error) {
	if reading.Flow < 0 || reading.BackpressureBar < 0 || allocated < 0 || maximumBackpressure <= 0 {
		return DrainProof{}, fmt.Errorf("invalid drain measurement")
	}
	proof := DrainProof{
		Reading:        reading,
		FlowWithinPlan: reading.Flow <= allocated+e.flowTolerance,
		PressureSafe:   reading.BackpressureBar <= maximumBackpressure,
		NoReturn:       !reading.ReturnDetected,
		EvaluatedAt:    now.UTC(),
	}
	return proof, nil
}

func EstimateBackpressure(totalFlow, capacity float64) float64 {
	if capacity <= 0 {
		return math.Inf(1)
	}
	ratio := totalFlow / capacity
	if ratio < 0 {
		ratio = 0
	}
	return 0.4 + ratio*ratio*1.4
}
