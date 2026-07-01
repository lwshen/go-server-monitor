package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/lwshen/go-server-monitor/internal/models"
)

// Admin server-management endpoints (JWT-guarded, REQ-ADMIN-06 / REQ-RES-00):
//
//	POST /api/admin/servers          — list servers (admin view)
//	POST /api/admin/servers/add      — create (generates a UUID)
//	POST /api/admin/servers/edit     — update fields
//	POST /api/admin/servers/delete   — delete (cascades metrics_history)
//	POST /api/admin/servers/reorder  — update sort_order
//
// NOTE: the admin group is JWT-guarded (middleware.JWTAuth), but with no
// JWT_SECRET configured the guard is pass-through in the skeleton — fine for local
// dev. TODO(P6): enforce auth + implement the remaining CRUD (edit/delete/reorder),
// with delete cascading to metrics_history.

// AdminServersAdd registers a new server and returns it (with a fresh UUID). This
// is brought forward from P6 so that /report — which now 404s on unknown ids —
// has a way to register servers.
func (h *Handlers) AdminServersAdd(c *gin.Context) {
	var body struct {
		Name        string `json:"name"`
		ServerGroup string `json:"server_group"`
		ExpireDate  string `json:"expire_date"`
		Notify      bool   `json:"notify"`
	}
	// Body is optional; an empty POST creates a default server.
	_ = c.ShouldBindJSON(&body)

	cfg := &models.ServerConfig{
		ID:          uuid.NewString(),
		Name:        body.Name,
		ServerGroup: body.ServerGroup,
		ExpireDate:  body.ExpireDate,
	}
	if err := h.deps.Store.CreateServer(c.Request.Context(), cfg); err != nil {
		ErrorFrom(c, err)
		return
	}
	JSON(c, http.StatusOK, cfg)
}

func (h *Handlers) AdminServers(c *gin.Context)        { notImplemented(c) }
func (h *Handlers) AdminServersEdit(c *gin.Context)    { notImplemented(c) }
func (h *Handlers) AdminServersDelete(c *gin.Context)  { notImplemented(c) }
func (h *Handlers) AdminServersReorder(c *gin.Context) { notImplemented(c) }
