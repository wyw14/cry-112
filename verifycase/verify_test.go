package verifycase

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/wyw14/cry-112/internal/api"
	"github.com/wyw14/cry-112/internal/cycle"
	"github.com/wyw14/cry-112/internal/model"
	"github.com/wyw14/cry-112/internal/vacuum"
)

func sendJSON(t *testing.T, client *http.Client, method, url string, body any, target any) {
	t.Helper()
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequest(method, url, bytes.NewReader(encoded))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		data, _ := io.ReadAll(response.Body)
		t.Fatalf("unexpected status %d: %s", response.StatusCode, data)
	}
	if target != nil {
		if err := json.NewDecoder(response.Body).Decode(target); err != nil {
			t.Fatal(err)
		}
	}
}

func TestAirRemovalRequiresVacuumDepthAndReboundProof(t *testing.T) {
	now := time.Now().UTC()
	controller, err := cycle.NewController(t.TempDir(), now)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(api.NewServer(controller).Handler())
	defer server.Close()
	client := server.Client()
	sendJSON(t, client, http.MethodPatch, server.URL+"/api/chambers/A", map[string]any{"pressure_kpa": 101.3, "temperature_c": 22, "jacket_temperature_c": 130, "drain_backpressure": 0}, nil)
	var created model.Cycle
	sendJSON(t, client, http.MethodPost, server.URL+"/api/cycles", map[string]any{"chamber_id": "A", "description": "Bowie-Dick validation load"}, &created)
	for sequence := 1; sequence <= 3; sequence++ {
		end := 10.5
		if sequence == 3 {
			end = 13
		}
		body := cycle.VacuumPulseInput{Sequence: sequence, MinimumKPa: 10, IsolationStart: 10, IsolationEnd: end, StartedAt: now.Add(time.Duration(sequence) * time.Minute), CompletedAt: now.Add(time.Duration(sequence)*time.Minute + 30*time.Second)}
		var result struct {
			Cycle model.Cycle            `json:"cycle"`
			Proof vacuum.AirRemovalProof `json:"proof"`
		}
		sendJSON(t, client, http.MethodPost, server.URL+"/api/cycles/"+created.ID.String()+"/vacuum-pulses", body, &result)
		if sequence == 3 {
			if result.Proof.ReboundValid {
				t.Fatal("rebound proof unexpectedly valid")
			}
			if result.Cycle.Phase != model.PhaseAirRemoval {
				t.Fatalf("cycle advanced to %s without rebound proof", result.Cycle.Phase)
			}
		}
	}
}
