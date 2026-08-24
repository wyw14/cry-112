package journal

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/wyw14/cry-112/internal/model"
)

func writeSnapshot(path string, snapshot model.Snapshot) error {
	if snapshot.Cycles == nil || snapshot.Chambers == nil || snapshot.Doors == nil {
		return fmt.Errorf("snapshot maps must be initialized")
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), "snapshot-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary snapshot: %w", err)
	}
	temporaryPath := temporary.Name()
	cleanup := func() {
		temporary.Close()
		os.Remove(temporaryPath)
	}
	encoder := json.NewEncoder(temporary)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(snapshot); err != nil {
		cleanup()
		return fmt.Errorf("encode snapshot: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		cleanup()
		return fmt.Errorf("sync snapshot: %w", err)
	}
	if err := temporary.Close(); err != nil {
		os.Remove(temporaryPath)
		return fmt.Errorf("close snapshot: %w", err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		os.Remove(temporaryPath)
		return fmt.Errorf("replace snapshot: %w", err)
	}
	return nil
}

func readSnapshot(path string, now time.Time) (model.Snapshot, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return model.EmptySnapshot(now), nil
		}
		return model.Snapshot{}, fmt.Errorf("open snapshot: %w", err)
	}
	defer file.Close()
	var snapshot model.Snapshot
	if err := json.NewDecoder(file).Decode(&snapshot); err != nil {
		return model.Snapshot{}, fmt.Errorf("decode snapshot: %w", err)
	}
	if snapshot.Cycles == nil {
		snapshot.Cycles = make(map[uuid.UUID]model.Cycle)
	}
	if snapshot.Chambers == nil {
		snapshot.Chambers = make(map[string]model.ChamberState)
	}
	if snapshot.Doors == nil {
		snapshot.Doors = make(map[string]model.DoorState)
	}
	if snapshot.Incidents == nil {
		snapshot.Incidents = make([]model.Incident, 0)
	}
	return snapshot, nil
}
