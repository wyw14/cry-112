package model

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Event struct {
	ID        uuid.UUID       `json:"id"`
	CycleID   uuid.UUID       `json:"cycle_id"`
	Kind      string          `json:"kind"`
	Payload   json.RawMessage `json:"payload"`
	CreatedAt time.Time       `json:"created_at"`
}

func NewEvent(cycleID uuid.UUID, kind string, payload any, now time.Time) (Event, error) {
	if kind == "" {
		return Event{}, fmt.Errorf("event kind is required")
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return Event{}, fmt.Errorf("encode event payload: %w", err)
	}
	return Event{ID: uuid.New(), CycleID: cycleID, Kind: kind, Payload: encoded, CreatedAt: now.UTC()}, nil
}

func (e Event) Decode(target any) error {
	if err := json.Unmarshal(e.Payload, target); err != nil {
		return fmt.Errorf("decode event %s: %w", e.ID, err)
	}
	return nil
}

func (e Event) Clone() Event {
	copyValue := e
	copyValue.Payload = append(json.RawMessage(nil), e.Payload...)
	return copyValue
}

type Snapshot struct {
	Version   int64                   `json:"version"`
	Cycles    map[uuid.UUID]Cycle     `json:"cycles"`
	Chambers  map[string]ChamberState `json:"chambers"`
	Doors     map[string]DoorState    `json:"doors"`
	Steam     SteamState              `json:"steam"`
	Incidents []Incident              `json:"incidents"`
	SavedAt   time.Time               `json:"saved_at"`
}

func EmptySnapshot(now time.Time) Snapshot {
	return Snapshot{
		Cycles:    make(map[uuid.UUID]Cycle),
		Chambers:  make(map[string]ChamberState),
		Doors:     make(map[string]DoorState),
		Incidents: make([]Incident, 0),
		SavedAt:   now.UTC(),
	}
}
