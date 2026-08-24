package verifycase

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/wyw14/cry-112/internal/api"
	"github.com/wyw14/cry-112/internal/cycle"
)

func TestConcurrentChambersShareDrainCapacity(t *testing.T) {
	controller, err := cycle.NewController(t.TempDir(), time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(api.NewServer(controller).Handler())
	defer server.Close()
	client := server.Client()
	start := make(chan struct{})
	results := make(chan cycle.DrainRequestResult, 2)
	errors := make(chan error, 2)
	var group sync.WaitGroup
	for _, chamberID := range []string{"A", "B"} {
		group.Add(1)
		go func(id string) {
			defer group.Done()
			<-start
			encoded, err := json.Marshal(map[string]any{"requested_flow": 80, "priority": 10})
			if err != nil {
				errors <- err
				return
			}
			response, err := client.Post(server.URL+"/api/chambers/"+id+"/drain", "application/json", bytes.NewReader(encoded))
			if err != nil {
				errors <- err
				return
			}
			defer response.Body.Close()
			var result cycle.DrainRequestResult
			if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
				errors <- err
				return
			}
			results <- result
		}(chamberID)
	}
	close(start)
	group.Wait()
	close(results)
	close(errors)
	for err := range errors {
		if err != nil {
			t.Fatal(err)
		}
	}
	count := 0
	for result := range results {
		count++
		if result.Reservation.Allocated < 0 || result.Reservation.Allocated > 80 {
			t.Fatalf("invalid chamber allocation %.2f", result.Reservation.Allocated)
		}
	}
	if count != 2 {
		t.Fatalf("received %d drain results", count)
	}
	diagnostics, err := controller.Diagnostics()
	if err != nil {
		t.Fatal(err)
	}
	if diagnostics.Drain.Committed > diagnostics.Drain.Capacity {
		t.Fatalf("committed %.2f exceeds capacity %.2f", diagnostics.Drain.Committed, diagnostics.Drain.Capacity)
	}
	var total float64
	for _, reservation := range diagnostics.Drain.Reservations {
		total += reservation.Allocated
	}
	if total > diagnostics.Drain.Capacity || total != diagnostics.Drain.Committed {
		t.Fatalf("reservation total %.2f is inconsistent with committed %.2f", total, diagnostics.Drain.Committed)
	}
}
