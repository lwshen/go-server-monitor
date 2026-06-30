package store

import (
	"context"
	"database/sql"
	"path/filepath"
	"sort"
	"testing"

	"go.uber.org/zap"

	"github.com/lwshen/go-server-monitor/internal/config"
)

// openTemp opens an embedded SQLite store backed by a fresh temp file.
func openTemp(t *testing.T) (Store, string) {
	t.Helper()
	dbFile := filepath.Join(t.TempDir(), "metrics.db")
	st, err := Open(context.Background(), &config.Config{DBPath: dbFile}, zap.NewNop())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st, dbFile
}

// TestMigrateCreatesSchema verifies the seam end to end on SQLite: Open -> Migrate
// creates the expected tables and indexes, records the schema version, and is
// idempotent. Swapping in libSQL/Postgres reuses this exact path with a different
// dialect.
func TestMigrateCreatesSchema(t *testing.T) {
	ctx := context.Background()
	st, dbFile := openTemp(t)

	if got := st.Backend(); got != "sqlite" {
		t.Fatalf("Backend() = %q, want sqlite", got)
	}
	if err := st.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if err := st.Migrate(ctx); err != nil { // idempotent (IF NOT EXISTS)
		t.Fatalf("Migrate (second run not idempotent): %v", err)
	}

	raw, err := sql.Open("sqlite", "file:"+dbFile)
	if err != nil {
		t.Fatalf("raw open: %v", err)
	}
	defer raw.Close()

	assertMasterNames(t, raw, "table", []string{"metrics_history", "servers", "settings"})
	assertMasterNames(t, raw, "index", []string{"idx_history_server_time", "idx_history_timestamp", "idx_servers_group"})

	var version string
	if err := raw.QueryRowContext(ctx,
		`SELECT value FROM settings WHERE key = 'schema_version'`).Scan(&version); err != nil {
		t.Fatalf("read schema_version: %v", err)
	}
	if version != "1" {
		t.Fatalf("schema_version = %q, want 1 (latestSchemaVersion=%d)", version, latestSchemaVersion)
	}
}

// TestSchemaColumns checks the migrated schema carries the full authoritative
// column set (a representative slice of report-types.ts fields), so P2 ingestion
// has somewhere to write every value.
func TestSchemaColumns(t *testing.T) {
	ctx := context.Background()
	st, dbFile := openTemp(t)
	if err := st.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	raw, err := sql.Open("sqlite", "file:"+dbFile)
	if err != nil {
		t.Fatalf("raw open: %v", err)
	}
	defer raw.Close()

	assertColumns(t, raw, "metrics_history", []string{
		"id", "server_id", "timestamp",
		"cpu", "load_avg", "cpu_cores", "cpu_info", "cpu_model",
		"memory_total", "memory_used", "swap_total", "swap_used", "hdd_total", "hdd_used",
		"network_rx", "network_tx", "network_in", "network_out", "last_network_in", "last_network_out",
		"ping_ct", "ping_cu", "ping_cm", "ping_bd", "loss_ct", "loss_cu", "loss_cm", "loss_bd",
		"online4", "online6", "os", "os_release", "kernel_version", "arch", "os_family",
		"uptime", "host_name", "gpu", "gpu_info", "region", "gid", "location",
		"vnstat", "custom", "disks_json",
	})
	assertColumns(t, raw, "servers", []string{
		"id", "name", "server_group", "expire_date", "report_interval", "ping_mode", "is_hidden",
		"sort_order", "last_online_state", "last_state_change", "expiration_notified",
		"sys_info_json", "ip_info_json", "created_at", "updated_at",
	})
}

// TestWALEnabled confirms the SQLite PRAGMA baseline took effect on the store's
// own connection.
func TestWALEnabled(t *testing.T) {
	ctx := context.Background()
	st, _ := openTemp(t)

	bs, ok := st.(*bunStore)
	if !ok {
		t.Fatalf("store is %T, want *bunStore", st)
	}
	var mode string
	if err := bs.db.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&mode); err != nil {
		t.Fatalf("PRAGMA journal_mode: %v", err)
	}
	if mode != "wal" {
		t.Fatalf("journal_mode = %q, want wal", mode)
	}
}

// TestMigrateForwardFromOlderVersion verifies pending migrations are (re)applied
// when the stored version is behind the binary's latest — the REQ-RES-09 upgrade
// path — and that doing so is harmless (idempotent DDL).
func TestMigrateForwardFromOlderVersion(t *testing.T) {
	ctx := context.Background()
	st, _ := openTemp(t)
	if err := st.Migrate(ctx); err != nil {
		t.Fatalf("initial Migrate: %v", err)
	}

	bs := st.(*bunStore)
	// Simulate a database written by an older binary.
	if err := bs.SetSetting(ctx, schemaVersionKey, "0"); err != nil {
		t.Fatalf("SetSetting: %v", err)
	}
	if v, _ := bs.currentSchemaVersion(ctx); v != 0 {
		t.Fatalf("precondition: currentSchemaVersion = %d, want 0", v)
	}

	if err := st.Migrate(ctx); err != nil {
		t.Fatalf("forward Migrate: %v", err)
	}
	if v, err := bs.currentSchemaVersion(ctx); err != nil || v != latestSchemaVersion {
		t.Fatalf("after forward Migrate: version = %d, err = %v; want %d", v, err, latestSchemaVersion)
	}
}

// TestSettingsRoundtrip covers the settings helpers Migrate relies on.
func TestSettingsRoundtrip(t *testing.T) {
	ctx := context.Background()
	st, _ := openTemp(t)
	if err := st.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	if v, err := st.GetSetting(ctx, "missing"); err != nil || v != "" {
		t.Fatalf("GetSetting(missing) = (%q, %v), want (\"\", nil)", v, err)
	}
	if err := st.SetSetting(ctx, "site_title", "Mon"); err != nil {
		t.Fatalf("SetSetting insert: %v", err)
	}
	if err := st.SetSetting(ctx, "site_title", "Monitor"); err != nil { // upsert
		t.Fatalf("SetSetting upsert: %v", err)
	}
	if v, err := st.GetSetting(ctx, "site_title"); err != nil || v != "Monitor" {
		t.Fatalf("GetSetting = (%q, %v), want (Monitor, nil)", v, err)
	}

	all, err := st.AllSettings(ctx)
	if err != nil {
		t.Fatalf("AllSettings: %v", err)
	}
	if all["site_title"] != "Monitor" || all[schemaVersionKey] != "1" {
		t.Fatalf("AllSettings = %v, want site_title=Monitor schema_version=1", all)
	}
}

// ── helpers ────────────────────────────────────────────────────────────────

// assertMasterNames checks every wanted sqlite_master object of the given type exists.
func assertMasterNames(t *testing.T, db *sql.DB, kind string, want []string) {
	t.Helper()
	rows, err := db.Query(`SELECT name FROM sqlite_master WHERE type = ?`, kind)
	if err != nil {
		t.Fatalf("query %ss: %v", kind, err)
	}
	defer rows.Close()

	have := map[string]bool{}
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			t.Fatalf("scan: %v", err)
		}
		have[n] = true
	}
	assertContains(t, kind, have, want)
}

// assertColumns checks every wanted column exists on the table (via PRAGMA table_info).
func assertColumns(t *testing.T, db *sql.DB, table string, want []string) {
	t.Helper()
	rows, err := db.Query(`SELECT name FROM pragma_table_info(?)`, table)
	if err != nil {
		t.Fatalf("table_info(%s): %v", table, err)
	}
	defer rows.Close()

	have := map[string]bool{}
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			t.Fatalf("scan: %v", err)
		}
		have[n] = true
	}
	assertContains(t, table+" column", have, want)
}

func assertContains(t *testing.T, what string, have map[string]bool, want []string) {
	t.Helper()
	var missing []string
	for _, w := range want {
		if !have[w] {
			missing = append(missing, w)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Fatalf("missing %ss: %v", what, missing)
	}
}
