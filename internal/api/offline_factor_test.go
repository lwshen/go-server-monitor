package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"
)

// TestOfflineFactorFromSettings proves the offline threshold is read from the
// settings table (admin-editable) rather than only config: a server last seen
// 200s ago is online at factor 5 (threshold 300s) but flips offline once
// offline_factor is set to 1 (threshold 60s) via the admin settings API.
func TestOfflineFactorFromSettings(t *testing.T) {
	h := newTestRouter(t) // Cfg.OfflineFactor = 5, empty JWT (admin passthrough)
	id := register(t, h, "host")

	seen := time.Now().Unix() - 200
	report := fmt.Sprintf(`{"id":%q,"secret":"s3cret","timestamp":%d,"data":{"cpu":1,"latest_ts":%d}}`, id, seen, seen)
	if code, body := do(t, h, http.MethodPost, "/report", report); code != 200 {
		t.Fatalf("report = %d %s", code, body)
	}

	online := func() bool {
		_, body := do(t, h, http.MethodGet, "/api/servers", "")
		var resp struct {
			Servers []struct {
				Online bool `json:"online"`
			} `json:"servers"`
		}
		if err := json.Unmarshal([]byte(body), &resp); err != nil || len(resp.Servers) != 1 {
			t.Fatalf("servers body = %s (err %v)", body, err)
		}
		return resp.Servers[0].Online
	}

	// Default factor 5 -> threshold 300s -> 200s ago is online.
	if !online() {
		t.Fatal("want online at factor 5")
	}

	// Tighten via settings -> factor 1 -> threshold 60s -> now offline.
	if code, body := do(t, h, http.MethodPost, "/api/admin/settings", `{"offline_factor":"1"}`); code != 200 {
		t.Fatalf("set offline_factor = %d %s", code, body)
	}
	if online() {
		t.Fatal("want offline after offline_factor=1")
	}
}
