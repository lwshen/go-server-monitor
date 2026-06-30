package store

import (
	"context"

	"github.com/uptrace/bun"
)

// latestSchemaVersion is the highest migration version baked into the binary. It
// equals the last entry in schemaMigrations and is what store.Migrate brings a
// database up to (REQ-RES-09).
const latestSchemaVersion = 1

// migration is one ordered, forward-only schema step. Migrations are
// additive-only (create table / add column / create index / backfill) and
// idempotent where possible (IF NOT EXISTS), so re-running Migrate is safe and a
// new binary can read an old database (REQ-RES-09).
type migration struct {
	version int
	name    string
	up      func(ctx context.Context, db *bun.DB) error
}

// schemaMigrations returns every migration in ascending version order. Append new
// migrations here (never edit or reorder shipped ones) and bump
// latestSchemaVersion to match. Each `up` runs against any backend because it
// uses Bun's dialect-aware builders, not raw dialect SQL.
func schemaMigrations() []migration {
	return []migration{
		{version: 1, name: "init schema", up: migrate0001Init},
	}
}

// migrate0001Init creates the base tables and indexes. The settings table is
// already ensured by Migrate (it tracks the version), so it is not recreated here.
func migrate0001Init(ctx context.Context, db *bun.DB) error {
	// servers must exist before metrics_history references it.
	if _, err := db.NewCreateTable().
		Model((*serverRow)(nil)).
		IfNotExists().
		Exec(ctx); err != nil {
		return err
	}

	if _, err := db.NewCreateTable().
		Model((*metricRow)(nil)).
		IfNotExists().
		ForeignKey(`("server_id") REFERENCES "servers" ("id") ON DELETE CASCADE`).
		Exec(ctx); err != nil {
		return err
	}

	indexes := []struct {
		name string
		cols []string
	}{
		{"idx_history_server_time", []string{"server_id", "timestamp"}}, // range scans (REQ-DB-04)
		{"idx_history_timestamp", []string{"timestamp"}},                // retention cleanup
		{"idx_servers_group", []string{"server_group"}},
		{"idx_servers_expire", []string{"expire_date"}}, // hourly expiration reminder (P7)
	}
	for _, idx := range indexes {
		model := any((*metricRow)(nil))
		if idx.name == "idx_servers_group" || idx.name == "idx_servers_expire" {
			model = (*serverRow)(nil)
		}
		if _, err := db.NewCreateIndex().
			Model(model).
			Index(idx.name).
			Column(idx.cols...).
			IfNotExists().
			Exec(ctx); err != nil {
			return err
		}
	}
	return nil
}
