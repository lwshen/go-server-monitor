package store

import (
	"context"

	"github.com/uptrace/bun"
)

// Migration 0001 — base schema (servers, metrics_history, settings + indexes).
// bun derives the name "0001" from this file's name.
func init() {
	schemaMigrations.MustRegister(up0001Init, down0001Init)
}

func up0001Init(ctx context.Context, db *bun.DB) error {
	// settings (key/value) — also holds bootstrapped admin creds later.
	if _, err := db.NewCreateTable().Model((*settingRow)(nil)).IfNotExists().Exec(ctx); err != nil {
		return err
	}
	// servers must exist before metrics_history references it.
	if _, err := db.NewCreateTable().Model((*serverRow)(nil)).IfNotExists().Exec(ctx); err != nil {
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

func down0001Init(ctx context.Context, db *bun.DB) error {
	// Drop in reverse dependency order (metrics_history references servers).
	for _, model := range []any{(*metricRow)(nil), (*serverRow)(nil), (*settingRow)(nil)} {
		if _, err := db.NewDropTable().Model(model).IfExists().Exec(ctx); err != nil {
			return err
		}
	}
	return nil
}
