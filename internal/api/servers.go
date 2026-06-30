package api

import "github.com/gin-gonic/gin"

// Servers returns the server list with latest stats (GET /api/servers). Public,
// but hidden servers are filtered for anonymous viewers (REQ-RES-00).
//
// P0 STUB: 501.
//
// TODO(P2): query servers joined with their latest metrics_history row; filter
// is_hidden for unauthenticated requests; return []models.Server.
func (h *Handlers) Servers(c *gin.Context) {
	notImplemented(c)
}
