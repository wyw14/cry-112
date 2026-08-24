package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (s *Server) listChambers(writer http.ResponseWriter, request *http.Request) {
	writeJSON(writer, http.StatusOK, map[string]any{"chambers": s.controller.Chambers()})
}

type updateChamberRequest struct {
	PressureKPa       float64 `json:"pressure_kpa"`
	TemperatureC      float64 `json:"temperature_c"`
	JacketTemperature float64 `json:"jacket_temperature_c"`
	DrainBackpressure float64 `json:"drain_backpressure"`
}

func (s *Server) updateChamber(writer http.ResponseWriter, request *http.Request) {
	var body updateChamberRequest
	if err := decodeJSON(request, &body); err != nil {
		writeError(writer, http.StatusBadRequest, err)
		return
	}
	state, err := s.controller.UpdateChamber(chi.URLParam(request, "chamberID"), body.PressureKPa, body.TemperatureC, body.JacketTemperature, body.DrainBackpressure, s.now())
	if err != nil {
		writeError(writer, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(writer, http.StatusOK, state)
}

type filterIntegrityRequest struct {
	PressureDrop float64 `json:"pressure_drop"`
	LeakRate     float64 `json:"leak_rate"`
	Passed       bool    `json:"passed"`
}

func (s *Server) recordFilterIntegrity(writer http.ResponseWriter, request *http.Request) {
	var body filterIntegrityRequest
	if err := decodeJSON(request, &body); err != nil {
		writeError(writer, http.StatusBadRequest, err)
		return
	}
	proof, err := s.controller.RecordFilterIntegrity(chi.URLParam(request, "chamberID"), body.PressureDrop, body.LeakRate, body.Passed, s.now())
	if err != nil {
		writeError(writer, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(writer, http.StatusOK, proof)
}

type drainRequest struct {
	RequestedFlow float64 `json:"requested_flow"`
	Priority      int     `json:"priority"`
}

func (s *Server) requestDrain(writer http.ResponseWriter, request *http.Request) {
	var body drainRequest
	if err := decodeJSON(request, &body); err != nil {
		writeError(writer, http.StatusBadRequest, err)
		return
	}
	result, err := s.controller.RequestSharedDrain(chi.URLParam(request, "chamberID"), body.RequestedFlow, body.Priority, s.now())
	if err != nil {
		writeError(writer, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(writer, http.StatusOK, result)
}

func (s *Server) getSteam(writer http.ResponseWriter, request *http.Request) {
	writeJSON(writer, http.StatusOK, s.controller.Steam())
}

func (s *Server) listDoors(writer http.ResponseWriter, request *http.Request) {
	writeJSON(writer, http.StatusOK, map[string]any{"doors": s.controller.Doors()})
}

type updateDoorRequest struct {
	DesiredClosed  bool    `json:"desired_closed"`
	PhysicalClosed bool    `json:"physical_closed"`
	Locked         bool    `json:"locked"`
	SealPressure   float64 `json:"seal_pressure_bar"`
}

func (s *Server) updateDoor(writer http.ResponseWriter, request *http.Request) {
	var body updateDoorRequest
	if err := decodeJSON(request, &body); err != nil {
		writeError(writer, http.StatusBadRequest, err)
		return
	}
	state, err := s.controller.UpdateDoor(chi.URLParam(request, "doorID"), body.DesiredClosed, body.PhysicalClosed, body.Locked, body.SealPressure, s.now())
	if err != nil {
		writeError(writer, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(writer, http.StatusOK, state)
}

type doorReleaseRequest struct {
	ChamberID string `json:"chamber_id"`
	PeerDoor  string `json:"peer_door"`
}

func (s *Server) checkDoorRelease(writer http.ResponseWriter, request *http.Request) {
	var body doorReleaseRequest
	if err := decodeJSON(request, &body); err != nil {
		writeError(writer, http.StatusBadRequest, err)
		return
	}
	result, err := s.controller.EvaluateDoorPermit(body.ChamberID, chi.URLParam(request, "doorID"), body.PeerDoor, s.now())
	if err != nil {
		writeError(writer, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(writer, http.StatusOK, result)
}

func (s *Server) listIncidents(writer http.ResponseWriter, request *http.Request) {
	writeJSON(writer, http.StatusOK, map[string]any{"incidents": s.controller.Incidents()})
}
