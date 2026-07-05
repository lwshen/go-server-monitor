package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/lwshen/go-server-monitor/internal/models"
	"github.com/lwshen/go-server-monitor/internal/store"
)

// Servers returns the server list with latest metrics + global stats
// (GET /api/servers). Hidden servers (is_hidden="1") are filtered out for public
// viewers (REQ-RES-00); online status is derived from the offline threshold.
func (h *Handlers) Servers(c *gin.Context) {
	ctx := c.Request.Context()
	all, err := h.deps.Store.ListServers(ctx)
	if err != nil {
		ErrorFrom(c, err)
		return
	}

	now := time.Now().Unix()
	factor := h.offlineFactor(ctx)
	visible := make([]models.Server, 0, len(all))
	online := 0
	for i := range all {
		s := all[i]
		if s.IsHidden == "1" {
			continue // TODO(P6): show hidden servers to authenticated admins
		}
		s.Online = isOnline(s.LastUpdated, s.ReportInterval, factor, now)
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

// offlineFactor reads the offline threshold factor from settings (admin-editable,
// env-seeded), falling back to the config default.
func (h *Handlers) offlineFactor(ctx context.Context) int {
	return store.IntSetting(ctx, h.deps.Store, models.SettingOfflineFactor, h.deps.Cfg.OfflineFactor)
}

// isOnline reports whether a server counts as online: its newest sample is within
// factor × report_interval of now (REQ-CRON-01 threshold, defaults 5×60s).
func isOnline(lastUpdated int64, reportInterval, factor int, now int64) bool {
	if lastUpdated <= 0 {
		return false
	}
	if reportInterval <= 0 {
		reportInterval = 60
	}
	if factor <= 0 {
		factor = 5
	}
	return now-lastUpdated <= int64(reportInterval*factor)
}
