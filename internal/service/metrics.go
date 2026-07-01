package service

import (
	"context"
	"time"

	"github.com/lwshen/go-server-monitor/internal/models"
	"github.com/lwshen/go-server-monitor/internal/store"
	"github.com/lwshen/go-server-monitor/internal/ws"
	"go.uber.org/zap"
)

// Ingest persists a probe report and broadcasts the realtime update. It is the
// single orchestration point between the store write and the WebSocket hub
// (REQ-API-06 / REQ-WS-07). Returns the number of metric rows written.
func Ingest(ctx context.Context, st store.Store, hub *ws.Hub, req *models.ReportRequest, log *zap.Logger) (int, error) {
	n, err := st.SaveReport(ctx, req)
	if err != nil {
		return 0, err
	}
	log.Info("上报已入库", zap.String("server_id", req.ID), zap.Int("samples", n))

	if hub != nil {
		if bd := broadcastData(req); bd != nil {
			hub.Broadcast(bd)
		}
	}
	return n, nil
}

// broadcastData builds the realtime frame from a report: dynamic metrics only
// (REQ-RES-06), one "update" for a single sample or a "batchUpdate" for several
// (REQ-RES-04). Scope is the server id so the Hub delivers to that server's
// subscribers and to "all". Returns nil when there is nothing to broadcast.
func broadcastData(req *models.ReportRequest) *models.BroadcastData {
	bd := &models.BroadcastData{ServerID: req.ID, Scope: req.ID}

	if len(req.Samples) > 0 {
		for _, s := range req.Samples {
			if s.Data == nil {
				continue
			}
			ts := normalizeTs(s.Timestamp, s.Data)
			bd.Samples = append(bd.Samples, models.BatchSample{Ts: ts, Data: models.DynamicData(s.Data)})
			if ts > bd.Ts {
				bd.Ts = ts
			}
		}
		switch len(bd.Samples) {
		case 0:
			return nil
		case 1: // collapse a single-sample batch to an "update"
			bd.Type = "update"
			bd.Data = bd.Samples[0].Data
			bd.Ts = bd.Samples[0].Ts
			bd.Samples = nil
		default:
			bd.Type = "batchUpdate"
		}
		return bd
	}

	if req.Data == nil {
		return nil
	}
	bd.Type = "update"
	bd.Ts = normalizeTs(req.Timestamp, req.Data)
	bd.Data = models.DynamicData(req.Data)
	return bd
}

// normalizeTs mirrors the store's rule: prefer the explicit ts, else the payload's
// latest_ts, else now; convert milliseconds to seconds.
func normalizeTs(ts int64, sr *models.StatReport) int64 {
	cand := ts
	if cand <= 0 && sr != nil {
		cand = sr.LatestTs
	}
	if cand <= 0 {
		return time.Now().Unix()
	}
	if cand > 1e10 {
		return cand / 1000
	}
	return cand
}
