package api

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/wyw14/cry-112/internal/cycle"
	"github.com/wyw14/cry-112/internal/model"
)

type createCycleRequest struct {
	ChamberID   string        `json:"chamber_id"`
	Description string        `json:"description"`
	Recipe      *model.Recipe `json:"recipe,omitempty"`
}

func (s *Server) listCycles(writer http.ResponseWriter, request *http.Request) {
	writeJSON(writer, http.StatusOK, map[string]any{"cycles": s.controller.List()})
}

func (s *Server) createCycle(writer http.ResponseWriter, request *http.Request) {
	var body createCycleRequest
	if err := decodeJSON(request, &body); err != nil {
		writeError(writer, http.StatusBadRequest, err)
		return
	}
	recipe := model.DefaultRecipe()
	if body.Recipe != nil {
		recipe = body.Recipe.Clone()
	}
	created, err := s.controller.Create(body.ChamberID, body.Description, recipe, s.now())
	if err != nil {
		writeError(writer, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(writer, http.StatusCreated, created)
}

func (s *Server) getCycle(writer http.ResponseWriter, request *http.Request) {
	id, err := cycleID(request)
	if err != nil {
		writeError(writer, http.StatusBadRequest, err)
		return
	}
	value, ok := s.controller.Get(id)
	if !ok {
		writeError(writer, http.StatusNotFound, http.ErrMissingFile)
		return
	}
	writeJSON(writer, http.StatusOK, value)
}

func (s *Server) recordVacuumPulse(writer http.ResponseWriter, request *http.Request) {
	id, err := cycleID(request)
	if err != nil {
		writeError(writer, http.StatusBadRequest, err)
		return
	}
	var body cycle.VacuumPulseInput
	if err := decodeJSON(request, &body); err != nil {
		writeError(writer, http.StatusBadRequest, err)
		return
	}
	if body.StartedAt.IsZero() {
		body.StartedAt = s.now().Add(-30 * time.Second)
	}
	if body.CompletedAt.IsZero() {
		body.CompletedAt = s.now()
	}
	result, err := s.controller.RecordVacuumPulse(id, body, s.now())
	if err != nil {
		writeError(writer, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(writer, http.StatusOK, result)
}

type vacuumRetryRequest struct {
	CondenserTemperatureC float64 `json:"condenser_temperature_c"`
	WaterFlow             float64 `json:"water_flow"`
}

func (s *Server) retryVacuum(writer http.ResponseWriter, request *http.Request) {
	id, err := cycleID(request)
	if err != nil {
		writeError(writer, http.StatusBadRequest, err)
		return
	}
	var body vacuumRetryRequest
	if err := decodeJSON(request, &body); err != nil {
		writeError(writer, http.StatusBadRequest, err)
		return
	}
	record, err := s.controller.RetryVacuumAfterCavitation(id, body.CondenserTemperatureC, body.WaterFlow, s.now())
	if err != nil {
		writeJSON(writer, http.StatusConflict, map[string]any{"error": err.Error(), "retry": record})
		return
	}
	writeJSON(writer, http.StatusOK, record)
}

func (s *Server) listVacuumRetries(writer http.ResponseWriter, request *http.Request) {
	id, err := cycleID(request)
	if err != nil {
		writeError(writer, http.StatusBadRequest, err)
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{"retries": s.controller.VacuumRetryHistory(id)})
}

type assignProbeRequest struct {
	ProbeID  uuid.UUID `json:"probe_id"`
	Position string    `json:"position"`
	Required bool      `json:"required"`
}

func (s *Server) assignProbe(writer http.ResponseWriter, request *http.Request) {
	id, err := cycleID(request)
	if err != nil {
		writeError(writer, http.StatusBadRequest, err)
		return
	}
	var body assignProbeRequest
	if err := decodeJSON(request, &body); err != nil {
		writeError(writer, http.StatusBadRequest, err)
		return
	}
	if body.ProbeID == uuid.Nil {
		body.ProbeID = uuid.New()
	}
	placement, err := s.controller.AssignProbe(id, body.ProbeID, body.Position, body.Required, s.now())
	if err != nil {
		writeError(writer, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(writer, http.StatusOK, placement)
}

type observeProbeRequest struct {
	ProbeID      uuid.UUID `json:"probe_id"`
	TemperatureC float64   `json:"temperature_c"`
	Moisture     float64   `json:"moisture"`
	ObservedAt   time.Time `json:"observed_at"`
}

func (s *Server) observeProbe(writer http.ResponseWriter, request *http.Request) {
	id, err := cycleID(request)
	if err != nil {
		writeError(writer, http.StatusBadRequest, err)
		return
	}
	var body observeProbeRequest
	if err := decodeJSON(request, &body); err != nil {
		writeError(writer, http.StatusBadRequest, err)
		return
	}
	observedAt := body.ObservedAt
	if observedAt.IsZero() {
		observedAt = s.now()
	}
	reading, err := s.controller.ObserveProbe(id, body.ProbeID, body.TemperatureC, body.Moisture, observedAt)
	if err != nil {
		writeError(writer, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(writer, http.StatusOK, reading)
}

func (s *Server) getColdspot(writer http.ResponseWriter, request *http.Request) {
	id, err := cycleID(request)
	if err != nil {
		writeError(writer, http.StatusBadRequest, err)
		return
	}
	proof, err := s.controller.Coldspot(id, s.now())
	if err != nil {
		writeError(writer, http.StatusNotFound, err)
		return
	}
	writeJSON(writer, http.StatusOK, proof)
}

type exposureRequest struct {
	SteamTemperatureC float64       `json:"steam_temperature_c"`
	NCGPercent        float64       `json:"ncg_percent"`
	Elapsed           time.Duration `json:"elapsed"`
}

func (s *Server) applyExposure(writer http.ResponseWriter, request *http.Request) {
	id, err := cycleID(request)
	if err != nil {
		writeError(writer, http.StatusBadRequest, err)
		return
	}
	var body exposureRequest
	if err := decodeJSON(request, &body); err != nil {
		writeError(writer, http.StatusBadRequest, err)
		return
	}
	result, err := s.controller.ApplyExposure(id, body.SteamTemperatureC, body.NCGPercent, body.Elapsed, s.now())
	if err != nil {
		writeError(writer, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(writer, http.StatusOK, result)
}

type exhaustRequest struct {
	RequestedFlow float64 `json:"requested_flow"`
	MeasuredFlow  float64 `json:"measured_flow"`
	Backpressure  float64 `json:"backpressure"`
	Returned      bool    `json:"returned"`
}

func (s *Server) applyExhaust(writer http.ResponseWriter, request *http.Request) {
	id, err := cycleID(request)
	if err != nil {
		writeError(writer, http.StatusBadRequest, err)
		return
	}
	var body exhaustRequest
	if err := decodeJSON(request, &body); err != nil {
		writeError(writer, http.StatusBadRequest, err)
		return
	}
	result, err := s.controller.ApplyExhaust(id, body.RequestedFlow, body.MeasuredFlow, body.Backpressure, body.Returned, s.now())
	if err != nil {
		writeError(writer, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(writer, http.StatusOK, result)
}
