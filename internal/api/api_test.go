package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"go.uber.org/zap"

	"github.com/lwshen/go-server-monitor/internal/config"
	"github.com/lwshen/go-server-monitor/internal/store"
	"github.com/lwshen/go-server-monitor/internal/ws"
)

// newTestRouter wires the real router over a temp SQLite store.
func newTestRouter(t *testing.T) http.Handler {
	t.Helper()
	log := zap.NewNop()
	st, err := store.Open(context.Background(), &config.Config{DBPath: filepath.Join(t.TempDir(), "m.db")}, log)
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if err := st.Migrate(context.Background()); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	return NewRouter(Deps{
		Cfg:   &config.Config{APISecret: "s3cret", OfflineFactor: 5},
		Store: st,
		Hub:   ws.NewHub(log),
		Log:   log,
	})
}

func do(t *testing.T, h http.Handler, method, path, body string) (int, string) {
	t.Helper()
	var r *http.Request
	if body != "" {
		r = httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w.Code, w.Body.String()
}

// register creates a server via the admin add endpoint and returns its id.
func register(t *testing.T, h http.Handler, name string) string {
	t.Helper()
	code, body := do(t, h, http.MethodPost, "/api/admin/servers/add", fmt.Sprintf(`{"name":%q}`, name))
	if code != 200 {
		t.Fatalf("POST /api/admin/servers/add = %d %s", code, body)
	}
	var out struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(body), &out); err != nil || out.ID == "" {
		t.Fatalf("add response = %s (err %v), want an id", body, err)
	}
	return out.ID
}

// TestReportToQueryRoundtrip is the P2 end-to-end loop over HTTP: register a
// server, upload via /report, then read it back through the query APIs.
func TestReportToQueryRoundtrip(t *testing.T) {
	h := newTestRouter(t)
	id := register(t, h, "prod-01")

	report := fmt.Sprintf(`{"id":%q,"secret":"s3cret","timestamp":2000000000,
		"data":{"name":"prod-01","cpu":25.5,"memory_total":8192,"memory_used":2048,
		"ping_ct":32,"ping_cu":-1,"loss_ct":0,"loss_cu":-1,"alias":"主力",
		"sys_info":{"host_name":"h1","os_name":"Linux"},"ip_info":{"country":"US"},"disks":[]}}`, id)

	if code, body := do(t, h, http.MethodPost, "/report", report); code != 200 || !strings.Contains(body, `"saved":1`) {
		t.Fatalf("POST /report = %d %s, want 200 saved:1", code, body)
	}

	// /api/servers
	code, body := do(t, h, http.MethodGet, "/api/servers", "")
	if code != 200 {
		t.Fatalf("GET /api/servers = %d %s", code, body)
	}
	var list struct {
		Servers []struct {
			ID            string `json:"id"`
			Alias         string `json:"alias"`
			Online        bool   `json:"online"`
			LatestMetrics *struct {
				Cpu    float64  `json:"cpu"`
				PingCt *float64 `json:"ping_ct"`
				PingCu *float64 `json:"ping_cu"`
			} `json:"latest_metrics"`
		} `json:"servers"`
		Stats struct{ Total, Online int } `json:"stats"`
	}
	if err := json.Unmarshal([]byte(body), &list); err != nil {
		t.Fatalf("unmarshal servers: %v", err)
	}
	if list.Stats.Total != 1 || len(list.Servers) != 1 {
		t.Fatalf("servers total = %d / len %d, want 1", list.Stats.Total, len(list.Servers))
	}
	s := list.Servers[0]
	if s.ID != id || s.Alias != "主力" || s.LatestMetrics == nil || s.LatestMetrics.Cpu != 25.5 {
		t.Fatalf("server[0] = %+v, want id=%s/主力/cpu25.5", s, id)
	}
	if s.LatestMetrics.PingCt == nil || *s.LatestMetrics.PingCt != 32 || s.LatestMetrics.PingCu != nil {
		t.Fatalf("ping_ct/ping_cu = %v/%v, want 32/nil", s.LatestMetrics.PingCt, s.LatestMetrics.PingCu)
	}

	// /api/server?id=
	code, body = do(t, h, http.MethodGet, "/api/server?id="+id, "")
	if code != 200 || !strings.Contains(body, `"host_name":"h1"`) || !strings.Contains(body, `"country":"US"`) {
		t.Fatalf("GET /api/server = %d %s, want sys_info+ip_info", code, body)
	}
	if code, _ := do(t, h, http.MethodGet, "/api/server?id=nope", ""); code != 404 {
		t.Fatalf("GET /api/server?id=nope = %d, want 404", code)
	}

	// /api/history
	if code, body := do(t, h, http.MethodGet, "/api/history?id="+id+"&range=24h", ""); code != 200 || !strings.Contains(body, `"samples"`) {
		t.Fatalf("GET /api/history = %d %s", code, body)
	}
	if code, _ := do(t, h, http.MethodGet, "/api/history?id="+id+"&range=bogus", ""); code != 400 {
		t.Fatalf("GET /api/history bogus range = %d, want 400", code)
	}
}

func TestReportAuth(t *testing.T) {
	h := newTestRouter(t)
	if code, _ := do(t, h, http.MethodPost, "/report", `{"id":"x","secret":"WRONG","data":{"name":"y"}}`); code != 401 {
		t.Fatalf("wrong secret = %d, want 401", code)
	}
	if code, _ := do(t, h, http.MethodPost, "/report", `{"secret":"s3cret","data":{"name":"y"}}`); code != 400 {
		t.Fatalf("missing id = %d, want 400", code)
	}
	if code, _ := do(t, h, http.MethodPost, "/report", `{not json`); code != 400 {
		t.Fatalf("bad json = %d, want 400", code)
	}
	// valid secret, but the server id was never registered -> 404 (no auto-create).
	if code, _ := do(t, h, http.MethodPost, "/report", `{"id":"ghost","secret":"s3cret","data":{"name":"y"}}`); code != 404 {
		t.Fatalf("unknown id = %d, want 404", code)
	}
}
