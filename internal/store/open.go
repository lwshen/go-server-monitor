package store

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/uptrace/bun/dialect/sqlitedialect"
	"go.uber.org/zap"

	// Pure-Go drivers (CGO_ENABLED=0 friendly), registered via blank import:
	_ "github.com/tursodatabase/libsql-client-go/libsql" // driver "libsql" (Turso, remote)
	_ "modernc.org/sqlite"                               // driver "sqlite" (embedded)

	"github.com/lwshen/go-server-monitor/internal/config"
)

// backendKind identifies the active database family.
type backendKind string

const (
	backendSQLite   backendKind = "sqlite"
	backendLibSQL   backendKind = "libsql"
	backendPostgres backendKind = "postgres"
)

// sqlitePragmas is appended to the SQLite DSN (modernc syntax). Matches the
// baseline in CONVENTIONS §8 / REQ-DB-01: WAL, NORMAL sync, 5s busy timeout,
// in-memory temp store, foreign keys on.
const sqlitePragmas = "_pragma=busy_timeout(5000)" +
	"&_pragma=journal_mode(WAL)" +
	"&_pragma=synchronous(NORMAL)" +
	"&_pragma=foreign_keys(ON)" +
	"&_pragma=temp_store(MEMORY)"

// Open selects a backend from the configuration and returns a ready Store.
//
// Selection (see config.Config.DatabaseURL / DB_PATH):
//   - empty DATABASE_URL          -> SQLite at DB_PATH (back-compat default)
//   - sqlite:// | sqlite: | file: -> SQLite (modernc, embedded)
//   - libsql:// | http(s):// | ws -> Turso/libSQL (libsql-client-go, remote)
//   - postgres:// | postgresql:// -> PostgreSQL (extension point; see docs/database.md)
func Open(ctx context.Context, cfg *config.Config, log *zap.Logger) (Store, error) {
	kind, target, err := classify(cfg)
	if err != nil {
		return nil, err
	}

	switch kind {
	case backendSQLite:
		if dir := filepath.Dir(target); dir != "" && dir != "." {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, fmt.Errorf("create db dir %q: %w", dir, err)
			}
		}
		sqldb, err := sql.Open("sqlite", "file:"+target+"?"+sqlitePragmas)
		if err != nil {
			return nil, fmt.Errorf("open sqlite: %w", err)
		}
		st := newBunStore(sqldb, sqlitedialect.New(), kind, log)
		return verify(ctx, st)

	case backendLibSQL:
		dsn := target
		if cfg.DatabaseAuthToken != "" && !strings.Contains(dsn, "authToken=") {
			sep := "?"
			if strings.Contains(dsn, "?") {
				sep = "&"
			}
			dsn += sep + "authToken=" + url.QueryEscape(cfg.DatabaseAuthToken)
		}
		sqldb, err := sql.Open("libsql", dsn)
		if err != nil {
			return nil, fmt.Errorf("open libsql: %w", err)
		}
		// libSQL is SQLite-compatible, so it uses the SQLite dialect.
		st := newBunStore(sqldb, sqlitedialect.New(), kind, log)
		return verify(ctx, st)

	case backendPostgres:
		// Extension point. To enable: add github.com/jackc/pgx/v5/stdlib (driver
		// "pgx") and github.com/uptrace/bun/dialect/pgdialect, then open with
		// sql.Open("pgx", target) + bun.NewDB(sqldb, pgdialect.New()).
		// See docs/database.md for the exact diff.
		return nil, fmt.Errorf("postgresql backend is an extension point, not compiled into this build (see docs/database.md)")

	default:
		return nil, fmt.Errorf("unknown database backend %q", kind)
	}
}

// verify pings the freshly-opened store, closing it on failure so we never leak
// a half-open handle. For the SQLite family it also logs the effective PRAGMAs so
// the operator can confirm WAL/busy_timeout actually took (they are applied
// per-connection via the DSN; PostgreSQL has no equivalent and is skipped).
func verify(ctx context.Context, st *bunStore) (Store, error) {
	if err := st.Ping(ctx); err != nil {
		_ = st.Close()
		return nil, fmt.Errorf("%s ping: %w", st.backend, err)
	}

	fields := []zap.Field{zap.String("backend", string(st.backend))}
	if st.backend == backendSQLite || st.backend == backendLibSQL {
		var journalMode string
		var busyTimeout int
		_ = st.db.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&journalMode)
		_ = st.db.QueryRowContext(ctx, "PRAGMA busy_timeout").Scan(&busyTimeout)
		fields = append(fields,
			zap.String("journal_mode", journalMode),
			zap.Int("busy_timeout_ms", busyTimeout))
	}
	st.log.Info("数据库已连接", fields...)
	return st, nil
}

// classify resolves the configured DATABASE_URL (or DB_PATH fallback) into a
// backend kind and a cleaned target (a file path for SQLite, a URL otherwise).
func classify(cfg *config.Config) (backendKind, string, error) {
	raw := strings.TrimSpace(cfg.DatabaseURL)
	if raw == "" {
		// Back-compat: no DATABASE_URL -> treat DB_PATH as an embedded SQLite file.
		return backendSQLite, cfg.DBPath, nil
	}

	scheme := raw
	if i := strings.Index(raw, "://"); i >= 0 {
		scheme = raw[:i]
	} else if i := strings.Index(raw, ":"); i >= 0 {
		scheme = raw[:i] // e.g. "sqlite:./x.db" or "file:x.db"
	}

	switch strings.ToLower(scheme) {
	case "sqlite", "sqlite3", "file":
		path := raw
		for _, p := range []string{"sqlite3://", "sqlite://", "sqlite3:", "sqlite:", "file:"} {
			path = strings.TrimPrefix(path, p)
		}
		return backendSQLite, path, nil
	case "libsql", "turso", "http", "https", "ws", "wss":
		return backendLibSQL, raw, nil
	case "postgres", "postgresql":
		return backendPostgres, raw, nil
	default:
		return "", "", fmt.Errorf("unsupported database URL scheme %q", scheme)
	}
}
