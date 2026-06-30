package service

import (
	"database/sql"

	"github.com/lwshen/go-server-monitor/internal/models"
	"github.com/lwshen/go-server-monitor/internal/ws"
	"github.com/lwshen/go-server-monitor/pkg/apperr"
	"go.uber.org/zap"
)

// SaveMetrics persists a probe report and broadcasts the realtime update
// (REQ-API-06 / REQ-WS-07).
//
// P0 STUB: logs "not implemented" and returns ErrNotImplemented.
//
// TODO(P2): in a transaction, upsert the servers row and INSERT every sample
// into metrics_history (each sample its own row, REQ-RES-04); store disks[] as
// disks_json (REQ-RES-02); map -1 sentinels to NULL.
// TODO(P4): after a successful write, build a BroadcastData with static fields
// removed (models.BroadcastDeleteFields, REQ-RES-06) and call hub.Broadcast.
func SaveMetrics(db *sql.DB, hub *ws.Hub, report *models.StatReport, log *zap.Logger) error {
	log.Warn("service.SaveMetrics not implemented (P2)")
	return apperr.ErrNotImplemented
}
