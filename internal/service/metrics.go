package service

import (
	"context"

	"github.com/lwshen/go-server-monitor/internal/models"
	"github.com/lwshen/go-server-monitor/internal/store"
	"github.com/lwshen/go-server-monitor/internal/ws"
	"go.uber.org/zap"
)

// Ingest persists a probe report and (P4) broadcasts the realtime update. It is
// the single orchestration point between the store write and the WebSocket hub
// (REQ-API-06 / REQ-WS-07). Returns the number of metric rows written.
//
// TODO(P4): after a successful write, build a BroadcastData with static fields
// removed (models.BroadcastDeleteFields, REQ-RES-06) and call hub.Broadcast —
// a single `update` frame for one sample, else a `batchUpdate` frame (REQ-RES-04).
func Ingest(ctx context.Context, st store.Store, hub *ws.Hub, req *models.ReportRequest, log *zap.Logger) (int, error) {
	n, err := st.SaveReport(ctx, req)
	if err != nil {
		return 0, err
	}
	log.Info("上报已入库", zap.String("server_id", req.ID), zap.Int("samples", n))
	_ = hub // TODO(P4): broadcast the dynamic metrics to subscribers
	return n, nil
}
