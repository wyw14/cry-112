package vacuum

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

type CondenserProof struct {
	TemperatureC float64   `json:"temperature_c"`
	WaterFlow    float64   `json:"water_flow"`
	ObservedAt   time.Time `json:"observed_at"`
}

func (p CondenserProof) Ready(maximumTemperature, minimumFlow float64, now time.Time) bool {
	return p.TemperatureC <= maximumTemperature && p.WaterFlow >= minimumFlow && now.Sub(p.ObservedAt) <= 5*time.Second
}

type RetryRecord struct {
	CycleID     uuid.UUID `json:"cycle_id"`
	Attempt     int       `json:"attempt"`
	Reason      string    `json:"reason"`
	Ready       bool      `json:"ready"`
	RequestedAt time.Time `json:"requested_at"`
}

type RetryCoordinator struct {
	mu                   sync.Mutex
	maximumAttempts      int
	maximumCondenserTemp float64
	minimumWaterFlow     float64
	records              map[uuid.UUID][]RetryRecord
}

func NewRetryCoordinator(maximumAttempts int, maximumCondenserTemp, minimumWaterFlow float64) *RetryCoordinator {
	return &RetryCoordinator{maximumAttempts: maximumAttempts, maximumCondenserTemp: maximumCondenserTemp, minimumWaterFlow: minimumWaterFlow, records: make(map[uuid.UUID][]RetryRecord)}
}

func (r *RetryCoordinator) Request(cycleID uuid.UUID, reason string, proof CondenserProof, now time.Time) (RetryRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	attempt := len(r.records[cycleID]) + 1
	if attempt > r.maximumAttempts {
		return RetryRecord{}, fmt.Errorf("vacuum retry limit reached")
	}
	ready := reason != "cavitation" || proof.Ready(r.maximumCondenserTemp, r.minimumWaterFlow, now)
	record := RetryRecord{CycleID: cycleID, Attempt: attempt, Reason: reason, Ready: ready, RequestedAt: now.UTC()}
	r.records[cycleID] = append(r.records[cycleID], record)
	if !ready {
		return record, fmt.Errorf("condenser is not ready after cavitation")
	}
	return record, nil
}

func (r *RetryCoordinator) Records(cycleID uuid.UUID) []RetryRecord {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]RetryRecord(nil), r.records[cycleID]...)
}

func (r *RetryCoordinator) Reset(cycleID uuid.UUID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.records, cycleID)
}
