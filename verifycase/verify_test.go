package verifycase

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/wyw14/cry-112/internal/api"
	"github.com/wyw14/cry-112/internal/cycle"
)

func invoke(t *testing.T, client *http.Client, method, url string, body any, target any) {
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
		t.Fatalf("unexpected status %d", response.StatusCode)
	}
	if target != nil {
		if err := json.NewDecoder(response.Body).Decode(target); err != nil {
			t.Fatal(err)
		}
	}
}

func TestPassThroughReleaseRequiresOppositeDoorPhysicalProof(t *testing.T) {
	now := time.Now().UTC()
	controller, err := cycle.NewController(t.TempDir(), now)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(api.NewServer(controller).Handler())
	defer server.Close()
	client := server.Client()
	invoke(t, client, http.MethodPatch, server.URL+"/api/chambers/A", map[string]any{"pressure_kpa": 101.3, "temperature_c": 40, "jacket_temperature_c": 40, "drain_backpressure": 0}, nil)
	invoke(t, client, http.MethodPatch, server.URL+"/api/doors/A-unload", map[string]any{"desired_closed": true, "physical_closed": true, "locked": true, "seal_pressure_bar": 0}, nil)
	invoke(t, client, http.MethodPatch, server.URL+"/api/doors/A-load", map[string]any{"desired_closed": true, "physical_closed": false, "locked": false, "seal_pressure_bar": 0}, nil)
	var result cycle.DoorPermitResult
	invoke(t, client, http.MethodPost, server.URL+"/api/doors/A-unload/release-check", map[string]any{"chamber_id": "A", "peer_door": "A-load"}, &result)
	if !result.Door.Unlock {
		t.Fatal("requested door setup should pass its own chamber release proof")
	}
	if result.Interlock.PeerProof.Valid() || result.Interlock.Permit || result.Permit {
		t.Fatalf("release accepted desired closed without physical proof: %#v", result)
	}
}
