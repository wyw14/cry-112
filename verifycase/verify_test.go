package verifycase

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/wyw14/cry-112/internal/api"
	"github.com/wyw14/cry-112/internal/cycle"
	"github.com/wyw14/cry-112/internal/model"
	"github.com/wyw14/cry-112/internal/temperature"
)

func call(t *testing.T, client *http.Client, method, url string, body any, target any) {
	t.Helper()
	var data []byte
	var err error
	if body != nil {
		data, err = json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
	}
	request, err := http.NewRequest(method, url, bytes.NewReader(data))
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
		t.Fatalf("unexpected status %d", response.StatusCode)
	}
	if target != nil {
		if err := json.NewDecoder(response.Body).Decode(target); err != nil {
			t.Fatal(err)
		}
	}
}

func createCycle(t *testing.T, client *http.Client, serverURL string) model.Cycle {
	t.Helper()
	var created model.Cycle
	call(t, client, http.MethodPost, serverURL+"/api/cycles", map[string]any{"chamber_id": "A", "description": "probe mapping load"}, &created)
	return created
}

func TestProbeRemapStartsFreshColdspotHistory(t *testing.T) {
	now := time.Now().UTC()
	controller, err := cycle.NewController(t.TempDir(), now)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(api.NewServer(controller).Handler())
	defer server.Close()
	client := server.Client()
	probeID := uuid.New()
	first := createCycle(t, client, server.URL)
	call(t, client, http.MethodPost, server.URL+"/api/cycles/"+first.ID.String()+"/probes", map[string]any{"probe_id": probeID, "position": "upper-rack", "required": true}, nil)
	for _, observedAt := range []time.Time{now.Add(-31 * time.Second), now} {
		call(t, client, http.MethodPost, server.URL+"/api/cycles/"+first.ID.String()+"/probe-readings", map[string]any{"probe_id": probeID, "temperature_c": 134, "moisture": 2, "observed_at": observedAt}, nil)
	}
	second := createCycle(t, client, server.URL)
	call(t, client, http.MethodPost, server.URL+"/api/cycles/"+second.ID.String()+"/probes", map[string]any{"probe_id": probeID, "position": "dense-pack-center", "required": true}, nil)
	call(t, client, http.MethodPost, server.URL+"/api/cycles/"+second.ID.String()+"/probe-readings", map[string]any{"probe_id": probeID, "temperature_c": 134, "moisture": 2, "observed_at": now}, nil)
	var proof temperature.ColdspotProof
	call(t, client, http.MethodGet, server.URL+"/api/cycles/"+second.ID.String()+"/coldspot", nil, &proof)
	if proof.Valid() || proof.Ready != 0 {
		t.Fatalf("remapped probe inherited old location history: %#v", proof)
	}
}
