package quality

import (
	"time"

	"github.com/wyw14/cry-112/internal/steam"
)

type SteamWindow struct {
	maximumAge time.Duration
}

func NewSteamWindow(maximumAge time.Duration) SteamWindow {
	return SteamWindow{maximumAge: maximumAge}
}

func (w SteamWindow) Valid(proof steam.QualityProof, now time.Time) bool {
	if !proof.Valid() || proof.Latest.ObservedAt.IsZero() {
		return false
	}
	return now.Sub(proof.Latest.ObservedAt) <= w.maximumAge
}

func (w SteamWindow) Reason(proof steam.QualityProof, now time.Time) string {
	if proof.SampleCount == 0 {
		return "steam quality has no samples"
	}
	if !proof.InstantValid {
		return "instantaneous noncondensable gas limit exceeded"
	}
	if !proof.TrendValid {
		return "noncondensable gas trend limit exceeded"
	}
	if !proof.TemperatureValid {
		return "steam temperature is below exposure requirement"
	}
	if now.Sub(proof.Latest.ObservedAt) > w.maximumAge {
		return "steam quality proof is stale"
	}
	return "steam quality is valid"
}
