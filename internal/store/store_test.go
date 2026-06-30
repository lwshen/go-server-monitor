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

// TestSQLiteOpenAndMigrate verifies the store seam end to end on the embedded
// SQLite backend: Open -> Migrate creates the expected tables and indexes and
// records schema_version. This is what proves Bun emits working DDL; swapping in
// libSQL/Postgres reuses the exact same Migrate path with a different dialect.
func TestSQLiteOpenAndMigrate(t *testing.T) {
	ctx := context.Background()
	dbFile := filepath.Join(t.TempDir(), "metrics.db")
	cfg := &config.Config{DBPath: dbFile} // empty DATABASE_URL -> SQLite at DBPath

	st, err := Open(ctx, cfg, zap.NewNop())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	if got := st.Backend(); got != "sqlite" {
		t.Fatalf("Backend() = %q, want sqlite", got)
	}
	if err := st.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if err := st.Migrate(ctx); err != nil { // idempotent (IF NOT EXISTS)
		t.Fatalf("Migrate (second run not idempotent): %v", err)
	}

	// Inspect the file with a raw connection so we test the actual on-disk schema.
	raw, err := sql.Open("sqlite", "file:"+dbFile)
	if err != nil {
		t.Fatalf("raw open: %v", err)
	}
	defer raw.Close()

	assertNames(t, raw, "table", []string{"metrics_history", "servers", "settings"})
	assertNames(t, raw, "index", []string{"idx_history_server_time", "idx_history_timestamp"})

	var version string
	if err := raw.QueryRowContext(ctx,
		`SELECT value FROM settings WHERE key = 'schema_version'`).Scan(&version); err != nil {
		t.Fatalf("read schema_version: %v", err)
	}
	if version != "1" {
		t.Fatalf("schema_version = %q, want 1", version)
	}
}

// assertNames checks that every wanted object of the given sqlite_master type exists.
func assertNames(t *testing.T, db *sql.DB, kind string, want []string) {
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
	var missing []string
	for _, w := range want {
		if !have[w] {
			missing = append(missing, w)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Fatalf("missing %ss: %v", kind, missing)
	}
}
