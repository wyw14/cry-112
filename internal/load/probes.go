package load

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/wyw14/cry-112/internal/model"
)

type ProbeMap struct {
	mu         sync.RWMutex
	placements map[uuid.UUID]model.ProbePlacement
	byLoad     map[uuid.UUID][]uuid.UUID
}

func NewProbeMap() *ProbeMap {
	return &ProbeMap{placements: make(map[uuid.UUID]model.ProbePlacement), byLoad: make(map[uuid.UUID][]uuid.UUID)}
}

func (m *ProbeMap) Assign(loadID, probeID uuid.UUID, position string, required bool, now time.Time) (model.ProbePlacement, error) {
	if loadID == uuid.Nil || probeID == uuid.Nil || position == "" {
		return model.ProbePlacement{}, fmt.Errorf("load, probe and position are required")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if previous, exists := m.placements[probeID]; exists {
		ids := m.byLoad[previous.LoadID]
		filtered := ids[:0]
		for _, id := range ids {
			if id != probeID {
				filtered = append(filtered, id)
			}
		}
		m.byLoad[previous.LoadID] = filtered
	}
	placement := model.ProbePlacement{ProbeID: probeID, LoadID: loadID, Position: position, Required: required, AssignedAt: now.UTC()}
	m.placements[probeID] = placement
	m.byLoad[loadID] = appendUnique(m.byLoad[loadID], probeID)
	return placement, nil
}

func appendUnique(ids []uuid.UUID, value uuid.UUID) []uuid.UUID {
	for _, id := range ids {
		if id == value {
			return ids
		}
	}
	return append(ids, value)
}

func (m *ProbeMap) Placement(probeID uuid.UUID) (model.ProbePlacement, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	placement, ok := m.placements[probeID]
	return placement, ok
}

func (m *ProbeMap) ForLoad(loadID uuid.UUID) []model.ProbePlacement {
	m.mu.RLock()
	defer m.mu.RUnlock()
	placements := make([]model.ProbePlacement, 0, len(m.byLoad[loadID]))
	for _, id := range m.byLoad[loadID] {
		placements = append(placements, m.placements[id])
	}
	return placements
}

func (m *ProbeMap) RemoveLoad(loadID uuid.UUID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, id := range m.byLoad[loadID] {
		delete(m.placements, id)
	}
	delete(m.byLoad, loadID)
}
