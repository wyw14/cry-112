package interlock

import (
	"time"

	"github.com/wyw14/cry-112/internal/model"
)

type DoorProof struct {
	DoorID          string    `json:"door_id"`
	PhysicalClosed  bool      `json:"physical_closed"`
	Locked          bool      `json:"locked"`
	SealEstablished bool      `json:"seal_established"`
	Fresh           bool      `json:"fresh"`
	ObservedAt      time.Time `json:"observed_at"`
}

func (p DoorProof) Valid() bool {
	return p.PhysicalClosed && p.Locked && p.SealEstablished && p.Fresh
}

type DoorEvaluator struct {
	minimumSealPressure float64
	maximumAge          time.Duration
}

func NewDoorEvaluator() DoorEvaluator {
	return DoorEvaluator{minimumSealPressure: 2.5, maximumAge: 5 * time.Second}
}

func (e DoorEvaluator) Evaluate(state model.DoorState, now time.Time) DoorProof {
	return DoorProof{
		DoorID:          state.ID,
		PhysicalClosed:  state.PhysicalClosed,
		Locked:          state.Locked,
		SealEstablished: state.SealPressureBar >= e.minimumSealPressure,
		Fresh:           now.Sub(state.UpdatedAt) <= e.maximumAge,
		ObservedAt:      now.UTC(),
	}
}
