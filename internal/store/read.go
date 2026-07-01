package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/lwshen/go-server-monitor/internal/models"
	"github.com/lwshen/go-server-monitor/pkg/apperr"
)

// ListServers returns every server ordered by sort weight, each joined with its
// most recent metrics_history row. Online status is left for the caller (it
// depends on config.OfflineFactor).
func (s *bunStore) ListServers(ctx context.Context) ([]models.Server, error) {
	var rows []serverRow
	if err := s.db.NewSelect().Model(&rows).
		OrderExpr("sort_order ASC").Order("id ASC").
		Scan(ctx); err != nil {
		return nil, err
	}

	out := make([]models.Server, 0, len(rows))
	for i := range rows {
		srv := serverFromRow(&rows[i], false)
		m, ts, err := s.latestMetric(ctx, rows[i].ID)
		if err != nil {
			return nil, err
		}
		srv.LatestMetrics = m
		srv.LastUpdated = ts
		out = append(out, srv)
	}
	return out, nil
}

// GetServer returns one server with its latest metrics and the structured
// sys_info / ip_info snapshots, or (nil, nil) when the id is unknown.
func (s *bunStore) GetServer(ctx context.Context, id string) (*models.Server, error) {
	var row serverRow
	err := s.db.NewSelect().Model(&row).Where("id = ?", id).Limit(1).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	srv := serverFromRow(&row, true)
	m, ts, err := s.latestMetric(ctx, id)
	if err != nil {
		return nil, err
	}
	srv.LatestMetrics = m
	srv.LastUpdated = ts
	return &srv, nil
}

// QueryHistory returns server-side downsampled buckets for the range key
// (REQ-RES-03): numeric metrics are bucket averages, ping_*/loss_* ignore NULLs.
func (s *bunStore) QueryHistory(ctx context.Context, id, rng string) ([]models.HistoryPoint, error) {
	window, bucket := rangeBuckets(rng)
	if window == 0 {
		return nil, apperr.New(400, "invalid range (want 1h/6h/24h/7d/30d/180d)")
	}
	cutoff := nowUnix() - window

	var points []models.HistoryPoint
	err := s.db.NewSelect().
		Model((*metricRow)(nil)).
		ColumnExpr("(timestamp / ?) * ? AS ts", bucket, bucket).
		ColumnExpr("AVG(cpu) AS cpu").
		ColumnExpr("AVG(memory_used) AS memory_used").
		ColumnExpr("AVG(memory_total) AS memory_total").
		ColumnExpr("AVG(swap_used) AS swap_used").
		ColumnExpr("AVG(swap_total) AS swap_total").
		ColumnExpr("AVG(hdd_used) AS hdd_used").
		ColumnExpr("AVG(hdd_total) AS hdd_total").
		ColumnExpr("AVG(network_rx) AS network_rx").
		ColumnExpr("AVG(network_tx) AS network_tx").
		ColumnExpr("AVG(tcp_conn) AS tcp_conn").
		ColumnExpr("AVG(processes) AS processes").
		ColumnExpr("AVG(ping_ct) AS ping_ct").
		ColumnExpr("AVG(ping_cu) AS ping_cu").
		ColumnExpr("AVG(ping_cm) AS ping_cm").
		ColumnExpr("AVG(ping_bd) AS ping_bd").
		ColumnExpr("AVG(loss_ct) AS loss_ct").
		ColumnExpr("AVG(loss_cu) AS loss_cu").
		ColumnExpr("AVG(loss_cm) AS loss_cm").
		ColumnExpr("AVG(loss_bd) AS loss_bd").
		Where("server_id = ?", id).
		Where("timestamp >= ?", cutoff).
		GroupExpr("timestamp / ?", bucket).
		OrderExpr("ts ASC").
		Scan(ctx, &points)
	if err != nil {
		return nil, err
	}
	return points, nil
}

// ListServerStates returns every server's offline/expiration state plus its
// last-seen timestamp (newest metric), for the P7 cron jobs. The correlated
// subquery for last_seen is dialect-portable (no GROUP BY).
func (s *bunStore) ListServerStates(ctx context.Context) ([]models.ServerState, error) {
	var states []models.ServerState
	err := s.db.NewSelect().
		Model((*serverRow)(nil)).
		ColumnExpr("id, name, location, report_interval, last_online_state, expire_date, expiration_notified").
		ColumnExpr("(SELECT COALESCE(MAX(mh.timestamp), 0) FROM metrics_history mh WHERE mh.server_id = srv.id) AS last_seen").
		Scan(ctx, &states)
	if err != nil {
		return nil, err
	}
	return states, nil
}

// latestMetric loads the newest metrics_history row for a server, returning
// (nil, 0, nil) when the server has never reported.
func (s *bunStore) latestMetric(ctx context.Context, serverID string) (*models.MetricsRow, int64, error) {
	var row metricRow
	err := s.db.NewSelect().Model(&row).
		Where("server_id = ?", serverID).
		OrderExpr("timestamp DESC").
		Limit(1).
		Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, 0, nil
	}
	if err != nil {
		return nil, 0, err
	}
	return metricRowToModel(&row), row.Timestamp, nil
}

// rangeBuckets maps a range key to (windowSeconds, bucketSeconds) per REQ-RES-03.
// Returns (0, 0) for an unknown range.
func rangeBuckets(rng string) (window, bucket int64) {
	switch rng {
	case "1h":
		return 3600, 30
	case "6h":
		return 21600, 60
	case "24h":
		return 86400, 300
	case "7d":
		return 604800, 1800
	case "30d":
		return 2592000, 7200
	case "180d":
		return 15552000, 43200
	default:
		return 0, 0
	}
}

// ── row -> model mappers ─────────────────────────────────────────────────────

func serverFromRow(r *serverRow, withSnapshots bool) models.Server {
	srv := models.Server{
		ServerConfig: models.ServerConfig{
			ID:              r.ID,
			Name:            r.Name,
			ServerGroup:     r.ServerGroup,
			Price:           r.Price,
			ExpireDate:      r.ExpireDate,
			Bandwidth:       r.Bandwidth,
			TrafficLimit:    r.TrafficLimit,
			TrafficCalcType: r.TrafficCalcType,
			ResetDay:        r.ResetDay,
			CollectInterval: r.CollectInterval,
			ReportInterval:  r.ReportInterval,
			PingMode:        r.PingMode,
			IsHidden:        r.IsHidden,
			SortOrder:       r.SortOrder,
			CreatedAt:       r.CreatedAt,
			UpdatedAt:       r.UpdatedAt,
		},
		Gid:      r.Gid,
		Alias:    r.Alias,
		Type:     r.Type,
		Location: r.Location,
		Notify:   r.Notify,
	}
	if withSnapshots {
		if r.SysInfoJSON != "" {
			var si models.SysInfo
			if json.Unmarshal([]byte(r.SysInfoJSON), &si) == nil {
				srv.SysInfo = &si
			}
		}
		if r.IpInfoJSON != "" {
			var ip models.IpInfo
			if json.Unmarshal([]byte(r.IpInfoJSON), &ip) == nil {
				srv.IpInfo = &ip
			}
		}
	}
	return srv
}

// metricRowToModel maps a persisted row back to the domain MetricsRow (the
// inverse of statReportToMetricRow). Fields line up 1:1.
func metricRowToModel(r *metricRow) *models.MetricsRow {
	return &models.MetricsRow{
		ID:             r.ID,
		ServerID:       r.ServerID,
		Timestamp:      r.Timestamp,
		Cpu:            r.Cpu,
		LoadAvg:        r.LoadAvg,
		Processes:      r.Processes,
		TcpConn:        r.TcpConn,
		UdpConn:        r.UdpConn,
		Thread:         r.Thread,
		CpuCores:       r.CpuCores,
		CpuInfo:        r.CpuInfo,
		CpuModel:       r.CpuModel,
		MemoryTotal:    r.MemoryTotal,
		MemoryUsed:     r.MemoryUsed,
		SwapTotal:      r.SwapTotal,
		SwapUsed:       r.SwapUsed,
		HddTotal:       r.HddTotal,
		HddUsed:        r.HddUsed,
		NetworkRx:      r.NetworkRx,
		NetworkTx:      r.NetworkTx,
		NetworkIn:      r.NetworkIn,
		NetworkOut:     r.NetworkOut,
		LastNetworkIn:  r.LastNetworkIn,
		LastNetworkOut: r.LastNetworkOut,
		PingCt:         r.PingCt,
		PingCu:         r.PingCu,
		PingCm:         r.PingCm,
		PingBd:         r.PingBd,
		LossCt:         r.LossCt,
		LossCu:         r.LossCu,
		LossCm:         r.LossCm,
		LossBd:         r.LossBd,
		Online4:        r.Online4,
		Online6:        r.Online6,
		Os:             r.Os,
		OsRelease:      r.OsRelease,
		KernelVersion:  r.KernelVersion,
		Arch:           r.Arch,
		OsFamily:       r.OsFamily,
		Uptime:         r.Uptime,
		HostName:       r.HostName,
		Gpu:            r.Gpu,
		GpuInfo:        r.GpuInfo,
		Region:         r.Region,
		Gid:            r.Gid,
		Location:       r.Location,
		Vnstat:         r.Vnstat,
		Custom:         r.Custom,
		DisksJSON:      r.DisksJSON,
	}
}
