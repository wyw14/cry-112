package cycle

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wyw14/cry-112/internal/filter"
	"github.com/wyw14/cry-112/internal/model"
	"github.com/wyw14/cry-112/internal/vacuum"
)

type VacuumPulseInput struct {
	Sequence       int       `json:"sequence"`
	MinimumKPa     float64   `json:"minimum_kpa"`
	IsolationStart float64   `json:"isolation_start"`
	IsolationEnd   float64   `json:"isolation_end"`
	StartedAt      time.Time `json:"started_at"`
	CompletedAt    time.Time `json:"completed_at"`
}

type AirRemovalResult struct {
	Cycle model.Cycle            `json:"cycle"`
	Proof vacuum.AirRemovalProof `json:"proof"`
}

func (c *Controller) RecordVacuumPulse(cycleID uuid.UUID, input VacuumPulseInput, now time.Time) (AirRemovalResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	cycle, ok := c.registry.Get(cycleID)
	if !ok {
		return AirRemovalResult{}, fmt.Errorf("cycle %s not found", cycleID)
	}
	if cycle.Phase != model.PhaseAirRemoval {
		return AirRemovalResult{}, fmt.Errorf("cycle is in %s, not air-removal", cycle.Phase)
	}
	proof, err := c.vacuum.RecordPulse(cycleID, input.Sequence, input.MinimumKPa, input.StartedAt, input.CompletedAt, input.IsolationStart, input.IsolationEnd, cycle.Recipe)
	if err != nil {
		return AirRemovalResult{}, err
	}
	if proof.Valid() {
		chamberState, _ := c.chambers.State(cycle.ChamberID)
		decision, err := c.steam.Condition(proof, chamberState.JacketTemperatureC, now)
		if err != nil {
			return AirRemovalResult{}, err
		}
		if decision.SupplyOpen {
			cycle, err = c.registry.Transition(cycleID, model.PhaseConditioning, now)
			if err != nil {
				return AirRemovalResult{}, err
			}
		}
	}
	result := AirRemovalResult{Cycle: cycle, Proof: proof}
	if err := c.recordLocked(cycleID, "vacuum-pulse", result, now); err != nil {
		return AirRemovalResult{}, err
	}
	return result, nil
}

func (c *Controller) RetryVacuumAfterCavitation(cycleID uuid.UUID, condenserTemperature, waterFlow float64, now time.Time) (vacuum.RetryRecord, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.registry.Get(cycleID); !ok {
		return vacuum.RetryRecord{}, fmt.Errorf("cycle %s not found", cycleID)
	}
	if _, err := c.cooling.ObserveCondenser(condenserTemperature, waterFlow, now); err != nil {
		return vacuum.RetryRecord{}, err
	}
	record, err := c.vacuum.CavitationRetry(cycleID, condenserTemperature, waterFlow, now, now)
	if err != nil {
		incident := c.addIncidentLocked(cycleID, "vacuum-cavitation", err.Error(), map[string]any{"temperature_c": condenserTemperature, "water_flow": waterFlow}, now)
		if persistErr := c.recordLocked(cycleID, "incident", incident, now); persistErr != nil {
			return vacuum.RetryRecord{}, persistErr
		}
		return record, err
	}
	if err := c.recordLocked(cycleID, "vacuum-retry", record, now); err != nil {
		return vacuum.RetryRecord{}, err
	}
	return record, nil
}

func (c *Controller) VacuumRetryHistory(cycleID uuid.UUID) []vacuum.RetryRecord {
	return c.vacuum.RetryHistory(cycleID)
}

func (c *Controller) RecordFilterIntegrity(chamberID string, pressureDrop, leakRate float64, passed bool, now time.Time) (filter.IntegrityProof, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	filterID, ok := c.filters[chamberID]
	if !ok {
		return filter.IntegrityProof{}, fmt.Errorf("filter for chamber %s not found", chamberID)
	}
	if !passed {
		c.cooling.InvalidateFilter(filterID)
	}
	proof, err := c.cooling.RecordFilter(filterID, pressureDrop, leakRate, passed, now)
	if err != nil {
		return filter.IntegrityProof{}, err
	}
	if err := c.recordLocked(uuid.Nil, "filter-integrity", proof, now); err != nil {
		return filter.IntegrityProof{}, err
	}
	return proof, nil
}
