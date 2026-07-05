package service

import (
	"context"
	"path/filepath"
	"testing"

	"go.uber.org/zap/zaptest"

	"github.com/lwshen/go-server-monitor/internal/config"
	"github.com/lwshen/go-server-monitor/internal/models"
	"github.com/lwshen/go-server-monitor/internal/store"
)

func TestBootstrapSettings(t *testing.T) {
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

	// First run seeds from the env/config values.
	if err := BootstrapSettings(ctx, st, 90, 3, log); err != nil {
		t.Fatalf("BootstrapSettings: %v", err)
	}
	if v, _ := st.GetSetting(ctx, models.SettingRetentionDays); v != "90" {
		t.Fatalf("retention_days = %q, want 90", v)
	}
	if v, _ := st.GetSetting(ctx, models.SettingOfflineFactor); v != "3" {
		t.Fatalf("offline_factor = %q, want 3", v)
	}

	// Simulate an admin change, then a restart with different env values.
	_ = st.SetSetting(ctx, models.SettingOfflineFactor, "7")
	if err := BootstrapSettings(ctx, st, 180, 5, log); err != nil {
		t.Fatalf("second BootstrapSettings: %v", err)
	}
	// DB values are authoritative — env no longer overwrites them.
	if v, _ := st.GetSetting(ctx, models.SettingOfflineFactor); v != "7" {
		t.Fatalf("offline_factor after restart = %q, want 7 (admin value preserved)", v)
	}
	if v, _ := st.GetSetting(ctx, models.SettingRetentionDays); v != "90" {
		t.Fatalf("retention_days after restart = %q, want 90 (seed preserved)", v)
	}
}
