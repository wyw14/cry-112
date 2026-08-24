package temperature

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

type HistoryKey struct {
	LoadID   uuid.UUID `json:"load_id"`
	ProbeID  uuid.UUID `json:"probe_id"`
	Position string    `json:"position"`
}

type Point struct {
	TemperatureC float64   `json:"temperature_c"`
	ObservedAt   time.Time `json:"observed_at"`
}

type Window struct {
	Key    HistoryKey `json:"key"`
	Points []Point    `json:"points"`
}

func (w Window) Clone() Window {
	copyValue := w
	copyValue.Points = append([]Point(nil), w.Points...)
	return copyValue
}

func (w Window) ContinuousAtOrAbove(threshold float64, duration time.Duration, now time.Time) bool {
	if len(w.Points) == 0 {
		return false
	}
	cutoff := now.Add(-duration)
	firstSeen := time.Time{}
	for _, point := range w.Points {
		if point.ObservedAt.Before(cutoff) {
			continue
		}
		if point.TemperatureC < threshold {
			return false
		}
		if firstSeen.IsZero() {
			firstSeen = point.ObservedAt
		}
	}
	return !firstSeen.IsZero() && !firstSeen.After(cutoff)
}

type HistoryRegistry struct {
	mu        sync.RWMutex
	windows   map[HistoryKey]Window
	retention time.Duration
}

func NewHistoryRegistry(retention time.Duration) *HistoryRegistry {
	return &HistoryRegistry{windows: make(map[HistoryKey]Window), retention: retention}
}

func (r *HistoryRegistry) Observe(key HistoryKey, point Point) (Window, error) {
	if key.LoadID == uuid.Nil || key.ProbeID == uuid.Nil || key.Position == "" {
		return Window{}, fmt.Errorf("complete history ownership is required")
	}
	if point.ObservedAt.IsZero() || point.TemperatureC < -20 {
		return Window{}, fmt.Errorf("temperature point is invalid")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	window := r.windows[key]
	window.Key = key
	window.Points = append(window.Points, point)
	cutoff := point.ObservedAt.Add(-r.retention)
	first := 0
	for first < len(window.Points) && window.Points[first].ObservedAt.Before(cutoff) {
		first++
	}
	window.Points = append([]Point(nil), window.Points[first:]...)
	r.windows[key] = window
	return window.Clone(), nil
}

func (r *HistoryRegistry) Window(key HistoryKey) (Window, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	window, ok := r.windows[key]
	return window.Clone(), ok
}

func (r *HistoryRegistry) RemoveLoad(loadID uuid.UUID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for key := range r.windows {
		if key.LoadID == loadID {
			delete(r.windows, key)
		}
	}
}
