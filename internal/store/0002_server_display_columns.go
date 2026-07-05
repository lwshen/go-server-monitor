package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/uptrace/bun"
)

// Migration 0002 — add the probe-pushed display columns to servers.
//
// gid/alias/type/location/notify were added to the serverRow model after 0001
// shipped, so databases created at 0001 (or by the earlier custom framework) lack
// them, which broke ListServers with "no such column: srv.gid". This ALTERs them
// in. Idempotent: on databases that already have the columns the "duplicate
// column" error is ignored. Identifiers are quoted and the types are portable
// across SQLite / libSQL / PostgreSQL.
func init() {
	schemaMigrations.MustRegister(up0002ServerDisplayColumns, down0002ServerDisplayColumns)
}

var serverDisplayColumns = []struct{ name, ddl string }{
	{"gid", `"gid" TEXT DEFAULT ''`},
	{"alias", `"alias" TEXT DEFAULT ''`},
	{"type", `"type" TEXT DEFAULT ''`},
	{"location", `"location" TEXT DEFAULT ''`},
	{"notify", `"notify" BOOLEAN DEFAULT FALSE`},
}

func up0002ServerDisplayColumns(ctx context.Context, db *bun.DB) error {
	for _, c := range serverDisplayColumns {
		if _, err := db.ExecContext(ctx, fmt.Sprintf(`ALTER TABLE "servers" ADD COLUMN %s`, c.ddl)); err != nil {
			if isDuplicateColumnErr(err) {
				continue // already present (fresh DB from the current 0001 model)
			}
			return fmt.Errorf("add servers.%s: %w", c.name, err)
		}
	}
	return nil
}

func down0002ServerDisplayColumns(ctx context.Context, db *bun.DB) error {
	for _, c := range serverDisplayColumns {
		if _, err := db.ExecContext(ctx, fmt.Sprintf(`ALTER TABLE "servers" DROP COLUMN "%s"`, c.name)); err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "no such column") ||
				strings.Contains(strings.ToLower(err.Error()), "does not exist") {
				continue
			}
			return fmt.Errorf("drop servers.%s: %w", c.name, err)
		}
	}
	return nil
}
