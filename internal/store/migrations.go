package store

import (
	"strings"

	"github.com/uptrace/bun/migrate"
)

// schemaMigrations is the ordered set of schema migrations, run by bun/migrate's
// Migrator (see bunStore.Migrate). Each migration lives in its own file named
// NNNN_name.go and registers itself in init() via schemaMigrations.MustRegister —
// bun derives the migration name from the file name (regex `^(\d{1,14})_([a-z0-9_-]+)\.`).
// bun records applied migrations in the bun_migrations table.
//
// Migrations are additive/idempotent so a database created by the previous custom
// (settings.schema_version) framework upgrades cleanly the first time this runs.
var schemaMigrations = migrate.NewMigrations()

// isDuplicateColumnErr reports whether err is an "add a column that already
// exists" error (SQLite: "duplicate column name"; PostgreSQL: "already exists").
func isDuplicateColumnErr(err error) bool {
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "duplicate column") || strings.Contains(s, "already exists")
}
