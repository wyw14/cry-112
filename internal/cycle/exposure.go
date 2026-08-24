package cycle

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wyw14/cry-112/internal/drying"
	"github.com/wyw14/cry-112/internal/model"
	"github.com/wyw14/cry-112/internal/steam"
	"github.com/wyw14/cry-112/internal/temperature"
)

type ExposureResult struct {
	Cycle       model.Cycle               `json:"cycle"`
	Coldspot    temperature.ColdspotProof `json:"coldspot"`
	Steam       steam.QualityProof        `json:"steam"`
	SteamReason string                    `json:"steam_reason"`
	AdvancedBy  time.Duration             `json:"advanced_by"`
}

func (c *Controller) AssignProbe(cycleID, probeID uuid.UUID, position string, required bool, now time.Time) (model.ProbePlacement, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	cycle, ok := c.registry.Get(cycleID)
	if !ok {
		return model.ProbePlacement{}, fmt.Errorf("cycle %s not found", cycleID)
	}
	placement, err := c.probeMap.Assign(cycle.LoadID, probeID, position, required, now)
	if err != nil {
		return model.ProbePlacement{}, err
	}
	cycle.Placements = c.probeMap.ForLoad(cycle.LoadID)
	batch, ok := c.loads.Get(cycle.LoadID)
	if !ok {
		return model.ProbePlacement{}, fmt.Errorf("load %s not found", cycle.LoadID)
	}
	batch.Placements = append([]model.ProbePlacement(nil), cycle.Placements...)
	if err := c.loads.Save(batch); err != nil {
		return model.ProbePlacement{}, err
	}
	if err := c.registry.Save(cycle); err != nil {
		return model.ProbePlacement{}, err
	}
	if err := c.recordLocked(cycleID, "probe-assigned", placement, now); err != nil {
		return model.ProbePlacement{}, err
	}
	return placement, nil
}

func (c *Controller) ObserveProbe(cycleID, probeID uuid.UUID, temperatureC, moisture float64, now time.Time) (model.ProbeReading, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	cycle, ok := c.registry.Get(cycleID)
	if !ok {
		return model.ProbeReading{}, fmt.Errorf("cycle %s not found", cycleID)
	}
	placement, ok := c.probeMap.Placement(probeID)
	if !ok || placement.LoadID != cycle.LoadID {
		return model.ProbeReading{}, fmt.Errorf("probe %s is not assigned to cycle load", probeID)
	}
	reading := model.ProbeReading{ProbeID: probeID, LoadID: cycle.LoadID, Position: placement.Position, TemperatureC: temperatureC, Moisture: moisture, ObservedAt: now.UTC()}
	if err := c.temperature.Observe(reading, placement); err != nil {
		return model.ProbeReading{}, err
	}
	if err := c.drying.ObserveLoad(dryingReading(reading)); err != nil {
		return model.ProbeReading{}, err
	}
	if err := c.recordLocked(cycleID, "probe-reading", reading, now); err != nil {
		return model.ProbeReading{}, err
	}
	return reading, nil
}

func (c *Controller) Coldspot(cycleID uuid.UUID, now time.Time) (temperature.ColdspotProof, error) {
	cycle, ok := c.registry.Get(cycleID)
	if !ok {
		return temperature.ColdspotProof{}, fmt.Errorf("cycle %s not found", cycleID)
	}
	return c.temperature.Coldspot(cycle.Placements, cycle.Recipe.ExposureTemperatureC, cycle.Recipe.ColdspotHold, now), nil
}

func dryingReading(reading model.ProbeReading) drying.MoistureReading {
	return drying.MoistureReading{ProbeID: reading.ProbeID, LoadID: reading.LoadID, Value: reading.Moisture, ObservedAt: reading.ObservedAt}
}

func (c *Controller) ApplyExposure(cycleID uuid.UUID, steamTemperature, ncgPercent float64, elapsed time.Duration, now time.Time) (ExposureResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	cycle, ok := c.registry.Get(cycleID)
	if !ok {
		return ExposureResult{}, fmt.Errorf("cycle %s not found", cycleID)
	}
	if cycle.Phase == model.PhaseConditioning {
		var err error
		cycle, err = c.registry.Transition(cycleID, model.PhaseExposure, now)
		if err != nil {
			return ExposureResult{}, err
		}
	}
	if cycle.Phase != model.PhaseExposure {
		return ExposureResult{}, fmt.Errorf("cycle is in %s, not exposure", cycle.Phase)
	}
	if _, err := c.steam.Observe(steamTemperature, ncgPercent, now); err != nil {
		return ExposureResult{}, err
	}
	steamProof := c.steam.Quality(cycle.Recipe.MaximumSteamNCGPercent, cycle.Recipe.ExposureTemperatureC, now)
	steamValid, reason := c.quality.SteamValid(steamProof, now)
	coldspot := c.temperature.Coldspot(cycle.Placements, cycle.Recipe.ExposureTemperatureC, cycle.Recipe.ColdspotHold, now)
	coldspotReady := coldspot.AllRequiredReady()
	advanced := time.Duration(0)
	// Effective exposure accumulates only once every coldspot probe the batch
	// requires has reached and is continuously holding the exposure temperature
	// for the recipe hold. A coldpoint that has not yet reached temperature, or
	// that drops below it mid-exposure, causes the hold proof to lapse; timing
	// resumes only after the hold is re-established. This prevents the chamber
	// average reaching setpoint from starting the count while the dense pack
	// center is still cold, and applies the recipe's hold rule on any drop.
	if steamValid && coldspotReady {
		cycle.EffectiveExposure += elapsed
		advanced = elapsed
		if cycle.EffectiveExposure >= cycle.Recipe.ExposureDuration {
			if err := cycle.Transition(model.PhaseExhaust, now); err != nil {
				return ExposureResult{}, err
			}
			c.steam.CloseSupply(now)
		}
		if err := c.registry.Save(cycle); err != nil {
			return ExposureResult{}, err
		}
	}
	if !steamValid {
		incident := c.addIncidentLocked(cycleID, "steam-quality", reason, map[string]any{"peak_ncg": steamProof.PeakNCG, "average_ncg": steamProof.AverageNCG}, now)
		_ = incident
	}
	if !coldspotReady {
		incident := c.addIncidentLocked(cycleID, "coldspot-hold", "coldspot hold not established", map[string]any{"required": coldspot.Required, "ready": coldspot.Ready}, now)
		_ = incident
	}
	result := ExposureResult{Cycle: cycle, Coldspot: coldspot, Steam: steamProof, SteamReason: reason, AdvancedBy: advanced}
	if err := c.recordLocked(cycleID, "exposure-evaluated", result, now); err != nil {
		return ExposureResult{}, err
	}
	return result, nil
}
