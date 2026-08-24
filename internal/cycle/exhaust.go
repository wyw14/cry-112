package cycle

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wyw14/cry-112/internal/condensate"
	"github.com/wyw14/cry-112/internal/model"
)

type ExhaustResult struct {
	Cycle       model.Cycle                   `json:"cycle"`
	Reservation condensate.Reservation        `json:"reservation"`
	Proof       condensate.DrainProof         `json:"proof"`
	Allocation  condensate.AllocationSnapshot `json:"allocation"`
}

type DrainRequestResult struct {
	Reservation condensate.Reservation        `json:"reservation"`
	Allocation  condensate.AllocationSnapshot `json:"allocation"`
}

func (c *Controller) RequestSharedDrain(chamberID string, requestedFlow float64, priority int, now time.Time) (DrainRequestResult, error) {
	if _, ok := c.chambers.State(chamberID); !ok {
		return DrainRequestResult{}, fmt.Errorf("chamber %s not found", chamberID)
	}
	reservation, err := c.condensate.Request(chamberID, requestedFlow, priority, now)
	if err != nil {
		return DrainRequestResult{}, err
	}
	return DrainRequestResult{Reservation: reservation, Allocation: c.condensate.Snapshot()}, nil
}

func (c *Controller) ApplyExhaust(cycleID uuid.UUID, requestedFlow, measuredFlow, backpressure float64, returned bool, now time.Time) (ExhaustResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	cycle, ok := c.registry.Get(cycleID)
	if !ok {
		return ExhaustResult{}, fmt.Errorf("cycle %s not found", cycleID)
	}
	if cycle.Phase != model.PhaseExhaust {
		return ExhaustResult{}, fmt.Errorf("cycle is in %s, not exhaust", cycle.Phase)
	}
	reservation, err := c.condensate.Request(cycle.ChamberID, requestedFlow, 10, now)
	if err != nil {
		return ExhaustResult{}, err
	}
	if backpressure == 0 {
		snapshot := c.condensate.Snapshot()
		backpressure = condensate.EstimateBackpressure(snapshot.Committed, snapshot.Capacity)
	}
	proof, err := c.condensate.Observe(cycle.ChamberID, condensate.DrainReading{Flow: measuredFlow, BackpressureBar: backpressure, ReturnDetected: returned, ObservedAt: now.UTC()}, cycle.Recipe.MaximumDrainBackpressure, now)
	if err != nil {
		return ExhaustResult{}, err
	}
	if proof.Valid() {
		cycle, err = c.registry.Transition(cycleID, model.PhaseDrying, now)
		if err != nil {
			return ExhaustResult{}, err
		}
	} else {
		incident := c.addIncidentLocked(cycleID, "drain-backpressure", "drain backpressure high", map[string]any{"flow": measuredFlow, "backpressure": backpressure, "return": returned}, now)
		_ = incident
	}
	result := ExhaustResult{Cycle: cycle, Reservation: reservation, Proof: proof, Allocation: c.condensate.Snapshot()}
	if err := c.recordLocked(cycleID, "exhaust-evaluated", result, now); err != nil {
		return ExhaustResult{}, err
	}
	return result, nil
}
