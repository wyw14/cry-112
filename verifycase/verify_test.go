package verifycase

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/wyw14/cry-112/internal/cooling"
	"github.com/wyw14/cry-112/internal/filter"
)

func TestCoolingMakeupAirRequiresSterileFilterProof(t *testing.T) {
	integrity := filter.NewIntegrityService()
	sequence, err := cooling.NewSequence(integrity)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	state, err := sequence.Tick(uuid.New(), "A", uuid.New(), 60, 12, now)
	if err != nil {
		t.Fatal(err)
	}
	if state.Makeup.SterilePermit {
		t.Fatal("missing filter proof produced a sterile permit")
	}
	if state.Makeup.OpenValve {
		t.Fatal("makeup valve opened through an unproved filter")
	}
	if !state.Makeup.SlowCooling || state.CoolingRate >= 12 {
		t.Fatalf("cooling did not enter the safe degraded path: %#v", state)
	}
}
