package drying

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

type MoistureReading struct {
	ProbeID    uuid.UUID `json:"probe_id"`
	LoadID     uuid.UUID `json:"load_id"`
	Value      float64   `json:"value"`
	ObservedAt time.Time `json:"observed_at"`
}

type MoistureProof struct {
	LoadID      uuid.UUID         `json:"load_id"`
	Readings    []MoistureReading `json:"readings"`
	Required    int               `json:"required"`
	Ready       int               `json:"ready"`
	Maximum     float64           `json:"maximum"`
	EvaluatedAt time.Time         `json:"evaluated_at"`
}

func (p MoistureProof) Valid() bool {
	return p.Required > 0 && p.Ready == p.Required
}

type MoistureTracker struct {
	mu       sync.RWMutex
	readings map[uuid.UUID]map[uuid.UUID]MoistureReading
}

func NewMoistureTracker() *MoistureTracker {
	return &MoistureTracker{readings: make(map[uuid.UUID]map[uuid.UUID]MoistureReading)}
}

func (t *MoistureTracker) Observe(reading MoistureReading) error {
	if reading.ProbeID == uuid.Nil || reading.LoadID == uuid.Nil || reading.Value < 0 || reading.ObservedAt.IsZero() {
		return fmt.Errorf("invalid load moisture reading")
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.readings[reading.LoadID] == nil {
		t.readings[reading.LoadID] = make(map[uuid.UUID]MoistureReading)
	}
	t.readings[reading.LoadID][reading.ProbeID] = reading
	return nil
}

func (t *MoistureTracker) Evaluate(loadID uuid.UUID, required []uuid.UUID, maximum float64, now time.Time) MoistureProof {
	t.mu.RLock()
	defer t.mu.RUnlock()
	proof := MoistureProof{LoadID: loadID, Required: len(required), Maximum: maximum, Readings: make([]MoistureReading, 0, len(required)), EvaluatedAt: now.UTC()}
	for _, probeID := range required {
		reading, ok := t.readings[loadID][probeID]
		if ok {
			proof.Readings = append(proof.Readings, reading)
			if reading.Value <= maximum && now.Sub(reading.ObservedAt) <= 10*time.Second {
				proof.Ready++
			}
		}
	}
	return proof
}

func (t *MoistureTracker) Reset(loadID uuid.UUID) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.readings, loadID)
}
