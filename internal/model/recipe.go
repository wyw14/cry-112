package model

import (
	"fmt"
	"time"
)

type Recipe struct {
	Name                     string        `json:"name"`
	ExposureTemperatureC     float64       `json:"exposure_temperature_c"`
	ExposureDuration         time.Duration `json:"exposure_duration"`
	ColdspotHold             time.Duration `json:"coldspot_hold"`
	VacuumDepthKPa           float64       `json:"vacuum_depth_kpa"`
	MaximumReboundKPa        float64       `json:"maximum_rebound_kpa"`
	RequiredVacuumPulses     int           `json:"required_vacuum_pulses"`
	MaximumSteamNCGPercent   float64       `json:"maximum_steam_ncg_percent"`
	MaximumLoadMoisture      float64       `json:"maximum_load_moisture"`
	MaximumReleaseTempC      float64       `json:"maximum_release_temp_c"`
	MaximumDrainBackpressure float64       `json:"maximum_drain_backpressure"`
}

func DefaultRecipe() Recipe {
	return Recipe{
		Name:                     "wrapped-instruments-134c",
		ExposureTemperatureC:     134,
		ExposureDuration:         4 * time.Minute,
		ColdspotHold:             30 * time.Second,
		VacuumDepthKPa:           12,
		MaximumReboundKPa:        1.3,
		RequiredVacuumPulses:     3,
		MaximumSteamNCGPercent:   3.5,
		MaximumLoadMoisture:      4,
		MaximumReleaseTempC:      80,
		MaximumDrainBackpressure: 1.8,
	}
}

func (r Recipe) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("recipe name is required")
	}
	if r.ExposureTemperatureC < 121 || r.ExposureTemperatureC > 140 {
		return fmt.Errorf("exposure temperature %.1f is outside the supported range", r.ExposureTemperatureC)
	}
	if r.ExposureDuration <= 0 || r.ColdspotHold <= 0 {
		return fmt.Errorf("exposure and coldspot durations must be positive")
	}
	if r.RequiredVacuumPulses < 1 || r.MaximumReboundKPa <= 0 {
		return fmt.Errorf("vacuum proof parameters are invalid")
	}
	if r.MaximumSteamNCGPercent <= 0 || r.MaximumLoadMoisture <= 0 {
		return fmt.Errorf("quality limits must be positive")
	}
	return nil
}

func (r Recipe) Clone() Recipe {
	return r
}
