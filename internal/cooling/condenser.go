package cooling

import (
	"fmt"
	"sync"
	"time"
)

type CondenserState struct {
	TemperatureC float64   `json:"temperature_c"`
	WaterFlow    float64   `json:"water_flow"`
	Ready        bool      `json:"ready"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Condenser struct {
	mu                 sync.RWMutex
	maximumTemperature float64
	minimumWaterFlow   float64
	state              CondenserState
}

func NewCondenser(maximumTemperature, minimumWaterFlow float64, now time.Time) *Condenser {
	return &Condenser{maximumTemperature: maximumTemperature, minimumWaterFlow: minimumWaterFlow, state: CondenserState{TemperatureC: 22, WaterFlow: minimumWaterFlow, Ready: true, UpdatedAt: now.UTC()}}
}

func (c *Condenser) Observe(temperature, waterFlow float64, now time.Time) (CondenserState, error) {
	if temperature < -20 || waterFlow < 0 {
		return CondenserState{}, fmt.Errorf("invalid condenser telemetry")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state = CondenserState{TemperatureC: temperature, WaterFlow: waterFlow, Ready: temperature <= c.maximumTemperature && waterFlow >= c.minimumWaterFlow, UpdatedAt: now.UTC()}
	return c.state, nil
}

func (c *Condenser) State() CondenserState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state
}
