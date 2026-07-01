package api

import (
	"context"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"go.uber.org/zap/zaptest"

	"github.com/lwshen/go-server-monitor/internal/config"
	"github.com/lwshen/go-server-monitor/internal/store"
	"github.com/lwshen/go-server-monitor/internal/ws"
)

// spaRouter builds a router with an in-memory SPA filesystem (mimicking the
// embedded web/dist) so the NoRoute static-serving logic can be tested without
// the `embed` build tag.
func spaRouter(t *testing.T) http.Handler {
	t.Helper()
	log := zaptest.NewLogger(t)
	st, err := store.Open(context.Background(), &config.Config{DBPath: filepath.Join(t.TempDir(), "m.db")}, log)
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if err := st.Migrate(context.Background()); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	spa := fstest.MapFS{
		"index.html":    {Data: []byte("<!doctype html><div id=app>SPA</div>")},
		"assets/app.js": {Data: []byte("console.log('app')")},
	}
	return NewRouter(Deps{
		Cfg:   &config.Config{APISecret: "s3cret", OfflineFactor: 5},
		Store: st,
		Hub:   ws.NewHub(log),
		Log:   log,
		SPA:   spa,
	})
}

func TestSPAServing(t *testing.T) {
	h := spaRouter(t)

	// root -> index.html
	if c, b := do(t, h, http.MethodGet, "/", ""); c != 200 || !strings.Contains(b, "id=app") {
		t.Fatalf("GET / = %d %q, want index.html", c, b)
	}
	// real static asset served directly
	if c, b := do(t, h, http.MethodGet, "/assets/app.js", ""); c != 200 || !strings.Contains(b, "console.log") {
		t.Fatalf("GET /assets/app.js = %d %q, want the asset", c, b)
	}
	// client-side route (no matching file) -> index.html fallback
	if c, b := do(t, h, http.MethodGet, "/server/123", ""); c != 200 || !strings.Contains(b, "id=app") {
		t.Fatalf("GET /server/123 = %d %q, want index.html fallback", c, b)
	}
	// unmatched API path -> JSON 404, NOT the SPA
	if c, b := do(t, h, http.MethodGet, "/api/nope", ""); c != 404 || strings.Contains(b, "id=app") {
		t.Fatalf("GET /api/nope = %d %q, want 404 json", c, b)
	}
	// real endpoints still take precedence over the SPA fallback
	if c, _ := do(t, h, http.MethodGet, "/api/config", ""); c != 200 {
		t.Fatalf("GET /api/config = %d, want 200", c)
	}
	if c, _ := do(t, h, http.MethodGet, "/health", ""); c != 200 {
		t.Fatalf("GET /health = %d, want 200", c)
	}
}
