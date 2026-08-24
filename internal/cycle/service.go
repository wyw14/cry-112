package cycle

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/wyw14/cry-112/internal/chamber"
	"github.com/wyw14/cry-112/internal/condensate"
	"github.com/wyw14/cry-112/internal/cooling"
	"github.com/wyw14/cry-112/internal/door"
	"github.com/wyw14/cry-112/internal/drying"
	"github.com/wyw14/cry-112/internal/filter"
	"github.com/wyw14/cry-112/internal/interlock"
	"github.com/wyw14/cry-112/internal/journal"
	"github.com/wyw14/cry-112/internal/load"
	"github.com/wyw14/cry-112/internal/model"
	"github.com/wyw14/cry-112/internal/quality"
	"github.com/wyw14/cry-112/internal/steam"
	"github.com/wyw14/cry-112/internal/temperature"
	"github.com/wyw14/cry-112/internal/vacuum"
)

type Controller struct {
	mu          sync.Mutex
	store       *journal.Store
	registry    *Registry
	chambers    *chamber.Service
	vacuum      *vacuum.Service
	steam       *steam.Service
	temperature *temperature.Service
	loads       *load.Service
	probeMap    *load.ProbeMap
	condensate  *condensate.Service
	doors       *door.Service
	drying      *drying.Service
	cooling     *cooling.Service
	quality     *quality.Service
	interlocks  *interlock.Service
	incidents   []model.Incident
	filters     map[string]uuid.UUID
	version     int64
}

type Diagnostics struct {
	Phases           []model.Phase                 `json:"phases"`
	JournalDirectory string                        `json:"journal_directory"`
	JournalEvents    int                           `json:"journal_events"`
	Loads            []load.Batch                  `json:"loads"`
	Drain            condensate.AllocationSnapshot `json:"drain"`
	DrainValves      []chamber.DrainValveState     `json:"drain_valves"`
	Condenser        cooling.CondenserState        `json:"condenser"`
	FilterProofs     []filter.IntegrityProof       `json:"filter_proofs"`
}

func NewController(dataDirectory string, now time.Time) (*Controller, error) {
	store, err := journal.NewStore(dataDirectory)
	if err != nil {
		return nil, err
	}
	chamberIDs := []string{"A", "B"}
	condensateService, err := condensate.NewService(chamberIDs, 100, 80, now)
	if err != nil {
		return nil, err
	}
	coolingService, err := cooling.NewService(now)
	if err != nil {
		return nil, err
	}
	controller := &Controller{
		store:       store,
		registry:    NewRegistry(),
		chambers:    chamber.NewService(chamberIDs, now),
		vacuum:      vacuum.NewService(),
		steam:       steam.NewService(now),
		temperature: temperature.NewService(),
		loads:       load.NewService(),
		probeMap:    load.NewProbeMap(),
		condensate:  condensateService,
		doors:       door.NewService([]string{"A-load", "A-unload", "B-load", "B-unload"}, model.DefaultRecipe().MaximumReleaseTempC, now),
		drying:      drying.NewService(),
		cooling:     coolingService,
		quality:     quality.NewService(),
		interlocks:  interlock.NewService(),
		incidents:   make([]model.Incident, 0),
		filters:     map[string]uuid.UUID{"A": uuid.New(), "B": uuid.New()},
	}
	if err := controller.restore(now); err != nil {
		return nil, err
	}
	for _, filterID := range controller.filters {
		if _, err := controller.cooling.RecordFilter(filterID, 0.8, 0.01, true, now); err != nil {
			return nil, err
		}
	}
	return controller, nil
}

func (c *Controller) restore(now time.Time) error {
	snapshot, err := c.store.LoadSnapshot(now)
	if err != nil {
		return err
	}
	c.registry.Restore(snapshot.Cycles)
	c.chambers.Restore(snapshot.Chambers)
	c.doors.Restore(snapshot.Doors)
	c.steam.Restore(snapshot.Steam)
	c.incidents = append([]model.Incident(nil), snapshot.Incidents...)
	c.version = snapshot.Version
	return nil
}

func (c *Controller) Create(chamberID, description string, recipe model.Recipe, now time.Time) (model.Cycle, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := recipe.Validate(); err != nil {
		return model.Cycle{}, err
	}
	if _, ok := c.chambers.State(chamberID); !ok {
		return model.Cycle{}, fmt.Errorf("chamber %s not found", chamberID)
	}
	batch, err := c.loads.Create(description, now)
	if err != nil {
		return model.Cycle{}, err
	}
	cycle := model.NewCycle(chamberID, recipe, now)
	cycle.LoadID = batch.ID
	if err := c.registry.Create(cycle); err != nil {
		return model.Cycle{}, err
	}
	if _, err := c.registry.Transition(cycle.ID, model.PhasePreheating, now); err != nil {
		return model.Cycle{}, err
	}
	cycle, err = c.registry.Transition(cycle.ID, model.PhaseAirRemoval, now)
	if err != nil {
		return model.Cycle{}, err
	}
	if err := c.recordLocked(cycle.ID, "cycle-created", cycle, now); err != nil {
		return model.Cycle{}, err
	}
	return cycle, nil
}

func (c *Controller) Get(id uuid.UUID) (model.Cycle, bool) {
	return c.registry.Get(id)
}

func (c *Controller) List() []model.Cycle {
	return c.registry.List()
}

func (c *Controller) Chambers() []model.ChamberState {
	return c.chambers.List()
}

func (c *Controller) Doors() []model.DoorState {
	return c.doors.List()
}

func (c *Controller) Steam() model.SteamState {
	return c.steam.State()
}

func (c *Controller) Incidents() []model.Incident {
	c.mu.Lock()
	defer c.mu.Unlock()
	incidents := make([]model.Incident, len(c.incidents))
	for index, incident := range c.incidents {
		incidents[index] = incident.Clone()
	}
	return incidents
}

func (c *Controller) Health() (journal.Health, error) {
	return c.store.Health()
}

func (c *Controller) Diagnostics() (Diagnostics, error) {
	events, err := c.store.ReadEvents()
	if err != nil {
		return Diagnostics{}, err
	}
	return Diagnostics{
		Phases:           model.AllPhases(),
		JournalDirectory: c.store.Directory(),
		JournalEvents:    len(events),
		Loads:            c.loads.List(),
		Drain:            c.condensate.Snapshot(),
		DrainValves:      c.condensate.ValveStates(),
		Condenser:        c.cooling.Condenser(),
		FilterProofs:     c.cooling.FilterProofs(),
	}, nil
}

func (c *Controller) recordLocked(cycleID uuid.UUID, kind string, payload any, now time.Time) error {
	event, err := model.NewEvent(cycleID, kind, payload, now)
	if err != nil {
		return err
	}
	if err := c.store.Append(event); err != nil {
		return err
	}
	c.version++
	return c.persistLocked(now)
}

func (c *Controller) persistLocked(now time.Time) error {
	snapshot := model.Snapshot{
		Version:   c.version,
		Cycles:    c.registry.Snapshot(),
		Chambers:  c.chambers.Snapshot(),
		Doors:     c.doors.Snapshot(),
		Steam:     c.steam.State(),
		Incidents: append([]model.Incident(nil), c.incidents...),
		SavedAt:   now.UTC(),
	}
	return c.store.SaveSnapshot(snapshot)
}

func (c *Controller) addIncidentLocked(cycleID uuid.UUID, kind, message string, details map[string]any, now time.Time) model.Incident {
	incident := model.NewIncident(cycleID, kind, message, details, now)
	c.incidents = append(c.incidents, incident)
	return incident
}
