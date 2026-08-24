package quality

import (
	"time"

	"github.com/google/uuid"
)

type ReleaseEvidence struct {
	CycleID          uuid.UUID `json:"cycle_id"`
	AirRemoved       bool      `json:"air_removed"`
	ColdspotComplete bool      `json:"coldspot_complete"`
	SteamValid       bool      `json:"steam_valid"`
	DrainSafe        bool      `json:"drain_safe"`
	LoadDry          bool      `json:"load_dry"`
	CoolingSafe      bool      `json:"cooling_safe"`
	DoorSafe         bool      `json:"door_safe"`
	EvaluatedAt      time.Time `json:"evaluated_at"`
}

func (e ReleaseEvidence) Valid() bool {
	return e.CycleID != uuid.Nil && e.AirRemoved && e.ColdspotComplete && e.SteamValid && e.DrainSafe && e.LoadDry && e.CoolingSafe && e.DoorSafe
}

func (e ReleaseEvidence) Missing() []string {
	missing := make([]string, 0)
	checks := []struct {
		name string
		ok   bool
	}{
		{"air-removal", e.AirRemoved},
		{"coldspot", e.ColdspotComplete},
		{"steam", e.SteamValid},
		{"drain", e.DrainSafe},
		{"drying", e.LoadDry},
		{"cooling", e.CoolingSafe},
		{"door", e.DoorSafe},
	}
	for _, check := range checks {
		if !check.ok {
			missing = append(missing, check.name)
		}
	}
	return missing
}
