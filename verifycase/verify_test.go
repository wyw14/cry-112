package verifycase

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/wyw14/cry-112/internal/cycle"
	"github.com/wyw14/cry-112/internal/model"
)

func TestExposureStartsOnlyAfterLoadColdspotReady(t *testing.T) {
	now := time.Now().UTC()
	controller, err := cycle.NewController(t.TempDir(), now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := controller.UpdateChamber("A", 101.3, 134, 130, 0, now); err != nil {
		t.Fatal(err)
	}
	value, err := controller.Create("A", "dense wrapped instrument load", model.DefaultRecipe(), now)
	if err != nil {
		t.Fatal(err)
	}
	for sequence := 1; sequence <= 3; sequence++ {
		input := cycle.VacuumPulseInput{Sequence: sequence, MinimumKPa: 10, IsolationStart: 10, IsolationEnd: 10.5, StartedAt: now.Add(time.Duration(sequence) * time.Minute), CompletedAt: now.Add(time.Duration(sequence)*time.Minute + 30*time.Second)}
		if _, err := controller.RecordVacuumPulse(value.ID, input, input.CompletedAt); err != nil {
			t.Fatal(err)
		}
	}
	probeID := uuid.New()
	if _, err := controller.AssignProbe(value.ID, probeID, "pack-center", true, now.Add(4*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := controller.ObserveProbe(value.ID, probeID, 128, 8, now.Add(4*time.Minute)); err != nil {
		t.Fatal(err)
	}
	result, err := controller.ApplyExposure(value.ID, 134, 1, time.Minute, now.Add(4*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if result.Coldspot.Valid() {
		t.Fatal("coldspot proof unexpectedly valid")
	}
	if result.AdvancedBy != 0 || result.Cycle.EffectiveExposure != 0 {
		t.Fatalf("exposure advanced by %s before coldspot readiness", result.AdvancedBy)
	}
}
