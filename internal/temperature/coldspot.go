package temperature

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wyw14/cry-112/internal/model"
)

type ProbeProof struct {
	Placement  model.ProbePlacement `json:"placement"`
	Latest     Point                `json:"latest"`
	Continuous bool                 `json:"continuous"`
	WindowSize int                  `json:"window_size"`
}

type ColdspotProof struct {
	LoadID      uuid.UUID    `json:"load_id"`
	Required    int          `json:"required"`
	Ready       int          `json:"ready"`
	Probes      []ProbeProof `json:"probes"`
	EvaluatedAt time.Time    `json:"evaluated_at"`
}

func (p ColdspotProof) Valid() bool {
	return p.Required > 0 && p.Ready == p.Required
}

type ColdspotEvaluator struct {
	history *HistoryRegistry
}

func NewColdspotEvaluator(history *HistoryRegistry) *ColdspotEvaluator {
	return &ColdspotEvaluator{history: history}
}

func (e *ColdspotEvaluator) Observe(reading model.ProbeReading, placement model.ProbePlacement) error {
	if reading.ProbeID != placement.ProbeID || reading.LoadID != placement.LoadID || reading.Position != placement.Position {
		return fmt.Errorf("probe reading does not match placement")
	}
	key := HistoryKey{LoadID: placement.LoadID, ProbeID: placement.ProbeID, Position: placement.Position}
	_, err := e.history.Observe(key, Point{TemperatureC: reading.TemperatureC, ObservedAt: reading.ObservedAt})
	return err
}

func (e *ColdspotEvaluator) Evaluate(placements []model.ProbePlacement, threshold float64, hold time.Duration, now time.Time) ColdspotProof {
	proof := ColdspotProof{Probes: make([]ProbeProof, 0), EvaluatedAt: now.UTC()}
	for _, placement := range placements {
		if !placement.Required {
			continue
		}
		proof.LoadID = placement.LoadID
		proof.Required++
		key := HistoryKey{LoadID: placement.LoadID, ProbeID: placement.ProbeID, Position: placement.Position}
		window, ok := e.history.Window(key)
		probeProof := ProbeProof{Placement: placement}
		if ok && len(window.Points) > 0 {
			probeProof.Latest = window.Points[len(window.Points)-1]
			probeProof.WindowSize = len(window.Points)
			probeProof.Continuous = window.ContinuousAtOrAbove(threshold, hold, now)
		}
		if probeProof.Continuous {
			proof.Ready++
		}
		proof.Probes = append(proof.Probes, probeProof)
	}
	return proof
}

func (e *ColdspotEvaluator) ResetLoad(loadID uuid.UUID) {
	e.history.RemoveLoad(loadID)
}
