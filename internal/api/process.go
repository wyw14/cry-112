package api

import (
	"net/http"
)

type dryingRequest struct {
	JacketHumidity float64 `json:"jacket_humidity"`
	MaximumJacket  float64 `json:"maximum_jacket"`
}

func (s *Server) applyDrying(writer http.ResponseWriter, request *http.Request) {
	id, err := cycleID(request)
	if err != nil {
		writeError(writer, http.StatusBadRequest, err)
		return
	}
	var body dryingRequest
	if err := decodeJSON(request, &body); err != nil {
		writeError(writer, http.StatusBadRequest, err)
		return
	}
	result, err := s.controller.ApplyDrying(id, body.JacketHumidity, body.MaximumJacket, s.now())
	if err != nil {
		writeError(writer, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(writer, http.StatusOK, result)
}

type coolingRequest struct {
	PressureKPa float64 `json:"pressure_kpa"`
	Rate        float64 `json:"rate"`
}

func (s *Server) applyCooling(writer http.ResponseWriter, request *http.Request) {
	id, err := cycleID(request)
	if err != nil {
		writeError(writer, http.StatusBadRequest, err)
		return
	}
	var body coolingRequest
	if err := decodeJSON(request, &body); err != nil {
		writeError(writer, http.StatusBadRequest, err)
		return
	}
	result, err := s.controller.ApplyCooling(id, body.PressureKPa, body.Rate, s.now())
	if err != nil {
		writeError(writer, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(writer, http.StatusOK, result)
}

func (s *Server) releaseCycle(writer http.ResponseWriter, request *http.Request) {
	id, err := cycleID(request)
	if err != nil {
		writeError(writer, http.StatusBadRequest, err)
		return
	}
	result, err := s.controller.Release(id, s.now())
	if err != nil {
		writeJSON(writer, http.StatusConflict, map[string]any{"error": err.Error(), "result": result})
		return
	}
	writeJSON(writer, http.StatusOK, result)
}
