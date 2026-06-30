package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/schema"
	"go.uber.org/zap"

	"github.com/lwshen/go-server-monitor/internal/models"
	"github.com/lwshen/go-server-monitor/pkg/apperr"
)

// bunStore is the Bun-backed Store implementation shared by every SQL backend.
// The dialect (passed in by Open) is the only thing that differs between SQLite,
// Turso/libSQL and PostgreSQL — the queries and models below are written once.
type bunStore struct {
	db      *bun.DB
	backend backendKind
	log     *zap.Logger
}

// compile-time assertion that bunStore satisfies the Store port.
var _ Store = (*bunStore)(nil)

// newBunStore wires a *sql.DB + dialect into Bun and applies the connection pool
// settings (REQ-DB-05).
func newBunStore(sqldb *sql.DB, dialect schema.Dialect, kind backendKind, log *zap.Logger) *bunStore {
	sqldb.SetMaxOpenConns(25)
	sqldb.SetMaxIdleConns(5)
	sqldb.SetConnMaxLifetime(0) // SQL connections need no recycling here

	return &bunStore{
		db:      bun.NewDB(sqldb, dialect),
		backend: kind,
		log:     log,
	}
}

// ── lifecycle ────────────────────────────────────────────────────────────────

func (s *bunStore) Backend() string { return string(s.backend) }

func (s *bunStore) Ping(ctx context.Context) error { return s.db.PingContext(ctx) }

func (s *bunStore) Close() error { return s.db.Close() }

// Migrate creates the tables and indexes if they do not already exist. Bun emits
// dialect-correct DDL from the models in schema.go, so this one method works for
// SQLite, libSQL and PostgreSQL alike.
//
// TODO(P1): evolve into versioned, additive-only migrations keyed on
// settings.schema_version (REQ-RES-09); for now it is create-if-not-exists.
func (s *bunStore) Migrate(ctx context.Context) error {
	start := time.Now()

	for _, model := range tableModels() {
		if _, err := s.db.NewCreateTable().Model(model).IfNotExists().Exec(ctx); err != nil {
			return fmt.Errorf("create table (%s): %w", s.backend, err)
		}
	}

	indexes := []struct {
		name string
		cols []string
	}{
		{"idx_history_server_time", []string{"server_id", "timestamp"}},
		{"idx_history_timestamp", []string{"timestamp"}},
	}
	for _, idx := range indexes {
		if _, err := s.db.NewCreateIndex().
			Model((*metricRow)(nil)).
			Index(idx.name).
			Column(idx.cols...).
			IfNotExists().
			Exec(ctx); err != nil {
			return fmt.Errorf("create index %s (%s): %w", idx.name, s.backend, err)
		}
	}

	// Record the schema version so P1 versioned migrations have a baseline.
	if err := s.SetSetting(ctx, "schema_version", fmt.Sprintf("%d", schemaVersion)); err != nil {
		return fmt.Errorf("write schema_version: %w", err)
	}

	s.log.Info("数据库迁移完成",
		zap.String("backend", string(s.backend)),
		zap.Duration("took", time.Since(start)))
	return nil
}

// ── ingest (P2) ────────────────────────────────────────────────────────────

// SaveReport persists a probe report.
//
// P2 STUB: returns ErrNotImplemented.
//
// TODO(P2): in s.db.RunInTx, upsert the serverRow then insert one metricRow per
// sample (map -1 sentinels to nil/NULL, store disks[] as disks_json).
func (s *bunStore) SaveReport(ctx context.Context, report *models.StatReport) error {
	s.log.Warn("store.SaveReport not implemented (P2)")
	return apperr.ErrNotImplemented
}

// ── public reads (P2) ──────────────────────────────────────────────────────

func (s *bunStore) ListServers(ctx context.Context) ([]models.Server, error) {
	s.log.Warn("store.ListServers not implemented (P2)")
	return nil, apperr.ErrNotImplemented
}

func (s *bunStore) GetServer(ctx context.Context, id string) (*models.Server, error) {
	s.log.Warn("store.GetServer not implemented (P2)")
	return nil, apperr.ErrNotImplemented
}

func (s *bunStore) QueryHistory(ctx context.Context, id, rng string) ([]models.MetricsRow, error) {
	s.log.Warn("store.QueryHistory not implemented (P2)")
	return nil, apperr.ErrNotImplemented
}

// ── admin server CRUD (P6) ───────────────────────────────────────────────────

func (s *bunStore) CreateServer(ctx context.Context, cfg *models.ServerConfig) error {
	s.log.Warn("store.CreateServer not implemented (P6)")
	return apperr.ErrNotImplemented
}

func (s *bunStore) UpdateServer(ctx context.Context, cfg *models.ServerConfig) error {
	s.log.Warn("store.UpdateServer not implemented (P6)")
	return apperr.ErrNotImplemented
}

func (s *bunStore) DeleteServer(ctx context.Context, id string) error {
	s.log.Warn("store.DeleteServer not implemented (P6)")
	return apperr.ErrNotImplemented
}

func (s *bunStore) ReorderServers(ctx context.Context, orderedIDs []string) error {
	s.log.Warn("store.ReorderServers not implemented (P6)")
	return apperr.ErrNotImplemented
}

// ── settings (P6) ────────────────────────────────────────────────────────────

// SetSetting upserts a key/value row. This one is implemented (real) because
// Migrate relies on it to persist schema_version.
func (s *bunStore) SetSetting(ctx context.Context, key, value string) error {
	row := &settingRow{Key: key, Value: value}
	_, err := s.db.NewInsert().
		Model(row).
		On("CONFLICT (key) DO UPDATE").
		Set("value = EXCLUDED.value").
		Exec(ctx)
	return err
}

func (s *bunStore) GetSetting(ctx context.Context, key string) (string, error) {
	s.log.Warn("store.GetSetting not implemented (P6)")
	return "", apperr.ErrNotImplemented
}

func (s *bunStore) AllSettings(ctx context.Context) (map[string]string, error) {
	s.log.Warn("store.AllSettings not implemented (P6)")
	return nil, apperr.ErrNotImplemented
}

// ── maintenance (P7/P9) ──────────────────────────────────────────────────────

func (s *bunStore) DeleteMetricsBefore(ctx context.Context, cutoffUnix int64) (int64, error) {
	s.log.Warn("store.DeleteMetricsBefore not implemented (P7)")
	return 0, apperr.ErrNotImplemented
}

func (s *bunStore) RebuildMetrics(ctx context.Context) error {
	s.log.Warn("store.RebuildMetrics not implemented (P9)")
	return apperr.ErrNotImplemented
}
