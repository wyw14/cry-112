package steam

import (
	"fmt"
	"sync"
	"time"
)

type Sample struct {
	TemperatureC float64   `json:"temperature_c"`
	NCGPercent   float64   `json:"ncg_percent"`
	ObservedAt   time.Time `json:"observed_at"`
}

type QualityProof struct {
	Latest           Sample    `json:"latest"`
	AverageNCG       float64   `json:"average_ncg"`
	PeakNCG          float64   `json:"peak_ncg"`
	SampleCount      int       `json:"sample_count"`
	InstantValid     bool      `json:"instant_valid"`
	TrendValid       bool      `json:"trend_valid"`
	TemperatureValid bool      `json:"temperature_valid"`
	EvaluatedAt      time.Time `json:"evaluated_at"`
}

func (p QualityProof) Valid() bool {
	return p.InstantValid && p.TrendValid && p.TemperatureValid && p.SampleCount > 0
}

type Analyzer struct {
	mu      sync.RWMutex
	window  time.Duration
	samples []Sample
}

func NewAnalyzer(window time.Duration) *Analyzer {
	return &Analyzer{window: window, samples: make([]Sample, 0, 64)}
}

func (a *Analyzer) Observe(sample Sample) error {
	if sample.TemperatureC < 0 || sample.NCGPercent < 0 || sample.ObservedAt.IsZero() {
		return fmt.Errorf("steam sample is outside physical range")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.samples = append(a.samples, sample)
	cutoff := sample.ObservedAt.Add(-a.window)
	first := 0
	for first < len(a.samples) && a.samples[first].ObservedAt.Before(cutoff) {
		first++
	}
	if first > 0 {
		a.samples = append([]Sample(nil), a.samples[first:]...)
	}
	return nil
}

func (a *Analyzer) Evaluate(maximumNCG, minimumTemperature float64, now time.Time) QualityProof {
	a.mu.RLock()
	defer a.mu.RUnlock()
	proof := QualityProof{EvaluatedAt: now.UTC(), InstantValid: true, TrendValid: true, TemperatureValid: true}
	if len(a.samples) == 0 {
		proof.InstantValid = false
		proof.TrendValid = false
		proof.TemperatureValid = false
		return proof
	}
	var total float64
	for _, sample := range a.samples {
		total += sample.NCGPercent
		if sample.NCGPercent > proof.PeakNCG {
			proof.PeakNCG = sample.NCGPercent
		}
		if sample.NCGPercent > maximumNCG {
			proof.InstantValid = false
		}
		if sample.TemperatureC < minimumTemperature {
			proof.TemperatureValid = false
		}
	}
	proof.Latest = a.samples[len(a.samples)-1]
	proof.SampleCount = len(a.samples)
	proof.AverageNCG = total / float64(len(a.samples))
	// The instantaneous and trend checks must stay independent: a single sample
	// over the limit is a dangerous event even when the rolling average passes,
	// so the trend result must not overwrite the per-sample instant result.
	proof.TrendValid = proof.AverageNCG <= maximumNCG
	return proof
}

func (a *Analyzer) Reset() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.samples = a.samples[:0]
}
