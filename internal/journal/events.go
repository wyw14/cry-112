package journal

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/wyw14/cry-112/internal/model"
)

func appendEvent(path string, event model.Event) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open event journal: %w", err)
	}
	encoded, err := json.Marshal(event)
	if err != nil {
		file.Close()
		return fmt.Errorf("encode journal event: %w", err)
	}
	encoded = append(encoded, '\n')
	if _, err := file.Write(encoded); err != nil {
		file.Close()
		return fmt.Errorf("append journal event: %w", err)
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return fmt.Errorf("sync event journal: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close event journal: %w", err)
	}
	return nil
}

func readEvents(path string) ([]model.Event, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []model.Event{}, nil
		}
		return nil, fmt.Errorf("open event journal: %w", err)
	}
	defer file.Close()
	reader := bufio.NewReader(file)
	events := make([]model.Event, 0)
	for {
		line, readErr := reader.ReadBytes('\n')
		if len(line) > 0 {
			var event model.Event
			if err := json.Unmarshal(line, &event); err != nil {
				return nil, fmt.Errorf("decode journal event %d: %w", len(events)+1, err)
			}
			events = append(events, event)
		}
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			return nil, fmt.Errorf("read event journal: %w", readErr)
		}
	}
	return events, nil
}
