package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// AdminInstallCommand generates a probe install command (REQ-ADMIN-08).
//
// NOTE: this endpoint is NOT in the frozen route table (REQ-RES-00), so the
// handler is defined here but intentionally NOT registered in router.go yet. It
// is kept so P6 has a home for the install-command feature.
//
// P0 STUB: 501.
//
// TODO(P6): decide the final route (e.g. POST /api/admin/install-command), wire
// it into the admin group, and generate the curl/.bat snippet from {server_id,os}.
func (h *Handlers) AdminInstallCommand(c *gin.Context) {
	notImplemented(c)
}

// AdminDBRebuild rebuilds metrics_history only (POST /api/admin/db/rebuild, JWT;
// REQ-RES-09). Dangerous — clears the time-series but keeps servers/settings — so
// it requires an explicit {"confirm": true} in the body.
func (h *Handlers) AdminDBRebuild(c *gin.Context) {
	var body struct {
		Confirm bool `json:"confirm"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || !body.Confirm {
		Error(c, http.StatusBadRequest, `refusing to rebuild without {"confirm": true}`)
		return
	}
	if err := h.deps.Store.RebuildMetrics(c.Request.Context()); err != nil {
		ErrorFrom(c, err)
		return
	}
	JSON(c, http.StatusOK, gin.H{"success": true})
}
