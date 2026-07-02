package store

import (
	"context"
	"testing"
)

// TestMigrateV2ReAddsServerColumns simulates a v1-era database that predates the
// gid/alias/type/location/notify columns (dropping gid + resetting the version),
// then verifies Migrate's v2 step ALTERs it back in and ListServers recovers.
func TestMigrateV2ReAddsServerColumns(t *testing.T) {
	ctx := context.Background()
	st, _ := openTemp(t)
	if err := st.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	bs := st.(*bunStore)

	// Simulate an old DB: drop ALL the P2 display columns (exercises every v2
	// ALTER, incl. the BOOLEAN one) and rewind the recorded version.
	for _, col := range []string{"gid", "alias", "type", "location", "notify"} {
		if _, err := bs.db.ExecContext(ctx, `ALTER TABLE "servers" DROP COLUMN "`+col+`"`); err != nil {
			t.Fatalf("simulate drop %s: %v", col, err)
		}
	}
	if err := bs.SetSetting(ctx, schemaVersionKey, "1"); err != nil {
		t.Fatalf("rewind version: %v", err)
	}

	// ListServers must now fail (column missing) — the exact reported symptom.
	if _, err := st.ListServers(ctx); err == nil {
		t.Fatal("expected ListServers to fail with gid missing")
	}

	// Re-migrate: v2 re-adds the column.
	if err := st.Migrate(ctx); err != nil {
		t.Fatalf("re-Migrate: %v", err)
	}
	for _, col := range []string{"gid", "alias", "type", "location", "notify"} {
		var n int
		if err := bs.db.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM pragma_table_info('servers') WHERE name=?`, col).Scan(&n); err != nil {
			t.Fatalf("pragma %s: %v", col, err)
		}
		if n != 1 {
			t.Fatalf("column %s count = %d, want 1 after v2", col, n)
		}
	}
	if _, err := st.ListServers(ctx); err != nil {
		t.Fatalf("ListServers still failing after v2: %v", err)
	}
	if v, _ := bs.currentSchemaVersion(ctx); v != latestSchemaVersion {
		t.Fatalf("schema_version = %d, want %d", v, latestSchemaVersion)
	}

	// v2 is idempotent: another Migrate is a no-op (duplicate-column ignored).
	if err := st.Migrate(ctx); err != nil {
		t.Fatalf("idempotent Migrate: %v", err)
	}
}
