package model

import "fmt"

type Phase string

const (
	PhaseLoaded       Phase = "loaded"
	PhasePreheating   Phase = "preheating"
	PhaseAirRemoval   Phase = "air-removal"
	PhaseConditioning Phase = "conditioning"
	PhaseExposure     Phase = "exposure"
	PhaseExhaust      Phase = "exhaust"
	PhaseDrying       Phase = "drying"
	PhaseCooling      Phase = "cooling"
	PhaseReleased     Phase = "released"
	PhaseFailed       Phase = "failed"
)

var phaseTransitions = map[Phase][]Phase{
	PhaseLoaded:       {PhasePreheating, PhaseFailed},
	PhasePreheating:   {PhaseAirRemoval, PhaseFailed},
	PhaseAirRemoval:   {PhaseConditioning, PhaseFailed},
	PhaseConditioning: {PhaseExposure, PhaseFailed},
	PhaseExposure:     {PhaseExhaust, PhaseFailed},
	PhaseExhaust:      {PhaseDrying, PhaseFailed},
	PhaseDrying:       {PhaseCooling, PhaseFailed},
	PhaseCooling:      {PhaseReleased, PhaseFailed},
	PhaseReleased:     {},
	PhaseFailed:       {},
}

func (p Phase) Terminal() bool {
	return p == PhaseReleased || p == PhaseFailed
}

func (p Phase) CanTransition(next Phase) bool {
	for _, candidate := range phaseTransitions[p] {
		if candidate == next {
			return true
		}
	}
	return false
}

func (p Phase) ValidateTransition(next Phase) error {
	if p.CanTransition(next) {
		return nil
	}
	return fmt.Errorf("phase transition %s to %s is not permitted", p, next)
}

func AllPhases() []Phase {
	return []Phase{PhaseLoaded, PhasePreheating, PhaseAirRemoval, PhaseConditioning, PhaseExposure, PhaseExhaust, PhaseDrying, PhaseCooling, PhaseReleased, PhaseFailed}
}
