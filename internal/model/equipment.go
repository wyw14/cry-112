package model

import (
	"time"

	"github.com/google/uuid"
)

type ChamberState struct {
	ID                 string    `json:"id"`
	PressureKPa        float64   `json:"pressure_kpa"`
	TemperatureC       float64   `json:"temperature_c"`
	JacketTemperatureC float64   `json:"jacket_temperature_c"`
	DrainBackpressure  float64   `json:"drain_backpressure"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type DoorState struct {
	ID              string    `json:"id"`
	DesiredClosed   bool      `json:"desired_closed"`
	PhysicalClosed  bool      `json:"physical_closed"`
	Locked          bool      `json:"locked"`
	SealPressureBar float64   `json:"seal_pressure_bar"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type SteamState struct {
	TemperatureC float64   `json:"temperature_c"`
	NCGPercent   float64   `json:"ncg_percent"`
	SupplyOpen   bool      `json:"supply_open"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Incident struct {
	ID        uuid.UUID      `json:"id"`
	CycleID   uuid.UUID      `json:"cycle_id"`
	Kind      string         `json:"kind"`
	Message   string         `json:"message"`
	Details   map[string]any `json:"details"`
	CreatedAt time.Time      `json:"created_at"`
}

func NewIncident(cycleID uuid.UUID, kind, message string, details map[string]any, now time.Time) Incident {
	copyDetails := make(map[string]any, len(details))
	for key, value := range details {
		copyDetails[key] = value
	}
	return Incident{ID: uuid.New(), CycleID: cycleID, Kind: kind, Message: message, Details: copyDetails, CreatedAt: now.UTC()}
}

func (i Incident) Clone() Incident {
	copyValue := i
	copyValue.Details = make(map[string]any, len(i.Details))
	for key, value := range i.Details {
		copyValue.Details[key] = value
	}
	return copyValue
}
