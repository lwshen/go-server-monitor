# Database backends

The app talks to storage only through the **`store.Store` interface**
(`internal/store/store.go`). No other package imports a database driver or
`*sql.DB`. The concrete implementation (`bunStore`) is built on
[Bun](https://bun.uptrace.dev/), whose per-dialect packages provide multi-database
support: the same model structs and queries emit dialect-correct SQL.

```
internal/store/
  store.go       # Store interface — the port every layer depends on
  open.go        # Open(): picks a backend from DATABASE_URL's scheme
  bun_store.go   # bunStore: the one implementation, shared by all SQL backends
  schema.go      # table models — Bun derives DDL for each dialect from these
  store_test.go  # seam smoke test (Open -> Migrate -> assert schema)
```

## Choosing a backend

The backend is selected by the **scheme of `DATABASE_URL`** (env) — no code change:

| Scheme | Backend | Driver | Status |
|---|---|---|---|
| _(empty)_ → `DB_PATH` | SQLite (embedded) | `modernc.org/sqlite` (pure Go) | ✅ built in |
| `sqlite:` / `file:` | SQLite (embedded) | `modernc.org/sqlite` | ✅ built in |
| `libsql://` `https://` `wss://` | Turso / libSQL (remote) | `libsql-client-go` (pure Go) | ✅ built in |
| `postgres://` `postgresql://` | PostgreSQL | `pgx/v5` | 🔌 extension point |

SQLite and Turso share Bun's `sqlitedialect` because libSQL is SQLite-compatible —
only the driver and connection string differ.

### Examples

```bash
# Embedded SQLite (default)
DATABASE_URL=                       # empty -> uses DB_PATH
DB_PATH=./data/metrics.db

# or explicitly
DATABASE_URL=sqlite:./data/metrics.db

# Turso / libSQL (remote)
DATABASE_URL=libsql://my-db.turso.io
DATABASE_AUTH_TOKEN=eyJhbGci...     # or append ?authToken=... to the URL
```

All builds stay `CGO_ENABLED=0` (static binary, easy cross-compile): the SQLite,
libSQL-remote and pgx drivers are all pure Go. *(Embedded Turso via
`go-libsql` would require cgo — not used here.)*

## Adding PostgreSQL (the extension point)

Postgres is deliberately **not compiled in** by default so the binary stays lean.
Wiring it is a localized, ~3-step change — no other layer is touched, because
everything depends on the `Store` interface:

1. Add the deps:
   ```bash
   go get github.com/jackc/pgx/v5/stdlib github.com/uptrace/bun/dialect/pgdialect
   ```
2. In `internal/store/open.go`, add the driver blank-import and replace the
   `backendPostgres` error branch with a real opener:
   ```go
   import (
       _ "github.com/jackc/pgx/v5/stdlib"          // driver "pgx"
       "github.com/uptrace/bun/dialect/pgdialect"
   )

   case backendPostgres:
       sqldb, err := sql.Open("pgx", target)       // target = the postgres:// URL
       if err != nil {
           return nil, fmt.Errorf("open postgres: %w", err)
       }
       st := newBunStore(sqldb, pgdialect.New(), kind, log)
       return verify(ctx, st)
   ```
3. Run `go mod tidy && go build ./... && go test ./internal/store/`.

`Migrate()` and every data method already work unchanged — Bun emits Postgres DDL
(`BIGSERIAL`, `$1` placeholders, etc.) from the same `schema.go` models. The
`metricRow.ID` `autoincrement` tag becomes `BIGSERIAL` on Postgres and
`AUTOINCREMENT` on SQLite automatically.

> When you implement the P2+ data methods in `bun_store.go`, keep them
> dialect-neutral (use Bun's query builder, not raw dialect-specific SQL) so they
> keep working across all three backends. Add a Postgres run to `store_test.go`
> (guarded by a `DATABASE_URL` env) to cover it in CI.
