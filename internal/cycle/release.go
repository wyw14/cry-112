package cycle

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wyw14/cry-112/internal/cooling"
	"github.com/wyw14/cry-112/internal/door"
	"github.com/wyw14/cry-112/internal/drying"
	"github.com/wyw14/cry-112/internal/interlock"
	"github.com/wyw14/cry-112/internal/model"
	"github.com/wyw14/cry-112/internal/quality"
)

type DryingResult struct {
	Cycle model.Cycle  `json:"cycle"`
	State drying.State `json:"state"`
}

type CoolingResult struct {
	Cycle model.Cycle           `json:"cycle"`
	State cooling.SequenceState `json:"state"`
}

type ReleaseResult struct {
	Cycle     model.Cycle                   `json:"cycle"`
	Interlock interlock.PassThroughDecision `json:"interlock"`
	Evidence  quality.ReleaseEvidence       `json:"evidence"`
}

type DoorPermitResult struct {
	Door      door.ReleaseDecision          `json:"door"`
	Interlock interlock.PassThroughDecision `json:"interlock"`
	Permit    bool                          `json:"permit"`
}

func (c *Controller) EvaluateDoorPermit(chamberID, requestedID, peerID string, now time.Time) (DoorPermitResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	chamberState, ok := c.chambers.State(chamberID)
	if !ok {
		return DoorPermitResult{}, fmt.Errorf("chamber %s not found", chamberID)
	}
	requested, ok := c.doors.State(requestedID)
	if !ok {
		return DoorPermitResult{}, fmt.Errorf("door %s not found", requestedID)
	}
	peer, ok := c.doors.State(peerID)
	if !ok {
		return DoorPermitResult{}, fmt.Errorf("door %s not found", peerID)
	}
	doorDecision, err := c.doors.EvaluateRelease(requestedID, chamberState, now)
	if err != nil {
		return DoorPermitResult{}, err
	}
	interlockDecision, err := c.interlocks.PermitRelease(requested, peer, now)
	if err != nil {
		return DoorPermitResult{}, err
	}
	return DoorPermitResult{Door: doorDecision, Interlock: interlockDecision, Permit: doorDecision.Unlock && interlockDecision.Permit}, nil
}

func (c *Controller) ApplyDrying(cycleID uuid.UUID, jacketHumidity, maximumJacket float64, now time.Time) (DryingResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	cycle, ok := c.registry.Get(cycleID)
	if !ok {
		return DryingResult{}, fmt.Errorf("cycle %s not found", cycleID)
	}
	if cycle.Phase != model.PhaseDrying {
		return DryingResult{}, fmt.Errorf("cycle is in %s, not drying", cycle.Phase)
	}
	required := requiredProbeIDs(cycle.Placements)
	state, err := c.drying.ApplyHumidity(cycleID, cycle.LoadID, jacketHumidity, required, maximumJacket, cycle.Recipe.MaximumLoadMoisture, now)
	if err != nil {
		return DryingResult{}, err
	}
	if state.Complete && c.quality.LoadDry(state.LoadProof, now) {
		c.condensate.Finish(cycle.ChamberID, now)
		cycle, err = c.registry.Transition(cycleID, model.PhaseCooling, now)
		if err != nil {
			return DryingResult{}, err
		}
	}
	result := DryingResult{Cycle: cycle, State: state}
	if err := c.recordLocked(cycleID, "drying-evaluated", result, now); err != nil {
		return DryingResult{}, err
	}
	return result, nil
}

func requiredProbeIDs(placements []model.ProbePlacement) []uuid.UUID {
	ids := make([]uuid.UUID, 0)
	for _, placement := range placements {
		if placement.Required {
			ids = append(ids, placement.ProbeID)
		}
	}
	return ids
}

func (c *Controller) ApplyCooling(cycleID uuid.UUID, pressureKPa, rate float64, now time.Time) (CoolingResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	cycle, ok := c.registry.Get(cycleID)
	if !ok {
		return CoolingResult{}, fmt.Errorf("cycle %s not found", cycleID)
	}
	if cycle.Phase != model.PhaseCooling {
		return CoolingResult{}, fmt.Errorf("cycle is in %s, not cooling", cycle.Phase)
	}
	filterID := c.filters[cycle.ChamberID]
	state, err := c.cooling.Tick(cycleID, cycle.ChamberID, filterID, pressureKPa, rate, now)
	if err != nil {
		return CoolingResult{}, err
	}
	result := CoolingResult{Cycle: cycle, State: state}
	if err := c.recordLocked(cycleID, "cooling-evaluated", result, now); err != nil {
		return CoolingResult{}, err
	}
	return result, nil
}

func (c *Controller) Release(cycleID uuid.UUID, now time.Time) (ReleaseResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	cycle, ok := c.registry.Get(cycleID)
	if !ok {
		return ReleaseResult{}, fmt.Errorf("cycle %s not found", cycleID)
	}
	if cycle.Phase != model.PhaseCooling {
		return ReleaseResult{}, fmt.Errorf("cycle is in %s, not cooling", cycle.Phase)
	}
	chamberState, _ := c.chambers.State(cycle.ChamberID)
	requestedID := cycle.ChamberID + "-unload"
	peerID := cycle.ChamberID + "-load"
	requested, _ := c.doors.State(requestedID)
	peer, _ := c.doors.State(peerID)
	interlockDecision, err := c.interlocks.PermitRelease(requested, peer, now)
	if err != nil {
		return ReleaseResult{}, err
	}
	doorDecision, err := c.doors.EvaluateRelease(requestedID, chamberState, now)
	if err != nil {
		return ReleaseResult{}, err
	}
	dryingState, dryingOK := c.drying.State(cycleID)
	coolingState, coolingOK := c.cooling.SequenceState(cycleID)
	vacuumProof, vacuumOK := c.vacuum.AirRemovalProof(cycleID)
	steamProof := c.steam.Quality(cycle.Recipe.MaximumSteamNCGPercent, cycle.Recipe.ExposureTemperatureC, now)
	drainProof, drainOK := c.condensate.Proof(cycle.ChamberID)
	coldspotProof := c.temperature.Coldspot(cycle.Placements, cycle.Recipe.ExposureTemperatureC, cycle.Recipe.ColdspotHold, now)
	evidence := quality.ReleaseEvidence{
		CycleID:          cycleID,
		AirRemoved:       vacuumOK && vacuumProof.Valid(),
		ColdspotComplete: coldspotProof.Valid(),
		SteamValid:       steamProof.Valid(),
		DrainSafe:        drainOK && drainProof.Valid(),
		LoadDry:          dryingOK && dryingState.Complete,
		CoolingSafe:      coolingOK && !coolingState.Makeup.SlowCooling,
		DoorSafe:         doorDecision.Unlock && interlockDecision.Permit,
		EvaluatedAt:      now.UTC(),
	}
	if err := c.quality.RecordRelease(evidence); err != nil {
		return ReleaseResult{}, err
	}
	evidence, _ = c.quality.Release(cycleID)
	if !evidence.Valid() {
		return ReleaseResult{Cycle: cycle, Interlock: interlockDecision, Evidence: evidence}, fmt.Errorf("release proof is incomplete: %v", evidence.Missing())
	}
	if _, err := c.doors.Unlock(requestedID, chamberState, now); err != nil {
		return ReleaseResult{}, err
	}
	cycle, err = c.registry.Transition(cycleID, model.PhaseReleased, now)
	if err != nil {
		return ReleaseResult{}, err
	}
	result := ReleaseResult{Cycle: cycle, Interlock: interlockDecision, Evidence: evidence}
	if err := c.recordLocked(cycleID, "cycle-released", result, now); err != nil {
		return ReleaseResult{}, err
	}
	c.probeMap.RemoveLoad(cycle.LoadID)
	c.temperature.ResetLoad(cycle.LoadID)
	c.drying.Finish(cycleID, cycle.LoadID)
	c.cooling.Finish(cycleID)
	if err := c.vacuum.Reset(cycleID); err != nil {
		return ReleaseResult{}, err
	}
	c.steam.ResetAnalyzer()
	return result, nil
}

func (c *Controller) UpdateDoor(id string, desiredClosed, physicalClosed, locked bool, sealPressure float64, now time.Time) (model.DoorState, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, err := c.doors.SetDesired(id, desiredClosed, now); err != nil {
		return model.DoorState{}, err
	}
	state, err := c.doors.ApplyPhysical(id, physicalClosed, locked, sealPressure, now)
	if err != nil {
		return model.DoorState{}, err
	}
	if err := c.recordLocked(uuid.Nil, "door-updated", state, now); err != nil {
		return model.DoorState{}, err
	}
	return state, nil
}

func (c *Controller) UpdateChamber(id string, pressure, temperature, jacket, drainBackpressure float64, now time.Time) (model.ChamberState, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	state, err := c.chambers.Update(id, pressure, temperature, jacket, drainBackpressure, now)
	if err != nil {
		return model.ChamberState{}, err
	}
	if err := c.recordLocked(uuid.Nil, "chamber-updated", state, now); err != nil {
		return model.ChamberState{}, err
	}
	return state, nil
}
