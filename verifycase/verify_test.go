package verifycase

import (
	"testing"
	"time"

	"github.com/wyw14/cry-112/internal/steam"
)

func TestNoncondensableSteamSpikeInvalidatesExposure(t *testing.T) {
	now := time.Now().UTC()
	analyzer := steam.NewAnalyzer(5 * time.Minute)
	for index := 0; index < 20; index++ {
		sample := steam.Sample{TemperatureC: 134, NCGPercent: 1, ObservedAt: now.Add(time.Duration(index-20) * time.Second)}
		if err := analyzer.Observe(sample); err != nil {
			t.Fatal(err)
		}
	}
	if err := analyzer.Observe(steam.Sample{TemperatureC: 134, NCGPercent: 10.5, ObservedAt: now}); err != nil {
		t.Fatal(err)
	}
	proof := analyzer.Evaluate(3.5, 134, now)
	if proof.AverageNCG >= 3.5 {
		t.Fatalf("test setup average %.2f is not below limit", proof.AverageNCG)
	}
	if proof.InstantValid || proof.Valid() {
		t.Fatalf("steam spike %.2f was accepted with average %.2f", proof.PeakNCG, proof.AverageNCG)
	}
}
