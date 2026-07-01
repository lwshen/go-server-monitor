package cron

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"go.uber.org/zap/zaptest"

	"github.com/lwshen/go-server-monitor/internal/config"
	"github.com/lwshen/go-server-monitor/internal/models"
	"github.com/lwshen/go-server-monitor/internal/service"
	"github.com/lwshen/go-server-monitor/internal/store"
)

// TestOfflineDetectionTransition drives the state machine: a server whose newest
// sample is stale flips online->offline exactly once, firing a webhook alert; a
// second run (no state change) fires nothing (REQ-CRON-05).
func TestOfflineDetectionTransition(t *testing.T) {
	ctx := context.Background()
	log := zaptest.NewLogger(t)

	st, err := store.Open(ctx, &config.Config{DBPath: filepath.Join(t.TempDir(), "m.db")}, log)
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if err := st.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	// Capture webhook alerts.
	recv := make(chan map[string]any, 8)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var m map[string]any
		_ = json.NewDecoder(r.Body).Decode(&m)
		recv <- m
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	_ = st.SetSetting(ctx, "notify_enabled", "true")
	_ = st.SetSetting(ctx, "webhook_url", srv.URL)

	// Register a server and report a STALE sample (older than 5×60s).
	if err := st.CreateServer(ctx, &models.ServerConfig{ID: "s1", Name: "n1"}); err != nil {
		t.Fatalf("CreateServer: %v", err)
	}
	stale := time.Now().Unix() - 1000
	if _, err := st.SaveReport(ctx, &models.ReportRequest{ID: "s1", Data: &models.StatReport{LatestTs: stale, Cpu: 1}}); err != nil {
		t.Fatalf("SaveReport: %v", err)
	}

	deps := Deps{
		Store:    st,
		Cfg:      &config.Config{OfflineFactor: 5, ReportRetentionDays: 180},
		Notifier: service.NewNotifier(st, log),
		Log:      log,
	}

	// First run: online(default) -> offline, one alert.
	DetectOfflineServers(deps)
	select {
	case m := <-recv:
		if m["type"] != "offline" || m["server_id"] != "s1" {
			t.Fatalf("alert = %v, want type=offline server_id=s1", m)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("expected an offline webhook alert")
	}

	states, _ := st.ListServerStates(ctx)
	for _, s := range states {
		if s.ID == "s1" && s.LastOnlineState != 0 {
			t.Fatalf("s1 state = %d, want 0 (offline)", s.LastOnlineState)
		}
	}

	// Second run: still offline, no transition -> no new alert.
	DetectOfflineServers(deps)
	select {
	case m := <-recv:
		t.Fatalf("unexpected duplicate alert: %v", m)
	case <-time.After(300 * time.Millisecond):
	}
}

// TestNotifierDisabled: with notify_enabled unset, Send is a no-op (no webhook hit).
func TestNotifierDisabled(t *testing.T) {
	ctx := context.Background()
	log := zaptest.NewLogger(t)
	st, err := store.Open(ctx, &config.Config{DBPath: filepath.Join(t.TempDir(), "m.db")}, log)
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if err := st.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	hit := make(chan struct{}, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hit <- struct{}{} }))
	defer srv.Close()
	_ = st.SetSetting(ctx, "webhook_url", srv.URL) // configured but notify_enabled not "true"

	if err := service.NewNotifier(st, log).Send(ctx, service.Alert{Type: "offline", ServerID: "x", Message: "m"}); err != nil {
		t.Fatalf("Send returned error while disabled: %v", err)
	}
	select {
	case <-hit:
		t.Fatal("webhook called while notifications disabled")
	case <-time.After(300 * time.Millisecond):
	}
}
