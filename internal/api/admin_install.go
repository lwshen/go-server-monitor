package api

import "github.com/gin-gonic/gin"

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
// REQ-RES-09). Dangerous: keeps servers/settings, clears time-series.
//
// P0 STUB: 501.
//
// TODO(P6): require a second-confirmation parameter, then TRUNCATE/recreate
// metrics_history while preserving servers and settings.
func (h *Handlers) AdminDBRebuild(c *gin.Context) {
	notImplemented(c)
}
