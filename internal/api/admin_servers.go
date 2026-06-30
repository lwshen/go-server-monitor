package api

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// newServerID generates a fresh server UUID (REQ-ADMIN-06). Wired now so the
// uuid dependency is in place; used by AdminServersAdd in P6.
func newServerID() string {
	return uuid.NewString()
}

var _ = newServerID // referenced by P6 AdminServersAdd; keep the wiring live

// Admin server-management endpoints (JWT-guarded, REQ-ADMIN-06 / REQ-RES-00):
//
//	POST /api/admin/servers          — list servers (admin view)
//	POST /api/admin/servers/add      — create (generates a UUID)
//	POST /api/admin/servers/edit     — update fields
//	POST /api/admin/servers/delete   — delete (cascades metrics_history)
//	POST /api/admin/servers/reorder  — update sort_order
//
// All P0 STUBs: 501.
//
// TODO(P6): implement CRUD against the servers table; use github.com/google/uuid
// for new ids; delete cascades to metrics_history.

func (h *Handlers) AdminServers(c *gin.Context)        { notImplemented(c) }
func (h *Handlers) AdminServersAdd(c *gin.Context)     { notImplemented(c) }
func (h *Handlers) AdminServersEdit(c *gin.Context)    { notImplemented(c) }
func (h *Handlers) AdminServersDelete(c *gin.Context)  { notImplemented(c) }
func (h *Handlers) AdminServersReorder(c *gin.Context) { notImplemented(c) }
