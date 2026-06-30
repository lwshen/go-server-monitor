package db

import (
	"database/sql"

	"go.uber.org/zap"
)

// schemaVersion is the current DB schema version baked into the binary
// (REQ-RES-09). On startup the live DB's settings.schema_version is compared to
// this and idempotent, additive-only migrations are applied as needed.
const schemaVersion = 1

// Migrate brings the database schema up to schemaVersion.
//
// P0 STUB: logs "not implemented" and returns nil. The real implementation runs
// migrations/schema.sql on a fresh DB, then applies ordered additive migrations
// (add column / create index / backfill only — never drop) per REQ-RES-09.
//
// TODO(P1): execute migrations/schema.sql, write settings.schema_version, and
// apply versioned diffs when the stored version is behind.
func Migrate(db *sql.DB, log *zap.Logger) error {
	log.Warn("db.Migrate not implemented (P1)", zap.Int("target_schema_version", schemaVersion))
	return nil
}
