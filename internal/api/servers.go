package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/lwshen/go-server-monitor/internal/models"
)

// Servers returns the server list with latest metrics + global stats
// (GET /api/servers). Hidden servers (is_hidden="1") are filtered out for public
// viewers (REQ-RES-00); online status is derived from the offline threshold.
func (h *Handlers) Servers(c *gin.Context) {
	all, err := h.deps.Store.ListServers(c.Request.Context())
	if err != nil {
		ErrorFrom(c, err)
		return
	}

	now := time.Now().Unix()
	visible := make([]models.Server, 0, len(all))
	online := 0
	for i := range all {
		s := all[i]
		if s.IsHidden == "1" {
			continue // TODO(P6): show hidden servers to authenticated admins
		}
		s.Online = h.isOnline(s.LastUpdated, s.ReportInterval, now)
		if s.Online {
			online++
		}
		visible = append(visible, s)
	}

	JSON(c, http.StatusOK, gin.H{
		"servers": visible,
		"stats": gin.H{
			"total":   len(visible),
			"online":  online,
			"offline": len(visible) - online,
		},
	})
}

// isOnline reports whether a server counts as online: its newest sample is within
// offline_factor × report_interval of now (REQ-CRON-01 threshold, defaults 5×60s).
func (h *Handlers) isOnline(lastUpdated int64, reportInterval int, now int64) bool {
	if lastUpdated <= 0 {
		return false
	}
	ri := reportInterval
	if ri <= 0 {
		ri = 60
	}
	factor := h.deps.Cfg.OfflineFactor
	if factor <= 0 {
		factor = 5
	}
	return now-lastUpdated <= int64(ri*factor)
}
