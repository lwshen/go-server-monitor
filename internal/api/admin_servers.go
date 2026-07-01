package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/lwshen/go-server-monitor/internal/models"
)

// Admin server-management endpoints (JWT-guarded, REQ-ADMIN-06 / REQ-RES-00):
//
//	POST /api/admin/servers          — list servers (admin view, incl. hidden)
//	POST /api/admin/servers/add      — create (generates a UUID)
//	POST /api/admin/servers/edit     — update admin-editable fields (partial)
//	POST /api/admin/servers/delete   — delete (also removes metrics_history)
//	POST /api/admin/servers/reorder  — set sort_order by id order
//
// The admin group is JWT-guarded (middleware.JWTAuth); with no JWT_SECRET the guard
// is pass-through (local dev). TODO(P8): always require auth.

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

// AdminServers lists every server (including hidden), with latest metrics + online.
func (h *Handlers) AdminServers(c *gin.Context) {
	servers, err := h.deps.Store.ListServers(c.Request.Context())
	if err != nil {
		ErrorFrom(c, err)
		return
	}
	now := time.Now().Unix()
	for i := range servers {
		servers[i].Online = h.isOnline(servers[i].LastUpdated, servers[i].ReportInterval, now)
	}
	JSON(c, http.StatusOK, gin.H{"servers": servers})
}

// AdminServersEdit applies a partial update: the current config is loaded, the
// provided JSON fields overlay it, then it is persisted. Body must include "id".
func (h *Handlers) AdminServersEdit(c *gin.Context) {
	var idOnly struct {
		ID string `json:"id"`
	}
	raw, err := c.GetRawData()
	if err != nil {
		Error(c, http.StatusBadRequest, "invalid payload")
		return
	}
	if err := bindJSON(raw, &idOnly); err != nil || idOnly.ID == "" {
		Error(c, http.StatusBadRequest, "missing id")
		return
	}

	ctx := c.Request.Context()
	current, err := h.deps.Store.GetServer(ctx, idOnly.ID)
	if err != nil {
		ErrorFrom(c, err)
		return
	}
	if current == nil {
		Error(c, http.StatusNotFound, "server not found")
		return
	}

	// Overlay the provided fields onto the existing config (partial update).
	cfg := current.ServerConfig
	if err := bindJSON(raw, &cfg); err != nil {
		Error(c, http.StatusBadRequest, "invalid payload")
		return
	}
	cfg.ID = idOnly.ID
	if err := h.deps.Store.UpdateServer(ctx, &cfg); err != nil {
		ErrorFrom(c, err)
		return
	}
	JSON(c, http.StatusOK, gin.H{"success": true, "server": cfg})
}

// AdminServersDelete deletes a server and its metrics. Body: {"id": "..."}.
func (h *Handlers) AdminServersDelete(c *gin.Context) {
	var body struct {
		ID string `json:"id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.ID == "" {
		Error(c, http.StatusBadRequest, "missing id")
		return
	}
	if err := h.deps.Store.DeleteServer(c.Request.Context(), body.ID); err != nil {
		ErrorFrom(c, err)
		return
	}
	JSON(c, http.StatusOK, gin.H{"success": true})
}

// AdminServersReorder sets sort_order from the given id order. Body: {"ids": [...]}.
func (h *Handlers) AdminServersReorder(c *gin.Context) {
	var body struct {
		IDs []string `json:"ids"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || len(body.IDs) == 0 {
		Error(c, http.StatusBadRequest, "missing ids")
		return
	}
	if err := h.deps.Store.ReorderServers(c.Request.Context(), body.IDs); err != nil {
		ErrorFrom(c, err)
		return
	}
	JSON(c, http.StatusOK, gin.H{"success": true})
}
