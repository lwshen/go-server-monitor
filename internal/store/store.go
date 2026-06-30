// Package store is the data-access boundary for the whole application.
//
// Every other layer (api, cron, service) depends ONLY on the Store interface,
// never on a concrete driver or *sql.DB. This is what makes the backend
// swappable: today SQLite (embedded, modernc) and Turso/libSQL (SQLite-compatible)
// are wired; PostgreSQL is a one-dialect extension point (see docs/database.md).
//
// The implementation is built on Bun (github.com/uptrace/bun), whose dialect
// packages (sqlitedialect / pgdialect / …) provide the multi-database support:
// the same model structs and queries emit dialect-correct SQL. SQLite and Turso
// share sqlitedialect because libSQL is SQLite-compatible.
package store

import (
	"context"

	"github.com/lwshen/go-server-monitor/internal/models"
)

// Store is the port every layer programs against. Lifecycle methods
// (Backend/Migrate/Ping/Close) are implemented; data methods are P2+ stubs that
// return apperr.ErrNotImplemented until their owning phase lands.
type Store interface {
	// ── lifecycle ────────────────────────────────────────────────────────────
	// Backend reports the active backend: "sqlite", "libsql" or "postgres".
	Backend() string
	// Migrate brings the schema up to date (creates tables + indexes if absent).
	Migrate(ctx context.Context) error
	// Ping verifies connectivity.
	Ping(ctx context.Context) error
	// Close releases the underlying connection pool.
	Close() error

	// ── ingest (P2) ──────────────────────────────────────────────────────────
	// SaveReport upserts the servers row and inserts every sample into
	// metrics_history (one row per sample, REQ-RES-04).
	SaveReport(ctx context.Context, report *models.StatReport) error

	// ── public reads (P2) ────────────────────────────────────────────────────
	ListServers(ctx context.Context) ([]models.Server, error)
	GetServer(ctx context.Context, id string) (*models.Server, error)
	// QueryHistory returns downsampled samples for a range key (1h/6h/24h/7d/30d/180d).
	QueryHistory(ctx context.Context, id, rng string) ([]models.MetricsRow, error)

	// ── admin server CRUD (P6) ───────────────────────────────────────────────
	CreateServer(ctx context.Context, cfg *models.ServerConfig) error
	UpdateServer(ctx context.Context, cfg *models.ServerConfig) error
	DeleteServer(ctx context.Context, id string) error
	ReorderServers(ctx context.Context, orderedIDs []string) error

	// ── settings (P6) ────────────────────────────────────────────────────────
	GetSetting(ctx context.Context, key string) (string, error)
	SetSetting(ctx context.Context, key, value string) error
	AllSettings(ctx context.Context) (map[string]string, error)

	// ── maintenance (P7/P9) ──────────────────────────────────────────────────
	// DeleteMetricsBefore prunes metrics_history older than cutoff (Unix seconds).
	DeleteMetricsBefore(ctx context.Context, cutoffUnix int64) (int64, error)
	// RebuildMetrics drops/recreates metrics_history only (REQ-RES-09).
	RebuildMetrics(ctx context.Context) error
}
