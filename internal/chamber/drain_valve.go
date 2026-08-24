package chamber

import (
	"fmt"
	"sync"
	"time"
)

type DrainValveState struct {
	ChamberID      string    `json:"chamber_id"`
	RequestedFlow  float64   `json:"requested_flow"`
	AllocatedFlow  float64   `json:"allocated_flow"`
	OpeningPercent float64   `json:"opening_percent"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type DrainValve struct {
	mu      sync.RWMutex
	maximum float64
	state   DrainValveState
}

func NewDrainValve(chamberID string, maximumFlow float64, now time.Time) *DrainValve {
	return &DrainValve{maximum: maximumFlow, state: DrainValveState{ChamberID: chamberID, UpdatedAt: now.UTC()}}
}

func (v *DrainValve) Apply(requested, allocated float64, now time.Time) (DrainValveState, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if requested < 0 || allocated < 0 || allocated > requested || allocated > v.maximum {
		return DrainValveState{}, fmt.Errorf("invalid drain flow request %.2f allocation %.2f", requested, allocated)
	}
	opening := 0.0
	if v.maximum > 0 {
		opening = allocated / v.maximum * 100
	}
	v.state.RequestedFlow = requested
	v.state.AllocatedFlow = allocated
	v.state.OpeningPercent = opening
	v.state.UpdatedAt = now.UTC()
	return v.state, nil
}

func (v *DrainValve) Close(now time.Time) DrainValveState {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.state.RequestedFlow = 0
	v.state.AllocatedFlow = 0
	v.state.OpeningPercent = 0
	v.state.UpdatedAt = now.UTC()
	return v.state
}

func (v *DrainValve) State() DrainValveState {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.state
}
