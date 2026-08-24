package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/wyw14/cry-112/internal/cycle"
)

type Server struct {
	controller *cycle.Controller
	router     chi.Router
	now        func() time.Time
}

func NewServer(controller *cycle.Controller) *Server {
	server := &Server{controller: controller, router: chi.NewRouter(), now: time.Now}
	server.routes()
	return server
}

func (s *Server) routes() {
	s.router.Use(middleware.RequestID)
	s.router.Use(middleware.RealIP)
	s.router.Use(middleware.Recoverer)
	s.router.Get("/healthz", s.health)
	s.router.Route("/api", func(router chi.Router) {
		router.Get("/cycles", s.listCycles)
		router.Post("/cycles", s.createCycle)
		router.Get("/cycles/{cycleID}", s.getCycle)
		router.Post("/cycles/{cycleID}/vacuum-pulses", s.recordVacuumPulse)
		router.Post("/cycles/{cycleID}/vacuum-retries", s.retryVacuum)
		router.Get("/cycles/{cycleID}/vacuum-retries", s.listVacuumRetries)
		router.Post("/cycles/{cycleID}/probes", s.assignProbe)
		router.Post("/cycles/{cycleID}/probe-readings", s.observeProbe)
		router.Get("/cycles/{cycleID}/coldspot", s.getColdspot)
		router.Post("/cycles/{cycleID}/exposure", s.applyExposure)
		router.Post("/cycles/{cycleID}/exhaust", s.applyExhaust)
		router.Post("/cycles/{cycleID}/drying", s.applyDrying)
		router.Post("/cycles/{cycleID}/cooling", s.applyCooling)
		router.Post("/cycles/{cycleID}/release", s.releaseCycle)
		router.Get("/chambers", s.listChambers)
		router.Patch("/chambers/{chamberID}", s.updateChamber)
		router.Post("/chambers/{chamberID}/drain", s.requestDrain)
		router.Post("/chambers/{chamberID}/filter-integrity", s.recordFilterIntegrity)
		router.Get("/steam", s.getSteam)
		router.Get("/doors", s.listDoors)
		router.Patch("/doors/{doorID}", s.updateDoor)
		router.Post("/doors/{doorID}/release-check", s.checkDoorRelease)
		router.Get("/incidents", s.listIncidents)
	})
}

func (s *Server) Handler() http.Handler {
	return s.router
}

func (s *Server) health(writer http.ResponseWriter, request *http.Request) {
	health, err := s.controller.Health()
	if err != nil {
		writeError(writer, http.StatusServiceUnavailable, err)
		return
	}
	diagnostics, err := s.controller.Diagnostics()
	if err != nil {
		writeError(writer, http.StatusServiceUnavailable, err)
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{"status": "ok", "journal": health, "diagnostics": diagnostics, "time": s.now().UTC()})
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}

func writeError(writer http.ResponseWriter, status int, err error) {
	writeJSON(writer, status, map[string]any{"error": err.Error()})
}

func decodeJSON(request *http.Request, target any) error {
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode request: %w", err)
	}
	return nil
}

func cycleID(request *http.Request) (uuid.UUID, error) {
	value := chi.URLParam(request, "cycleID")
	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid cycle identity")
	}
	return id, nil
}
