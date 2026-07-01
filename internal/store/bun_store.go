package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/schema"
	"go.uber.org/zap"

	"github.com/lwshen/go-server-monitor/internal/models"
	"github.com/lwshen/go-server-monitor/pkg/apperr"
)

// schemaVersionKey is the settings row that tracks the applied migration version.
const schemaVersionKey = "schema_version"

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

// Migrate brings the schema up to latestSchemaVersion by applying every pending
// migration in order (REQ-RES-09). It is safe to call on every boot: already-
// applied migrations are skipped, and the migrations themselves use IF NOT
// EXISTS, so a re-run is a no-op. Bun emits dialect-correct DDL, so this one path
// works for SQLite, libSQL and PostgreSQL alike.
func (s *bunStore) Migrate(ctx context.Context) error {
	start := time.Now()

	// The settings table tracks the schema version, so it must exist first
	// (chicken/egg: it is not part of a numbered migration).
	if _, err := s.db.NewCreateTable().Model((*settingRow)(nil)).IfNotExists().Exec(ctx); err != nil {
		return fmt.Errorf("ensure settings table (%s): %w", s.backend, err)
	}

	current, err := s.currentSchemaVersion(ctx)
	if err != nil {
		return fmt.Errorf("read schema version: %w", err)
	}

	applied := 0
	for _, m := range schemaMigrations() {
		if m.version <= current {
			continue
		}
		if err := m.up(ctx, s.db); err != nil {
			return fmt.Errorf("migration %d (%s) on %s: %w", m.version, m.name, s.backend, err)
		}
		if err := s.SetSetting(ctx, schemaVersionKey, strconv.Itoa(m.version)); err != nil {
			return fmt.Errorf("record schema version %d: %w", m.version, err)
		}
		current = m.version
		applied++
		s.log.Info("已应用数据库迁移", zap.Int("version", m.version), zap.String("name", m.name))
	}

	s.log.Info("数据库迁移完成",
		zap.String("backend", string(s.backend)),
		zap.Int("schema_version", current),
		zap.Int("applied", applied),
		zap.Duration("took", time.Since(start)))
	return nil
}

// currentSchemaVersion reads settings.schema_version, returning 0 when it has
// never been set (a fresh database).
func (s *bunStore) currentSchemaVersion(ctx context.Context) (int, error) {
	var raw string
	err := s.db.NewSelect().
		Model((*settingRow)(nil)).
		Column("value").
		Where("key = ?", schemaVersionKey).
		Scan(ctx, &raw)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return 0, nil
	case err != nil:
		return 0, err
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid schema_version %q: %w", raw, err)
	}
	return n, nil
}

// ── ingest (P2) ────────────────────────────────────────────────────────────

// SaveReport records a probe upload for an ALREADY-REGISTERED server: it refreshes
// the probe-pushed display metadata + sys_info/ip_info snapshots and inserts one
// metrics_history row per sample, in a single transaction (REQ-API-06 / REQ-RES-04).
// An unknown server id yields a 404 AppError — servers are admin-created
// (CONVENTIONS §6 / 03-report-protocol §2.7.2), not auto-created here. Returns the
// number of metric rows written.
func (s *bunStore) SaveReport(ctx context.Context, req *models.ReportRequest) (int, error) {
	samples := resolveSamples(req)
	if len(samples) == 0 {
		return 0, apperr.New(400, "report has no metrics")
	}

	// The newest-timestamp sample drives the server's "current" snapshot (REQ-RES-04 §3).
	latest := samples[0]
	for _, smp := range samples[1:] {
		if smp.ts > latest.ts {
			latest = smp
		}
	}

	rows := make([]*metricRow, len(samples))
	for i, smp := range samples {
		rows[i] = statReportToMetricRow(req.ID, smp.ts, smp.data)
	}
	sr := latest.data

	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		exists, err := tx.NewSelect().Model((*serverRow)(nil)).Where("id = ?", req.ID).Exists(ctx)
		if err != nil {
			return fmt.Errorf("check server: %w", err)
		}
		if !exists {
			return apperr.New(404, "server not found") // register via /api/admin/servers/add first
		}

		// Refresh probe-managed columns only — never the admin-managed name /
		// server_group / created_at.
		if _, err := tx.NewUpdate().Model((*serverRow)(nil)).
			Set("gid = ?", sr.Gid).
			Set("alias = ?", sr.Alias).
			Set("type = ?", sr.Type).
			Set("location = ?", sr.Location).
			Set("notify = ?", sr.Notify).
			Set("sort_order = ?", sr.Weight).
			Set("sys_info_json = ?", marshalSnapshot(sr.SysInfo)).
			Set("ip_info_json = ?", marshalSnapshot(sr.IpInfo)).
			Set("updated_at = ?", nowISO()).
			Where("id = ?", req.ID).
			Exec(ctx); err != nil {
			return fmt.Errorf("update server: %w", err)
		}

		if _, err := tx.NewInsert().Model(&rows).Exec(ctx); err != nil {
			return fmt.Errorf("insert %d samples: %w", len(rows), err)
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return len(rows), nil
}

// public reads (ListServers / GetServer / QueryHistory) live in read.go.

// ── admin server CRUD (P6) ───────────────────────────────────────────────────

// CreateServer inserts a new admin-registered server row (REQ-API-09). The caller
// supplies the id (a fresh UUID); sensible defaults fill unset columns.
func (s *bunStore) CreateServer(ctx context.Context, cfg *models.ServerConfig) error {
	now := nowISO()
	row := &serverRow{
		ID:              cfg.ID,
		Name:            orDefaultStr(cfg.Name, "New Server"),
		ServerGroup:     orDefaultStr(cfg.ServerGroup, "Default"),
		Price:           cfg.Price,
		ExpireDate:      cfg.ExpireDate,
		Bandwidth:       cfg.Bandwidth,
		TrafficLimit:    cfg.TrafficLimit,
		TrafficCalcType: orDefaultStr(cfg.TrafficCalcType, "total"),
		ResetDay:        orDefaultInt(cfg.ResetDay, 1),
		CollectInterval: cfg.CollectInterval,
		ReportInterval:  orDefaultInt(cfg.ReportInterval, 60),
		PingMode:        orDefaultStr(cfg.PingMode, "http"),
		IsHidden:        orDefaultStr(cfg.IsHidden, "0"),
		SortOrder:       cfg.SortOrder,
		LastOnlineState: 1,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if _, err := s.db.NewInsert().Model(row).Exec(ctx); err != nil {
		return fmt.Errorf("create server: %w", err)
	}
	return nil
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

// GetSetting returns the value for key, or ("", nil) when the key is absent.
func (s *bunStore) GetSetting(ctx context.Context, key string) (string, error) {
	var value string
	err := s.db.NewSelect().
		Model((*settingRow)(nil)).
		Column("value").
		Where("key = ?", key).
		Scan(ctx, &value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return value, err
}

// AllSettings returns every key/value pair. (Callers redact secret-class keys
// before returning them over the admin API — REQ-RES-01.)
func (s *bunStore) AllSettings(ctx context.Context) (map[string]string, error) {
	var rows []settingRow
	if err := s.db.NewSelect().Model(&rows).Scan(ctx); err != nil {
		return nil, err
	}
	out := make(map[string]string, len(rows))
	for _, r := range rows {
		out[r.Key] = r.Value
	}
	return out, nil
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
