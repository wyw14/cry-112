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

func request(t *testing.T, client *http.Client, method, url string, body any, target any) {
	t.Helper()
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(method, url, bytes.NewReader(encoded))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	response, err := client.Do(req)
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

func TestDoorUnlockRequiresSealPressureReleased(t *testing.T) {
	now := time.Now().UTC()
	controller, err := cycle.NewController(t.TempDir(), now)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(api.NewServer(controller).Handler())
	defer server.Close()
	client := server.Client()
	request(t, client, http.MethodPatch, server.URL+"/api/chambers/A", map[string]any{"pressure_kpa": 101.3, "temperature_c": 40, "jacket_temperature_c": 40, "drain_backpressure": 0}, nil)
	request(t, client, http.MethodPatch, server.URL+"/api/doors/A-unload", map[string]any{"desired_closed": true, "physical_closed": true, "locked": true, "seal_pressure_bar": 2.4}, nil)
	var result cycle.DoorPermitResult
	request(t, client, http.MethodPost, server.URL+"/api/doors/A-unload/release-check", map[string]any{"chamber_id": "A", "peer_door": "A-load"}, &result)
	if result.Door.Proof.SealReleased {
		t.Fatal("pressurized seal was reported released")
	}
	if result.Door.Unlock || result.Permit {
		t.Fatal("door release permitted while seal remained pressurized")
	}
}
