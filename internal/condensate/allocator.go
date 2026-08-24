package condensate

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

type Reservation struct {
	ChamberID string    `json:"chamber_id"`
	Requested float64   `json:"requested"`
	Allocated float64   `json:"allocated"`
	Priority  int       `json:"priority"`
	UpdatedAt time.Time `json:"updated_at"`
}

type AllocationSnapshot struct {
	Capacity     float64       `json:"capacity"`
	Committed    float64       `json:"committed"`
	Available    float64       `json:"available"`
	Reservations []Reservation `json:"reservations"`
	Version      uint64        `json:"version"`
}

type Allocator struct {
	mu           sync.Mutex
	capacity     float64
	reservations map[string]Reservation
	version      uint64
}

func NewAllocator(capacity float64) (*Allocator, error) {
	if capacity <= 0 {
		return nil, fmt.Errorf("drain capacity must be positive")
	}
	return &Allocator{capacity: capacity, reservations: make(map[string]Reservation)}, nil
}

func (a *Allocator) Reserve(chamberID string, requested float64, priority int, now time.Time) (Reservation, AllocationSnapshot, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if chamberID == "" || requested < 0 || priority < 0 {
		return Reservation{}, AllocationSnapshot{}, fmt.Errorf("invalid drain reservation")
	}
	delete(a.reservations, chamberID)
	committed := a.committedLocked()
	available := a.capacity - committed
	allocated := requested
	if allocated > available {
		allocated = available
	}
	if allocated < 0 {
		allocated = 0
	}
	reservation := Reservation{ChamberID: chamberID, Requested: requested, Allocated: allocated, Priority: priority, UpdatedAt: now.UTC()}
	a.reservations[chamberID] = reservation
	a.rebalanceLocked()
	reservation = a.reservations[chamberID]
	a.version++
	return reservation, a.snapshotLocked(), nil
}

func (a *Allocator) Release(chamberID string) AllocationSnapshot {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.reservations, chamberID)
	a.rebalanceLocked()
	a.version++
	return a.snapshotLocked()
}

func (a *Allocator) Snapshot() AllocationSnapshot {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.snapshotLocked()
}

func (a *Allocator) committedLocked() float64 {
	var total float64
	for _, reservation := range a.reservations {
		total += reservation.Allocated
	}
	return total
}

func (a *Allocator) rebalanceLocked() {
	reservations := make([]Reservation, 0, len(a.reservations))
	for _, reservation := range a.reservations {
		reservations = append(reservations, reservation)
	}
	sort.SliceStable(reservations, func(i, j int) bool {
		if reservations[i].Priority == reservations[j].Priority {
			return reservations[i].UpdatedAt.Before(reservations[j].UpdatedAt)
		}
		return reservations[i].Priority > reservations[j].Priority
	})
	remaining := a.capacity
	for _, reservation := range reservations {
		allocation := reservation.Requested
		if allocation > remaining {
			allocation = remaining
		}
		if allocation < 0 {
			allocation = 0
		}
		reservation.Allocated = allocation
		a.reservations[reservation.ChamberID] = reservation
		remaining -= allocation
	}
}

func (a *Allocator) snapshotLocked() AllocationSnapshot {
	reservations := make([]Reservation, 0, len(a.reservations))
	for _, reservation := range a.reservations {
		reservations = append(reservations, reservation)
	}
	sort.Slice(reservations, func(i, j int) bool { return reservations[i].ChamberID < reservations[j].ChamberID })
	committed := a.committedLocked()
	return AllocationSnapshot{Capacity: a.capacity, Committed: committed, Available: a.capacity - committed, Reservations: reservations, Version: a.version}
}
