package quality

import (
	"time"

	"github.com/wyw14/cry-112/internal/drying"
)

type LoadMoisture struct {
	maximumAge time.Duration
}

func NewLoadMoisture(maximumAge time.Duration) LoadMoisture {
	return LoadMoisture{maximumAge: maximumAge}
}

func (l LoadMoisture) Evaluate(proof drying.MoistureProof, now time.Time) bool {
	if !proof.Valid() {
		return false
	}
	for _, reading := range proof.Readings {
		if now.Sub(reading.ObservedAt) > l.maximumAge || reading.Value > proof.Maximum {
			return false
		}
	}
	return true
}
