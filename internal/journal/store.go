package journal

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/wyw14/cry-112/internal/model"
)

type Store struct {
	directory    string
	eventsPath   string
	snapshotPath string
	mu           sync.Mutex
}

type Health struct {
	Directory      string    `json:"directory"`
	EventBytes     int64     `json:"event_bytes"`
	SnapshotBytes  int64     `json:"snapshot_bytes"`
	LastModifiedAt time.Time `json:"last_modified_at"`
}

func NewStore(directory string) (*Store, error) {
	if directory == "" {
		return nil, fmt.Errorf("journal directory is required")
	}
	abs, err := filepath.Abs(directory)
	if err != nil {
		return nil, fmt.Errorf("resolve journal directory: %w", err)
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return nil, fmt.Errorf("create journal directory: %w", err)
	}
	return &Store{
		directory:    abs,
		eventsPath:   filepath.Join(abs, "events.jsonl"),
		snapshotPath: filepath.Join(abs, "snapshot.json"),
	}, nil
}

func (s *Store) Directory() string {
	return s.directory
}

func (s *Store) Append(event model.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return appendEvent(s.eventsPath, event)
}

func (s *Store) ReadEvents() ([]model.Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return readEvents(s.eventsPath)
}

func (s *Store) SaveSnapshot(snapshot model.Snapshot) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return writeSnapshot(s.snapshotPath, snapshot)
}

func (s *Store) LoadSnapshot(now time.Time) (model.Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return readSnapshot(s.snapshotPath, now)
}

func (s *Store) Health() (Health, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	health := Health{Directory: s.directory}
	paths := []string{s.eventsPath, s.snapshotPath}
	for index, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return Health{}, fmt.Errorf("stat journal file: %w", err)
		}
		if index == 0 {
			health.EventBytes = info.Size()
		} else {
			health.SnapshotBytes = info.Size()
		}
		if info.ModTime().After(health.LastModifiedAt) {
			health.LastModifiedAt = info.ModTime().UTC()
		}
	}
	return health, nil
}
