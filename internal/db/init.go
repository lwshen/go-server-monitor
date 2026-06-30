// Package db handles SQLite initialization, connection pooling, transactions and
// migrations. For the P0 skeleton everything is stubbed: nothing actually opens a
// database file or executes DDL.
package db

import (
	"database/sql"

	_ "modernc.org/sqlite" // pure-Go SQLite driver, registered as "sqlite"

	"github.com/lwshen/go-server-monitor/internal/config"
	"go.uber.org/zap"
)

// InitDB opens (and on first run initializes) the SQLite database.
//
// P0 STUB: this does NOT open a real file. It logs "not implemented" and returns
// (nil, nil) so the boot sequence can proceed without a working DB.
//
// TODO(P1): open sql.Open("sqlite", "file:"+cfg.DBPath+"?...&_journal_mode=WAL"),
// apply the PRAGMA baseline (WAL, synchronous=NORMAL, busy_timeout=5000,
// cache_size, temp_store=MEMORY), execute migrations/schema.sql, run migrations
// (migrations.go), configure the pool (pool.go) and return the live *sql.DB.
func InitDB(cfg *config.Config, log *zap.Logger) (*sql.DB, error) {
	log.Warn("db.InitDB not implemented (P1)", zap.String("db_path", cfg.DBPath))
	return nil, nil
}

// Close closes the database if it is non-nil. Safe to call with a nil handle
// (the P0 InitDB returns nil).
func Close(db *sql.DB) error {
	if db == nil {
		return nil
	}
	return db.Close()
}
