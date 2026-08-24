package load

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/wyw14/cry-112/internal/model"
)

type Batch struct {
	ID          uuid.UUID              `json:"id"`
	Description string                 `json:"description"`
	Placements  []model.ProbePlacement `json:"placements"`
	CreatedAt   time.Time              `json:"created_at"`
}

func NewBatch(description string, now time.Time) Batch {
	return Batch{ID: uuid.New(), Description: description, Placements: make([]model.ProbePlacement, 0), CreatedAt: now.UTC()}
}

func (b Batch) Clone() Batch {
	copyValue := b
	copyValue.Placements = append([]model.ProbePlacement(nil), b.Placements...)
	return copyValue
}

type Service struct {
	mu      sync.RWMutex
	batches map[uuid.UUID]Batch
}

func NewService() *Service {
	return &Service{batches: make(map[uuid.UUID]Batch)}
}

func (s *Service) Create(description string, now time.Time) (Batch, error) {
	if description == "" {
		return Batch{}, fmt.Errorf("load description is required")
	}
	batch := NewBatch(description, now)
	s.mu.Lock()
	s.batches[batch.ID] = batch
	s.mu.Unlock()
	return batch.Clone(), nil
}

func (s *Service) Save(batch Batch) error {
	if batch.ID == uuid.Nil {
		return fmt.Errorf("load identity is required")
	}
	s.mu.Lock()
	s.batches[batch.ID] = batch.Clone()
	s.mu.Unlock()
	return nil
}

func (s *Service) Get(id uuid.UUID) (Batch, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	batch, ok := s.batches[id]
	return batch.Clone(), ok
}

func (s *Service) List() []Batch {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Batch, 0, len(s.batches))
	for _, batch := range s.batches {
		result = append(result, batch.Clone())
	}
	return result
}
