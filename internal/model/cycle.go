package model

import (
	"time"

	"github.com/google/uuid"
)

type ProbePlacement struct {
	ProbeID    uuid.UUID `json:"probe_id"`
	LoadID     uuid.UUID `json:"load_id"`
	Position   string    `json:"position"`
	Required   bool      `json:"required"`
	AssignedAt time.Time `json:"assigned_at"`
}

type ProbeReading struct {
	ProbeID      uuid.UUID `json:"probe_id"`
	LoadID       uuid.UUID `json:"load_id"`
	Position     string    `json:"position"`
	TemperatureC float64   `json:"temperature_c"`
	Moisture     float64   `json:"moisture"`
	ObservedAt   time.Time `json:"observed_at"`
}

type Cycle struct {
	ID                uuid.UUID        `json:"id"`
	LoadID            uuid.UUID        `json:"load_id"`
	ChamberID         string           `json:"chamber_id"`
	Recipe            Recipe           `json:"recipe"`
	Phase             Phase            `json:"phase"`
	StartedAt         time.Time        `json:"started_at"`
	UpdatedAt         time.Time        `json:"updated_at"`
	FinishedAt        *time.Time       `json:"finished_at,omitempty"`
	EffectiveExposure time.Duration    `json:"effective_exposure"`
	Failure           string           `json:"failure,omitempty"`
	Placements        []ProbePlacement `json:"placements"`
}

func NewCycle(chamberID string, recipe Recipe, now time.Time) Cycle {
	return Cycle{
		ID:         uuid.New(),
		LoadID:     uuid.New(),
		ChamberID:  chamberID,
		Recipe:     recipe,
		Phase:      PhaseLoaded,
		StartedAt:  now.UTC(),
		UpdatedAt:  now.UTC(),
		Placements: make([]ProbePlacement, 0),
	}
}

func (c Cycle) Clone() Cycle {
	copyValue := c
	copyValue.Placements = append([]ProbePlacement(nil), c.Placements...)
	return copyValue
}

func (c *Cycle) Transition(next Phase, now time.Time) error {
	if err := c.Phase.ValidateTransition(next); err != nil {
		return err
	}
	c.Phase = next
	c.UpdatedAt = now.UTC()
	if next.Terminal() {
		finished := now.UTC()
		c.FinishedAt = &finished
	}
	return nil
}

func (c *Cycle) Fail(reason string, now time.Time) error {
	if c.Phase.Terminal() {
		return nil
	}
	c.Failure = reason
	return c.Transition(PhaseFailed, now)
}
