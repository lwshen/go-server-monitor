package api

import "github.com/gin-gonic/gin"

// History returns downsampled history (GET /api/history?id=<id>&range=<r>).
// range is one of 1h/6h/24h/7d/30d/180d.
//
// P0 STUB: 501.
//
// TODO(P2): map range -> bucket size (REQ-RES-03: 1h=30s, 6h=1m, 24h=5m, 7d=30m,
// 30d=2h, 180d=12h), GROUP BY timestamp/bucket with AVG over metrics, ignoring
// -1/NULL for ping_*/loss_*; cache per (id,range) for one bucket period.
func (h *Handlers) History(c *gin.Context) {
	notImplemented(c)
}
