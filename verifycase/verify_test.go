package verifycase

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/wyw14/cry-112/internal/drying"
)

func TestDryingCompletionRequiresLoadMoistureProof(t *testing.T) {
	now := time.Now().UTC()
	service := drying.NewService()
	cycleID := uuid.New()
	loadID := uuid.New()
	probeID := uuid.New()
	reading := drying.MoistureReading{ProbeID: probeID, LoadID: loadID, Value: 12, ObservedAt: now}
	if err := service.ObserveLoad(reading); err != nil {
		t.Fatal(err)
	}
	state, err := service.ApplyHumidity(cycleID, loadID, 2, []uuid.UUID{probeID}, 3, 4, now)
	if err != nil {
		t.Fatal(err)
	}
	if !state.JacketReady {
		t.Fatal("jacket setup should be dry")
	}
	if state.LoadProof.Valid() {
		t.Fatal("wet load unexpectedly has a valid moisture proof")
	}
	if state.Complete {
		t.Fatal("drying completed from jacket humidity while package remained wet")
	}
}
