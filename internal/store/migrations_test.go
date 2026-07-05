package store

import (
	"context"
	"testing"
)

// TestMigrateRecordsInBunMigrations verifies a fresh Migrate applies both
// migrations and records them in bun's tracking table.
func TestMigrateRecordsInBunMigrations(t *testing.T) {
	ctx := context.Background()
	st, _ := openTemp(t)
	if err := st.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	bs := st.(*bunStore)

	names := map[string]bool{}
	rows, err := bs.db.QueryContext(ctx, `SELECT name FROM bun_migrations ORDER BY name`)
	if err != nil {
		t.Fatalf("query bun_migrations: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var n string
		_ = rows.Scan(&n)
		names[n] = true
	}
	if !names["0001"] || !names["0002"] {
		t.Fatalf("recorded migrations = %v, want 0001 + 0002", names)
	}
}

// TestMigrateFromLegacyCustomDB simulates a database created by the earlier custom
// (settings.schema_version) framework: app tables exist WITHOUT the P2 display
// columns and there is NO bun_migrations table. The new bun/migrate-based Migrate
// must take over cleanly — create its tracking table, no-op 0001, and 0002 adds
// the missing columns so ListServers recovers.
func TestMigrateFromLegacyCustomDB(t *testing.T) {
	ctx := context.Background()
	st, _ := openTemp(t)
	if err := st.Migrate(ctx); err != nil {
		t.Fatalf("initial Migrate: %v", err)
	}
	bs := st.(*bunStore)

	// Rewind to a legacy state:
	//  - drop the P2 display columns (as if 0002 never ran),
	//  - drop bun's tracking tables (as if bun/migrate was never used),
	//  - leave a stale settings.schema_version (what the old framework wrote).
	for _, col := range []string{"gid", "alias", "type", "location", "notify"} {
		if _, err := bs.db.ExecContext(ctx, `ALTER TABLE "servers" DROP COLUMN "`+col+`"`); err != nil {
			t.Fatalf("drop %s: %v", col, err)
		}
	}
	for _, tbl := range []string{"bun_migrations", "bun_migration_locks"} {
		if _, err := bs.db.ExecContext(ctx, `DROP TABLE IF EXISTS `+tbl); err != nil {
			t.Fatalf("drop %s: %v", tbl, err)
		}
	}
	if err := bs.SetSetting(ctx, "schema_version", "1"); err != nil {
		t.Fatalf("stale schema_version: %v", err)
	}

	if _, err := st.ListServers(ctx); err == nil {
		t.Fatal("precondition: ListServers should fail with columns missing")
	}

	// The new Migrate takes over.
	if err := st.Migrate(ctx); err != nil {
		t.Fatalf("takeover Migrate: %v", err)
	}
	for _, col := range []string{"gid", "alias", "type", "location", "notify"} {
		var n int
		bs.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM pragma_table_info('servers') WHERE name=?`, col).Scan(&n)
		if n != 1 {
			t.Fatalf("column %s not restored (count=%d)", col, n)
		}
	}
	if _, err := st.ListServers(ctx); err != nil {
		t.Fatalf("ListServers still failing after takeover: %v", err)
	}
	var applied int
	bs.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM bun_migrations`).Scan(&applied)
	if applied != 2 {
		t.Fatalf("bun_migrations rows = %d, want 2", applied)
	}

	// Idempotent afterwards.
	if err := st.Migrate(ctx); err != nil {
		t.Fatalf("idempotent Migrate: %v", err)
	}
}
